/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package hugo_site

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
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
	"net/url"
	"strings"
	"time"
)

type Config struct {
	Interval string   `config:"interval"`
	Urls     []string `config:"urls"`
}

type Plugin struct {
	api.Handler
	Enabled  bool               `config:"enabled"`
	Queue    *queue.QueueConfig `config:"queue"`
	Interval string             `config:"interval"`
	PageSize int                `config:"page_size"`
}

func (this *Plugin) Setup() {
	ok, err := env.ParseConfig("connector.hugo_site", &this)
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

// ParseTimestamp safely parses a timestamp string into a *time.Time.
// Returns nil if parsing fails.
func ParseTimestamp(timestamp string) *time.Time {
	layout := time.RFC3339 // ISO 8601 format
	parsedTime, err := time.Parse(layout, timestamp)
	if err != nil {
		return nil
	}
	return &parsedTime
}

func (this *Plugin) Start() error {

	if this.Enabled {
		task.RegisterScheduleTask(task.ScheduleTask{
			ID:          util.GetUUID(),
			Group:       "connectors",
			Singleton:   true,
			Interval:    util.GetDurationOrDefault(this.Interval, time.Second*30).String(), //connector's task interval
			Description: "indexing hugo json docs",
			Task: func(ctx context.Context) {
				connector := common.Connector{}
				connector.ID = "hugo_site"
				exists, err := orm.Get(&connector)
				if !exists || err != nil {
					panic("invalid hugo_site connector")
				}

				q := orm.Query{}
				q.Size = this.PageSize
				q.Conds = orm.And(orm.Eq("connector.id", connector.ID))
				var results []common.DataSource

				err, _ = orm.SearchWithJSONMapper(&results, &q)
				if err != nil {
					panic(err)
				}

				for _, item := range results {
					log.Debugf("ID: %s, Name: %s, Other: %s", item.ID, item.Name, util.MustToJSON(item))
					this.fetch_site(&connector, &item)
				}
			},
		})
	}

	return nil
}

func (this *Plugin) fetch_site(connector *common.Connector, datasource *common.DataSource) {

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

	log.Debugf("handle hugo_site's datasource: %v", obj)

	for _, myURL := range obj.Urls {

		if global.ShuttingDown() {
			break
		}

		log.Debugf("connect to hugo site: %v", myURL)

		res, err := util.HttpGet(myURL)
		if err != nil {
			panic(err)
		}

		if res.Body != nil {
			var documents []HugoDocument

			// Unmarshal JSON into the slice
			err := util.FromJSONBytes(res.Body, &documents)
			if err != nil {
				panic(errors.Errorf("Failed to parse JSON: %v", err))
			}

			// Output the parsed data
			for i, v := range documents {

				if global.ShuttingDown() {
					break
				}

				doc := common.Document{Source: common.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name}}

				if v.Created != "" {
					doc.Created = ParseTimestamp(v.Created)
				}

				if v.Updated != "" {
					doc.Created = ParseTimestamp(v.Updated)
				}

				doc.Type = "web_page"
				doc.Icon = "web"
				doc.Title = v.Title
				doc.Lang = v.Lang
				doc.Content = v.Content
				doc.Category = v.Category
				doc.Subcategory = v.Subcategory
				doc.Summary = v.Summary
				doc.Tags = v.Tags
				v2, er := getFullURL(myURL, v.URL)
				if er != nil {
					panic(er)
				}
				doc.URL = v2
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

			log.Infof("fetched %v docs from hugo site: %v", len(documents), myURL)
		}
	}

}

// Function to construct the full URL using only the domain from the seed URL
func getFullURL(seedURL, relativePath string) (string, error) {
	// Parse the seed URL
	parsedURL, err := url.Parse(seedURL)
	if err != nil {
		return "", fmt.Errorf("invalid seed URL: %w", err)
	}

	// Extract the domain (scheme and host)
	domain := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Remove any leading "/" from relativePath to avoid duplication
	relativePath = strings.TrimPrefix(relativePath, "/")

	// Combine the domain with the relative path
	fullURL := fmt.Sprintf("%s/%s", domain, relativePath)

	return fullURL, nil
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return "hugo_site"
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
