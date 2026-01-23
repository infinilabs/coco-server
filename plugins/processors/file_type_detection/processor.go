/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_type_detection

import (
	"path/filepath"
	"strings"

	"mime"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/coco/plugins/connectors/local_fs"
	"infini.sh/coco/plugins/connectors/s3"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "file_type_detection"
const FieldMimeType = "mime_type"
const FieldContentType = "content_type"
const FieldContentCategory = "content_category"

// Supported connector IDs for file type detection
var supportedConnectors = map[string]bool{
	s3.ConnectorS3:            true,
	local_fs.ConnectorLocalFs: true,
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type FileTypeDetectionProcessor struct {
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

	p := &FileTypeDetectionProcessor{config: &cfg}

	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}

	return p, nil
}

func (p *FileTypeDetectionProcessor) Name() string {
	return ProcessorName
}

func (p *FileTypeDetectionProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		log.Warnf("processor [%s] receives an empty pipeline context", p.Name())
		return nil
	}

	messages, ok := obj.([]queue.Message)
	if !ok {
		return nil
	}

	// Track which documents have been enqueued
	enqueued := make(map[int]bool)

	for i := range messages {
		// Check shutdown before processing each document
		if global.ShuttingDown() {
			log.Debugf("[%s] shutting down, skipping remaining %d documents", p.Name(), len(messages)-i)
			return errors.New("shutting down")
		}

		doc := core.Document{}

		docBytes := messages[i].Data
		err := util.FromJSONBytes(docBytes, &doc)
		if err != nil {
			log.Errorf("processor [%s] failed to deserialize document: %v", p.Name(), err)
			continue
		}

		// Only process documents from s3 or local_fs connectors
		connectorID, err := utils.GetConnectorID(&doc)
		if err != nil {
			log.Warnf("processor [%s] failed to get connector ID for document [%s/%s]: %v", p.Name(), doc.Title, doc.ID, err)
			continue
		}

		if !supportedConnectors[connectorID] || doc.Type != connectors.TypeFile {
			log.Debugf("processor [%s] skipping document [%s/%s] as it is not a [file] that come from [local_fs/s3], connector [%s]", p.Name(), doc.Title, doc.ID, connectorID)
			continue
		}

		// Initialize Metadata map if nil
		if doc.Metadata == nil {
			doc.Metadata = make(map[string]interface{})
		}

		// Set MIME type/content type/content category if needed
		if isMetadataEmpty(doc.Metadata, FieldMimeType) || isMetadataEmpty(doc.Metadata, FieldContentType) {
			mimeType, contentType := detectFileTypes(doc.Title)
			doc.Metadata[FieldMimeType] = mimeType
			doc.Metadata[FieldContentType] = contentType
			log.Infof("processor [%s] detected mime_type=%s, content_type=%s for document [%s/%s]", p.Name(), mimeType, contentType, doc.Title, doc.ID)
		}
		if isMetadataEmpty(doc.Metadata, FieldContentCategory) {
			contentCategory := categorizeContentType(getStringFromMetadata(doc.Metadata, FieldContentType))
			doc.Metadata[FieldContentCategory] = contentCategory
			log.Infof("processor [%s] detected content_category=%s for document [%s/%s]", p.Name(), contentCategory, doc.Title, doc.ID)
		}

		// Update message data in-place
		messages[i].Data = util.MustToJSONBytes(doc)

		// Enqueue immediately after processing
		if p.outputQueue != nil {
			if err := queue.Push(p.outputQueue, messages[i].Data); err != nil {
				log.Errorf("processor [%s] failed to push document [%s/%s] to output queue: %v", p.Name(), doc.Title, doc.ID, err)
			} else {
				enqueued[i] = true
			}
		}
	}

	// Enqueue any documents that were skipped (not enqueued during processing)
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

// detectFileTypes returns mime_type and content_type based on file extension.
//
// An empty string means the type is unknown
func detectFileTypes(filename string) (string, string) {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeType := mime.TypeByExtension(ext)
	contentType := getContentType(ext)

	return mimeType, contentType
}

// getContentType returns the coarse-grained content type for a given extension.
// Returns empty string if the extension is not recognized.
func getContentType(ext string) string {
	switch ext {
	// Images
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".tiff", ".tif", ".ico":
		return "image"
	// Videos
	case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".mkv", ".m4v":
		return "video"
	// Markdown
	case ".md", ".markdown":
		return "markdown"
	// PDF
	case ".pdf":
		return "pdf"
	// DOCX (includes .doc for backward compatibility)
	case ".doc", ".docx":
		return "docx"
	// PPTX (includes .ppt for backward compatibility)
	case ".ppt", ".pptx":
		return "pptx"
	// XLSX (includes .xls for backward compatibility)
	case ".xls", ".xlsx":
		return "xlsx"
	default:
		return ""
	}
}

// Categorize content types into content categories.
func categorizeContentType(contentType string) string {
	switch contentType {
	case "image":
		return "image"
	case "video":
		return "video"
	// Markdown
	case "markdown", "pdf", "docx", "pptx", "xlsx":
		return "doc"
	default:
		return ""
	}
}

// isMetadataEmpty returns true if the value is nil, empty string, or not a string type.
// This safely handles map[string]interface{} lookups where the key may not exist
// or the value may be of a different type.
func isMetadataEmpty(metadata map[string]interface{}, key string) bool {
	if metadata == nil {
		return true
	}
	val, exists := metadata[key]
	if !exists {
		return true
	}
	if val == nil {
		return true
	}
	str, ok := val.(string)
	if !ok {
		return true
	}
	return str == ""
}

// getStringFromMetadata safely returns a string value from metadata.
// Returns empty string if key doesn't exist, value is nil, or value is not a string.
func getStringFromMetadata(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	val, exists := metadata[key]
	if !exists {
		return ""
	}
	if val == nil {
		return ""
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}
