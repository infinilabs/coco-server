/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rss

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/mmcdole/gofeed"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/module"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ConnectorRss = "rss"

type Config struct {
	Interval string   `config:"interval"`
	Urls     []string `config:"urls"`
}

type Plugin struct {
	connectors.BasePlugin
}

func (p *Plugin) Setup() {
	p.BasePlugin.Init("connector.rss", "indexing rss feeds", p)
}

func (p *Plugin) Start() error {
	return p.BasePlugin.Start(connectors.DefaultSyncInterval)
}

func (p *Plugin) Scan(connector *common.Connector, datasource *common.DataSource) {
	p.fetchRssFeed(connector, datasource)
}

// fetchRssFeed handles the logic of fetching, parsing, and indexing a single RSS feed.
func (p *Plugin) fetchRssFeed(connector *common.Connector, datasource *common.DataSource) {
	if connector == nil || datasource == nil {
		panic("invalid rss connector or datasource config")
	}

	cfg, err := config3.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		log.Errorf("[%v connector] Failed to create config from datasource [%s]: %v", ConnectorRss, datasource.Name, err)
		panic(err)
	}

	obj := Config{}
	err = cfg.Unpack(&obj)
	if err != nil {
		log.Errorf("[%v connector] Failed to unpack config for datasource [%s]: %v", ConnectorRss, datasource.Name, err)
		panic(err)
	}

	log.Debugf("[%v connector] Handling datasource: %v", ConnectorRss, obj)

	fp := gofeed.NewParser()

	for _, theUrl := range obj.Urls {
		if global.ShuttingDown() {
			break
		}

		log.Debugf("[%v connector] Connecting to RSS feed: %v", ConnectorRss, theUrl)

		feed, err := fp.ParseURL(theUrl)
		if err != nil {
			log.Errorf("[%v connector] Failed to parse RSS feed from URL [%s]: %v", ConnectorRss, theUrl, err)
			continue
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
				Type:    "rss",
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
				log.Tracef("[%v connector] Queuing document: %s", ConnectorRss, string(data))
			}

			if err := queue.Push(queue.SmartGetOrInitConfig(p.Queue), data); err != nil {
				log.Errorf("[%v connector] Failed to push document to queue for datasource [%s]: %v", ConnectorRss, datasource.Name, err)
				// just panic? or continue
				panic(err)
			}
		}

		log.Infof("[%v connector] Fetched %d items from RSS feed: %s", ConnectorRss, len(feed.Items), theUrl)
	}
}

func (p *Plugin) Stop() error {
	return nil
}

func (p *Plugin) Name() string {
	return ConnectorRss
}

func init() {
	module.RegisterUserPlugin(&Plugin{})
}
