/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package text_attachment_extraction

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	"infini.sh/coco/plugins/processors/fileproc"
	"infini.sh/framework/core/global"
)

// buildImageDescriptionPrompt returns a prompt that asks the vision model to
// describe the image in the given language.
func buildImageDescriptionPrompt(lang string) string {
	return fmt.Sprintf(`You are an expert image analyst. Describe the content of this image in detail.
Focus on:
1. Main subjects and objects visible in the image
2. Colors, composition, and visual elements
3. Text content if any is visible
4. The overall context or purpose of the image

Provide a comprehensive description that would help someone understand what this image contains without seeing it.
Be factual and descriptive. Do not make assumptions about things not visible in the image.

IMPORTANT: Your response MUST be in %s.`, lang)
}

// processImage uses a vision model to generate a text description of the image
// and returns it as a single-page Extraction.
func (p *TextAttachmentExtractionProcessor) processImage(ctx context.Context, imagePath string) (fileproc.Extraction, error) {
	modelId := llmmodule.ResolveModel(core.LLMTypeVision, &core.ModelId{
		ProviderID: p.config.VisionModelProviderID,
		ID:         p.config.VisionModelName,
	})
	if modelId == nil {
		return fileproc.Extraction{}, fmt.Errorf("[%s] no vision model configured: set vision_model_provider/vision_model in pipeline config or configure a default vision model in settings", p.Name())
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		return fileproc.Extraction{}, fmt.Errorf("failed to get vision model provider: %w", err)
	}

	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, modelId.ID, provider.APIKey, "")

	imagePart, err := fileproc.LoadLocalImageToContentPart(imagePath, p.config.ImageContentFormat)
	if err != nil {
		return fileproc.Extraction{}, fmt.Errorf("failed to convert image to content part: %w", err)
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
		return fileproc.Extraction{}, fmt.Errorf("failed to generate image description: %w", err)
	}

	descriptionText := strings.TrimSpace(description.String())
	log.Debugf("generated image description for [%s]: %s", imagePath, truncateString(descriptionText, 100))

	return fileproc.Extraction{
		Pages:       []string{descriptionText},
		Attachments: []string{},
	}, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
