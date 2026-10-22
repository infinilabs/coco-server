/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package generate_cover

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core/codecs"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/coco/plugins/processors/fileproc"
	fwconfig "infini.sh/framework/core/config"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

const AttachmentProcessorName = "generate_attachment_cover"

func init() {
	pipeline.RegisterProcessorPluginWithConfigMetadata(AttachmentProcessorName, NewAttachmentProcessor, AttachmentCoverConfig{})
	pipeline.RegisterProcessorContract(AttachmentProcessorName, utils.AttachmentRecordContract(pipeline.GranularityRecord))
}

// AttachmentCoverConfig holds the configuration for GenerateAttachmentCoverProcessor.
type AttachmentCoverConfig struct {
	MessageField param.ParaKey `config:"message_field"`
}

// GenerateAttachmentCoverProcessor reads serialized attachments from the pipeline context,
// generates cover/thumbnail images, and updates the attachment metadata.
type GenerateAttachmentCoverProcessor struct {
	config *AttachmentCoverConfig
}

func NewAttachmentProcessor(c *fwconfig.Config) (pipeline.Processor, error) {
	cfg := AttachmentCoverConfig{MessageField: core.PipelineContextDocuments}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack config of %s processor: %w", AttachmentProcessorName, err)
	}
	if cfg.MessageField == "" {
		cfg.MessageField = core.PipelineContextDocuments
	}
	return &GenerateAttachmentCoverProcessor{config: &cfg}, nil
}

func (p *GenerateAttachmentCoverProcessor) Name() string {
	return AttachmentProcessorName
}

// ProcessBatch: record-convention channel (attachment namespace). The
// record's attachment subtree is serialized into a synthetic message and
// flows through the exact same processMessage path as the legacy
// convention — one core, two doors.
func (p *GenerateAttachmentCoverProcessor) ProcessBatch(ctx *pipeline.Context, records []*event.Event) error {
	for _, rec := range records {
		if rec == nil {
			continue
		}
		if global.ShuttingDown() {
			return fmt.Errorf("shutting down")
		}
		v, err := rec.Fields.GetValue(codecs.FieldAttachment)
		if err != nil || v == nil {
			continue // not an attachment-family record
		}
		b, err := util.ToJSONBytes(v)
		if err != nil {
			continue
		}
		updated, err := p.processMessage(ctx.Context, queue.Message{Data: b})
		if err != nil {
			log.Errorf("processor [%s] failed to process attachment record: %v", p.Name(), err)
			continue
		}
		if updated == nil {
			continue // skipped
		}
		nb, err := util.ToJSONBytes(updated)
		if err != nil {
			continue
		}
		nm := util.MapStr{}
		if err := util.FromJSONBytes(nb, &nm); err != nil {
			continue
		}
		rec.Fields[codecs.FieldAttachment] = nm
	}
	return nil
}

func (p *GenerateAttachmentCoverProcessor) Process(ctx *pipeline.Context) error {
	// record-scope path (attachment family)
	if rec, ok := pipeline.CurrentRecord(ctx); ok {
		return p.ProcessBatch(ctx, []*event.Event{rec})
	}

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
		updatedAtt, err := p.processMessage(ctx.Context, messages[i])
		if err != nil {
			log.Errorf("processor [%s] failed to process message %d: %v", p.Name(), i, err)
			continue
		}
		// Write the updated attachment back to the message for the caller to persist.
		if updatedAtt != nil {
			messages[i].Data = util.MustToJSONBytes(updatedAtt)
		}
	}
	return nil
}

// processMessage handles a single serialized attachment message.
func (p *GenerateAttachmentCoverProcessor) processMessage(ctx context.Context, msg queue.Message) (*core.Attachment, error) {
	// Deserialize attachment from message data.
	att := &core.Attachment{}
	if err := util.FromJSONBytes(msg.Data, att); err != nil {
		return nil, fmt.Errorf("failed to deserialize attachment: %w", err)
	}
	if att.ID == "" {
		log.Warnf("processor [%s] received an attachment with empty ID, skipping", p.Name())
		return nil, nil
	}
	if att.Deleted {
		log.Debugf("processor [%s] attachment [%s] is marked as deleted, skipping", p.Name(), att.ID)
		return nil, nil
	}

	data, err := kv.GetValue(core.AttachmentKVBucket, []byte(att.ID))
	if err != nil || len(data) == 0 {
		log.Warnf("processor [%s] binary data for attachment [%s] not found in blob store, skipping", p.Name(), att.ID)
		return nil, nil
	}

	// Process the attachment: generate cover and thumbnail.
	if err := p.processAttachment(ctx, att, data); err != nil {
		return nil, err
	}

	log.Debugf("processor [%s] attachment [%s] cover generated", p.Name(), att.ID)
	return att, nil
}

// processAttachment takes binary content already in memory, writes it to a
// temp file, generates a cover and (for image files) a thumbnail, uploads both
// to the blob store, and records the resulting URLs in att.Metadata["cover"]
// and att.Metadata["thumbnail"].
func (p *GenerateAttachmentCoverProcessor) processAttachment(ctx context.Context, att *core.Attachment, data []byte) error {
	tempDir, err := os.MkdirTemp("", "coco-generate-cover-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Use the original filename so that GenerateCoverAndThumbnail can detect
	// the file type from its extension. filepath.Base guards against att.Name
	// being an absolute or relative path (e.g. "../etc/passwd").
	filename := filepath.Base(att.Name)
	if filename == "" || filename == "." {
		filename = att.ID
	}
	localPath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(localPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write attachment to temp file: %w", err)
	}

	if global.ShuttingDown() {
		return fmt.Errorf("shutting down")
	}

	coverFilename := att.ID + "_cover.png"
	coverPath := filepath.Join(tempDir, coverFilename)
	thumbnailFilename := att.ID + "_thumbnail.png"
	thumbnailPath := filepath.Join(tempDir, thumbnailFilename)

	if err := GenerateCoverAndThumbnail(localPath, coverPath, thumbnailPath); err != nil {
		log.Warnf("processor [%s] failed to generate cover for attachment [%s]: %v", p.Name(), att.ID, err)
		return nil // non-fatal
	}

	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.DirectAccess()
	ormCtx.PermissionScope(security.PermissionScopePlatform)

	ownerID := att.GetOwnerID()

	if att.Metadata == nil {
		att.Metadata = make(map[string]interface{})
	}

	coverFile, err := os.Open(coverPath)
	if err != nil {
		log.Warnf("processor [%s] failed to open cover for attachment [%s]: %v", p.Name(), att.ID, err)
		return nil
	}
	coverID, err := attachment.UploadToBlobStore(ormCtx, "", coverFile, nil, coverFilename, ownerID, nil, "", true)
	coverFile.Close()
	if err != nil {
		log.Warnf("processor [%s] failed to upload cover for attachment [%s]: %v", p.Name(), att.ID, err)
	} else {
		att.Metadata["cover"] = "attachment://" + coverID
		log.Debugf("processor [%s] uploaded cover for attachment [%s]: %s", p.Name(), att.ID, att.Metadata["cover"])
	}

	if fileproc.IsImage(filename) {
		thumbnailFile, err := os.Open(thumbnailPath)
		if err != nil {
			log.Warnf("processor [%s] failed to open thumbnail for attachment [%s]: %v", p.Name(), att.ID, err)
		} else {
			thumbnailID, err := attachment.UploadToBlobStore(ormCtx, "", thumbnailFile, nil, thumbnailFilename, ownerID, nil, "", true)
			thumbnailFile.Close()
			if err != nil {
				log.Warnf("processor [%s] failed to upload thumbnail for attachment [%s]: %v", p.Name(), att.ID, err)
			} else {
				att.Metadata["thumbnail"] = "attachment://" + thumbnailID
				log.Debugf("processor [%s] uploaded thumbnail for attachment [%s]: %s", p.Name(), att.ID, att.Metadata["thumbnail"])
			}
		}
	}

	return nil
}

// Ensure the interface is satisfied at compile time.
var _ pipeline.Processor = (*GenerateAttachmentCoverProcessor)(nil)
