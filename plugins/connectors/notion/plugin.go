// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package notion

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/api"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"time"
)

type Config struct {
	Token string `config:"token"`
}

type Plugin struct {
	api.Handler
	Enabled  bool               `config:"enabled"`
	Queue    *queue.QueueConfig `config:"queue"`
	Interval string             `config:"interval"`
	PageSize int                `config:"page_size"`
}

func (this *Plugin) Setup() {
	ok, err := env.ParseConfig("connector.notion", &this)
	if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
		panic(err)
	}

	if !this.Enabled {
		return
	}

	if this.PageSize <= 0 {
		this.PageSize = 1000
	}

	if this.Queue == nil {
		this.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}

	this.Queue = queue.SmartGetOrInitConfig(this.Queue)
}

func (this *Plugin) Start() error {

	if this.Enabled {
		task.RegisterScheduleTask(task.ScheduleTask{
			ID:          util.GetUUID(),
			Group:       "connectors",
			Singleton:   true,
			Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(), //connector's task interval
			Description: "indexing notion docs",
			Task: func(ctx context.Context) {
				connector := common.Connector{}
				connector.ID = "notion"
				exists, err := orm.Get(&connector)
				if !exists || err != nil {
					panic("invalid hugo_site connector")
				}

				q := orm.Query{}
				q.Size = this.PageSize
				q.Conds = orm.And(orm.Eq("connector.id", connector.ID), orm.Eq("sync_enabled", true))
				var results []common.DataSource

				err, _ = orm.SearchWithJSONMapper(&results, &q)
				if err != nil {
					panic(err)
				}

				for _, item := range results {
					toSync, err := connectors.CanDoSync(item)
					if err != nil {
						_ = log.Errorf("error checking syncable with datasource [%s]: %v", item.Name, err)
						continue
					}
					if !toSync {
						continue
					}
					log.Debugf("ID: %s, Name: %s, Other: %s", item.ID, item.Name, util.MustToJSON(item))
					this.fetch_notion(&connector, &item)
				}
			},
		})
	}

	return nil
}

func (this *Plugin) fetch_notion(connector *common.Connector, datasource *common.DataSource) {

	if connector == nil || datasource == nil {
		panic("invalid connector config")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		panic(err)
	}

	obj := Config{}
	err = cfg.Unpack(&obj)
	if err != nil {
		panic(err)
	}

	log.Debugf("handle notion's datasource: %v", obj)
	// Define the callback function to handle each page of results
	handlePage := func(result *SearchResult) {
		// Process the current page's results here
		for i, res := range result.Results {
			// Process individual result
			doc := common.Document{Source: common.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name}}

			doc.Created = &res.Created
			doc.Created = &res.Updated

			doc.Type = res.Object
			doc.Icon = res.Object
			doc.Title = extractTitle(&res)
			//doc.Content = v.Content
			//doc.Category = v.Category
			//doc.Subcategory = v.Subcategory
			//doc.Summary = v.Summary
			//doc.Tags = v.Tags
			doc.Payload = res.Properties
			doc.URL = res.Url

			log.Debugf("save document: %d: %+v %v", i+1, doc.Title, doc.URL)

			doc.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", connector.ID, datasource.ID, doc.URL))

			data := util.MustToJSONBytes(doc)

			if global.Env().IsDebug {
				log.Tracef(string(data))
			}

			err := queue.Push(queue.SmartGetOrInitConfig(this.Queue), data)
			if err != nil {
				panic(err)
			}
		}

		log.Info("fetched ", len(result.Results), " notion results")
	}
	search(obj.Token, "", handlePage)

}

// SearchResult represents the response from the Notion Search API
type SearchResult struct {
	Object         string       `json:"object"`
	Results        []SearchItem `json:"results"`
	NextCursor     string       `json:"next_cursor"`
	HasMore        bool         `json:"has_more"`
	Type           string       `json:"type"`
	PageOrDatabase interface{}  `json:"page_or_database"`
}

// SearchItem represents an individual item in the search results
type SearchItem struct {
	Object     string                 `json:"object"`
	ID         string                 `json:"id"`
	Created    time.Time              `json:"created_time"`
	Updated    time.Time              `json:"last_edited_time"`
	Archived   bool                   `json:"archived"`
	Properties map[string]interface{} `json:"properties"`

	//database
	Title []TitleItem `json:"title"`

	Parent       ParentInfo  `json:"parent"`
	Cover        CoverImage  `json:"cover"`
	Icon         interface{} `json:"icon"`
	CreatedBy    interface{} `json:"created_by"`
	LastEditedBy interface{} `json:"last_edited_by"`
	Url          string      `json:"url"`
}

// Extract the title from any property in SearchItem
func extractTitle(item *SearchItem) string {
	if item != nil {

		if len(item.Title) > 0 {
			for _, v := range item.Title {
				if v.PlainText != "" {
					return v.PlainText
				}
			}
		}

		// Iterate over all properties in the properties map
		for _, value := range item.Properties {
			// Type assert value to a map to check for a title property
			if propMap, ok := value.(map[string]interface{}); ok {
				// Check if the "type" is "title"
				if propType, ok := propMap["type"].(string); ok && propType == "title" {
					// Extract the title array from the property map
					if titleArray, ok := propMap["title"].([]interface{}); ok {
						// Loop through each title item in the array
						for _, titleItem := range titleArray {
							// Type assert titleItem to a map
							if titleMap, ok := titleItem.(map[string]interface{}); ok {
								// Extract the "plain_text" field from the title item
								if text, ok := titleMap["plain_text"].(string); ok {
									// Log or return the key and its title
									return text
								}
							}
						}
					}
				}
			}
		}
	}
	return ""
}

// Property represents the various property types in a Notion page (e.g., title, rich_text, etc.)
type Property struct {
	ID    string      `json:"id"`
	Type  string      `json:"type"`
	Title []TitleItem `json:"title"`
}

// TitleItem represents each item in a title property
type TitleItem struct {
	Type        string      `json:"type"`
	Text        TextContent `json:"text"`
	Annotations Annotations `json:"annotations"`
	PlainText   string      `json:"plain_text"`
	Href        string      `json:"href"`
}

// TextContent holds the text content of the title item
type TextContent struct {
	Content string `json:"content"`
	Link    *Link  `json:"link"`
}

// Annotations stores styling information for the text (e.g., bold, italic)
type Annotations struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color"`
}

// Link represents a hyperlink inside the text
type Link struct {
	URL string `json:"url"`
}

// ParentInfo contains information about the parent of a page or database
type ParentInfo struct {
	Type       string `json:"type"`
	DatabaseID string `json:"database_id,omitempty"`
	PageID     string `json:"page_id,omitempty"`
}

// CoverImage represents the cover image for a page
type CoverImage struct {
	Type     string `json:"type"`
	External struct {
		URL string `json:"url"`
	} `json:"external"`
}

// SearchOptions allows for filtering and customizing search queries
type SearchOptions struct {
	Query       string `json:"query,omitempty"`
	Filter      Filter `json:"filter,omitempty"`
	Sort        Sort   `json:"sort,omitempty"`
	StartCursor string `json:"start_cursor,omitempty"`
	PageSize    int    `json:"page_size,omitempty"`
}

// Filter represents the filter options used in a search query
type Filter struct {
	Value    string `json:"value,omitempty"`
	Property string `json:"property,omitempty"`
}

// Sort defines how the results should be ordered
type Sort struct {
	Property  string `json:"property,omitempty"`
	Direction string `json:"direction,omitempty"`
}

func search(token, nextCursor string, handlePage func(result *SearchResult)) {
	// Prepare the request payload based on the next_cursor
	var requestBody map[string]interface{}
	if nextCursor != "" {
		requestBody = map[string]interface{}{
			"start_cursor": nextCursor,
		}
	} else {
		requestBody = map[string]interface{}{}
	}

	// Prepare the request
	req := util.NewPostRequest("https://api.notion.com/v1/search", util.MustToJSONBytes(requestBody))
	req.AddHeader("Authorization", fmt.Sprintf("Bearer %v", token))
	req.AddHeader("Notion-Version", "2022-06-28")
	req.AddHeader("Content-Type", "application/json")

	// Execute the request
	res, err := util.ExecuteRequest(req)
	if err != nil {
		panic(err)
	}

	// Handle non-successful status codes
	if res != nil {
		if res.StatusCode > 300 {
			panic(errors.Errorf("%v,%v", res.StatusCode, string(res.Body)))
		}
	}

	// Parse the response into a Result struct
	var result SearchResult
	err = json.Unmarshal(res.Body, &result)
	if err != nil {
		panic(errors.Errorf("Error parsing response: %v", err))
	}

	// Process the current page of results
	handlePage(&result)

	// If there's a next_cursor, fetch the next page
	if result.NextCursor != "" {
		// Recursively call search to get more results
		search(token, result.NextCursor, handlePage)
	}
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return "notion"
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
