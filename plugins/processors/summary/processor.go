/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package summary

import (
	"context"
	"fmt
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

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

const ProcessorName = "document_summarization"

// We set a minimum context length limit, this is a reasonable limit, it is rare
// to see a model whose context length is smaller than this value. Even small
// local LLMs have context of 8k tokens.
const MinimumModelContextLength = 4000

type Config struct {
	MessageField           param.ParaKey      `config:"message_field"`
	OutputQueue            *queue.QueueConfig `config:"output_queue"`
	MinInputDocumentLength uint32             `config:"min_input_document_length"`
	MaxInputDocumentLength uint32             `config:"max_input_document_length"`

	ModelProviderID     string `config:"model_provider"`
	ModelName           string `config:"model"`
	ModelContextLength  uint32 `config:"model_context_length"`
	AIInsightsMaxLength uint32 `config:"ai_insights_max_length"`
}

type DocumentSummarizationProcessor struct {
	config             *Config
	outputQueue        *queue.QueueConfig
	removeThinkPattern *regexp.Regexp
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{MessageField: core.PipelineContextDocuments, MinInputDocumentLength: 100, MaxInputDocumentLength: 100000, AIInsightsMaxLength: 500}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of %s processor: %s", ProcessorName, err)
	}

	/*
		Validate configuration
	*/
	if cfg.MessageField == "" {
		cfg.MessageField = core.PipelineContextDocuments
	}

	if cfg.ModelProviderID == "" {
		panic("model_provider can't be empty")
	}
	if cfg.ModelName == "" {
		panic("model can't be empty")
	}
	// This is rare, or unreachable in reality. Even small local LLMs have
	// context of 8k tokens.
	if cfg.ModelContextLength < MinimumModelContextLength {
		panic("Model's context length is too low")
	}

	processor := DocumentSummarizationProcessor{config: &cfg}

	if cfg.OutputQueue != nil {
		processor.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}
	// Regular expression to remove <think> content
	processor.removeThinkPattern = regexp.MustCompile(`(?s)<think>.*?</think>`)

	return &processor, nil
}

func (processor *DocumentSummarizationProcessor) Name() string {
	return ProcessorName
}

func (processor *DocumentSummarizationProcessor) Process(ctx *pipeline.Context) error {
	// get message from queue
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
		docLen := uint32(len(pop))

		if docLen >= processor.config.MinInputDocumentLength {
			doc := core.Document{}
			err := util.FromJSONBytes(pop, &doc)
			if err != nil {
				log.Error("error on handle document:", i, err)
				continue
			}

			log.Infof("processor [%s] start summarizing document [%s/%s]", processor.Name(), doc.Title, doc.ID)
			start := time.Now()
			err = summarizeDocument(llmCtx, &doc, processor.config, llm, processor.removeThinkPattern)
			if err != nil {
				log.Errorf("[%s] failed to summarize document [%s/%s], error [%s]", processor.Name(), doc.Title, doc.ID, err)
				continue
			}
			log.Infof("[%s] finished summarizing doc [%s/%s], elapsed: [%v], short_summary: [%v], ai_insights_length: [%v]",
				processor.Name(), doc.Title, doc.ID, util.Since(start), doc.Summary,
				len(fmt.Sprintf("%v", doc.Metadata["ai_insights"])))
			message.Data = util.MustToJSONBytes(doc)

			// Enqueue immediately after processing
			if processor.outputQueue != nil {
				if err := queue.Push(processor.outputQueue, message.Data); err != nil {
					log.Errorf("processor [%s] failed to push document [%s/%s] to output queue: %v", processor.Name(), doc.Title, doc.ID, err)
				} else {
					enqueued[i] = true
				}
			}
		} else {
			// Document was skipped (too short), will be enqueued in final pass
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

// Helper function to set both document.Summary and document.Metadata["ai_insights"]
func setSummaryAndInsights(document *core.Document, shortSummary string, aiInsights string) {
	// Initialize Metadata if nil
	if document.Metadata == nil {
		document.Metadata = make(map[string]interface{})
	}

	document.Summary = shortSummary
	document.Metadata["ai_insights"] = aiInsights
}

// Main logic of this processor, generate AI insights and summary for this document.
func summarizeDocument(ctx context.Context, document *core.Document, config *Config, llm llms.Model, regexpToRemoveThink *regexp.Regexp) error {
	// Stage 1: Generate ai_insights (~500 tokens, Markdown+Mermaid)
	var aiInsights string
	var err error

	doneInOnePass, summary, err := tryGenerateAIInsightsOnePass(ctx, document, config, llm, regexpToRemoveThink)
	if err != nil {
		return err
	}

	if doneInOnePass {
		log.Trace("ai_insights generated in one pass")
		aiInsights = summary
	} else {
		log.Trace("ai_insights needs to be generated in multiple passes")
		aiInsights, err = summarizeDocumentMultiPasses(ctx, document, config, llm, regexpToRemoveThink)
		if err != nil {
			return err
		}
	}

	// Stage 2: Generate short_summary (~50 tokens) from ai_insights
	log.Trace("generating short_summary from ai_insights")
	shortSummary, err := generateShortSummaryFromInsights(ctx, llm, aiInsights, regexpToRemoveThink)
	if err != nil {
		return err
	}

	// Stage 3: Store both in document
	setSummaryAndInsights(document, shortSummary, aiInsights)
	return nil
}

func summarizeDocumentMultiPasses(ctx context.Context, document *core.Document, config *Config, llm llms.Model, regexpToRemoveThink *regexp.Regexp) (string, error) {
	chunkSystemPrompt := "You are an expert summarizer. Generate a concise, factual summary."
	chunkUserPromptPrefix := "Summarize the following content without exceeding 500 tokens. Focus on the main points and key details.\n\nContent:\n"

	chunkBudget, err := calculateContentBudget(chunkSystemPrompt, chunkUserPromptPrefix, config.ModelContextLength)
	if err != nil {
		return "", err
	}

	// Build larger chunks from embedding-sized chunks to better utilize model context.
	log.Tracef("processor [%s] chunking document", ProcessorName)
	originalChunks := make([]string, 0, len(document.Chunks))
	for _, c := range document.Chunks {
		text := strings.TrimSpace(c.Text)
		if text == "" {
			continue
		}
		originalChunks = append(originalChunks, text)
	}
	if len(originalChunks) == 0 {
		return "", fmt.Errorf("document has no chunk text to summarize")
	}

	mergedChunks := aggregateTexts(originalChunks, chunkBudget)
	if len(mergedChunks) == 0 {
		return "", fmt.Errorf("failed to merge document chunks for summarization")
	}

	log.Debugf("processor [%s]: document got split into [%d] chunks", ProcessorName, len(mergedChunks))

	// Map phase: Summarize each chunk
	log.Tracef("processor [%s] summarizing chunks", ProcessorName)
	chunkSummaries := make([]string, 0, len(mergedChunks))
	for idx, chunkText := range mergedChunks {
		log.Tracef("processor [%s] summarizing chunk [%d]", ProcessorName, idx)
		userPrompt := chunkUserPromptPrefix + chunkText
		summary, err := generateSummaryFromPrompt(ctx, llm, chunkSystemPrompt, userPrompt, regexpToRemoveThink)
		if err != nil {
			return "", err
		}
		chunkSummaries = append(chunkSummaries, strings.TrimSpace(summary))
	}

	if len(chunkSummaries) == 0 {
		return "", fmt.Errorf("no summaries generated for document chunks")
	}
	if len(chunkSummaries) == 1 {
		// Single chunk: directly generate analysis with Mermaid
		return generateAIInsightsFromSummary(ctx, llm, chunkSummaries[0], config.AIInsightsMaxLength, regexpToRemoveThink)
	}

	// Reduce phase: Combine summaries recursively
	log.Tracef("processor [%s] need to summarize summaries", ProcessorName)
	combineSystemPrompt := chunkSystemPrompt
	combineUserPromptPrefix := "You are given summaries of a larger document. Combine them into a concise summary without exceeding 500 tokens. Keep only the essential points.\n\nSummaries:\n"
	combineBudget, err := calculateContentBudget(combineSystemPrompt, combineUserPromptPrefix, config.ModelContextLength)
	if err != nil {
		return "", err
	}

	current := chunkSummaries
	for len(current) > 1 {
		log.Debugf("processor [%s]: [%d] summaries to summarize", ProcessorName, len(current))

		grouped := aggregateTexts(current, combineBudget)

		// Final pass: Generate full Markdown+Mermaid analysis
		if len(grouped) == 1 {
			return generateAIInsightsFromSummary(ctx, llm, grouped[0], config.AIInsightsMaxLength, regexpToRemoveThink)
		}

		// Intermediate pass: Plain text summary
		next := make([]string, 0, len(grouped))
		for _, groupText := range grouped {
			userPrompt := combineUserPromptPrefix + groupText
			summary, err := generateSummaryFromPrompt(ctx, llm, combineSystemPrompt, userPrompt, regexpToRemoveThink)
			if err != nil {
				return "", err
			}
			next = append(next, strings.TrimSpace(summary))
		}
		current = next
	}

	// Final single summary: Convert to full analysis
	return generateAIInsightsFromSummary(ctx, llm, current[0], config.AIInsightsMaxLength, regexpToRemoveThink)
}

func calculateContentBudget(systemPrompt, userPromptPrefix string, modelContextLength uint32) (int, error) {
	promptLength := utf8.RuneCountInString(systemPrompt) + utf8.RuneCountInString(userPromptPrefix)
	budget := int(modelContextLength) - promptLength
	if budget <= 0 {
		return 0, fmt.Errorf("model context length is too small for prompts (%d)", promptLength)
	}
	return budget, nil
}

// aggregateTexts merges texts into larger groups without exceeding the provided budget (rune-based).
func aggregateTexts(texts []string, budget int) []string {
	if budget <= 0 {
		return nil
	}

	var groups []string
	var builder strings.Builder
	currentLen := 0
	separator := "\n"
	separatorLen := utf8.RuneCountInString(separator)

	flush := func() {
		if builder.Len() > 0 {
			groups = append(groups, builder.String())
			builder.Reset()
			currentLen = 0
		}
	}

	for _, raw := range texts {
		text := strings.TrimSpace(raw)
		if text == "" {
			continue
		}

		runes := []rune(text)
		textLen := len(runes)

		if textLen > budget {
			flush()
			for len(runes) > 0 {
				take := budget
				if take > len(runes) {
					take = len(runes)
				}
				groups = append(groups, string(runes[:take]))
				runes = runes[take:]
			}
			continue
		}

		needed := textLen
		if currentLen > 0 {
			needed += separatorLen
		}

		if currentLen > 0 && currentLen+needed > budget {
			flush()
		}

		if builder.Len() > 0 {
			builder.WriteString(separator)
			currentLen += separatorLen
		}
		builder.WriteString(text)
		currentLen += textLen
	}

	flush()
	return groups
}

// Base/helper function to do summary generation.
func generateSummaryFromPrompt(ctx context.Context, llm llms.Model, systemPrompt, userPrompt string, regexpToRemoveThink *regexp.Regexp) (string, error) {
	message := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	summaryBuilder := strings.Builder{}
	completion, err := llm.GenerateContent(ctx, message, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		if global.ShuttingDown() {
			ctx.Done()
			return errors.New("shutting down")
		}
		summaryBuilder.Write(chunk)
		return nil
	}))
	if err != nil {
		return "", err
	}
	_ = completion

	summary := summaryBuilder.String()
	summary = regexpToRemoveThink.ReplaceAllLiteralString(summary, "")

	return summary, nil
}

// generateAIInsightsFromSummary converts a plain summary into full Markdown+Mermaid analysis
func generateAIInsightsFromSummary(ctx context.Context, llm llms.Model, summary string, maxLength uint32, regexpToRemoveThink *regexp.Regexp) (string, error) {
	systemPrompt := "You are an expert document analyst. Generate comprehensive analysis in Markdown format with Mermaid mind maps."
	userPrompt := buildFinalAnalysisPrompt(summary, maxLength)
	return generateSummaryFromPrompt(ctx, llm, systemPrompt, userPrompt, regexpToRemoveThink)
}

// generateShortSummaryFromInsights generates a ~50 token summary from the analysis
func generateShortSummaryFromInsights(ctx context.Context, llm llms.Model, aiInsights string, regexpToRemoveThink *regexp.Regexp) (string, error) {
	systemPrompt := "You are an expert summarizer. Generate highly concise summaries."
	userPrompt := buildShortSummaryPrompt(aiInsights)
	return generateSummaryFromPrompt(ctx, llm, systemPrompt, userPrompt, regexpToRemoveThink)
}

// Try generating ai_insights in one pass
func tryGenerateAIInsightsOnePass(ctx context.Context, document *core.Document, config *Config, llm llms.Model, regexpToRemoveThink *regexp.Regexp) (bool, string, error) {
	const SystemPrompt = "You are an expert document analyst. Generate comprehensive analysis in Markdown format with Mermaid mind maps."

	documentJson := util.MustToJSON(document)
	userPrompt := buildAnalysisPrompt(documentJson, config.AIInsightsMaxLength)

	// Check if doable in one pass
	if uint32(len(userPrompt)+len(SystemPrompt)) > config.ModelContextLength {
		return false, "", nil
	}

	message := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, SystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	builder := strings.Builder{}
	_, err := llm.GenerateContent(ctx, message, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		if global.ShuttingDown() {
			ctx.Done()
			return errors.New("shutting down")
		}
		builder.Write(chunk)
		return nil
	}))
	if err != nil {
		return false, "", err
	}

	result := builder.String()
	result = regexpToRemoveThink.ReplaceAllLiteralString(result, "")
	return true, result, nil
}

// buildAnalysisPrompt generates prompt for deep analysis with Mermaid mind map
//
// Used in [tryGenerateAIInsightsOnePass]
func buildAnalysisPrompt(documentJson string, maxLength uint32) string {
	return fmt.Sprintf(
		"You are an expert document analyst. Analyze the following document and generate a comprehensive analysis in Markdown format.\n\n"+
			"Requirements:\n"+
			"- Length: Approximately %d tokens\n"+
			"- Content: Detailed document interpretation including key insights, themes, and relationships\n"+
			"- MUST include: A Mermaid mind map in a code block (```mermaid ... ```) showing the document structure\n\n"+
			"Document JSON:\n%s\n\n"+
			"Generate the analysis now. End with a ```mermaid mindmap``` block.",
		maxLength,
		documentJson,
	)
}

// buildShortSummaryPrompt generates prompt for concise summary from analysis
func buildShortSummaryPrompt(analysis string) string {
	return fmt.Sprintf(
		"You are an expert summarizer. Based on the following document analysis, generate a highly concise summary.\n\n"+
			"Requirements:\n"+
			"- Length: Approximately 50 tokens (1-2 sentences)\n"+
			"- Content: Pure text, no markdown formatting\n"+
			"- Focus: The core message or takeaway\n\n"+
			"Analysis to summarize:\n%s",
		analysis,
	)
}

// buildFinalAnalysisPrompt generates prompt for final Markdown+Mermaid output (from combined summaries)
func buildFinalAnalysisPrompt(combinedSummary string, maxLength uint32) string {
	return fmt.Sprintf(
		"You are an expert document analyst. Transform the following combined summary into a comprehensive Markdown analysis.\n\n"+
			"Requirements:\n"+
			"- Length: Approximately %d tokens\n"+
			"- Content: Detailed interpretation with key insights\n"+
			"- MUST include: A Mermaid mind map in a code block (```mermaid ... ```) showing document structure\n\n"+
			"Combined Summary:\n%s\n\n"+
			"Generate the final analysis with Mermaid mind map.",
		maxLength,
		combinedSummary,
	)
}
