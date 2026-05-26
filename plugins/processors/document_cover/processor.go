/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document_cover

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/coco/plugins/connectors/local_fs"
	"infini.sh/coco/plugins/connectors/s3"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/coco/plugins/processors/fileproc"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

const ProcessorName = "document_cover"

var supportedConnectors = map[string]bool{
	s3.ConnectorS3:            true,
	local_fs.ConnectorLocalFs: true,
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type DocumentCoverProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

type Config struct {
	MessageField param.ParaKey      `config:"message_field"`
	OutputQueue  *queue.QueueConfig `config:"output_queue"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		MessageField: core.PipelineContextDocuments,
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, err
	}

	p := &DocumentCoverProcessor{config: &cfg}
	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}
	return p, nil
}

func (p *DocumentCoverProcessor) Name() string {
	return ProcessorName
}

func (p *DocumentCoverProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		log.Warnf("processor [%s] receives an empty pipeline context", p.Name())
		return nil
	}

	messages, ok := obj.([]queue.Message)
	if !ok {
		return nil
	}

	enqueued := make(map[int]bool)

	for i := range messages {
		if global.ShuttingDown() {
			log.Debugf("[%s] shutting down, skipping remaining %d documents", p.Name(), len(messages)-i)
			return fmt.Errorf("shutting down")
		}

		doc := core.Document{}
		if err := util.FromJSONBytes(messages[i].Data, &doc); err != nil {
			log.Errorf("processor [%s] failed to deserialize document: %s", p.Name(), err)
			continue
		}

		connectorID, err := utils.GetConnectorID(&doc)
		if err != nil {
			log.Warnf("processor [%s] failed to get connector ID for document [%s]: %v", p.Name(), doc.ID, err)
			continue
		}

		if !supportedConnectors[connectorID] || doc.Type != connectors.TypeFile {
			log.Debugf("processor [%s] skipping document [%s/%s]: not a supported file connector [%s]", p.Name(), doc.Title, doc.ID, connectorID)
			continue
		}

		log.Infof("processor [%s] processing file [%s/%s] from connector [%s]", p.Name(), doc.Title, doc.ID, connectorID)
		if err := p.processDocument(ctx.Context, &doc, connectorID); err != nil {
			log.Errorf("processor [%s] failed to process [%s/%s]: %s", p.Name(), doc.Title, doc.ID, err)
			continue
		}

		messages[i].Data = util.MustToJSONBytes(doc)

		if p.outputQueue != nil {
			if err := queue.Push(p.outputQueue, messages[i].Data); err != nil {
				log.Errorf("processor [%s] failed to push document [%s/%s] to output queue: %v", p.Name(), doc.Title, doc.ID, err)
			} else {
				enqueued[i] = true
			}
		}
	}

	if p.outputQueue != nil {
		for i := range messages {
			if !enqueued[i] {
				if err := queue.Push(p.outputQueue, messages[i].Data); err != nil {
					log.Errorf("processor [%s] failed to push skipped document [%d] to output queue: %v", p.Name(), i, err)
				}
			}
		}
	}

	return nil
}

// processDocument downloads the file, generates its cover/thumbnail, then uploads
// the results to the blob store and records the attachment references in doc.
func (p *DocumentCoverProcessor) processDocument(ctx context.Context, doc *core.Document, connectorID string) error {
	tempDir, err := os.MkdirTemp("", "coco-document-cover-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	log.Tracef("[%s] downloading file for [%s/%s]", p.Name(), doc.Title, doc.ID)
	localPath, err := fileproc.DownloadToLocal(ctx, doc, connectorID, tempDir)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	if global.ShuttingDown() {
		return fmt.Errorf("shutting down")
	}

	coverFilename := doc.ID + "_cover.png"
	coverPath := filepath.Join(tempDir, coverFilename)
	thumbnailFilename := doc.ID + "_thumbnail.png"
	thumbnailPath := filepath.Join(tempDir, thumbnailFilename)

	if err := GenerateCoverAndThumbnail(localPath, coverPath, thumbnailPath); err != nil {
		log.Warnf("processor [%s] failed to generate cover for [%s/%s]: %v", p.Name(), doc.Title, doc.ID, err)
		return nil // non-fatal: skip cover but don't fail the pipeline
	}

	log.Tracef("[%s] uploading cover/thumbnail for [%s/%s]", p.Name(), doc.Title, doc.ID)
	coverFile, err := os.Open(coverPath)
	if err != nil {
		log.Warnf("processor [%s] failed to open cover file for [%s/%s]: %v", p.Name(), doc.Title, doc.ID, err)
		return nil
	}

	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.DirectAccess()
	ormCtx.PermissionScope(security.PermissionScopePlatform)
	ownerID := doc.GetOwnerID()

	attachmentID, err := attachment.UploadToBlobStore(ormCtx, "", coverFile, nil, coverFilename, ownerID, nil, "", true)
	if err != nil {
		log.Warnf("processor [%s] failed to upload cover for [%s/%s]: %v", p.Name(), doc.Title, doc.ID, err)
	} else {
		doc.Cover = "attachment://" + attachmentID
		log.Debugf("processor [%s] uploaded cover for [%s/%s]: %s", p.Name(), doc.Title, doc.ID, doc.Cover)
	}

	if fileproc.ContentTypeFromURL(doc.URL) == "image" {
		thumbnailFile, err := os.Open(thumbnailPath)
		if err != nil {
			log.Warnf("processor [%s] failed to open thumbnail for [%s/%s]: %v", p.Name(), doc.Title, doc.ID, err)
		} else {
			thumbnailID, err := attachment.UploadToBlobStore(ormCtx, "", thumbnailFile, nil, thumbnailFilename, ownerID, nil, "", true)
			if err != nil {
				log.Warnf("processor [%s] failed to upload thumbnail for [%s/%s]: %v", p.Name(), doc.Title, doc.ID, err)
			} else {
				doc.Thumbnail = "attachment://" + thumbnailID
				log.Debugf("processor [%s] uploaded thumbnail for [%s/%s]: %s", p.Name(), doc.Title, doc.ID, doc.Thumbnail)
			}
			thumbnailFile.Close()
		}
	}

	return nil
}
