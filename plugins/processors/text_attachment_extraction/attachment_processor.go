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

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/coco/plugins/processors/fileproc"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const AttachmentProcessorName = "attachment_text_extraction"

func init() {
	pipeline.RegisterProcessorPlugin(AttachmentProcessorName, NewAttachmentProcessor)
}

// AttachmentTextExtractionProcessor extracts text from attachment files.
// Unlike DocumentTextAttachmentExtractionProcessor, this processor:
// - Reads serialized core.Attachment from []queue.Message
// - Gets binary data from KV store
// - Only extracts text content (never extracts embedded attachments)
// - Sets extracted text to att.Text and writes the updated attachment back to the message
type AttachmentTextExtractionProcessor struct {
	config *AttachmentConfig
}

// AttachmentConfig holds configuration for the attachment_text_extraction processor.
type AttachmentConfig struct {
	MessageField param.ParaKey `config:"message_field"`

	TikaEndpoint         string `config:"tika_endpoint"`
	TikaTimeoutInSeconds int    `config:"tika_timeout_in_seconds"`

	// Vision model used for image-file description
	VisionModelProviderID string `config:"vision_model_provider"`
	VisionModelName       string `config:"vision_model"`
	ImageContentFormat    string `config:"image_content_format"`

	// BCP 47 language tag for LLM-generated content (e.g. "en-US", "zh-CN")
	LLMGenerationLang string `config:"llm_generation_lang"`
}

func NewAttachmentProcessor(c *config.Config) (pipeline.Processor, error) {
	cfg := AttachmentConfig{
		MessageField:         core.PipelineContextDocuments,
		TikaEndpoint:         "http://127.0.0.1:9998",
		TikaTimeoutInSeconds: 120,
		ImageContentFormat:   "data_uri",
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unpack config of %s processor: %w", AttachmentProcessorName, err)
	}
	if cfg.MessageField == "" {
		cfg.MessageField = core.PipelineContextDocuments
	}

	if cfg.LLMGenerationLang == "" {
		if appCfg := common.AppConfig(); appCfg.DocumentProcessing != nil && appCfg.DocumentProcessing.LLMGenerationLanguage != "" {
			cfg.LLMGenerationLang = appCfg.DocumentProcessing.LLMGenerationLanguage
		}
	}
	cfg.LLMGenerationLang = utils.ValidateAndNormalizeLLMLang(AttachmentProcessorName, cfg.LLMGenerationLang)

	return &AttachmentTextExtractionProcessor{config: &cfg}, nil
}

func (p *AttachmentTextExtractionProcessor) Name() string {
	return AttachmentProcessorName
}

func (p *AttachmentTextExtractionProcessor) Process(ctx *pipeline.Context) error {
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
			log.Debugf("[%s] shutting down, skipping remaining %d attachments", p.Name(), len(messages)-i)
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
func (p *AttachmentTextExtractionProcessor) processMessage(ctx context.Context, msg queue.Message) (*core.Attachment, error) {
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

	// Get binary data from KV store.
	data, err := kv.GetValue(core.AttachmentKVBucket, []byte(att.ID))
	if err != nil || len(data) == 0 {
		log.Warnf("processor [%s] binary data for attachment [%s] not found in blob store, skipping", p.Name(), att.ID)
		return nil, nil
	}

	// Extract text from the attachment.
	text, err := p.extractText(ctx, att, data)
	if err != nil {
		return nil, err
	}

	att.Text = text
	log.Debugf("processor [%s] extracted text for attachment [%s]: %s", p.Name(), att.ID, truncateString(text, 100))
	return att, nil
}

// extractText extracts text content from attachment binary data.
func (p *AttachmentTextExtractionProcessor) extractText(ctx context.Context, att *core.Attachment, data []byte) (string, error) {
	tempDir, err := os.MkdirTemp("", "coco-attachment-text-extraction-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Use original filename for extension detection.
	filename := filepath.Base(att.Name)
	if filename == "" || filename == "." {
		filename = att.ID
	}
	localPath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(localPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write attachment to temp file: %w", err)
	}

	if global.ShuttingDown() {
		return "", fmt.Errorf("shutting down")
	}

	ext := strings.ToLower(filepath.Ext(localPath))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		return p.extractTextFromImage(ctx, localPath)
	default:
		// Use Tika for PDF, DOCX, and other document types.
		return p.extractTextWithTika(ctx, localPath)
	}
}

// extractTextWithTika uses Apache Tika to extract text from a document file.
// This method never extracts embedded attachments (images, etc.) from the document.
func (p *AttachmentTextExtractionProcessor) extractTextWithTika(ctx context.Context, localPath string) (string, error) {
	htmlReader, err := fileproc.TikaGetTextHtml(ctx, p.config.TikaEndpoint, p.config.TikaTimeoutInSeconds, localPath)
	if err != nil {
		return "", fmt.Errorf("failed to extract text using tika: %w", err)
	}
	defer fileproc.DeferClose(htmlReader)

	docHTML, err := goquery.NewDocumentFromReader(htmlReader)
	if err != nil {
		return "", fmt.Errorf("failed to parse tika response: %w", err)
	}

	// Extract page content without processing embedded images.
	var pages []string
	pagesSelection := docHTML.Find("div.page")
	for i := 0; i < pagesSelection.Length(); i++ {
		s := pagesSelection.Eq(i)
		// Remove img tags entirely since we don't extract embedded attachments.
		s.Find("img").Remove()
		pages = append(pages, strings.TrimSpace(s.Text()))
	}
	if len(pages) == 0 {
		s := docHTML.Find("body")
		s.Find("img").Remove()
		pages = append(pages, strings.TrimSpace(s.Text()))
	}

	return strings.Join(pages, " "), nil
}

// extractTextFromImage uses a vision model to generate a text description of the image.
func (p *AttachmentTextExtractionProcessor) extractTextFromImage(ctx context.Context, imagePath string) (string, error) {
	modelId := llmmodule.ResolveModel(core.LLMTypeVision, &core.ModelId{
		ProviderID: p.config.VisionModelProviderID,
		ID:         p.config.VisionModelName,
	})
	if modelId == nil {
		return "", fmt.Errorf("[%s] no vision model configured: set vision_model_provider/vision_model in pipeline config or configure a default vision model in settings", p.Name())
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		return "", fmt.Errorf("failed to get vision model provider: %w", err)
	}

	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, modelId.ID, provider.APIKey, "")

	imagePart, err := fileproc.LoadLocalImageToContentPart(imagePath, p.config.ImageContentFormat)
	if err != nil {
		return "", fmt.Errorf("failed to convert image to content part: %w", err)
	}

	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(buildImageDescriptionPrompt(p.config.LLMGenerationLang)),
				imagePart,
			},
		},
	}

	var description strings.Builder
	_, err = llm.GenerateContent(ctx, messages, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		if global.ShuttingDown() {
			return fmt.Errorf("shutting down")
		}
		description.Write(chunk)
		return nil
	}))
	if err != nil {
		return "", fmt.Errorf("failed to generate image description: %w", err)
	}

	return strings.TrimSpace(description.String()), nil
}

// Ensure the interface is satisfied at compile time.
var _ pipeline.Processor = (*AttachmentTextExtractionProcessor)(nil)
