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
	MaxSummaryLength       uint32             `config:"max_summary_length"`

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
	llmCtx := context.Background()

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
			err = summarizeDocument(&doc, processor.config, llm, llmCtx, processor.removeThinkPattern)
			if err != nil {
				log.Errorf("[%s] failed to summarize document [%s/%s]", processor.Name(), doc.Title, doc.ID)
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
func summarizeDocument(document *core.Document, config *Config, llm llms.Model, llmCtx context.Context, regexpToRemoveThink *regexp.Regexp) error {
	/*
		excluding the content field as it could be binary data, which is
		not helpful for generating document summary. In cases where it is
		a text string, field `Document.Text` already provides that, so
		it is not needed either.

		NOTE: you should restore this field whenever you return from this
		function
	*/
	documentContent := document.Content
	document.Content = ""

	/*
		If the length of prompt and document won't exceed model's context
		length, we do it in one pass, i.e., only one LLM call. Otherwise,
		we need to chunk the document and summarize chunks.
	*/
	doneInOnePass, summary, err := trySummarizeDocumentOnePass(document, config, llm, llmCtx, regexpToRemoveThink)
	if err != nil {
		// restore the Content field
		document.Content = documentContent
		return err
	}
	if doneInOnePass {
		setSummary(document, config, summary)
		// restore the Content field
		document.Content = documentContent
		return nil
	}

	/*
		Otherwise, we have to chunk the document and invoke multiple LLM calls.
	*/
	summary, err = summarizeDocumentTwoPasses(document, config, llm, llmCtx, regexpToRemoveThink)
	if err != nil {
		// restore the Content field
		document.Content = documentContent
		return err
	}
	setSummary(document, config, summary)
	// restore the Content field
	document.Content = documentContent
	return nil
}

func summarizeDocumentTwoPasses(document *core.Document, config *Config, llm llms.Model, llmCtx context.Context, regexpToRemoveThink *regexp.Regexp) (string, error) {
	return "", nil
}

// Summarize the passed text chunk
func summarizeChunk(llm llms.Model, llmCtx context.Context, regexpToRemoveThink *regexp.Regexp, chunk string, chunkRange core.ChunkRange, tokenLimit uint32) (string, error) {
	userPrompt := fmt.Sprintf(
		"Summarize pages %d-%d concisely within %d tokens. Focus on key facts and main ideas. Content:\n%s",
		chunkRange.Start, chunkRange.End, tokenLimit, chunk,
	)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an expert summarizer."),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	builder := strings.Builder{}
	_, err := llm.GenerateContent(llmCtx, messages, llms.WithStreamingFunc(func(ctx context.Context, data []byte) error {
		if global.ShuttingDown() {
			llmCtx.Done()
			return errors.New("shutting down")
		}
		builder.Write(data)
		return nil
	}))
	if err != nil {
		return "", err
	}

	summary := regexpToRemoveThink.ReplaceAllLiteralString(builder.String(), "")
	return summary, nil
}

// Try summarizing the document in one pass
func trySummarizeDocumentOnePass(document *core.Document, config *Config, llm llms.Model, llmCtx context.Context, regexpToRemoveThink *regexp.Regexp) (bool, string, error) {
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
	completion, err := llm.GenerateContent(llmCtx, message, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
		if global.ShuttingDown() {
			llmCtx.Done()
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
