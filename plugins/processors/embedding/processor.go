/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package embedding

import (
	"context"
	"fmt"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/embeddings"
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

const ProcessorName = "document_embedding"

type Config struct {
	MessageField           param.ParaKey      `config:"message_field"`
	OutputQueue            *queue.QueueConfig `config:"output_queue"`
	ModelProviderID        string             `config:"model_provider"`
	ModelName              string             `config:"model"`
	MinInputDocumentLength int                `config:"min_input_document_length"`
	MaxInputDocumentLength int                `config:"max_input_document_length"`
}

type DocumentEmbeddingProcessor struct {
	config   *Config
	outCfg   *queue.QueueConfig
	producer queue.ProducerAPI
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{MessageField: core.PipelineContextDocuments, MinInputDocumentLength: 10, MaxInputDocumentLength: 100000}

	if err := c.Unpack(&cfg); err != nil {
		log.Error(err)
		return nil, fmt.Errorf("failed to unpack the configuration of %s processor: %s", ProcessorName, err)
	}

	if cfg.MessageField == "" {
		panic("message field is empty")
	}

	if cfg.OutputQueue.Name == "" {
		panic(errors.New("name of output_queue can't be nil"))
	}
	if cfg.ModelProviderID == "" {
		panic(errors.New("model_provider can't be empty"))
	}
	if cfg.ModelName == "" {
		panic(errors.New("model can't be empty"))
	}

	runner := DocumentEmbeddingProcessor{config: &cfg}

	queueConfig := queue.AdvancedGetOrInitConfig("", cfg.OutputQueue.Name, cfg.OutputQueue.Labels)
	queueConfig.ReplaceLabels(cfg.OutputQueue.Labels)

	producer, err := queue.AcquireProducer(queueConfig)
	if err != nil {
		panic(err)
	}

	runner.outCfg = queue.AdvancedGetOrInitConfig("", cfg.OutputQueue.Name, cfg.OutputQueue.Labels)
	runner.producer = producer

	return &runner, nil
}

func (processor *DocumentEmbeddingProcessor) Name() string {
	return ProcessorName
}

func (processor *DocumentEmbeddingProcessor) Process(ctx *pipeline.Context) error {
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
	c := context.Background()

	// Check if the LLM client supports embeddings
	embedder, ok := llm.(embeddings.EmbedderClient)
	if !ok {
		log.Errorf("Model [%s/%s] does not support embeddings", processor.config.ModelProviderID, processor.config.ModelName)
		return nil
	}

	for i := range messages {
		message := &messages[i]
		pop := message.Data
		var outputBytes []byte

		doc := core.Document{}
		err := util.FromJSONBytes(pop, &doc)
		if err != nil {
			log.Error("error on handle document:", i, err)
			continue
		}

		// Skip if text is too short or too long
		if len(doc.Text) < processor.config.MinInputDocumentLength {
			log.Debugf("skipping document %s: text length %d < min %d", doc.ID, len(doc.Text), processor.config.MinInputDocumentLength)
			// Still push to output queue? Maybe yes, but without embedding.
			// For now, let's push it as is.
			outputBytes = pop
		} else {
			log.Info("start embedding doc: ", doc.ID, ",", doc.Title)
			start := time.Now()

			// Truncate if too long (simple truncation, ideally should chunk)
			textToEmbed := doc.Text
			if len(textToEmbed) > processor.config.MaxInputDocumentLength {
				textToEmbed = textToEmbed[:processor.config.MaxInputDocumentLength]
			}

			embeddings, err := embedder.CreateEmbedding(c, []string{textToEmbed})
			if err != nil {
				log.Errorf("failed to create embedding for doc %s: %v", doc.ID, err)
				// Push original doc without embedding
				outputBytes = pop
			} else if len(embeddings) > 0 {
				// Convert []float32 to []float64
				var embedding64 []float64
				for _, v := range embeddings[0] {
					embedding64 = append(embedding64, float64(v))
				}
				doc.Embedding = embedding64
				outputBytes = util.MustToJSONBytes(doc)
				message.Data = outputBytes
				log.Infof("finished embedding doc, %v, %v, elapsed: %v, dims: %v", doc.ID, doc.Title, util.Since(start), len(doc.Embedding))
			} else {
				outputBytes = pop
			}
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

	return nil
}
