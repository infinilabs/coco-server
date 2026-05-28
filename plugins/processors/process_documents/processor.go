/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package process_documents

import (
	"fmt"

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

	// No datasource reference — nothing to look up.
	if doc.Source.ID == "" {
		return p.pushWithProcessed(msg.Data, false)
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
		return p.pushWithProcessed(msg.Data, false)
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
		return p.pushWithProcessed(msg.Data, false)
	}

	// Load the pipeline config (pipeline name == ES document ID).
	pipelineCfg := pipeline.PipelineConfigV2{}
	pipelineCfg.ID = pipelineName
	exists, err = orm.GetV2(ormCtx, &pipelineCfg)
	if err != nil || !exists {
		log.Warnf("processor [%s] pipeline [%s] not found (err=%v), passing through", p.Name(), pipelineName, err)
		return p.pushWithProcessed(msg.Data, false)
	}

	// Compile the processor chain from the stored config.
	processorCfgs, err := pipelineCfg.GetProcessorsConfig()
	if err != nil {
		log.Errorf("processor [%s] failed to build processor configs for pipeline [%s]: %v", p.Name(), pipelineName, err)
		return p.pushWithProcessed(msg.Data, false)
	}

	procs, err := pipeline.NewPipeline(processorCfgs)
	if err != nil {
		log.Errorf("processor [%s] failed to instantiate pipeline [%s]: %v", p.Name(), pipelineName, err)
		return p.pushWithProcessed(msg.Data, false)
	}

	// Run the enrichment pipeline synchronously in the current goroutine.
	subCtx := pipeline.AcquireContext(pipelineCfg)
	subCtx.Set(p.config.MessageField, []queue.Message{msg})
	// Downstream processors consult the framework pipeline state via ShouldContinue.
	// This synchronous caller does not rely on runtime lifecycle semantics itself,
	// but the sub-context must be marked started so the processor chain can run.
	subCtx.Started()

	pipelineSucceeded := true
	if err := procs.Process(subCtx); err != nil {
		log.Errorf("processor [%s] pipeline [%s] returned error: %v — forwarding whatever was enriched", p.Name(), pipelineName, err)
		pipelineSucceeded = false
	}

	// Retrieve the (possibly processed) messages from the sub-context.
	enriched, ok := subCtx.Get(p.config.MessageField).([]queue.Message)
	if !ok || len(enriched) == 0 {
		log.Warnf("processor [%s] sub-pipeline [%s] produced no output, passing through original", p.Name(), pipelineName)
		return p.pushWithProcessed(msg.Data, false)
	}

	// Write every processed document to the output queue.
	// A sub-pipeline processor (e.g. a splitter or "duplicate" processor) may
	// expand one input document into multiple output documents, so we iterate
	// over the full slice rather than assuming a 1-to-1 mapping.
	for _, em := range enriched {
		if err := p.pushWithProcessed(em.Data, pipelineSucceeded); err != nil {
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

// pushWithProcessed stamps doc.Processed, re-serializes, and pushes to the
// output queue. Falls back to the raw bytes if (de)serialization fails.
func (p *ProcessDocumentsProcessor) pushWithProcessed(data []byte, processed bool) error {
	if p.outputQueue == nil {
		return nil
	}
	doc := core.Document{}
	if err := util.FromJSONBytes(data, &doc); err != nil {
		return queue.Push(p.outputQueue, data)
	}
	doc.Processed = processed
	return queue.Push(p.outputQueue, util.MustToJSONBytes(&doc))
}
