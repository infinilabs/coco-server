/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package notion

import (
	"encoding/json"
	"fmt"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/pipeline"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
)

type Config struct {
	Token string `config:"token"`
}

type Plugin struct {
	cmn.ConnectorProcessorBase
}

const Name = "notion"

func init() {
	pipeline.RegisterProcessorPlugin(Name, New)
}

func New(c *config3.Config) (pipeline.Processor, error) {
	runner := Plugin{}
	if err := c.Unpack(&runner); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of processor %v, error: %s", Name, err)
	}

	runner.Init(c, &runner)
	return &runner, nil
}

func (processor *Plugin) Name() string {
	return Name
}

func (this *Plugin) Fetch(pipeCtx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
	cfg := Config{}
	this.MustParseConfig(datasource, &cfg)

	log.Debugf("handle notion's datasource: %v", cfg)
	// Define the callback function to handle each page of results
	handlePage := func(result *SearchResult) {
		// Process the current page's results here
		for i, res := range result.Results {
			// Process individual result
			doc := common.Document{Source: common.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name}}

			doc.Created = &res.Created
			doc.Created = &res.Updated
			doc.System = datasource.System

			doc.Type = res.Object
			doc.Icon = res.Object
			doc.Title = extractTitle(&res)

			// Fetch and set content for pages
			if res.Object == "page" {
				content, err := fetchPageContent(cfg.Token, res.ID)
				if err != nil {
					_ = log.Warnf("Failed to fetch content for Notion page %s: %v", res.ID, err)
				} else {
					doc.Content = content
				}
			}

			//doc.Category = v.Category
			//doc.Subcategory = v.Subcategory
			//doc.Summary = v.Summary
			//doc.Tags = v.Tags
			doc.Payload = res.Properties
			doc.URL = res.Url

			log.Debugf("save document: %d: %+v %v", i+1, doc.Title, doc.URL)

			doc.ID = util.MD5digest(fmt.Sprintf("%v-%v-%v", connector.ID, datasource.ID, doc.URL))

			this.Collect(pipeCtx, connector, datasource, doc)
		}

		log.Info("fetched ", len(result.Results), " notion results")
	}

	search(cfg.Token, "", handlePage)

	return nil
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

	// Execute the request
	res, err := executeNotionRequest(req, token)
	if err != nil {
		panic(err)
	}

	var result SearchResult
	err = json.Unmarshal(res, &result)
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

func fetchPageContent(token, blockID string) (string, error) {
	var contentBuilder strings.Builder
	var nextCursor string
	for {
		// Prepare the request
		endpoint := fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children", blockID)
		if nextCursor != "" {
			endpoint = fmt.Sprintf("%s?start_cursor=%s", endpoint, nextCursor)
		}

		req := util.NewGetRequest(endpoint, nil)

		res, err := executeNotionRequest(req, token)
		if err != nil {
			return "", err
		}

		var blockChild BlockChildrenResponse
		err = json.Unmarshal(res, &blockChild)
		if err != nil {
			panic(errors.Errorf("Error parsing response: %v", err))
		}

		// Extract text from blocks
		for _, block := range blockChild.Results {
			extractTextFromBlock(&block, &contentBuilder)
		}

		if blockChild.HasMore && blockChild.NextCursor != "" {
			nextCursor = blockChild.NextCursor
		} else {
			break
		}

	}
	return strings.TrimSpace(contentBuilder.String()), nil
}

func extractTextFromBlock(block *Block, builder *strings.Builder) {
	blockType := block.Type()
	if blockType == "" {
		return
	}

	richText := block.GetRichTextSliceBy(blockType)

	for _, rt := range richText {
		builder.WriteString(rt.PlainText)
	}
	// Add a newline after each block to separate content
	builder.WriteString("\n")
}

// executeNotionRequest is a helper function to execute requests to the Notion API.
// It sets the required headers, executes the request, checks for errors.
func executeNotionRequest(req *util.Request, token string) ([]byte, error) {
	req.AddHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	req.AddHeader("Notion-Version", "2022-06-28")
	req.AddHeader("Content-Type", "application/json")

	res, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.Errorf("Notion API error, no response")
	}

	if res.StatusCode >= 300 {
		return nil, errors.Errorf("Notion API error: status %d, body: %s", res.StatusCode, string(res.Body))
	}

	return res.Body, nil
}
