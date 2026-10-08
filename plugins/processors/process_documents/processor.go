/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package process_documents

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	fwconfig "infini.sh/framework/core/config"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

const ProcessorName = "process_documents"

// pipelineRunsCap bounds the run history kept on the document (W1):
// newest first, older runs drop off — a timeline, not an audit log.
const pipelineRunsCap = 5

type Config struct {
	MessageField param.ParaKey      `config:"message_field"`
	OutputQueue  *queue.QueueConfig `config:"output_queue"`
}

// ProcessDocumentsProcessor routes each incoming document through a
// per-datasource (or globally-configured) pipeline, then writes
// the processed document to the configured output queue.
//
// If no pipeline is configured, the document is passed through unchanged.
type ProcessDocumentsProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

// New creates a new ProcessDocumentsProcessor from the given config.
func New(c *fwconfig.Config) (pipeline.Processor, error) {
	cfg := Config{MessageField: core.PipelineContextDocuments}

	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack config of %s processor: %s", ProcessorName, err)
	}

	if cfg.MessageField == "" {
		cfg.MessageField = core.PipelineContextDocuments
	}

	p := &ProcessDocumentsProcessor{config: &cfg}
	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}

	return p, nil
}

func (p *ProcessDocumentsProcessor) Name() string {
	return ProcessorName
}

func (p *ProcessDocumentsProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		log.Warnf("processor [%s] receives an empty pipeline context", p.Name())
		return nil
	}

	messages, ok := obj.([]queue.Message)
	if !ok {
		log.Warnf("processor [%s] context value is not []queue.Message", p.Name())
		return nil
	}

	for i := range messages {
		if err := p.processMessage(messages[i]); err != nil {
			log.Errorf("processor [%s] failed to process message %d: %v", p.Name(), i, err)
		}
	}

	return nil
}

// processMessage handles a single document message: resolves the
// pipeline for its datasource, runs it, and writes the processed document to
// the output queue. Falls back to a direct passthrough on any error or when
// no pipeline is configured.
func (p *ProcessDocumentsProcessor) processMessage(msg queue.Message) error {
	// Deserialize to read the source ID.
	doc := core.Document{}
	if err := util.FromJSONBytes(msg.Data, &doc); err != nil {
		log.Errorf("processor [%s] failed to deserialize document: %v", p.Name(), err)
		return p.passthrough(msg)
	}

	// Ingest dedup gate (W1): a different-id document with the same
	// content fingerprint in the same datasource is a true duplicate
	// (renamed copy, re-uploaded file) — drop it and refresh the
	// existing row's updated timestamp instead of indexing a twin.
	if ingestDedupGateEnabled() && doc.ContentHash != "" && doc.Source.ID != "" {
		if dupID, found := findDuplicateByFingerprint(&doc); found {
			log.Infof("processor [%s] dropping duplicate of document [%s] (same content_hash in datasource [%s])", p.Name(), dupID, doc.Source.ID)
			bumpExistingUpdated(dupID)
			return nil
		}
	}

	// No datasource reference — nothing to look up.
	if doc.Source.ID == "" {
		return p.pushPassthrough(msg.Data, "no-datasource")
	}

	ormCtx := orm.NewContext()
	ormCtx.DirectReadAccess()
	ormCtx.PermissionScope(security.PermissionScopePlatform)

	// Fetch the datasource to read its processing config.
	ds := core.DataSource{}
	ds.ID = doc.Source.ID
	exists, err := orm.GetV2(ormCtx, &ds)
	if err != nil || !exists {
		log.Debugf("processor [%s] datasource [%s] not found (err=%v), passing through", p.Name(), doc.Source.ID, err)
		return p.pushPassthrough(msg.Data, "datasource-not-found")
	}

	// Resolve the enrichment pipeline name: datasource-level first, then global default.
	pipelineName := ""
	if ds.DocumentProcessingConfig.Enabled && ds.DocumentProcessingConfig.Pipeline != "" {
		pipelineName = ds.DocumentProcessingConfig.Pipeline
	} else {
		appCfg := common.AppConfig()
		if appCfg.DocumentProcessing != nil {
			pipelineName = appCfg.DocumentProcessing.DefaultPipelineForDocument
		}
	}

	if pipelineName == "" {
		// No pipeline configured — pass through directly.
		return p.pushPassthrough(msg.Data, "no-pipeline-configured")
	}

	// Load the pipeline config (pipeline name == ES document ID).
	pipelineCfg := pipeline.PipelineConfigV2{}
	pipelineCfg.ID = pipelineName
	exists, err = orm.GetV2(ormCtx, &pipelineCfg)
	if err != nil || !exists {
		log.Warnf("processor [%s] pipeline [%s] not found (err=%v), passing through", p.Name(), pipelineName, err)
		return p.pushPassthrough(msg.Data, "pipeline-not-found")
	}

	// Compile the processor chain from the stored config.
	processorCfgs, err := pipelineCfg.GetProcessorsConfig()
	if err != nil {
		log.Errorf("processor [%s] failed to build processor configs for pipeline [%s]: %v", p.Name(), pipelineName, err)
		return p.pushPassthrough(msg.Data, "pipeline-config-invalid")
	}

	procs, err := pipeline.NewPipeline(processorCfgs)
	if err != nil {
		log.Errorf("processor [%s] failed to instantiate pipeline [%s]: %v", p.Name(), pipelineName, err)
		return p.pushPassthrough(msg.Data, "pipeline-init-failed")
	}

	// Run the enrichment pipeline synchronously in the current goroutine.
	subCtx := pipeline.AcquireContext(pipelineCfg)
	subCtx.Set(p.config.MessageField, []queue.Message{msg})
	// Downstream processors consult the framework pipeline state via ShouldContinue.
	// This synchronous caller does not rely on runtime lifecycle semantics itself,
	// but the sub-context must be marked started so the processor chain can run.
	subCtx.Started()

	started := time.Now()
	pipelineSucceeded := true
	pipelineErr := ""
	if err := procs.Process(subCtx); err != nil {
		log.Errorf("processor [%s] pipeline [%s] returned error: %v — forwarding whatever was enriched", p.Name(), pipelineName, err)
		pipelineSucceeded = false
		pipelineErr = err.Error()
	}
	tookMs := time.Since(started).Milliseconds()

	// Retrieve the (possibly processed) messages from the sub-context.
	enriched, ok := subCtx.Get(p.config.MessageField).([]queue.Message)
	if !ok || len(enriched) == 0 {
		log.Warnf("processor [%s] sub-pipeline [%s] produced no output, passing through original", p.Name(), pipelineName)
		return p.pushPassthrough(msg.Data, "pipeline-no-output")
	}

	// Write every processed document to the output queue.
	// A sub-pipeline processor (e.g. a splitter or "duplicate" processor) may
	// expand one input document into multiple output documents, so we iterate
	// over the full slice rather than assuming a 1-to-1 mapping.
	for _, em := range enriched {
		if err := p.pushEnriched(em.Data, pipelineSucceeded, pipelineName, tookMs, pipelineErr); err != nil {
			log.Errorf("processor [%s] failed to push enriched document to output queue: %v", p.Name(), err)
		}
	}

	return nil
}

// passthrough writes the original message to the output queue without
// modification. Used only when the message cannot be deserialized, making
// it impossible to stamp the Processed field.
func (p *ProcessDocumentsProcessor) passthrough(msg queue.Message) error {
	if p.outputQueue == nil {
		return nil
	}
	return queue.Push(p.outputQueue, msg.Data)
}

// pushEnriched stamps the terminal lifecycle outcome of an enrichment run
// (W1) and pushes to the output queue. Falls back to the raw bytes if
// (de)serialization fails.
func (p *ProcessDocumentsProcessor) pushEnriched(data []byte, success bool, pipelineName string, tookMs int64, errMsg string) error {
	if p.outputQueue == nil {
		return nil
	}
	doc := core.Document{}
	if err := util.FromJSONBytes(data, &doc); err != nil {
		return queue.Push(p.outputQueue, data)
	}
	applyProcessingOutcome(&doc, success, pipelineName, tookMs, errMsg)
	// W11 liaison: notify the curated layer on every enrichment completion
	// — handlers decide (fingerprint compare) whether anything changed
	common.FireDocumentProcessed(&doc)
	return queue.Push(p.outputQueue, util.MustToJSONBytes(&doc))
}

// pushPassthrough stamps a terminal-without-enrichment outcome — the
// document leaves the queue for indexing, but no enrichment ran (no
// datasource, no pipeline, broken config). Processed stays false; the run
// record carries the passthrough reason so the timeline (W13a) can show
// why. Legacy statuses (failed on an earlier attempt) are preserved.
func (p *ProcessDocumentsProcessor) pushPassthrough(data []byte, reason string) error {
	if p.outputQueue == nil {
		return nil
	}
	doc := core.Document{}
	if err := util.FromJSONBytes(data, &doc); err != nil {
		return queue.Push(p.outputQueue, data)
	}
	applyPassthroughOutcome(&doc, reason)
	return queue.Push(p.outputQueue, util.MustToJSONBytes(&doc))
}

// applyProcessingOutcome stamps the enrichment terminal state and prepends
// a run record to Metadata["pipeline_runs"] (capped, newest first).
func applyProcessingOutcome(doc *core.Document, success bool, pipelineName string, tookMs int64, errMsg string) {
	if success {
		doc.Status = core.DocumentStatusCompleted
		doc.ErrorMessage = ""
		doc.Processed = true
	} else {
		doc.Status = core.DocumentStatusFailed
		doc.ErrorMessage = truncateRunError(errMsg)
		doc.Processed = false
	}
	prependRunRecord(doc, util.MapStr{
		"pipeline": pipelineName,
		"took_ms":  tookMs,
		"success":  success,
		"at":       time.Now().UTC().Format(time.RFC3339),
		"error":    truncateRunError(errMsg),
	})
}

// applyPassthroughOutcome marks the lifecycle terminal without claiming
// enrichment ran. A prior failure is not overwritten: re-running a failed
// document whose datasource lost its pipeline keeps the honest failed state.
func applyPassthroughOutcome(doc *core.Document, reason string) {
	if doc.Status != core.DocumentStatusFailed {
		doc.Status = core.DocumentStatusCompleted
	}
	doc.Processed = false
	prependRunRecord(doc, util.MapStr{
		"pipeline":    "",
		"passthrough": reason,
		"success":     true,
		"at":          time.Now().UTC().Format(time.RFC3339),
	})
}

func prependRunRecord(doc *core.Document, rec util.MapStr) {
	if doc.Metadata == nil {
		doc.Metadata = map[string]interface{}{}
	}
	runs, _ := doc.Metadata["pipeline_runs"].([]interface{})
	out := make([]interface{}, 0, pipelineRunsCap+1)
	out = append(out, rec)
	out = append(out, runs...)
	if len(out) > pipelineRunsCap {
		out = out[:pipelineRunsCap]
	}
	doc.Metadata["pipeline_runs"] = out
}

func truncateRunError(msg string) string {
	if len(msg) > 500 {
		return msg[:500]
	}
	return msg
}

// ingestDedupGateEnabled reads INGEST_DEDUP_GATE; anything but the explicit
// off values keeps the gate on (fail-closed against duplicates).
func ingestDedupGateEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("INGEST_DEDUP_GATE"))) {
	case "off", "false", "0":
		return false
	}
	return true
}

// findDuplicateByFingerprint looks for another document in the same
// datasource carrying the same content hash.
func findDuplicateByFingerprint(doc *core.Document) (string, bool) {
	ormCtx := orm.NewContext()
	ormCtx.DirectReadAccess()
	ormCtx.PermissionScope(security.PermissionScopePlatform)

	q := orm.Query{}
	var err error
	q.RawQuery, err = core.RewriteQueryWithFilter(q.RawQuery, util.MapStr{
		"bool": util.MapStr{
			"filter": []util.MapStr{
				{"term": util.MapStr{"content_hash": doc.ContentHash}},
				{"term": util.MapStr{"source.id": doc.Source.ID}},
			},
			"must_not": []util.MapStr{
				{"term": util.MapStr{"id": doc.ID}},
			},
		},
	})
	if err != nil {
		return "", false
	}
	q.From, q.Size = 0, 1

	docs := []core.Document{}
	if err, _ = orm.SearchWithJSONMapper(&docs, &q); err != nil {
		log.Warnf("processor [%s] dedup gate search failed (err=%v), indexing anyway", ProcessorName, err)
		return "", false
	}
	if len(docs) == 0 {
		return "", false
	}
	return docs[0].ID, true
}

// bumpExistingUpdated refreshes the surviving duplicate's updated timestamp
// (best-effort — a failure here never blocks ingestion).
func bumpExistingUpdated(id string) {
	ormCtx := orm.NewContext()
	ormCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ormCtx.Refresh = orm.WaitForRefresh

	doc := core.Document{}
	doc.ID = id
	exists, err := orm.GetV2(ormCtx, &doc)
	if err != nil || !exists {
		return
	}
	now := time.Now().UTC()
	doc.Updated = &now
	if err := orm.Save(ormCtx, &doc); err != nil {
		log.Debugf("processor [%s] failed to bump duplicate [%s] updated timestamp: %v", ProcessorName, id, err)
	}
}
