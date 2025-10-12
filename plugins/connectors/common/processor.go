/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

type ConnectorProcessorConfigBase struct {
	MessageField param.ParaKey      `config:"message_field"`
	Queue        *queue.QueueConfig `config:"queue"`
}

type ConnectorProcessorBase struct {
	ConnectorProcessorConfigBase
}

func (base *ConnectorProcessorBase) InitBaseConfig(c *config.Config) {
	cfg := ConnectorProcessorConfigBase{MessageField: core.PipelineContextDocuments}
	if err := c.Unpack(&cfg); err != nil {
		panic(err)
	}

	if cfg.MessageField == "" {
		panic("message field is empty")
	}

	if cfg.Queue == nil {
		cfg.Queue = &queue.QueueConfig{Name: "indexing_documents"}
	}
	base.Queue = queue.SmartGetOrInitConfig(cfg.Queue)
}

func (base *ConnectorProcessorBase) GetBasicInfo(ctx *pipeline.Context) (connector *common.Connector, datasource *common.DataSource) {
	tempConnector := ctx.Get(core.PipelineContextConnector)
	connector, ok := tempConnector.(*common.Connector)
	if !ok || connector == nil {
		panic(errors.Errorf("invalid connector in pipeline context [%s][%s]", ctx.Config.Name, ctx.ID()))
	}

	tempDatasource := ctx.Get(core.PipelineContextDatasource)
	datasource, ok = tempDatasource.(*common.DataSource)
	if !ok || datasource == nil {
		panic(errors.Errorf("invalid datasource in pipeline context [%s][%s]", ctx.Config.Name, ctx.ID()))
	}
	return connector, datasource
}

func (processor *ConnectorProcessorBase) ProcessMessages(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource, docs []common.Document) {

	//append enrichment pipeline process
	if datasource.EnrichmentPipeline != nil {
		err := pipeline.RunPipelineSync(*datasource.EnrichmentPipeline, ctx)
		if err != nil {
			panic(err)
		}
	}

	for _, doc := range docs {
		data := util.MustToJSONBytes(doc)
		err := queue.Push(queue.SmartGetOrInitConfig(processor.Queue), data)
		if err != nil {
			panic(err)
		}
	}
}
