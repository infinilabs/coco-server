/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

// Package process_attachments provides the process_attachments pipeline
// processor, which consumes attachment IDs from a queue, retrieves each
// attachment's metadata from Easysearch and its binary content from the
// KV blob store, runs a user-configured sub-pipeline against them, and
// writes the updated metadata back to Elasticsearch.
package process_attachments

import (
	"fmt"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	attachmentmod "infini.sh/coco/modules/attachment"
	"infini.sh/coco/modules/common"
	fwconfig "infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

const ProcessorName = "process_attachments"

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

// Config holds the configuration for ProcessAttachmentsProcessor.
type Config struct {
	MessageField param.ParaKey `config:"message_field"`
}

// ProcessAttachmentsProcessor reads attachment IDs from the pipeline context,
// retrieves each attachment's metadata and binary data, invokes the
// globally-configured attachment processing sub-pipeline, and persists the
// updated metadata back to Easysearch.
type ProcessAttachmentsProcessor struct {
	config *Config
}

func New(c *fwconfig.Config) (pipeline.Processor, error) {
	cfg := Config{MessageField: core.PipelineContextDocuments}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack config of %s processor: %w", ProcessorName, err)
	}
	if cfg.MessageField == "" {
		cfg.MessageField = core.PipelineContextDocuments
	}
	return &ProcessAttachmentsProcessor{config: &cfg}, nil
}

func (p *ProcessAttachmentsProcessor) Name() string {
	return ProcessorName
}

func (p *ProcessAttachmentsProcessor) Process(ctx *pipeline.Context) error {
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
		if global.ShuttingDown() {
			log.Debugf("[%s] shutting down, skipping remaining %d messages", p.Name(), len(messages)-i)
			return fmt.Errorf("shutting down")
		}
		if err := p.processMessage(messages[i]); err != nil {
			log.Errorf("processor [%s] failed to process message %d: %v", p.Name(), i, err)
		}
	}
	return nil
}

// processMessage handles a single attachment ID message.
func (p *ProcessAttachmentsProcessor) processMessage(msg queue.Message) error {
	attachmentID := string(msg.Data)
	if attachmentID == "" {
		log.Warnf("processor [%s] received an empty attachment ID, skipping", p.Name())
		return nil
	}

	// Resolve the sub-pipeline name from global settings.
	pipelineName := ""
	if appCfg := common.AppConfig(); appCfg.DocumentProcessing != nil {
		pipelineName = appCfg.DocumentProcessing.DefaultPipelineForAttachment
	}
	// When no attachment processing pipeline is configured, there is nothing
	// further to do for this attachment. Mark it as completed so that
	// downstream consumers (e.g. chat flows that block on attachment status)
	// do not wait forever for a pipeline that will never run.
	if pipelineName == "" {
		log.Debugf("processor [%s] no attachment pipeline configured, marking attachment [%s] as completed", p.Name(), attachmentID)
		attachmentmod.UpdateAttachmentStats(attachmentID, util.MapStr{
			core.AttachmentStageInitialParsing: core.StatusCompleted,
		})
		return nil
	}

	ormCtx := orm.NewContext()
	ormCtx.DirectReadAccess()
	ormCtx.PermissionScope(security.PermissionScopePlatform)

	// Fetch attachment metadata from Easysearch.
	attachment := core.Attachment{}
	attachment.ID = attachmentID
	exists, err := orm.GetV2(ormCtx, &attachment)
	if err != nil {
		return fmt.Errorf("failed to fetch attachment [%s]: %w", attachmentID, err)
	}
	if !exists || attachment.Deleted {
		log.Debugf("processor [%s] attachment [%s] not found or already deleted, skipping", p.Name(), attachmentID)
		return nil
	}

	// Fetch binary data from the KV blob store.
	data, err := kv.GetValue(core.AttachmentKVBucket, []byte(attachmentID))
	if err != nil || len(data) == 0 {
		log.Warnf("processor [%s] binary data for attachment [%s] not found in blob store, skipping", p.Name(), attachmentID)
		return nil
	}

	// Load and compile the sub-pipeline.
	pipelineCfg := pipeline.PipelineConfigV2{}
	pipelineCfg.ID = pipelineName
	exists, err = orm.GetV2(ormCtx, &pipelineCfg)
	if err != nil || !exists {
		log.Warnf("processor [%s] pipeline [%s] not found (err=%v), skipping attachment [%s]", p.Name(), pipelineName, err, attachmentID)
		return nil
	}

	processorCfgs, err := pipelineCfg.GetProcessorsConfig()
	if err != nil {
		return fmt.Errorf("failed to build processor configs for pipeline [%s]: %w", pipelineName, err)
	}

	procs, err := pipeline.NewPipeline(processorCfgs)
	if err != nil {
		return fmt.Errorf("failed to instantiate pipeline [%s]: %w", pipelineName, err)
	}

	// Mark the attachment as processing before invoking the sub-pipeline.
	attachmentmod.UpdateAttachmentStats(attachmentID, util.MapStr{
		core.AttachmentStageInitialParsing: core.StatusProcessing,
	})

	// Run the sub-pipeline with attachment metadata serialized in []queue.Message
	// and binary data in PipelineContextAttachmentData.
	subCtx := pipeline.AcquireContext(pipelineCfg)
	subCtx.Set(core.PipelineContextDocuments, []queue.Message{{Data: util.MustToJSONBytes(&attachment)}})
	subCtx.Set(core.PipelineContextAttachmentData, data)

	if err := procs.Process(subCtx); err != nil {
		log.Errorf("processor [%s] pipeline [%s] returned error for attachment [%s]: %v — skipping write-back", p.Name(), pipelineName, attachmentID, err)
		attachmentmod.UpdateAttachmentStats(attachmentID, util.MapStr{
			core.AttachmentStageInitialParsing: core.StatusFailed,
		})
		return nil
	}

	// Retrieve the updated metadata set by the sub-pipeline.
	updatedAttachment, ok := subCtx.Get(core.PipelineContextAttachmentMeta).(*core.Attachment)
	if !ok || updatedAttachment == nil {
		log.Warnf("processor [%s] sub-pipeline [%s] produced no updated metadata for attachment [%s], skipping write-back", p.Name(), pipelineName, attachmentID)
		return nil
	}

	// Best-effort guard against a concurrent soft-delete: re-check the deleted
	// flag just before writing back. If the attachment was deleted while the
	// sub-pipeline was running, discard the result.
	//
	// NOTE: This does NOT fully eliminate the race — a TOCTOU (Time-of-Check
	// to Time-of-Use) window still exists between this read and the orm.Update
	// call below. Because the framework provides no attachment-level locking or
	// optimistic-concurrency mechanism (e.g. ES _seq_no / _primary_term), we
	// cannot close this window entirely. The re-check here merely shrinks it to
	// a sub-millisecond gap, making accidental resurrection of a deleted
	// attachment vanishingly unlikely in practice.
	guard := core.Attachment{}
	guard.ID = attachmentID
	exists, err = orm.GetV2(ormCtx, &guard)
	if err != nil {
		return fmt.Errorf("failed to re-check attachment [%s] before write-back: %w", attachmentID, err)
	}
	if !exists || guard.Deleted {
		log.Debugf("processor [%s] attachment [%s] was deleted during processing, discarding write-back", p.Name(), attachmentID)
		return nil
	}

	// Persist the updated metadata.
	writeCtx := orm.NewContext()
	writeCtx.PermissionScope(security.PermissionScopePlatform)
	if err := orm.Update(writeCtx, updatedAttachment); err != nil {
		attachmentmod.UpdateAttachmentStats(attachmentID, util.MapStr{
			core.AttachmentStageInitialParsing: core.StatusFailed,
		})
		return fmt.Errorf("failed to write back updated attachment [%s]: %w", attachmentID, err)
	}

	log.Debugf("processor [%s] attachment [%s] successfully processed and updated", p.Name(), attachmentID)

	// Mark the attachment as completed now that metadata has been persisted.
	attachmentmod.UpdateAttachmentStats(attachmentID, util.MapStr{
		core.AttachmentStageInitialParsing: core.StatusCompleted,
	})

	return nil
}

// Ensure the interface is satisfied at compile time.
var _ pipeline.Processor = (*ProcessAttachmentsProcessor)(nil)
