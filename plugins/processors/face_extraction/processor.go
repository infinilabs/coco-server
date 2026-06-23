/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package face_extraction

import (
	"context"
	"fmt"
	"os"

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

const ProcessorName = "face_extraction"

var supportedConnectors = map[string]bool{
	s3.ConnectorS3:            true,
	local_fs.ConnectorLocalFs: true,
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type FaceExtractionProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

type Config struct {
	MessageField param.ParaKey      `config:"message_field"`
	OutputQueue  *queue.QueueConfig `config:"output_queue"`

	// Tika is used to extract embedded images from compound documents (DOCX, PDF…)
	TikaEndpoint         string `config:"tika_endpoint"`
	TikaTimeoutInSeconds int    `config:"tika_timeout_in_seconds"`

	// PigoFacefinderPath is optional. When empty, face extraction is skipped.
	PigoFacefinderPath string `config:"pigo_facefinder_path"`

	// Vision model for face recognition
	VisionModelProviderID string `config:"vision_model_provider"`
	VisionModelName       string `config:"vision_model"`
	ImageContentFormat    string `config:"image_content_format"`

	// BCP 47 language tag for LLM-generated content
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

	if cfg.PigoFacefinderPath == "" {
		log.Warnf("processor [%s] pigo_facefinder_path is not set; face extraction will be skipped", ProcessorName)
	} else if _, err := os.Stat(cfg.PigoFacefinderPath); os.IsNotExist(err) {
		log.Warnf("processor [%s] pigo_facefinder_path [%s] does not exist; face extraction will be skipped", ProcessorName, cfg.PigoFacefinderPath)
		cfg.PigoFacefinderPath = ""
	}

	p := &FaceExtractionProcessor{config: &cfg}
	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}
	return p, nil
}

func (p *FaceExtractionProcessor) Name() string {
	return ProcessorName
}

func (p *FaceExtractionProcessor) Process(ctx *pipeline.Context) error {
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

var allowedContentTypes = map[string]bool{
	"image": true,
	"pptx":  true,
	"xlsx":  true,
	"pdf":   true,
	"docx":  true,
}

func (p *FaceExtractionProcessor) processDocument(ctx context.Context, doc *core.Document, connectorID string) error {
	// Skip if pigo is not configured
	if p.config.PigoFacefinderPath == "" {
		log.Debugf("processor [%s] skipping [%s/%s]: pigo_facefinder_path not configured", p.Name(), doc.Title, doc.ID)
		return nil
	}

	contentType := fileproc.ContentTypeFromURL(doc.URL)
	if !allowedContentTypes[contentType] {
		log.Debugf("processor [%s] skipping [%s/%s]: file type [%s] not eligible for face extraction", p.Name(), doc.Title, doc.ID, contentType)
		return nil
	}

	tempDir, err := os.MkdirTemp("", "coco-face-extraction-*")
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

	return p.extractFacesAndRecognizeNames(ctx, doc, localPath, contentType)
}
