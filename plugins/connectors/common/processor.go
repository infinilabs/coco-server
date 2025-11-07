/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/datasource"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"net/http"
)

type ConnectorProcessorConfigBase struct {
	MinimumVersion string             `config:"minimum_version"`
	MessageField   param.ParaKey      `config:"message_field"`
	Queue          *queue.QueueConfig `config:"queue"`
}

type ConnectorProcessorBase struct {
	api.Handler
	ConnectorProcessorConfigBase
	connector ConnectorAPI
}

func (base *ConnectorProcessorBase) MustParseConfig(datasource *core.DataSource, cfgObj interface{}) {
	cfg, err := config.NewConfigFrom(datasource.Connector.Config)
	if err != nil {
		log.Errorf("Failed to create config from datasource [%s]: %v", datasource.Name, err)
		panic(err)
	}
	err = cfg.Unpack(cfgObj)
	if err != nil {
		log.Errorf("Failed to unpack config for datasource [%s]: %v", datasource.Name, err)
		panic(err)
	}
}

func (base *ConnectorProcessorBase) Init(c *config.Config, connector ConnectorAPI) {
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
	base.connector = connector
	base.Queue = queue.SmartGetOrInitConfig(cfg.Queue)
}

func (processor *ConnectorProcessorBase) Process(ctx *pipeline.Context) error {
	conn, ds := processor.GetBasicInfo(ctx)
	if conn == nil {
		return errors.New("connector is not found")
	}
	if ds == nil {
		return errors.New("datasource is not found")
	}
	if processor.connector == nil {
		return errors.New("connector is not found")
	}
	return processor.connector.Fetch(ctx, conn, ds)
}

func (base *ConnectorProcessorBase) GetBasicInfo(ctx *pipeline.Context) (connector *core.Connector, datasource *core.DataSource) {
	tempConnector := ctx.Get(core.PipelineContextConnector)
	connector, ok := tempConnector.(*core.Connector)
	if !ok || connector == nil {
		panic(errors.Errorf("invalid connector in pipeline context [%s][%s]", ctx.Config.Name, ctx.ID()))
	}

	tempDatasource := ctx.Get(core.PipelineContextDatasource)
	datasource, ok = tempDatasource.(*core.DataSource)
	if !ok || datasource == nil {
		panic(errors.Errorf("invalid datasource in pipeline context [%s][%s]", ctx.Config.Name, ctx.ID()))
	}
	return connector, datasource
}

func (processor *ConnectorProcessorBase) Collect(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource, doc core.Document) {
	processor.BatchCollect(ctx, connector, datasource, []core.Document{doc})
}

func (processor *ConnectorProcessorBase) BatchCollect(ctx *pipeline.Context, connector *core.Connector, datasource *core.DataSource, docs []core.Document) {

	//append enrichment pipeline process
	if datasource.EnrichmentPipeline != nil {
		log.Error("running enrichment pipeline")
		ctx.Set("documents", docs)
		err := pipeline.RunPipelineSync(*datasource.EnrichmentPipeline, ctx)
		if err != nil {
			panic(err)
		}
	}

	for _, doc := range docs {

		if global.ShuttingDown() {
			break
		}

		if common.IsDatasourceDeleted(datasource.ID) {
			panic("datasource has been deleted, skip further collect")
		}

		log.Infof("collect: [%v] [%v] [%v] [%v] [%v]", connector.Name, datasource.Name, doc.ID, doc.Category, doc.Title)

		data := util.MustToJSONBytes(doc)
		err := queue.Push(processor.Queue, data)
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

	return "", err
	return string(data), nil
}

func init() {
	updatePermission := security.GetSimplePermission("datasource", "reset_last_modified_time", string(security.Delete))
	api.HandleUIMethod(api.GET, "/datasource/:id/reset_last_modified_time", reset, api.RequirePermission(updatePermission), api.RequireLogin())

}

func reset(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	datasourceID := ps.MustGetParameter("id")

	//check datasource's permission
	v, err := datasource.GetDatasourceByID([]string{datasourceID})
	if len(v) != 0 && err == nil {
		err := kv.DeleteKey("/datasource/increments/lastModifiedTime", []byte(datasourceID))
		if err != nil {
			panic(err)
		}
		api.WriteJSON(w, api.NewAckJSON(true), 200)
		return
	}
	api.WriteJSON(w, api.NewAckJSON(false), 404)
}
