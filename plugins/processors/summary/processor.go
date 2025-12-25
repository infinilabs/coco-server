/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package summary

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "document_summarization"

// Users are allowed to set the limit of the summary length, which is a soft limit.
// This is the hard limit.
const SummaryLengthHardLimit = 300

// We set a minimum context length limit, this is a reasonable limit, it is rare
// to see a model whose context length is smaller than this value. Even small
// local LLMs have context of 8k tokens.
const MinimumModelContextLength = 4000

type Config struct {
	MessageField           param.ParaKey      `config:"message_field"`
	OutputQueue            *queue.QueueConfig `config:"output_queue"`
	MinInputDocumentLength uint32             `config:"min_input_document_length"`
	MaxInputDocumentLength uint32             `config:"max_input_document_length"`
	// Soft limit
	MaxSummaryLength uint32 `config:"max_summary_length"`

	ModelProviderID    string `config:"model_provider"`
	ModelName          string `config:"model"`
	ModelContextLength uint32 `config:"model_context_length"`

	PreviousSummaryField       string `config:"previous_summary_field"`
	KeepPreviousSummaryContent bool   `config:"keep_previous_summary_content"`

	IncludeSkippedDocumentToOutputQueue bool `config:"include_skipped_document_to_output_queue"`
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
	cfg := Config{MessageField: core.PipelineContextDocuments, MinInputDocumentLength: 100, MaxInputDocumentLength: 100000, MaxSummaryLength: 10000, IncludeSkippedDocumentToOutputQueue: true}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
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
	if cfg.MaxSummaryLength > SummaryLengthHardLimit {
		log.Warnf("processor [%s] config [MaxSummaryLength] cannot exceed [%d], setting it to [%d]", ProcessorName, SummaryLengthHardLimit, SummaryLengthHardLimit)
		cfg.MaxSummaryLength = SummaryLengthHardLimit
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

	for i := range messages {
		message := &messages[i]
		pop := message.Data
		docLen := uint32(len(pop))

		if docLen > processor.config.MinInputDocumentLength {
			doc := core.Document{}
			err := util.FromJSONBytes(pop, &doc)
			if err != nil {
				log.Error("error on handle document:", i, err)
				continue
			}

			log.Info("start summarize doc: ", doc.ID, ",", doc.Title)
			start := time.Now()
			err = summarizeDocument(llmCtx, &doc, processor.config, llm, processor.removeThinkPattern)
			if err != nil {
				log.Errorf("[%s] failed to summarize document [%s/%s], error [%s]", processor.Name(), doc.Title, doc.ID, err)
				continue
			}
			log.Infof("[%s] finished summarize doc, %v, %v, elapsed: %v, summary: %v", processor.Name(), doc.ID, doc.Title, util.Since(start), doc.Summary)
			message.Data = util.MustToJSONBytes(doc)
		} else {
			if !processor.config.IncludeSkippedDocumentToOutputQueue {
				continue
			}
		}

		// push to output queue
		if processor.outputQueue != nil {
			if err := queue.Push(processor.outputQueue, message.Data); err != nil {
				log.Errorf("failed to push document to [%s]'s output queue: %v", processor.Name(), err)
			}
		}
	}
	return nil
}

// Helper function to set document.Summary to summary and keep if the previous
// summary if needed.
func setSummary(document *core.Document, config *Config, summary string) {
	/*
		Do not discard the previous summary if specified
	*/
	previousSummary := document.Summary
	if previousSummary != "" && config.KeepPreviousSummaryContent && config.PreviousSummaryField != "" {
		document.Payload[config.PreviousSummaryField] = previousSummary
	}

	document.Summary = summary
}

// Main logic of this processor, generate document summary and store it in
// "document.Summary".
func summarizeDocument(ctx context.Context, document *core.Document, config *Config, llm llms.Model, regexpToRemoveThink *regexp.Regexp) error {
	/*
		For local files, excluding the content field as it could be binary
		data, which is not helpful for generating document summary. In cases
		where it is a text string, field `Document.DocumentChunk` already
		provides that, so it is not needed either.
	*/
	if document.Type == connectors.TypeFile {
		documentContent := document.Content
		document.Content = ""

		defer func() {
			document.Content = documentContent
		}()
	}

	/*
		If the length of prompt and document won't exceed model's context
		length, we do it in one pass, i.e., only one LLM call. Otherwise,
		we need to chunk the document and summarize chunks.
	*/
	doneInOnePass, summary, err := trySummarizeDocumentOnePass(ctx, document, config, llm, regexpToRemoveThink)
	if err != nil {
		return err
	}
	if doneInOnePass {
		log.Trace("summary generated in one pass")
		setSummary(document, config, summary)
		return nil
	}

	/*
		Otherwise, we have to chunk the document and invoke multiple LLM calls.
	*/
	log.Trace("summary needs to be generated in multiple passes")
	summary, err = summarizeDocumentMultiPasses(ctx, document, config, llm, regexpToRemoveThink)
	if err != nil {
		return err
	}
	setSummary(document, config, summary)
	return nil
}

func summarizeDocumentMultiPasses(ctx context.Context, document *core.Document, config *Config, llm llms.Model, regexpToRemoveThink *regexp.Regexp) (string, error) {
	chunkSystemPrompt := "You are an expert summarizer. Generate a concise, factual summary."
	chunkUserPromptPrefix := fmt.Sprintf("Summarize the following content without exceeding %d tokens. Focus on the main points and key details.\n\nContent:\n", config.MaxSummaryLength)

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
		return chunkSummaries[0], nil
	}

	/*
		Reduce multiple chunk summaries into a single final summary, recursively
		if needed.
	*/
	log.Tracef("processor [%s] need to summarize summaries", ProcessorName)
	combineSystemPrompt := chunkSystemPrompt
	combineUserPromptPrefix := fmt.Sprintf("You are given summaries of a larger document. Combine them into a concise summary without exceeding %d tokens. Keep only the essential points.\n\nSummaries:\n", config.MaxSummaryLength)
	combineBudget, err := calculateContentBudget(combineSystemPrompt, combineUserPromptPrefix, config.ModelContextLength)
	if err != nil {
		return "", err
	}

	current := chunkSummaries
	for len(current) > 1 {
		log.Debugf("processor [%s]: [%d] summaries to summarize", ProcessorName, len(current))
		grouped := aggregateTexts(current, combineBudget)
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

	return current[0], nil
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

// Try summarizing the document in one pass
func trySummarizeDocumentOnePass(ctx context.Context, document *core.Document, config *Config, llm llms.Model, regexpToRemoveThink *regexp.Regexp) (bool, string, error) {
	const SystemPrompt = "You are an expert summarizer, and your task is to generate a concise summary of the document."

	documentJson := util.MustToJSON(document)
	userPrompt := onePassUserPrompt(config.MaxSummaryLength, documentJson)
	// Check if do it in one pass
	if uint32(len(userPrompt)+len(SystemPrompt)) > config.ModelContextLength {
		// Unfortunately, we cann't.
		return false, "", nil
	}

	/*
		Summary can be generated in one pass
	*/
	message := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, SystemPrompt),
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
		panic(err)
	}
	_ = completion

	summary := summaryBuilder.String()
	summary = regexpToRemoveThink.ReplaceAllLiteralString(summary, "")

	return true, summary, nil
}

// Return the user prompt when we can summarize this document in 1 LLM call.
func onePassUserPrompt(max_token uint32, documentJson string) string {
	return fmt.Sprintf(
		"You are an expert summarizer tasked with summarizing documents. Your "+
			"job is to read the provided document below and generate a clean "+
			"concise, and accurate summary for it, considering all fields "+
			"provided. Make sure your summary reflects the most important points "+
			"from the document as rich as possible without exceeding %v tokens."+
			"\n"+
			"The provided document is in JSON format: %s"+
			"\n"+
			"Please use all of the available fields in the document to generate "+
			"the summary. If any of these fields are missing or incomplete, focus "+
			"on the available ones and fill in the gaps logically, based on "+
			"your understanding of the document."+
			"\n"+
			"Make sure the final summary is clear, concise, and easy to understand."+
			"\n"+
			"No need return how you think.",

		// Arguments
		max_token,
		documentJson,
	)
}
