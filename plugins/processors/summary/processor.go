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

type Config struct {
	MessageField               param.ParaKey      `config:"message_field"`
	OutputQueue                *queue.QueueConfig `config:"output_queue"`
	MaxRunningTimeoutInSeconds time.Duration
	MinInputDocumentLength     uint32 `config:"min_input_document_length"`
	MaxInputDocumentLength     uint32 `config:"max_input_document_length"`
	MaxOutputDocumentLength    uint32 `config:"max_output_document_length"`
	ModelProviderID            string `config:"model_provider"`
	ModelName                  string `config:"model"`
	OutputSummaryField         string `config:"output_summary_field"`
	PreviousSummaryField       string `config:"previous_summary_field"`

	KeepPreviousSummaryContent          bool `config:"keep_previous_summary_content"`
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
	cfg := Config{MessageField: core.PipelineContextDocuments, MinInputDocumentLength: 100, MaxInputDocumentLength: 100000, MaxOutputDocumentLength: 10000, IncludeSkippedDocumentToOutputQueue: true}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
	}

	if cfg.MessageField == "" {
		cfg.MessageField = "messages"
	}

	if cfg.ModelProviderID == "" {
		panic(errors.New("model_provider can't be empty"))
	}
	if cfg.ModelName == "" {
		panic(errors.New("model can't be empty"))
	}

	processor := DocumentSummarizationProcessor{config: &cfg}

	if cfg.OutputQueue.Name != "" {
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

			// Create a copy of the document for the prompt, excluding
			// the content field as it could be binary bytes, which is
			// not helpful for generating document summary. In cases where
			// it is a text, field `Document.Text` already provides that,
			// so it is not needed either.
			docForPrompt := doc
			docForPrompt.Content = ""

			promptStr := humanPrompt(processor.config.MaxOutputDocumentLength, docForPrompt)

			content := []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeSystem, "You are an expert summarizer, and your task is to generate a concise summary of the document."),
				llms.TextParts(llms.ChatMessageTypeHuman, promptStr),
			}

			summary := strings.Builder{}
			completion, err := llm.GenerateContent(llmCtx, content, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				if global.ShuttingDown() {
					llmCtx.Done()
					return errors.New("shutting down")
				}
				summary.Write(chunk)
				return nil
			}))
			if err != nil {
				panic(err)
			}
			_ = completion

			text := summary.String()
			text = processor.removeThinkPattern.ReplaceAllString(text, "")

			if len(text) > 0 {
				previousSummary := doc.Summary
				if previousSummary != "" && processor.config.KeepPreviousSummaryContent && processor.config.PreviousSummaryField != "" {
					doc.Payload[processor.config.PreviousSummaryField] = previousSummary
				} else {
					doc.Summary = text
				}
			}
			message.Data = util.MustToJSONBytes(doc)
			log.Infof("[%s] finished summarize doc, %v, %v, elapsed: %v, summary: %v", processor.Name(), doc.ID, doc.Title, util.Since(start), text)
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

// Helper function to construct the human/user prompt.
func humanPrompt(max_token uint32, document core.Document) string {
	docBytes := util.MustToJSONBytes(document)
	return fmt.Sprintf(
		"You are an expert summarizer tasked with summarizing documents. Your "+
			"job is to read the provided information below and generate a clean "+
			"concise, and accurate summary of the document, considering all fields "+
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
		string(docBytes),
	)
}
