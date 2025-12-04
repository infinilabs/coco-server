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
	config      *Config
	outputQueue *queue.QueueConfig
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
		cfg.MessageField = "messages"
	}

	if cfg.ModelProviderID == "" {
		panic(errors.New("model_provider can't be empty"))
	}
	if cfg.ModelName == "" {
		panic(errors.New("model can't be empty"))
	}

	processor := DocumentEmbeddingProcessor{config: &cfg}

	if cfg.OutputQueue != nil {
		processor.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}

	return &processor, nil
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

		doc := core.Document{}
		err := util.FromJSONBytes(pop, &doc)
		if err != nil {
			log.Error("error on handle document:", i, err)
			continue
		}

		// Skip if text is too short or too long
		if len(doc.Text) < processor.config.MinInputDocumentLength {
			log.Debugf("skipping document %s: text length %d < min %d", doc.ID, len(doc.Text), processor.config.MinInputDocumentLength)
			continue
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
				panic(fmt.Sprintf("failed to generate embeddings: %s\n", err))
			}

			// Convert []float32 to []float64
			var embedding64 []float64
			for _, v := range embeddings[0] {
				embedding64 = append(embedding64, float64(v))
			}
			doc.Embedding = embedding64
			message.Data = util.MustToJSONBytes(doc)
			log.Infof("finished embedding doc, %v, %v, elapsed: %v, dims: %v", doc.ID, doc.Title, util.Since(start), len(doc.Embedding))
		}

		//push to output queue
		if processor.outputQueue != nil {
			if err := queue.Push(processor.outputQueue, message.Data); err != nil {
				log.Errorf("failed to push document to [%s]'s output queue: %v", processor.Name(), err)
			}
		}
	}

	return nil
}
