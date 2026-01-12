/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/coco/plugins/connectors/local_fs"
	"infini.sh/coco/plugins/connectors/s3"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "file_extraction"

// Supported connector IDs for file extraction
var supportedConnectors = map[string]bool{
	s3.ConnectorS3:            true,
	local_fs.ConnectorLocalFs: true,
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type FileExtractionProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

type Config struct {
	MessageField     param.ParaKey      `config:"message_field"`
	OutputQueue      *queue.QueueConfig `config:"output_queue"`
	TikaEndpoint     string             `config:"tika_endpoint"`
	TimeoutInSeconds int                `config:"timeout_in_seconds"`
	ChunkSize        int                `config:"chunk_size"`

	// Vision model configuration for image processing
	VisionModelProviderID string `config:"vision_model_provider"`
	VisionModelName       string `config:"vision_model"`

	// Pigo face finder configuration for face detection
	PigoFacefinderPath string `config:"pigo_facefinder_path"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	// Default values
	cfg := Config{
		MessageField:     core.PipelineContextDocuments,
		TikaEndpoint:     "http://127.0.0.1:9998",
		TimeoutInSeconds: 120,
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, err
	}

	p := &FileExtractionProcessor{config: &cfg}

	/*
		Validate configuration
	*/
	if p.config.VisionModelProviderID == "" || p.config.VisionModelName == "" {
		panic(fmt.Sprintf("Processor [%s]: Vision model is not configured", ProcessorName))
	}

	// Validate pigo_facefinder_path
	if cfg.PigoFacefinderPath == "" {
		panic(fmt.Sprintf("Processor [%s] pigo_facefinder_path not set", ProcessorName))
	}
	if _, err := os.Stat(cfg.PigoFacefinderPath); os.IsNotExist(err) {
		panic(fmt.Sprintf("Processor [%s] pigo_facefinder_path file does not exist: %s", ProcessorName, cfg.PigoFacefinderPath))
	}

	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}

	return p, nil
}

func (p *FileExtractionProcessor) Name() string {
	return ProcessorName
}

func (p *FileExtractionProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		log.Warnf("processor [%s] receives an empty pipeline context", p.Name())
		return nil
	}

	messages, ok := obj.([]queue.Message)
	if !ok {
		return nil
	}

	for i := range messages {
		// Check shutdown before processing each document
		if global.ShuttingDown() {
			log.Debugf("[%s] shutting down, skipping remaining %d documents", p.Name(), len(messages)-i)
			return fmt.Errorf("shutting down")
		}

		doc := core.Document{}

		docBytes := messages[i].Data
		err := util.FromJSONBytes(docBytes, &doc)
		if err != nil {
			log.Errorf("processor [%s] failed to deserialize document from bytes: [%s]", p.Name(), err)
			continue
		}

		// Check if document is from a supported connector
		connectorID, err := utils.GetConnectorID(&doc)
		if err != nil {
			log.Warnf("processor [%s] failed to get connector ID for document [%s]: %v", p.Name(), doc.ID, err)
			continue
		}

		// Only process [file] documents from [local_fs] and [s3]
		if !supportedConnectors[connectorID] || doc.Type != connectors.TypeFile {
			log.Debugf("processor [%s] skipping document [%s/%s] as it is not a [file] that come from [local_fs/s3], connector [%s]", p.Name(), doc.ID, doc.Title, connectorID)
			continue
		}

		log.Infof("processor [%s] processing file [%s] from connector [%s]", p.Name(), doc.Title, connectorID)
		err = p.processDocument(ctx.Context, &doc, connectorID)
		if err != nil {
			log.Errorf("processor [%s] failed to process file [%s]: %s", p.Name(), doc.Title, err)
			continue
		}

		// Update messages[i].Data in-place with the new document content
		messages[i].Data = util.MustToJSONBytes(doc)
	}

	// Push all processed messages to output queue in batch
	if p.outputQueue != nil {
		for i := range messages {
			if err := queue.Push(p.outputQueue, messages[i].Data); err != nil {
				log.Errorf("failed to push document to [%s]'s output queue: %v", p.Name(), err)
			}
		}
	}

	return nil
}

// processDocument is the main processing logic for a document.
//
// It performs the following steps:
//
// 1. Download/copy file to temp directory
// 2. Extract dominant colors (for images)
// 3. Store image width/height
// 4. Generate and upload cover
// 5. Extract text and attachments
//
// If a step fails, we skip that step and continue with the rest if we can, try
// our best to do perform all the steps.
func (p *FileExtractionProcessor) processDocument(ctx context.Context, doc *core.Document, connectorID string) error {
	// Create temp directory for processing
	tempDir, err := os.MkdirTemp("", "file-extraction-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	/*
	 * Step 1: Download/copy file to temp directory
	 */
	docLocalPath, err := p.downloadToLocal(ctx, doc, connectorID, tempDir)
	if err != nil {
		// The following steps are dependent on this, if this fails, error out.
		return fmt.Errorf("failed to download file to local: %w", err)
	}
	log.Debugf("processor [%s] file downloaded to [%s]", p.Name(), docLocalPath)

	// Initialize metadata if nil as we need to write to it.
	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}

	// Check shutdown before proceeding with extraction steps
	if global.ShuttingDown() {
		log.Debugf("[%s] shutting down, skipping extraction steps for document [%s]", p.Name(), doc.Title)
		return fmt.Errorf("shutting down")
	}

	/*
	 * Step 2: Extract dominant colors (for images)
	 */
	contentType, _ := doc.Metadata["content_type"].(string)
	if contentType == "image" {
		img, err := loadImageFile(docLocalPath)
		if err != nil {
			log.Warnf("processor [%s] failed to load image for color extraction [%s]: %v", p.Name(), doc.Title, err)
		} else {
			colors, err := ExtractDominantColors(img)
			if err != nil {
				log.Warnf("processor [%s] failed to extract colors for [%s]: %v", p.Name(), doc.Title, err)
			} else {
				doc.Metadata["colors"] = colors
				log.Debugf("processor [%s] extracted colors for [%s]: %v", p.Name(), doc.Title, colors)
			}
		}
	}

	// Check shutdown before next step
	if global.ShuttingDown() {
		log.Debugf("[%s] shutting down, skipping extraction steps for document [%s]", p.Name(), doc.Title)
		return fmt.Errorf("shutting down")
	}

	/*
	 * Step 3: Store image width/height
	 */
	if contentType == "image" {
		img, err := loadImageFile(docLocalPath)
		if err != nil {
			log.Warnf("processor [%s] failed to load image for dimension extraction [%s]: %v", p.Name(), doc.Title, err)
		} else {
			bounds := img.Bounds()
			doc.Metadata["width"] = bounds.Dx()
			doc.Metadata["height"] = bounds.Dy()
			log.Debugf("processor [%s] extracted dimensions for [%s]: %dx%d", p.Name(), doc.Title, bounds.Dx(), bounds.Dy())
		}
	}

	// Check shutdown before next step
	if global.ShuttingDown() {
		log.Debugf("[%s] shutting down, skipping extraction steps for document [%s]", p.Name(), doc.Title)
		return fmt.Errorf("shutting down")
	}

	/*
	 * Step 4: Generate and upload cover and thumbnail (if it is an image)
	 */
	coverFilename := doc.ID + "_cover.png"
	coverPath := filepath.Join(tempDir, coverFilename)
	thumbnailFilename := doc.ID + "_thumbnail.png"
	thumbnailPath := filepath.Join(tempDir, thumbnailFilename)
	err = GenerateCoverAndThumbnail(docLocalPath, coverPath, thumbnailPath)
	if err != nil {
		log.Warnf("processor [%s] failed to generate cover for [%s]: %v", p.Name(), doc.Title, err)
	} else {
		// Open cover file for upload
		coverFile, err := os.Open(coverPath)
		if err != nil {
			log.Warnf("processor [%s] failed to open cover file for [%s]: %v", p.Name(), doc.Title, err)
		} else {
			// Create ORM context with direct access (required for background processor)
			ormCtx := orm.NewContextWithParent(ctx)
			ormCtx.DirectAccess()

			// Get owner ID from document
			ownerID := doc.GetOwnerID()

			// Upload to blob store
			attachmentID, err := attachment.UploadToBlobStore(ormCtx, "", coverFile, coverFilename, ownerID, nil, "", true)
			if err != nil {
				log.Warnf("processor [%s] failed to upload cover for [%s]: %v", p.Name(), doc.Title, err)
			} else {
				// Set cover as attachment reference
				doc.Cover = "attachment://" + attachmentID
				log.Debugf("processor [%s] uploaded cover for [%s]: %s", p.Name(), doc.Title, doc.Cover)
			}

			// Upload thumbnail as well, only images have thumbnail
			if contentType == "image" {
				thumbnailFile, err := os.Open(thumbnailPath)
				if err != nil {
					log.Warnf("processor [%s] failed to open thumbnail file for [%s]: %v", p.Name(), doc.Title, err)
				} else {
					thumbnailAttachmentID, err := attachment.UploadToBlobStore(ormCtx, "", thumbnailFile, thumbnailFilename, ownerID, nil, "", true)
					if err != nil {
						log.Warnf("processor [%s] failed to upload thumbnail for [%s]: %v", p.Name(), doc.Title, err)
					} else {
						// Set thumbnail as attachment reference
						doc.Thumbnail = "attachment://" + thumbnailAttachmentID
						log.Debugf("processor [%s] uploaded thumbnail for [%s]: %s", p.Name(), doc.Title, doc.Thumbnail)
					}
					thumbnailFile.Close()
				}
			}
		}
	}

	// Check shutdown before next step
	if global.ShuttingDown() {
		log.Debugf("[%s] shutting down, skipping extraction steps for document [%s]", p.Name(), doc.Title)
		return fmt.Errorf("shutting down")
	}

	/*
	 * Step 5: Extract text and attachments
	 */
	err = p.extractTextAndAttachment(ctx, doc, docLocalPath)
	if err != nil {
		log.Warnf("processor [%s] failed to extract text/attachments for [%s]: %v", p.Name(), doc.Title, err)
	} else {
		log.Debugf("processor [%s] extracted text/attachments for [%s]: %v", p.Name(), doc.Title)
	}

	// Check shutdown before next step
	if global.ShuttingDown() {
		log.Debugf("[%s] shutting down, skipping extraction steps for document [%s]", p.Name(), doc.Title)
		return fmt.Errorf("shutting down")
	}

	/*
	 * Step 6: Extract faces and recognize names (optional, skip on error)
	 */
	allowedTypes := map[string]bool{
		"image": true,
		"pptx":  true,
		"xlsx":  true,
		"pdf":   true,
		"docx":  true,
	}
	if allowedTypes[contentType] {
		err = p.extractFacesAndRecognizeNames(ctx, doc, docLocalPath, contentType)
		if err != nil {
			log.Warnf("processor [%s] failed to extract faces for [%s]: %v", p.Name(), doc.Title, err)
		}
	}

	return nil
}

// downloadToLocal downloads or copies [doc]'s file to the local temp
// directory, then returns the its local path.
func (p *FileExtractionProcessor) downloadToLocal(ctx context.Context, doc *core.Document, connectorID string, tempDir string) (string, error) {
	// Determine local file path
	fileName := filepath.Base(doc.URL)
	if fileName == "" || fileName == "." {
		fileName = doc.ID + filepath.Ext(doc.URL)
	}
	localPath := filepath.Join(tempDir, fileName)

	switch connectorID {
	case s3.ConnectorS3:
		return p.downloadFromS3Connector(ctx, doc, localPath)
	case local_fs.ConnectorLocalFs:
		// For local files, doc.URL is the file path
		if err := copyLocalFile(doc.URL, localPath); err != nil {
			return "", fmt.Errorf("failed to copy local file: %w", err)
		}
		return localPath, nil
	default:
		return "", fmt.Errorf("unsupported connector: %s", connectorID)
	}
}

// downloadFromS3Connector downloads a file from S3 using the datasource connector configuration.
func (p *FileExtractionProcessor) downloadFromS3Connector(ctx context.Context, doc *core.Document, localPath string) (string, error) {
	// Get datasource to access connector config
	ds, err := utils.GetDatasource(doc)
	if err != nil {
		return "", fmt.Errorf("failed to get datasource: %w", err)
	}

	// Parse connector config
	connectorConfig, ok := ds.Connector.Config.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid connector config type")
	}

	// Extract S3 configuration
	cfg := S3Config{
		Endpoint:        getStringFromMap(connectorConfig, "endpoint"),
		AccessKeyID:     getStringFromMap(connectorConfig, "access_key_id"),
		SecretAccessKey: getStringFromMap(connectorConfig, "secret_access_key"),
		Bucket:          getStringFromMap(connectorConfig, "bucket"),
		UseSSL:          getBoolFromMap(connectorConfig, "use_ssl"),
	}

	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.Bucket == "" {
		log.Errorf("Processor [%s]: datasource [%s]'s s3 connector configuration is incomplete: %v", p.Name(), ds.ID, cfg)
		return "", fmt.Errorf("incomplete S3 configuration")
	}

	// Parse object key from URL
	// URL format: http://{bucket}.{endpoint}/{key} or s3://{bucket}.{endpoint}/{key}
	objectKey, err := parseS3ObjectKey(doc.URL, cfg.Bucket, cfg.Endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse S3 object key from URL: %w", err)
	}

	// Download the file
	if err := downloadFromS3(ctx, &cfg, objectKey, localPath); err != nil {
		return "", err
	}

	return localPath, nil
}

// parseS3ObjectKey extracts the object key from an S3 URL.
func parseS3ObjectKey(url, bucket, endpoint string) (string, error) {
	// Try format: http(s)://{bucket}.{endpoint}/{key}
	prefix := fmt.Sprintf("http://%s.%s/", bucket, endpoint)
	if strings.HasPrefix(url, prefix) {
		return strings.TrimPrefix(url, prefix), nil
	}

	prefix = fmt.Sprintf("https://%s.%s/", bucket, endpoint)
	if strings.HasPrefix(url, prefix) {
		return strings.TrimPrefix(url, prefix), nil
	}

	// Try format: s3://{bucket}.{endpoint}/{key}
	prefix = fmt.Sprintf("s3://%s.%s/", bucket, endpoint)
	if strings.HasPrefix(url, prefix) {
		return strings.TrimPrefix(url, prefix), nil
	}

	// Fallback: try to extract from path component
	// URL might be in format: http://{endpoint}/{bucket}/{key}
	prefix = fmt.Sprintf("http://%s/%s/", endpoint, bucket)
	if strings.HasPrefix(url, prefix) {
		return strings.TrimPrefix(url, prefix), nil
	}

	prefix = fmt.Sprintf("https://%s/%s/", endpoint, bucket)
	if strings.HasPrefix(url, prefix) {
		return strings.TrimPrefix(url, prefix), nil
	}

	return "", fmt.Errorf("unable to parse object key from URL: %s", url)
}

// getStringFromMap safely extracts a string from a map
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getBoolFromMap safely extracts a bool from a map
func getBoolFromMap(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// Extraction represents the result of extracting content from a document.
type Extraction struct {
	// Pages contains the text content of each page in the document.
	Pages []string
	// For every attachment contained in the document, we create one [core.Attachment]
	// for it. This field contains their IDs.
	Attachments []string
}

// extractTextAndAttachment extracts text content and attachments from a document.
// This is the renamed processLocalFile function with support for image processing.
func (p *FileExtractionProcessor) extractTextAndAttachment(ctx context.Context, doc *core.Document, localPath string) error {
	ext := strings.ToLower(filepath.Ext(localPath))

	var extraction Extraction
	var err error

	switch ext {
	case ".pdf":
		extraction, err = p.processPdf(ctx, doc, localPath)
	case ".pptx", ".ppt", ".pptm":
		extraction, err = p.processPptx(ctx, doc, localPath)
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		// Use vision model for image description
		extraction, err = p.processImage(ctx, localPath)
	default:
		// Use the PDF implementation as a fallback, as it uses Tika for extracting
		// both text and attachment, which should work with many file types, though
		// it may not work well.
		extraction, err = p.processPdf(ctx, doc, localPath)
	}

	if err != nil {
		return err
	}

	doc.Chunks = SplitPagesToChunks(extraction.Pages, p.config.ChunkSize)
	doc.Attachments = extraction.Attachments

	return nil
}
