/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
)

// recognizeFacesWithAI uses vision model to identify names for each detected face
func recognizeFacesWithAI(ctx context.Context, processor *FileExtractionProcessor, originalImgPath string, faceImages []string, surroundingText SurroundingText) ([]FaceRecognitionResult, error) {
	// Get vision model
	provider, err := common.GetModelProvider(processor.config.VisionModelProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vision model provider: %w", err)
	}

	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, processor.config.VisionModelName, provider.APIKey, "")

	var parts []llms.ContentPart

	// Add original image
	originalImgPart, err := localImageToDataURI(originalImgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load original image: %w", err)
	}
	parts = append(parts, llms.TextPart("Original image:"))
	parts = append(parts, originalImgPart)
	parts = append(parts, llms.TextPart("\n\n"))

	// Add each face image with numbering
	for i, faceImgPath := range faceImages {
		faceImgPart, err := localImageToDataURI(faceImgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load face image %s: %w", faceImgPath, err)
		}
		parts = append(parts, llms.TextPart(fmt.Sprintf("Face %d:", i)))
		parts = append(parts, faceImgPart)
		parts = append(parts, llms.TextPart("\n"))
	}

	// Build context from surrounding text
	contextHint := ""
	if surroundingText.Before != "" || surroundingText.After != "" {
		contextHint = "\nSurrounding text context:\n"
		if surroundingText.Before != "" {
			contextHint += fmt.Sprintf("Before image: %s\n", surroundingText.Before)
		}
		if surroundingText.After != "" {
			contextHint += fmt.Sprintf("After image: %s\n", surroundingText.After)
		}
	}

	prompt := fmt.Sprintf(`%s
Task: Identify the person's name for each face (numbered 0, 1, 2...) based on the original image and surrounding text context.

Output strict JSON format:
[
    {"face_index": 0, "name": "张三"},
    {"face_index": 1, "name": "李四"},
    {"face_index": 2, "name": ""}
]

CRITICAL CONSTRAINTS:
- You have been provided with exactly %d cropped face images (numbered 0 to %d)
- Your JSON output MUST ONLY contain face_index values from 0 to %d
- Do NOT output face_index values outside this range
- Each face image should have exactly one entry in your response

Notes:
- face_index corresponds to the face number (0, 1, 2...) in the cropped face images provided above
- IMPORTANT: Use the EXACT name from the surrounding text context. Do NOT translate or transliterate names. If the context contains "张三", output "张三" NOT "Zhang San".
- If the person's name cannot be determined from the context, use empty string for name: "name": ""
- The surrounding text context contains names in their original language - extract them directly
- Context hint: These photos are typically manually taken of real people (not AI-generated or heavily edited). In most cases, each face in an image represents a different person. However, use your judgment - if the same person appears multiple times intentionally, reflect that in your answer.
- Important: The cropped face images provided are automated detections and may contain errors. Some cropped regions might not actually contain a human face (false positives). If a cropped image does not clearly show a person's face, return empty string for that face_index instead of guessing.
`, contextHint, len(faceImages), len(faceImages)-1, len(faceImages)-1)

	parts = append(parts, llms.TextPart(prompt))

	content := []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeHuman,
			Parts: parts,
		},
	}

	completion, err := llm.GenerateContent(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("vision model API call failed: %w", err)
	}

	// Parse response
	resp := completion.Choices[0].Content

	// Clean and parse JSON - handle markdown code blocks
	jsonStr := resp
	if strings.HasPrefix(jsonStr, "```json") {
		jsonStr = strings.TrimPrefix(jsonStr, "```json")
		jsonStr = strings.TrimSuffix(jsonStr, "```")
		jsonStr = strings.TrimSpace(jsonStr)
	} else if strings.HasPrefix(jsonStr, "```") {
		jsonStr = strings.TrimPrefix(jsonStr, "```")
		jsonStr = strings.TrimSuffix(jsonStr, "```")
		jsonStr = strings.TrimSpace(jsonStr)
	}

	start := strings.Index(jsonStr, "[")
	end := strings.LastIndex(jsonStr, "]")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no valid JSON array found in response: %s", resp)
	}
	cleanJson := jsonStr[start : end+1]

	var results []FaceRecognitionResult
	if err := json.Unmarshal([]byte(cleanJson), &results); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w (raw: %s)", err, cleanJson)
	}

	return results, nil
}
