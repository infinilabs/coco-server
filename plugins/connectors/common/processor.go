/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/common/fingerprint"
	"infini.sh/coco/modules/datasource"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
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
		//log.Error("running enrichment pipeline")
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

		// sync run journal (W8): per-document progress — the shared choke
		// every connector passes through; the journal lives in
		// modules/common (the neutral import floor). Counted before the
		// skip gate: a fingerprint-skipped document was still seen and
		// handled by this run.
		common.RecordSyncBatch(datasource.ID, 1)

		// deletion reconciliation: record the observed ID before the skip
		// gate — a gate-dropped document still exists at the source
		common.RecordSeenDocID(datasource.ID, doc.ID)

		// sync skip gate: the deterministic doc id resolves the indexed
		// copy, and a matching source fingerprint means the source has not
		// changed since its last successful enrichment — drop the message
		// here so the re-collect cycle neither re-runs the enrichment chain
		// nor refills the queues. Failed or never-enriched documents always
		// re-enter (retry); a deleted document no longer resolves and
		// falls through to normal ingest.
		if doc.ID != "" && doc.SourceFingerprint != "" && engineDocUnchanged(doc.ID, doc.SourceFingerprint, datasource.EnrichmentPipeline == nil) {
			log.Infof("collect: skipping unchanged document [%v] [%v]", doc.ID, doc.URL)
			continue
		}

		log.Infof("collect: [%v] [%v] [%v] [%v] [%v]", connector.Name, datasource.Name, doc.ID, doc.Category, doc.Title)

		// stamp the dedup fingerprint at the source (D1.5): this queue
		// feeds the framework's indexing_merge/bulk_indexing processors,
		// which index straight to the engine and never touch orm — the
		// document module's pre-hook cannot fire for connector-sourced
		// documents. Same math as the hook (modules/common/fingerprint);
		// !ok clears the fields so a shrunken re-ingest drops its stale
		// fingerprint instead of keeping the previous write's
		if h, s, ok := fingerprint.Compute(doc.Content); ok {
			doc.ContentHash, doc.ContentSimhash = h, int64(s)
		} else {
			doc.ContentHash, doc.ContentSimhash = "", 0
		}

		data := util.MustToJSONBytes(doc)
		err := queue.Push(processor.Queue, data)
		if err != nil {
			panic(err)
		}
	}
}

// engineDocUnchanged reports whether the indexed copy of this document
// completed enrichment with the same source fingerprint the connector just
// observed. Errors and misses return false so collection falls through to
// a normal ingest — the gate may skip work, never lose it.
//
// noPipeline: the datasource has no enrichment pipeline configured, so its
// documents deterministically exit the chain via the passthrough outcome
// (Processed stays false there, by design — the timeline records why). For
// those sources a completed document is already terminal: without this,
// the gate would never open and every re-collect cycle would re-ingest
// every document forever. A pipeline added later still re-enriches via
// _reprocess (which clears the stored fingerprint).
func engineDocUnchanged(docID, sourceFingerprint string, noPipeline bool) bool {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.Document{})

	existing := core.Document{}
	existing.ID = docID
	exists, err := orm.GetV2(ctx, &existing)
	if err != nil || !exists {
		return false
	}
	if existing.SourceFingerprint != sourceFingerprint {
		return false
	}
	// a failed or half-processed document must retry even though the
	// source itself never changed
	if existing.Status == core.DocumentStatusFailed {
		return false
	}
	return existing.Processed || noPipeline
}

// incrementWatermarkKV is the kv bucket holding the per-datasource
// last-modified watermark. This is the canonical key: /datasource/_sync_status
// (modules/document) reads the same bucket, so keep them in sync.
const incrementWatermarkKV = "/datasource/increment/lastModifiedTime"

func (processor *ConnectorProcessorBase) SaveLastModifiedTime(datasourceID string, lastModifiedTime string) error {
	return kv.AddValue(incrementWatermarkKV, []byte(datasourceID), []byte(lastModifiedTime))
}

func (processor *ConnectorProcessorBase) GetLastModifiedTime(datasourceID string) (string, error) {
	data, err := kv.GetValue(incrementWatermarkKV, []byte(datasourceID))
	if err != nil {
		// a missing watermark is a normal first-sync state; callers treat
		// "" as "no cursor" and fall back to a full scan
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
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
		// legacy key (plural) cleanup, then the canonical ones:
		// - lastModifiedTime: cloud-drive connectors' watermark (feishu etc.)
		// - lastSyncValue: RDBMS scanners' source-side watermark (mysql/pg/mssql/oracle)
		_ = kv.DeleteKey("/datasource/increments/lastModifiedTime", []byte(datasourceID))
		if err := kv.DeleteKey(incrementWatermarkKV, []byte(datasourceID)); err != nil {
			panic(err)
		}
		if err := kv.DeleteKey(lastSyncValueKV, []byte(datasourceID)); err != nil {
			panic(err)
		}
		api.WriteJSON(w, api.NewAckJSON(true), 200)
		return
	}
	api.WriteJSON(w, api.NewAckJSON(false), 404)
}
