/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package embedding

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms/ollama"
	"infini.sh/coco/core"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"regexp"
	"time"
)

type Config struct {
	MessageField param.ParaKey `config:"message_field"`
	OutputQueue  struct {
		Name   string                 `config:"name"`
		Labels map[string]interface{} `config:"label" json:"label,omitempty"`
	} `config:"output_queue"`
	MaxRunningTimeoutInSeconds          time.Duration
	MinInputDocumentLength              int    `config:"min_input_document_length"`
	MaxInputDocumentLength              int    `config:"max_input_document_length"`
	MaxOutputDocumentLength             int    `config:"max_output_document_length"`
	SummaryModel                        string `config:"model"`
	IncludeSkippedDocumentToOutputQueue bool   `config:"include_skipped_document_to_output_queue"`
}

type DocumentSummaryProcessor struct {
	config             *Config
	outCfg             *queue.QueueConfig
	producer           queue.ProducerAPI
	removeThinkPattern *regexp.Regexp
}

const Name = "document_embedding"

func init() {
	pipeline.RegisterProcessorPlugin(Name, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{MessageField: core.PipelineContextDocuments, MinInputDocumentLength: 100, MaxInputDocumentLength: 100000, MaxOutputDocumentLength: 10000, IncludeSkippedDocumentToOutputQueue: true}

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

	runner := DocumentSummaryProcessor{config: &cfg}

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

func (processor *DocumentSummaryProcessor) Name() string {
	return Name
}

func (processor *DocumentSummaryProcessor) Process(ctx *pipeline.Context) error {

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

				doc := core.Document{}
				err := util.FromJSONBytes(pop, &doc)
				if err != nil {
					log.Error("error on handle document:", i, err)
					continue
				}

				log.Info("start summarize doc: ", doc.ID, ",", doc.Title)
				start := time.Now()

				content := []string{}
				embedding, err := llm.CreateEmbedding(ctx, content)
				if err != nil {
					panic(err)
				}

				if len(embedding) > 0 {
					previousSummary := doc.Summary
					if previousSummary != "" {
						doc.Payload["previous_summary"] = previousSummary
					}
					//doc.Embedding = embedding //TODO
				}

				outputBytes = util.MustToJSONBytes(doc)

				log.Infof("finished embedding doc, %v, %v, elapsed: %v", doc.ID, doc.Title, util.Since(start))
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
