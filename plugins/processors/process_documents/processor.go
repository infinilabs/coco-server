/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package process_documents

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/core/codecs"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/document"
	"infini.sh/coco/plugins/connectors"
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

	// Mode selects the sub-chain execution convention: "messages" (legacy,
	// default until P8) or "records" (record scope — the sub-chain runs
	// against *event.Event with the document subtree, processors use their
	// ProcessBatch/record paths).
	Mode string `config:"mode"`
	// CompileCacheTTLInMs caches compiled sub-chains per pipeline name for
	// this long (0 = disabled, per-message compile like today). The stored
	// pipeline's semantics still take effect within TTL+change-latency.
	CompileCacheTTLInMs int64 `config:"compile_cache_ttl_ms"`
}

// ProcessDocumentsProcessor routes each incoming document through a
// per-datasource (or globally-configured) pipeline, then writes
// the processed document to the configured output queue.
//
// If no pipeline is configured, the document is passed through unchanged.
type compiledEntry struct {
	procs   *pipeline.Processors
	expires time.Time
}

type ProcessDocumentsProcessor struct {
	compileCache map[string]compiledEntry
	config       *Config
	outputQueue  *queue.QueueConfig
}

func init() {
	pipeline.RegisterProcessorPluginWithConfigMetadata(ProcessorName, New, Config{})
	pipeline.RegisterProcessorContract(ProcessorName, pipeline.RecordContract{Scope: pipeline.ScopeBatch})
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

	// Idempotent retry gate: a queued message for a document that is already
	// indexed, enriched, and content-identical is a stale backlog message
	// (crashed slice, re-collect) — drop it instead of paying the LLM chain
	// again. Changed content re-enters with a new hash and enriches normally.
	if doc.ID != "" && doc.ContentHash != "" {
		done, err := alreadyEnriched(&doc)
		if err != nil {
			log.Debugf("processor [%s] enriched-state check failed for [%s] (%v), enriching anyway", p.Name(), doc.ID, err)
		} else if done {
			log.Debugf("processor [%s] dropping stale message for enriched document [%s/%s]", p.Name(), doc.Title, doc.ID)
			return nil
		}
	}

	// Folder documents carry no content to enrich: exempt them from the
	// enrichment chain entirely — the tree aggregation keys on folder_path,
	// not enrichment fields, and every chain processor would otherwise
	// wake up per folder per sync just to no-op.
	if doc.Type == connectors.TypeFolder {
		return p.pushPassthrough(msg.Data, "folder-doc")
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

	// resolveCompiled returns the compiled sub-chain, using the TTL cache
	// when enabled (stores refetch on expiry — the version-aware compile
	// cache, task #132).
	resolveCompiled := func() (*pipeline.Processors, string) {
		if p.config.CompileCacheTTLInMs > 0 {
			if e, ok := p.compileCache[pipelineName]; ok && time.Now().Before(e.expires) {
				return e.procs, ""
			}
		}
		pipelineCfg := pipeline.PipelineConfigV2{}
		pipelineCfg.ID = pipelineName
		exists2, err2 := orm.GetV2(ormCtx, &pipelineCfg)
		if err2 != nil || !exists2 {
			return nil, "pipeline-not-found"
		}
		processorCfgs, err2 := pipelineCfg.GetProcessorsConfig()
		if err2 != nil {
			return nil, "pipeline-config-invalid"
		}
		var procs2 *pipeline.Processors
		if p.config.Mode == "records" {
			// record-scope compile: the stored chain runs against document-
			// namespace records (its processors use their record paths)
			procs2, err2 = pipeline.NewPipelineWithScope(processorCfgs, pipeline.ScopeInfo{
				Kind:      pipeline.ScopeRecord,
				Namespace: "document",
				Strict:    pipelineCfg.StrictContract != nil && *pipelineCfg.StrictContract,
			}, true)
		} else {
			procs2, err2 = pipeline.NewPipeline(processorCfgs)
		}
		if err2 != nil {
			return nil, "pipeline-init-failed"
		}
		if p.config.CompileCacheTTLInMs > 0 {
			p.compileCache[pipelineName] = compiledEntry{procs: procs2, expires: time.Now().Add(time.Duration(p.config.CompileCacheTTLInMs) * time.Millisecond)}
		}
		return procs2, ""
	}
	procs, failure := resolveCompiled()
	if failure != "" {
		log.Warnf("processor [%s] pipeline [%s] %s, passing through", p.Name(), pipelineName, failure)
		return p.pushPassthrough(msg.Data, failure)
	}

	// Run the enrichment pipeline synchronously in the current goroutine.
	subCtx := pipeline.AcquireContext(pipelineCfg0(p.Name()))
	if p.config.Mode == "records" {
		// record convention: decode once, run the chain in record scope,
		// encode once at exit — the sub-chain's processors see document.*
		rec, derr := codecs.DocumentCodec{}.Decode(msg.Data)
		if derr != nil {
			return p.pushPassthrough(msg.Data, "record-decode-failed")
		}
		subCtx.Set(pipeline.RecordContextKey, rec)
		subCtx.Started()
		if rerr := procs.Process(subCtx); rerr != nil {
			log.Errorf("processor [%s] records-mode pipeline error: %v", p.Name(), rerr)
		}
		encoded, eerr := codecs.DocumentCodec{}.Encode(rec)
		if eerr != nil {
			return p.pushPassthrough(msg.Data, "record-encode-failed")
		}
		msg.Data = encoded
		msg.Size = len(encoded)
		pipeline.ReleaseContext(subCtx)
		if err := p.pushEnriched(msg.Data, true, pipelineName, 0, "", nil); err != nil {
			log.Errorf("processor [%s] failed to push enriched document to output queue: %v", p.Name(), err)
		}
		return nil
	}
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
	// Per-processor step stats recorded by the framework Process loop —
	// persisted on the run record for document-level pipeline observability.
	steps := subCtx.GetStepStats()

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
		if err := p.pushEnriched(em.Data, pipelineSucceeded, pipelineName, tookMs, pipelineErr, steps); err != nil {
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
func (p *ProcessDocumentsProcessor) pushEnriched(data []byte, success bool, pipelineName string, tookMs int64, errMsg string, steps []pipeline.StepStat) error {
	if p.outputQueue == nil {
		return nil
	}
	doc := core.Document{}
	if err := util.FromJSONBytes(data, &doc); err != nil {
		return queue.Push(p.outputQueue, data)
	}
	applyProcessingOutcome(&doc, success, pipelineName, tookMs, errMsg, steps)
	// W3 方案 c: mirror the (re-)chunked document into the standalone
	// KnowledgeChunk index — the retrieval face for chunk-level legs
	if len(doc.Chunks) > 0 {
		document.ProjectDocumentChunksFn(context.Background(), &doc)
	}
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
func applyProcessingOutcome(doc *core.Document, success bool, pipelineName string, tookMs int64, errMsg string, steps []pipeline.StepStat) {
	if success {
		doc.Status = core.DocumentStatusCompleted
		doc.ErrorMessage = ""
		doc.Processed = true
	} else {
		doc.Status = core.DocumentStatusFailed
		doc.ErrorMessage = truncateRunError(errMsg)
		doc.Processed = false
	}
	rec := util.MapStr{
		"pipeline": pipelineName,
		"took_ms":  tookMs,
		"success":  success,
		"at":       time.Now().UTC().Format(time.RFC3339),
		"error":    truncateRunError(errMsg),
	}
	// Per-processor steps (name/took_ms/error) for the timeline UI; only
	// attached when the framework recorded any, keeping passthrough records lean.
	if len(steps) > 0 {
		stepArr := make([]interface{}, 0, len(steps))
		for _, s := range steps {
			m := util.MapStr{"name": s.Name, "took_ms": s.TookUs / 1000}
			if s.Error != "" {
				m["error"] = s.Error
			}
			stepArr = append(stepArr, m)
		}
		rec["steps"] = stepArr
	}
	prependRunRecord(doc, rec)
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

// alreadyEnriched reports whether the indexed copy of this document is
// already enriched (Processed) with the same content hash — meaning the
// queued message is a stale retry. Errors (missing index, store trouble)
// are returned so the caller can fall through to normal enrichment.
func alreadyEnriched(doc *core.Document) (bool, error) {
	ormCtx := orm.NewContext()
	ormCtx.DirectReadAccess()
	ormCtx.PermissionScope(security.PermissionScopePlatform)
	orm.WithModel(ormCtx, &core.Document{})

	existing := core.Document{}
	existing.ID = doc.ID
	exists, err := orm.GetV2(ormCtx, &existing)
	if err != nil {
		if exists == false && strings.Contains(err.Error(), "record not found") {
			return false, nil
		}
		return false, err
	}
	if !exists {
		return false, nil
	}
	// Processed is the strict signal; a stored summary with a matching hash
	// also counts — an attempt that crashed mid-chain (flag still false)
	// must not force the whole backlog through the LLM chain again.
	return (existing.Processed || existing.Summary != "") && existing.ContentHash == doc.ContentHash, nil
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
	// the background pipeline has no request user; the get/search security
	// hooks bypass on the READ flag and panic without it
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
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

// pipelineCfg0 builds a minimal PipelineConfigV2 for sub-contexts.
func pipelineCfg0(name string) pipeline.PipelineConfigV2 {
	return pipeline.PipelineConfigV2{Name: name}
}
