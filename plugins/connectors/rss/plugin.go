/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rss

import (
	"fmt"
	"infini.sh/coco/core"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/pipeline"

	log "github.com/cihub/seelog"
	"github.com/mmcdole/gofeed"
	config3 "infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
)

const ConnectorRss = "rss"

type Config struct {
	Interval string   `config:"interval"`
	Urls     []string `config:"urls"`
}

// Fetch handles the logic of fetching, parsing, and indexing a single RSS feed.
func (p *Plugin) Fetch(pipeCtx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {
	config := Config{}
	p.MustParseConfig(datasource, &config)

	log.Debugf("[%v connector] Handling datasource: %v", ConnectorRss, config)

	fp := gofeed.NewParser()

	for _, theUrl := range config.Urls {
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

			doc := core.Document{
				Source: core.DataSourceReference{
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
			doc.System = datasource.System
			doc.Created = item.PublishedParsed
			doc.Updated = item.UpdatedParsed
			if doc.Updated == nil {
				doc.Updated = doc.Created
			}

			if len(item.Authors) > 0 {
				doc.Owner = &core.UserInfo{
					UserName: item.Authors[0].Name,
				}
			}

			// Use the item's GUID or Link for a stable ID
			idContent := item.GUID
			if idContent == "" {
				idContent = item.Link
			}
			doc.ID = util.MD5digest(fmt.Sprintf("%s-%s-%s", connector.ID, datasource.ID, idContent))

			p.Collect(pipeCtx, connector, datasource, doc)
		}

		log.Infof("[%v connector] Fetched %d items from RSS feed: %s", ConnectorRss, len(feed.Items), theUrl)
	}

	return nil
}

type Plugin struct {
	cmn.ConnectorProcessorBase
}

const Name = "rss"

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
