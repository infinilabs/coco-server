/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package face_extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	"infini.sh/coco/plugins/processors/fileproc"
)

// recognizeFacesWithAI sends the original image and individual face crops to a
// vision model and returns the name identified for each face.
func recognizeFacesWithAI(ctx context.Context, p *FaceExtractionProcessor, originalImgPath string, faceImages []string, surroundingText SurroundingText) ([]FaceRecognitionResult, error) {
	modelId := llmmodule.ResolveModel(core.LLMTypeVision, &core.ModelId{
		ProviderID: p.config.VisionModelProviderID,
		ID:         p.config.VisionModelName,
	})
	if modelId == nil {
		return nil, fmt.Errorf("[%s] no vision model configured: set vision_model_provider/vision_model in pipeline config or configure a default vision model in settings", ProcessorName)
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vision model provider: %w", err)
	}

	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, modelId.ID, provider.APIKey, "")

	var parts []llms.ContentPart

	originalPart, err := fileproc.LoadLocalImageToContentPart(originalImgPath, p.config.ImageContentFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to convert original image: %w", err)
	}
	parts = append(parts, llms.TextPart("Original image:"), originalPart, llms.TextPart("\n\n"))

	for i, faceImgPath := range faceImages {
		facePart, err := fileproc.LoadLocalImageToContentPart(faceImgPath, p.config.ImageContentFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to load face image %s: %w", faceImgPath, err)
		}
		parts = append(parts, llms.TextPart(fmt.Sprintf("Face %d:", i)), facePart, llms.TextPart("\n"))
	}

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
		{Role: llms.ChatMessageTypeHuman, Parts: parts},
	}

	completion, err := llm.GenerateContent(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("vision model API call failed: %w", err)
	}

	resp := completion.Choices[0].Content

	// Strip optional markdown code fence
	jsonStr := resp
	if strings.HasPrefix(jsonStr, "```json") {
		jsonStr = strings.TrimPrefix(jsonStr, "```json")
		jsonStr = strings.TrimSuffix(strings.TrimSpace(jsonStr), "```")
	} else if strings.HasPrefix(jsonStr, "```") {
		jsonStr = strings.TrimPrefix(jsonStr, "```")
		jsonStr = strings.TrimSuffix(strings.TrimSpace(jsonStr), "```")
	}

	start := strings.Index(jsonStr, "[")
	end := strings.LastIndex(jsonStr, "]")
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no valid JSON array found in response: %s", resp)
	}

	var results []FaceRecognitionResult
	if err := json.Unmarshal([]byte(jsonStr[start:end+1]), &results); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w (raw: %s)", err, jsonStr[start:end+1])
	}
	return results, nil
}
