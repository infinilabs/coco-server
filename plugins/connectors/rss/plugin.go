/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rss

import (
	"context"
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mmcdole/gofeed"
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
	ok, err := env.ParseConfig("connector.rss", &this)
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
	if !this.Enabled {
		return nil
	}

	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          util.GetUUID(),
		Group:       "connectors",
		Singleton:   true,
		Interval:    util.GetDurationOrDefault(this.Interval, time.Minute*5).String(),
		Description: "indexing rss feeds",
		Task: func(ctx context.Context) {
			connector := common.Connector{}
			connector.ID = "rss"
			exists, err := orm.Get(&connector)
			if !exists || err != nil {
				panic(errors.Errorf("invalid %s connector:%v", connector.ID, err))
			}

			q := orm.Query{}
			q.Size = this.PageSize
			q.Conds = orm.And(orm.Eq("connector.id", connector.ID), orm.Eq("sync_enabled", true))
			var results []common.DataSource

			err, _ = orm.SearchWithJSONMapper(&results, &q)
			if err != nil {
				log.Errorf("Failed to search for RSS data source: %v", err)
				panic(err)
			}

			for _, item := range results {
				toSync, err := connectors.CanDoSync(item)
				if err != nil {
					log.Errorf("error checking sync status with data source [%s]: %v", item.Name, err)
					continue
				}
				if !toSync {
					continue
				}
				log.Debugf("ID: %s, Name: %s, Other: %s", item.ID, item.Name, util.MustToJSON(item))
				this.fetchRssFeed(&connector, &item)
			}
		},
	})

	return nil
}

// fetchRssFeed handles the logic of fetching, parsing, and indexing a single RSS feed.
func (this *Plugin) fetchRssFeed(connector *common.Connector, datasource *common.DataSource) {
	if connector == nil || datasource == nil {
		panic("invalid connector or datasource config")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		log.Errorf("Failed to create config from datasource [%s]: %v", datasource.Name, err)
		panic(err)
	}

	obj := Config{}
	err = cfg.Unpack(&obj)
	if err != nil {
		log.Errorf("Failed to unpack config for datasource [%s]: %v", datasource.Name, err)
		panic(err)
	}

	log.Debugf("Handling RSS datasource: %v", obj)

	fp := gofeed.NewParser()

	for _, theUrl := range obj.Urls {
		if global.ShuttingDown() {
			break
		}

		log.Debugf("Connecting to RSS feed: %v", theUrl)

		feed, err := fp.ParseURL(theUrl)
		if err != nil {
			log.Errorf("Failed to parse RSS feed from URL [%s]: %v", theUrl, err)
			continue
			// panic(err)
		}

		for _, item := range feed.Items {
			if global.ShuttingDown() {
				break
			}

			doc := common.Document{
				Source: common.DataSourceReference{
					ID:   datasource.ID,
					Type: "connector",
					Name: datasource.Name,
				},
				Type:    "rss", // feed
				Icon:    "default",
				Title:   item.Title,
				Summary: item.Description,
				Content: item.Content,
				URL:     item.Link,
				Tags:    item.Categories,
			}
			doc.Created = item.PublishedParsed
			doc.Updated = item.UpdatedParsed
			if doc.Updated == nil {
				doc.Updated = doc.Created
			}

			if len(item.Authors) > 0 {
				doc.Owner = &common.UserInfo{
					UserName: item.Authors[0].Name,
				}
			}

			// Use the item's GUID or Link for a stable ID
			idContent := item.GUID
			if idContent == "" {
				idContent = item.Link
			}
			doc.ID = util.MD5digest(fmt.Sprintf("%s-%s-%s", connector.ID, datasource.ID, idContent))

			data := util.MustToJSONBytes(doc)
			if global.Env().IsDebug {
				log.Tracef("Queuing document: %s", string(data))
			}

			if err := queue.Push(queue.SmartGetOrInitConfig(this.Queue), data); err != nil {
				log.Errorf("Failed to push document to queue for datasource [%s]: %v", datasource.Name, err)
				// just panic? or continue
				panic(err)
			}
		}

		log.Infof("Fetched %d items from RSS feed: %s", len(feed.Items), theUrl)
	}
}

func (this *Plugin) Stop() error {
	return nil
}

func (this *Plugin) Name() string {
	return "rss"
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
