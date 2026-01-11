/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package embedding

import (
	"context"
	"fmt"
	"slices"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/embeddings"
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

const ProcessorName = "document_embedding"

type Config struct {
	MessageField       param.ParaKey      `config:"message_field"`
	OutputQueue        *queue.QueueConfig `config:"output_queue"`
	ModelProviderID    string             `config:"model_provider"`
	ModelName          string             `config:"model"`
	EmbeddingDimension int32              `config:"embedding_dimension"`
	ChunkSize          int                `config:"chunk_size"`
}

type DocumentEmbeddingProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
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
	if cfg.EmbeddingDimension == 0 {
		panic("embedding_dimension is not specified or set to 0, which is not allowed")
	}
	if !slices.Contains(core.SupportedEmbeddingDimensions, cfg.EmbeddingDimension) {
		panic(fmt.Sprintf("invalid embedding_dimension, available values %v", core.SupportedEmbeddingDimensions))
	}
	if cfg.ChunkSize == 0 {
		panic("chunk_size is not specified or set to 0, which is not allowed")
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
		log.Tracef("processor [%s] get %v messages from context", processor.Name(), len(messages))
	}

	if len(messages) == 0 {
		return nil
	}

	for i := range messages {
		// Check shutdown before processing each document
		if global.ShuttingDown() {
			log.Debugf("[%s] shutting down, skipping remaining %d documents", processor.Name(), len(messages)-i)
			return errors.New("shutting down")
		}

		docBytes := messages[i].Data
		doc := core.Document{}
		err := util.FromJSONBytes(docBytes, &doc)
		if err != nil {
			log.Errorf("processor [%s] failed to deserialize document: %s", processor.Name(), err)
			continue
		}

		// Only local file have this now.
		if doc.Type == connectors.TypeFile && doc.Chunks != nil {
			err := generateEmbedding(ctx.Context, &doc, processor.config)
			if err != nil {
				log.Errorf("processor [%s] failed to generate embeddings for document [%s/%s] due to error [%s]", processor.Name(), doc.ID, doc.Title, err)
			}
			log.Infof("processor [%s] embeddings of document [%s/%s] generated", processor.Name(), doc.ID, doc.Title)

			// Update messages[i].Data in-place
			messages[i].Data = util.MustToJSONBytes(doc)
		}
	}

	// Push all processed messages to output queue in batch
	if processor.outputQueue != nil {
		for i := range messages {
			if err := queue.Push(processor.outputQueue, messages[i].Data); err != nil {
				log.Error("failed to push document to [%s]'s output queue: %v\n", processor.Name(), err)
			}
		}
	}

	return nil
}

// Generate embeddings for [document.Chunks].
func generateEmbedding(ctx context.Context, document *core.Document, processorConfig *Config) error {
	embedder, err := getEmbedderClient(processorConfig)
	if err != nil {
		return err
	}

	if err := generateChunkEmbeddings(ctx, embedder, document.Chunks); err != nil {
		return err
	}

	return nil
}

func generateChunkEmbeddings(ctx context.Context, embedder embeddings.EmbedderClient, chunks []core.DocumentChunk) error {
	nChunks := len(chunks)
	if nChunks == 0 {
		return nil
	}

	batchSize := 10
	batch := make([]string, 0, batchSize)
	for batchStart := 0; batchStart < nChunks; batchStart += batchSize {
		batch = batch[:0]

		batchEnd := batchStart + batchSize
		if batchEnd > nChunks {
			batchEnd = nChunks
		}

		for _, chunk := range chunks[batchStart:batchEnd] {
			batch = append(batch, chunk.Text)
		}

		embeddings, err := embedder.CreateEmbedding(ctx, batch)
		if err != nil {
			return errors.New(fmt.Sprintf("failed to generated embeddings due to error: %s", err))
		}

		for relativeIdx, embedding := range embeddings {
			idx := batchStart + relativeIdx
			embeddingWrapper := core.Embedding{}
			embeddingWrapper.SetValue(embedding)
			chunks[idx].Embedding = embeddingWrapper
		}
	}

	return nil
}

// According to the specified configuration, init the "EmbedderClient" and
// return it.
func getEmbedderClient(cfg *Config) (embeddings.EmbedderClient, error) {
	provider, err := common.GetModelProvider(cfg.ModelProviderID)
	if err != nil {
		log.Error("failed to get model provider: ", err)
		return nil, err
	}

	model := langchain.GetLLM(provider.BaseURL, provider.APIType, cfg.ModelName, provider.APIKey, "")
	// Check if the LLM client supports embeddings
	embedder, ok := model.(embeddings.EmbedderClient)

	if !ok {
		errorMsg := fmt.Sprintf("Model [%s/%s] does not support embeddings", cfg.ModelProviderID, cfg.ModelName)
		log.Error(errorMsg)
		return nil, errors.New(errorMsg)
	}

	return embedder, nil
}
