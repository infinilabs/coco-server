/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_metadata

import (
	"context"
	"fmt"

	"os"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/coco/plugins/connectors/local_fs"
	"infini.sh/coco/plugins/connectors/s3"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/coco/plugins/processors/fileproc"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "file_metadata"

var supportedConnectors = map[string]bool{
	s3.ConnectorS3:            true,
	local_fs.ConnectorLocalFs: true,
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type FileMetadataProcessor struct {
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

	p := &FileMetadataProcessor{config: &cfg}
	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}
	return p, nil
}

func (p *FileMetadataProcessor) Name() string {
	return ProcessorName
}

func (p *FileMetadataProcessor) Process(ctx *pipeline.Context) error {
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

// processDocument executes steps 1–3: download, extract dominant colors, record dimensions.
func (p *FileMetadataProcessor) processDocument(ctx context.Context, doc *core.Document, connectorID string) error {
	tempDir, err := os.MkdirTemp("", "coco-file-metadata-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Step 1: Download file
	log.Tracef("[%s] step 1/3: downloading file for [%s/%s]", p.Name(), doc.Title, doc.ID)
	localPath, err := fileproc.DownloadToLocal(ctx, doc, connectorID, tempDir)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	log.Debugf("processor [%s] file downloaded to [%s]", p.Name(), localPath)

	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}

	if global.ShuttingDown() {
		return fmt.Errorf("shutting down")
	}

	contentType := fileproc.ContentTypeFromURL(doc.URL)

	// Step 2+3: Extract dominant colors and dimensions (images only, single decode)
	log.Tracef("[%s] step 2/3: extracting image metadata for [%s/%s]", p.Name(), doc.Title, doc.ID)
	if contentType == "image" {
		img, err := loadImageFile(localPath)
		if err != nil {
			log.Warnf("processor [%s] failed to load image [%s/%s]: %v", p.Name(), doc.Title, doc.ID, err)
		} else {
			colors, err := ExtractDominantColors(img)
			if err != nil {
				log.Warnf("processor [%s] failed to extract colors for [%s/%s]: %v", p.Name(), doc.Title, doc.ID, err)
			} else {
				doc.Metadata["colors"] = colors
				log.Debugf("processor [%s] extracted colors for [%s/%s]: %v", p.Name(), doc.Title, doc.ID, colors)
			}

			bounds := img.Bounds()
			doc.Metadata["width"] = bounds.Dx()
			doc.Metadata["height"] = bounds.Dy()
			log.Debugf("processor [%s] extracted dimensions for [%s/%s]: %dx%d", p.Name(), doc.Title, doc.ID, bounds.Dx(), bounds.Dy())
		}
	}

	return nil
}
