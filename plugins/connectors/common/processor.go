/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"net/http"
)

type ConnectorProcessorConfigBase struct {
	MessageField param.ParaKey      `config:"message_field"`
	Queue        *queue.QueueConfig `config:"queue"`
}

type ConnectorProcessorBase struct {
	api.Handler
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

func (processor *ConnectorProcessorBase) ProcessMessage(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource, doc common.Document) {
	processor.ProcessMessages(ctx, connector, datasource, []common.Document{doc})
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

func (processor *ConnectorProcessorBase) SaveLastModifiedTime(datasourceID string, lastModifiedTime string) error {
	err := kv.AddValue("/datasource/increment/lastModifiedTime", []byte(datasourceID), []byte(lastModifiedTime))
	return err
}

func (processor *ConnectorProcessorBase) GetLastModifiedTime(datasourceID string) (string, error) {
	data, err := kv.GetValue("/datasource/increment/lastModifiedTime", []byte(datasourceID))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func init() {
	api.HandleUIMethod(api.GET, "/datasource/:id/reset_last_modified_time", reset, api.RequireLogin())

}

func reset(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//TODO check permission

	//from context
	datasourceID := ps.MustGetParameter("id")

	err := kv.DeleteKey("/datasource/increments/lastModifiedTime", []byte(datasourceID))
	if err != nil {
		panic(err)
	}

	api.WriteJSON(w, api.NewAckJSON(true), 200)
}
