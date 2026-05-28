/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package text_attachment_extraction

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
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

const ProcessorName = "text_attachment_extraction"

var supportedConnectors = map[string]bool{
	s3.ConnectorS3:            true,
	local_fs.ConnectorLocalFs: true,
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type TextAttachmentExtractionProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

// Config holds configuration for the text_attachment_extraction processor.
// TikaEndpoint / TikaTimeoutInSeconds are only used when processing file types
// that require Apache Tika (e.g. PDF, DOCX).  PPTX and plain images are
// handled without Tika.
type Config struct {
	MessageField param.ParaKey      `config:"message_field"`
	OutputQueue  *queue.QueueConfig `config:"output_queue"`

	TikaEndpoint         string `config:"tika_endpoint"`
	TikaTimeoutInSeconds int    `config:"tika_timeout_in_seconds"`
	ChunkSize            int    `config:"chunk_size"`

	// ExtractAttachments controls whether embedded attachments (images, etc.)
	// are extracted from documents. When false, attachment extraction, OCR,
	//把这个 package 的文件夹的名字改成 text attachment extraction，package 的名字也改成这个 image markers, and upload are all skipped. Defaults to true.
	ExtractAttachments *bool `config:"extract_attachments"`

	// Vision model used for image-file description
	VisionModelProviderID string `config:"vision_model_provider"`
	VisionModelName       string `config:"vision_model"`
	ImageContentFormat    string `config:"image_content_format"`

	// BCP 47 language tag for LLM-generated content (e.g. "en-US", "zh-CN")
	LLMGenerationLang string `config:"llm_generation_lang"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		MessageField:         core.PipelineContextDocuments,
		TikaEndpoint:         "http://127.0.0.1:9998",
		TikaTimeoutInSeconds: 120,
		ImageContentFormat:   "data_uri",
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, err
	}

	if cfg.LLMGenerationLang == "" {
		if appCfg := common.AppConfig(); appCfg.DocumentProcessing != nil && appCfg.DocumentProcessing.LLMGenerationLanguage != "" {
			cfg.LLMGenerationLang = appCfg.DocumentProcessing.LLMGenerationLanguage
		}
	}
	cfg.LLMGenerationLang = utils.ValidateAndNormalizeLLMLang(ProcessorName, cfg.LLMGenerationLang)

	if cfg.ChunkSize <= 0 {
		panic(fmt.Sprintf("processor [%s] configuration [chunk_size] is not set or invalid, should be a positive number", ProcessorName))
	}

	// Default ExtractAttachments to true if not explicitly set.
	if cfg.ExtractAttachments == nil {
		defaultTrue := true
		cfg.ExtractAttachments = &defaultTrue
	}

	p := &TextAttachmentExtractionProcessor{config: &cfg}
	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}
	return p, nil
}

func (p *TextAttachmentExtractionProcessor) Name() string {
	return ProcessorName
}

func (p *TextAttachmentExtractionProcessor) Process(ctx *pipeline.Context) error {
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

func (p *TextAttachmentExtractionProcessor) processDocument(ctx context.Context, doc *core.Document, connectorID string) error {
	tempDir, err := os.MkdirTemp("", "coco-text-extraction-*")
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

	if err := p.extractTextAndAttachment(ctx, doc, localPath); err != nil {
		return err
	}

	log.Debugf("processor [%s] extracted text/attachments for [%s/%s]", p.Name(), doc.Title, doc.ID)
	return nil
}

// extractTextAndAttachment dispatches to the correct extractor based on file extension.
func (p *TextAttachmentExtractionProcessor) extractTextAndAttachment(ctx context.Context, doc *core.Document, localPath string) error {
	ext := strings.ToLower(filepath.Ext(localPath))

	var (
		extraction fileproc.Extraction
		err        error
	)

	switch ext {
	case ".pdf":
		extraction, err = p.processPdf(ctx, doc, localPath)
	case ".pptx", ".ppt", ".pptm":
		extraction, err = p.processPptx(ctx, doc, localPath)
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		extraction, err = p.processImage(ctx, localPath)
	default:
		// Tika handles most document types; fall back to the PDF/Tika code path.
		extraction, err = p.processPdf(ctx, doc, localPath)
	}

	if err != nil {
		return err
	}

	doc.Chunks = fileproc.SplitPagesToChunks(extraction.Pages, p.config.ChunkSize)
	doc.Attachments = extraction.Attachments
	doc.Content = strings.Join(extraction.Pages, " ")
	return nil
}
