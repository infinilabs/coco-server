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

	for _, msg := range messages {
		docBytes := msg.Data
		doc := core.Document{}
		err := util.FromJSONBytes(docBytes, &doc)
		if err != nil {
			log.Errorf("processor [%s] failed to deserialize document: %s", processor.Name(), err)
			continue
		}

		// Only local file have this now.
		if doc.Type == connectors.TypeFile && doc.Text != nil {
			embeddings, err := generateEmbedding(doc.Text, processor.config)
			if err != nil {
				log.Errorf("processor [%s] failed to generate embeddings for document [%s/%s] due to error [%s]", processor.Name(), doc.ID, doc.Title, err)
			}
			log.Infof("processor [%s] embeddings of document [%s/%s] generated", processor.Name(), doc.ID, doc.Title)

			doc.Embedding = []core.Embedding{embeddings}
			// Update msg
			updatedDocBytes := util.MustToJSONBytes(doc)
			msg.Data = updatedDocBytes
		}

		if processor.outputQueue != nil {
			if err := queue.Push(processor.outputQueue, msg.Data); err != nil {
				log.Error("failed to push document to [%s]'s output queue: %v\n", processor.Name(), err)
			}
		}
	}

	return nil
}

// Generate embeddings for "pages".
func generateEmbedding(pages []core.PageText, processorConfig *Config) (core.Embedding, error) {
	chunks := make([]core.ChunkEmbedding, 0, 10)
	embedder, err := getEmbedderClient(processorConfig)
	if err != nil {
		return core.Embedding{}, err
	}

	/*
		Split pages texts to text chunks
	*/
	textChunks, chunkRanges := SplitPagesToChunks(pages, processorConfig.ChunkSize)

	/*
		Generate embeddings for text chunks, in batch
	*/
	ctx := context.Background()
	nChunks := len(textChunks)
	batchSize := 10
	for batchStart := 0; batchStart < nChunks; batchStart += batchSize {
		// batchEnd is exclusive
		batchEnd := batchStart + batchSize
		if batchEnd > nChunks {
			batchEnd = nChunks
		}

		batch := textChunks[batchStart:batchEnd]
		embeddings, err := embedder.CreateEmbedding(ctx, batch)
		if err != nil {
			return core.Embedding{}, errors.New(fmt.Sprintf("failed to generated embeddings due to error: %s", err))
		}

		for relative_idx, embedding := range embeddings {
			idx := batchStart + relative_idx
			chunkRange := chunkRanges[idx]

			chunk := core.ChunkEmbedding{
				Range:     chunkRange,
				Embedding: embedding,
			}
			chunks = append(chunks, chunk)
		}
	}

	/*
		Set the corresponding EmbeddingXxx field and return
	*/
	embedding := core.Embedding{
		ModelProvider:      processorConfig.ModelProviderID,
		Model:              processorConfig.ModelName,
		EmbeddingDimension: processorConfig.EmbeddingDimension,
	}
	embedding.SetEmbeddings(chunks)
	return embedding, nil
}

// Splits page texts into chunks using character count as a token proxy
// and tracks the page range for each chunk.
func SplitPagesToChunks(pages []core.PageText, chunkSize int) ([]string, []core.ChunkRange) {
	// Early return
	if chunkSize <= 0 {
		return nil, nil
	}
	if len(pages) == 0 {
		return make([]string, 0), make([]core.ChunkRange, 0)
	}

	var chunks []string
	var ranges []core.ChunkRange

	buf := make([]rune, 0, chunkSize)
	// Value 0 means `startPage`` and `lastPage` are not initialized
	startPage := 0
	lastPage := 0

	for _, page := range pages {
		pageNumber := page.PageNumber
		pageChars := []rune(page.Content)

		for len(pageChars) > 0 {
			nCharsWeWant := chunkSize - len(buf)
			nCharsWeCanTake := min(nCharsWeWant, len(pageChars))
			chars := pageChars[:nCharsWeCanTake]
			buf = append(buf, chars...)

			// Update page range after modifying `buf`
			if startPage == 0 {
				startPage = pageNumber
			}
			if len(buf) == chunkSize && lastPage == 0 {
				lastPage = pageNumber

				// `buf` is ready
				textChunk := string(buf)
				chunkRange := core.ChunkRange{
					Start: startPage,
					End:   lastPage,
				}
				chunks = append(chunks, textChunk)
				ranges = append(ranges, chunkRange)

				// clear buf and states
				buf = buf[:0]
				startPage = 0
				lastPage = 0
			}

			// Remove the consumed bytes from `pageChars`
			pageChars = pageChars[nCharsWeCanTake:]
		}
	}

	// We may have a chunk whose size is smaller than `chunkSize`
	if len(buf) != 0 {
		// startPage should be updated
		if startPage == 0 {
			panic("unreachable: buf got updated but startPage is still 0")
		}
		// Set lastPage
		if lastPage == 0 {
			lastPage = len(pages)
		}

		// `buf` is ready
		textChunk := string(buf)
		chunkRange := core.ChunkRange{
			Start: startPage,
			End:   lastPage,
		}
		chunks = append(chunks, textChunk)
		ranges = append(ranges, chunkRange)
	}

	if len(chunks) != len(ranges) {
		panic("chunks and ranges should have the same length")
	}

	return chunks, ranges
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
