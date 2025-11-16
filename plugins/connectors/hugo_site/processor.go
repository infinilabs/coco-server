/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package hugo_site

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	cmn "infini.sh/coco/plugins/connectors/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
)

type Processor struct {
	cmn.ConnectorProcessorBase
}

func init() {
	pipeline.RegisterProcessorPlugin(processorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of processor [%v], err: %s", processorName, err)
	}

	runner := Processor{}
	runner.Init(c, &runner)
	return &runner, nil
}

const processorName = "hugo_site"

func (processor *Processor) Name() string {
	return processorName
}

func (processor *Processor) Fetch(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource) error {

	cfg := Config{}
	processor.MustParseConfig(datasource, &cfg)

	//core processor logic
	for _, myURL := range cfg.Urls {

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

			outputDocs := []core.Document{}

			// Output the parsed data
			for i, v := range documents {

				if global.ShuttingDown() {
					break
				}

				doc := core.Document{Source: core.DataSourceReference{ID: datasource.ID, Type: "connector", Name: datasource.Name}}

				if v.Created != "" {
					doc.Created = ParseTimestamp(v.Created)
				}

				if v.Updated != "" {
					doc.Created = ParseTimestamp(v.Updated)
				}
				doc.System = datasource.System
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

				outputDocs = append(outputDocs, doc)
			}

			if len(outputDocs) > 0 {
				//put doc back to context or call enrichment pipeline if it exists
				ctx.Set(core.PipelineContextDocuments, &outputDocs)
				processor.BatchCollect(ctx, connector, datasource, outputDocs)
			}

			log.Infof("fetched %v docs from hugo site: %v", len(documents), myURL)
		}
	}

	return nil
}
