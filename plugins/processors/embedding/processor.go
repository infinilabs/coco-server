/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package embedding

import (
	"context"
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/embeddings"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	utils "infini.sh/coco/plugins/processors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/event"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "document_embedding"

type Config struct {
	MessageField    param.ParaKey      `config:"message_field"`
	OutputQueue     *queue.QueueConfig `config:"output_queue"`
	ModelProviderID string             `config:"model_provider"`
	ModelName       string             `config:"model"`
}

type DocumentEmbeddingProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

func init() {
	pipeline.RegisterProcessorPluginWithConfigMetadata(ProcessorName, New, Config{})
	pipeline.RegisterProcessorContract(ProcessorName, utils.DocumentRecordContract(pipeline.GranularityBatchCapable))
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

	processor := DocumentEmbeddingProcessor{config: &cfg}

	if cfg.OutputQueue != nil {
		processor.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}

	return &processor, nil
}

func (processor *DocumentEmbeddingProcessor) Name() string {
	return ProcessorName
}

// embedDocument is the per-document core shared by both conventions.
// generateEmbedding resolves its own model config lazily (docs without
// chunks skip it entirely).
func (processor *DocumentEmbeddingProcessor) embedDocument(ctx context.Context, doc *core.Document) (bool, error) {
	// langchaingo's embedding HTTP client has no overall timeout and the
	// pipeline context carries no deadline — bound the call like every
	// other pipeline LLM step, or one stalled provider connection hangs
	// the whole serial consumer (observed live).
	llmCtx, llmCancel := utils.PipelineLLMContext(ctx)
	modified, errs := generateEmbedding(llmCtx, doc, processor.config)
	llmCancel()
	if len(errs) != 0 {
		if modified {
			log.Errorf("processor [%s] failed to generate embeddings for document [%s/%s] _partially_ due to error [%v]", processor.Name(), doc.Title, doc.ID, errs)
		} else {
			log.Errorf("processor [%s] failed to generate embeddings for document [%s/%s] due to error [%v]", processor.Name(), doc.Title, doc.ID, errs)
		}
	}
	if modified {
		log.Infof("processor [%s] embeddings of document [%s/%s] generated", processor.Name(), doc.Title, doc.ID)
	}
	return modified, nil
}

// ProcessBatch: record-convention batch channel.
func (processor *DocumentEmbeddingProcessor) ProcessBatch(ctx *pipeline.Context, records []*event.Event) error {
	return utils.EachDocumentRecord(ctx.Context, records, func(dctx context.Context, doc *core.Document) (bool, error) {
		if global.ShuttingDown() {
			return false, errors.New("shutting down")
		}
		return processor.embedDocument(dctx, doc)
	})
}

func (processor *DocumentEmbeddingProcessor) Process(ctx *pipeline.Context) error {
	// record-scope fallback: batch_capable degrades to per-record when nested
	if rec, ok := pipeline.CurrentRecord(ctx); ok {
		return utils.EachDocumentRecord(ctx.Context, []*event.Event{rec}, func(dctx context.Context, doc *core.Document) (bool, error) {
			return processor.embedDocument(dctx, doc)
		})
	}

	obj := ctx.Get(processor.config.MessageField)

	if obj == nil {
		log.Warnf("processor [%s] receives an empty pipeline context", processor.Name())
		return nil
	}

	messages, ok := obj.([]queue.Message)
	if !ok {
		log.Warnf("processor [%s] context value is not []queue.Message", processor.Name())
		return nil
	}
	if global.Env().IsDebug {
		log.Tracef("processor [%s] get %v messages from context", processor.Name(), len(messages))
	}

	if len(messages) == 0 {
		return nil
	}

	// Track which documents have been enqueued
	enqueued := make(map[int]bool)

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

		modified, _ := processor.embedDocument(ctx.Context, &doc) // logging inside

		if modified {
			// Update messages[i].Data if doc gets modified
			messages[i].Data = util.MustToJSONBytes(doc)

			// Enqueue immediately after processing
			if processor.outputQueue != nil {
				if err := queue.Push(processor.outputQueue, messages[i].Data); err != nil {
					log.Errorf("processor [%s] failed to push document [%s/%s] to output queue: %v", processor.Name(), doc.Title, doc.ID, err)
				} else {
					enqueued[i] = true
				}
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

// Generate embeddings for [document.Chunks] and [document.AiInsights.Embedding].
//
// Return a bool value indicating if `document` gets modified or not.
func generateEmbedding(ctx context.Context, document *core.Document, processorConfig *Config) (bool, []error) {
	var modified bool
	var finalErrs []error

	// nothing to embed — skip before resolving any model configuration
	// (skip-only batches must not require embedder readiness)
	if len(document.Chunks) == 0 && document.AiInsights.Text == "" {
		return false, nil
	}

	embedder, modelId, err := getEmbedderClient(processorConfig)
	if err != nil {
		return false, []error{err}
	}

	chunkGenerated, err := generateChunkEmbeddings(ctx, embedder, document.Chunks)
	if chunkGenerated {
		modified = true
	}
	if err != nil {
		finalErrs = append(finalErrs, err)
	}

	// Generate embedding for document.AiInsights.Text
	if document.AiInsights.Text != "" {
		embedding, err := embedder.CreateEmbedding(ctx, []string{document.AiInsights.Text})
		if err != nil {
			finalErrs = append(finalErrs, err)
		}
		if len(embedding) > 0 {
			if err := validateEmbeddingDimension(embedding[0], "ai_insights"); err != nil {
				finalErrs = append(finalErrs, err)
			} else {
				document.AiInsights.Embedding.SetValue(embedding[0])
				modified = true
			}
		}
	}

	// Stamp the embedding model identity (W1): a later default-model change
	// can detect stale vectors per document and offer batch reprocessing
	// instead of silently mixing models in one index.
	if modified {
		document.EmbeddingModel = modelId.ProviderID + "/" + modelId.ID
	}

	return modified, finalErrs
}

// generateChunkEmbeddings generates embeddings for document chunks.
//
// Return a bool value indicating if `chunks` get modified or not.
func generateChunkEmbeddings(ctx context.Context, embedder embeddings.EmbedderClient, chunks []core.DocumentChunk) (bool, error) {
	nChunks := len(chunks)
	if nChunks == 0 {
		return false, nil
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
			return false, errors.New(fmt.Sprintf("failed to generated embeddings due to error: %s", err))
		}

		for relativeIdx, embedding := range embeddings {
			idx := batchStart + relativeIdx
			if err := validateEmbeddingDimension(embedding, "chunk"); err != nil {
				return false, err
			}
			embeddingWrapper := core.Embedding{}
			embeddingWrapper.SetValue(embedding)
			chunks[idx].Embedding = embeddingWrapper
		}
	}

	return true, nil
}

// helper function to verify that an embedding vector has the dimension
// required by Coco semantic search, so that documents with wrong-sized
// vectors are rejected instead of being indexed silently.
func validateEmbeddingDimension(embedding []float32, source string) error {
	if len(embedding) != core.RequiredEmbeddingDimension {
		return fmt.Errorf("embedding dimension mismatch for %s: got %d, want %d", source, len(embedding), core.RequiredEmbeddingDimension)
	}
	return nil
}

// According to the specified configuration, init the "EmbedderClient" and
// return it, along with the resolved model identity (for the W1 embedding
// model stamp).
func getEmbedderClient(cfg *Config) (embeddings.EmbedderClient, *core.ModelId, error) {
	modelId := llmmodule.ResolveModel(core.LLMTypeEmbedding, &core.ModelId{ProviderID: cfg.ModelProviderID, ID: cfg.ModelName})
	if modelId == nil {
		return nil, nil, fmt.Errorf("no embedding model configured: set model_provider/model in pipeline config or configure a default embedding model in settings")
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		log.Error("failed to get model provider: ", err)
		return nil, nil, err
	}

	model := langchain.GetEmbeddingLLM(provider.BaseURL, provider.APIType, modelId.ID, provider.APIKey, core.RequiredEmbeddingDimension)
	// Check if the LLM client supports embeddings
	embedder, ok := model.(embeddings.EmbedderClient)

	if !ok {
		errorMsg := fmt.Sprintf("Model [%s/%s] does not support embeddings", cfg.ModelProviderID, cfg.ModelName)
		log.Error(errorMsg)
		return nil, nil, errors.New(errorMsg)
	}

	return embedder, modelId, nil
}
