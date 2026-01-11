/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"context"
	"fmt"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/global"
)

const imageDescriptionPrompt = `You are an expert image analyst. Describe the content of this image in detail.
Focus on:
1. Main subjects and objects visible in the image
2. Colors, composition, and visual elements
3. Text content if any is visible
4. The overall context or purpose of the image

Provide a comprehensive description that would help someone understand what this image contains without seeing it.
Be factual and descriptive. Do not make assumptions about things not visible in the image.`

// processImage processes an image file using a vision model to extract text description.
// Returns an Extraction with the vision model's description as text content.
func (p *FileExtractionProcessor) processImage(ctx context.Context, imagePath string) (Extraction, error) {
	// Get model provider
	provider, err := common.GetModelProvider(p.config.VisionModelProviderID)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to get vision model provider: %w", err)
	}

	// Create LLM client
	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, p.config.VisionModelName, provider.APIKey, "")

	// Convert image to data URI
	imagePart, err := localImageToDataURI(imagePath)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to convert image to data URI: %w", err)
	}

	// Build message with image
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextPart(imageDescriptionPrompt),
				imagePart,
			},
		},
	}

	// Generate description
	var description strings.Builder
	_, err = llm.GenerateContent(ctx, messages, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		if global.ShuttingDown() {
			return fmt.Errorf("shutting down")
		}
		description.Write(chunk)
		return nil
	}))

	if err != nil {
		return Extraction{}, fmt.Errorf("failed to generate image description: %w", err)
	}

	descriptionText := strings.TrimSpace(description.String())
	log.Debugf("generated image description for [%s]: %s", imagePath, truncateString(descriptionText, 100))

	// Return as single page (will be chunked by the caller)
	return Extraction{
		Pages:       []string{descriptionText},
		Attachments: []string{}, // Images don't have attachments
	}, nil
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
