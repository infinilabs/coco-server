/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package extract_tags

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "extract_tags"

const MinimumModelContextLength = 4000

type Config struct {
	MessageField param.ParaKey      `config:"message_field"`
	OutputQueue  *queue.QueueConfig `config:"output_queue"`

	ModelProviderID    string `config:"model_provider"`
	ModelName          string `config:"model"`
	ModelContextLength uint32 `config:"model_context_length"`
}

type ExtractTagsProcessor struct {
	config             *Config
	outputQueue        *queue.QueueConfig
	removeThinkPattern *regexp.Regexp
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{MessageField: core.PipelineContextDocuments}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of %s processor: %s", ProcessorName, err)
	}

	if cfg.MessageField == "" {
		cfg.MessageField = core.PipelineContextDocuments
	}

	if cfg.ModelProviderID == "" {
		panic("model_provider can't be empty")
	}
	if cfg.ModelName == "" {
		panic("model can't be empty")
	}
	if cfg.ModelContextLength < MinimumModelContextLength {
		panic("Model's context length is too low")
	}

	processor := ExtractTagsProcessor{config: &cfg}

	if cfg.OutputQueue != nil {
		processor.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}
	processor.removeThinkPattern = regexp.MustCompile(`(?s)`)
	return &processor, nil
}

func (processor *ExtractTagsProcessor) Name() string {
	return ProcessorName
}

func (processor *ExtractTagsProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(processor.config.MessageField)

	if obj == nil {
		log.Warnf("processor [] receives an empty pipeline context", processor.Name())
		return nil
	}

	messages := obj.([]queue.Message)
	if global.Env().IsDebug {
		log.Tracef("get %v messages from context", len(messages))
	}

	if len(messages) == 0 {
		return nil
	}

	provider, err := common.GetModelProvider(processor.config.ModelProviderID)
	if err != nil {
		log.Error("failed to get model provider:", err)
		return err
	}

	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, processor.config.ModelName, provider.APIKey, "")
	llmCtx, cancelFunc := context.WithCancel(ctx.Context)
	defer cancelFunc()

	// Track which documents have been enqueued
	enqueued := make(map[int]bool)

	for i := range messages {
		// Check shutdown before processing each document
		if global.ShuttingDown() {
			log.Debugf("[%s] shutting down, skipping remaining %d documents", processor.Name(), len(messages)-i)
			return errors.New("shutting down")
		}

		message := &messages[i]
		pop := message.Data

		doc := core.Document{}
		err := util.FromJSONBytes(pop, &doc)
		if err != nil {
			log.Error("error on handle document:", i, err)
			continue
		}

		aiInsights, hasAIInsights := doc.Metadata["ai_insights"]
		if !hasAIInsights {
			log.Debugf("[%s] document [%s/%s] has no ai_insights, skipping tag extraction", processor.Name(), doc.Title, doc.ID)
			continue
		}

		aiInsightsStr, ok := aiInsights.(string)
		if !ok || strings.TrimSpace(aiInsightsStr) == "" {
			log.Debugf("[%s] document [%s/%s] has empty ai_insights, skipping tag extraction", processor.Name(), doc.Title, doc.ID)
			continue
		}

		log.Infof("processor [%s] start extracting tags for document [%s/%s]", processor.Name(), doc.Title, doc.ID)
		start := time.Now()
		tags, err := extractTagsFromInsights(llmCtx, aiInsightsStr, processor.config, llm, processor.removeThinkPattern)
		if err != nil {
			log.Errorf("[%s] failed to extract tags for document [%s/%s], error [%s]", processor.Name(), doc.Title, doc.ID, err)
			continue
		}
		doc.Tags = tags
		log.Infof("[%s] finished extracting tags for doc, %v, %v, elapsed: %v, tags: %v",
			processor.Name(), doc.Title, doc.ID, util.Since(start), tags)
		message.Data = util.MustToJSONBytes(doc)

		// Enqueue immediately after processing
		if processor.outputQueue != nil {
			if err := queue.Push(processor.outputQueue, message.Data); err != nil {
				log.Errorf("processor [%s] failed to push document [%s/%s] to output queue: %v", processor.Name(), doc.Title, doc.ID, err)
			} else {
				enqueued[i] = true
			}
		}
	}

	// Enqueue any documents that were skipped (not enqueued during processing)
	if processor.outputQueue != nil {
		for i := range messages {
			if !enqueued[i] {
				if err := queue.Push(processor.outputQueue, messages[i].Data); err != nil {
					log.Errorf("processor [%s] failed to push skipped document [%d] to output queue: %v", processor.Name(), i, err)
				}
			}
		}
	}

	return nil
}

func extractTagsFromInsights(ctx context.Context, aiInsights string, config *Config, llm llms.Model, regexpToRemoveThink *regexp.Regexp) ([]string, error) {
	systemPrompt := "You are an expert tag extractor. Analyze document insights and extract relevant tags."
	userPrompt := buildTagExtractionPrompt(aiInsights)

	message := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	responseBuilder := strings.Builder{}
	_, err := llm.GenerateContent(ctx, message, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		if global.ShuttingDown() {
			ctx.Done()
			return errors.New("shutting down")
		}
		responseBuilder.Write(chunk)
		return nil
	}))
	if err != nil {
		return nil, err
	}

	response := responseBuilder.String()
	response = regexpToRemoveThink.ReplaceAllLiteralString(response, "")

	tags, err := parseTagsFromResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tags from LLM response: %w", err)
	}

	return normalizeTags(tags), nil
}

func buildTagExtractionPrompt(aiInsights string) string {
	return fmt.Sprintf(
		"Extract 3-8 relevant tags from the following document analysis.\n"+
			"Focus on: main topics, technologies, domains, and key concepts.\n\n"+
			"Requirements:\n"+
			"- Return ONLY a valid JSON array of strings\n"+
			"- Each tag should be 1-3 words\n"+
			"- Tags should be descriptive and specific\n"+
			"- Format: [\"tag1\", \"tag2\", \"tag3\"]\n\n"+
			"Document Analysis:\n%s\n\n"+
			"Generate the JSON array of tags now.",
		aiInsights,
	)
}

func parseTagsFromResponse(response string) ([]string, error) {
	trimmed := strings.TrimSpace(response)

	jsonStart := strings.Index(trimmed, "[")
	jsonEnd := strings.LastIndex(trimmed, "]")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart > jsonEnd {
		return nil, fmt.Errorf("no valid JSON array found in response")
	}

	jsonStr := trimmed[jsonStart : jsonEnd+1]

	var tags []string
	if err := json.Unmarshal([]byte(jsonStr), &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return tags, nil
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(tags))

	for _, tag := range tags {
		normalized := strings.TrimSpace(strings.ToLower(tag))
		if normalized == "" {
			continue
		}

		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}

	return result
}
