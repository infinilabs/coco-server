/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package summary

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	MessageField param.ParaKey `config:"message_field"`
	OutputQueue  struct {
		Name   string                 `config:"name"`
		Labels map[string]interface{} `config:"label" json:"label,omitempty"`
	} `config:"output_queue"`
	MaxRunningTimeoutInSeconds time.Duration
	MinInputDocumentLength     int    `config:"min_input_document_length"`
	MaxInputDocumentLength     int    `config:"max_input_document_length"`
	MaxOutputDocumentLength    int    `config:"max_output_document_length"`
	SummaryModel               string `config:"model"`
	OutputSummaryField         string `config:"output_summary_field"`
	PreviousSummaryField       string `config:"previous_summary_field"`

	KeepPreviousSummaryContent          bool `config:"keep_previous_summary_content"`
	IncludeSkippedDocumentToOutputQueue bool `config:"include_skipped_document_to_output_queue"`
}

type DocumentSummarizationProcessor struct {
	config             *Config
	outCfg             *queue.QueueConfig
	producer           queue.ProducerAPI
	removeThinkPattern *regexp.Regexp
}

func init() {
	pipeline.RegisterProcessorPlugin("document_summarization", New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{MessageField: "messages", MinInputDocumentLength: 100, MaxInputDocumentLength: 100000, MaxOutputDocumentLength: 10000, IncludeSkippedDocumentToOutputQueue: true}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of flow_replay processor: %s", err)
	}

	if cfg.MessageField == "" {
		panic("message field is empty")
	}

	if cfg.OutputQueue.Name == "" {
		panic(errors.New("name of output_queue can't be nil"))
	}
	if cfg.SummaryModel == "" {
		panic(errors.New("summary model can't be empty"))
	}

	runner := DocumentSummarizationProcessor{config: &cfg}

	queueConfig := queue.AdvancedGetOrInitConfig("", cfg.OutputQueue.Name, cfg.OutputQueue.Labels)
	queueConfig.ReplaceLabels(cfg.OutputQueue.Labels)

	producer, err := queue.AcquireProducer(queueConfig)
	if err != nil {
		panic(err)
	}

	runner.outCfg = queue.AdvancedGetOrInitConfig("", cfg.OutputQueue.Name, cfg.OutputQueue.Labels)
	runner.producer = producer
	// Regular expression to remove <think> content
	runner.removeThinkPattern = regexp.MustCompile(`(?s)<think>.*?</think>`)

	return &runner, nil
}

func (processor DocumentSummarizationProcessor) Stop() error {
	return nil
}

func (processor *DocumentSummarizationProcessor) Name() string {
	return "document_enrichment"
}

func (processor *DocumentSummarizationProcessor) Process(ctx *pipeline.Context) error {

	//get message from queue
	obj := ctx.Get(processor.config.MessageField)
	if obj != nil {
		messages := obj.([]queue.Message)
		if global.Env().IsDebug {
			log.Tracef("get %v messages from context", len(messages))
		}

		if len(messages) == 0 {
			return nil
		}

		llm, err := ollama.New(ollama.WithModel(processor.config.SummaryModel))
		if err != nil {
			panic(err)
		}
		ctx := context.Background()

		for i, message := range messages {

			pop := message.Data
			var outputBytes []byte

			if len(pop) > processor.config.MinInputDocumentLength {

				doc := common.Document{}
				err := util.FromJSONBytes(pop, &doc)
				if err != nil {
					log.Error("error on handle document:", i, err)
					continue
				}

				log.Info("start summarize doc: ", doc.ID, ",", doc.Title)
				start := time.Now()

				prompt := fmt.Sprintf(`You are an expert summarizer tasked with summarizing documents. Your job is to read the provided information below and generate a clean, concise, 
and accurate summary of the document, considering all fields provided. Make sure your summary reflects the most important points from the document as rich as possible without exceeding %v tokens.

Please use all of the available fields in the document to generate the summary, the document is in JSON format:
%s

If any of these fields are missing or incomplete, focus on the available ones and fill in the gaps logically, based on your understanding of the document. 
Make sure the final summary is clear, concise, and easy to understand.

No need return how you think.

Summary:`, processor.config.MaxOutputDocumentLength, util.SubStringWithSuffix(string(pop), processor.config.MaxInputDocumentLength, "..."))

				content := []llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeSystem, "You are an expert summarizer, and your task is to generate a concise summary of the document."),
					llms.TextParts(llms.ChatMessageTypeHuman, prompt),
				}

				summary := strings.Builder{}
				completion, err := llm.GenerateContent(ctx, content, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
					if global.ShuttingDown() {
						ctx.Done()
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

				outputBytes = util.MustToJSONBytes(doc)

				log.Infof("finished summarize doc, %v, %v, elapsed: %v, summary: %v", doc.ID, doc.Title, util.Since(start), text)
			} else {

				if !processor.config.IncludeSkippedDocumentToOutputQueue {
					continue
				}

				outputBytes = pop
			}

			if outputBytes == nil {
				panic("invalid output")
			}

			//push to output queue
			r := queue.ProduceRequest{Topic: processor.outCfg.ID, Data: outputBytes}
			res := []queue.ProduceRequest{r}
			_, err = processor.producer.Produce(&res)
			if err != nil {
				panic(errors.Errorf("failed to push message to output queue: %v, %s, offset:%v, size:%v, err:%v", processor.outCfg.Name, processor.outCfg.ID, message.Offset.String(), len(outputBytes), err))
			}
		}
	}
	return nil
}
