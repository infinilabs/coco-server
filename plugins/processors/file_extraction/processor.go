/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"context"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "file_extraction"

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type FileExtractionProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

type Config struct {
	MessageField     param.ParaKey      `config:"message_field"`
	OutputQueue      *queue.QueueConfig `config:"output_queue"`
	TikaEndpoint     string             `config:"tika_endpoint"`
	TimeoutInSeconds int                `config:"timeout_in_seconds"`
	ChunkSize        int                `config:"chunk_size"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		MessageField:     core.PipelineContextDocuments,
		TikaEndpoint:     "http://127.0.0.1:9998",
		TimeoutInSeconds: 120,
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, err
	}

	p := &FileExtractionProcessor{config: &cfg}

	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}

	return p, nil
}

func (p *FileExtractionProcessor) Name() string {
	return ProcessorName
}

func (p *FileExtractionProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		log.Warnf("processor [%s] receives an empty pipeline context", p.Name())
		return nil
	}

	messages, ok := obj.([]queue.Message)
	if !ok {
		return nil
	}

	for i := range messages {
		msg := &messages[i]
		doc := core.Document{}

		docBytes := msg.Data
		err := util.FromJSONBytes(docBytes, &doc)
		if err != nil {
			log.Error("processor [%s] failed to deserialize document from bytes: [%s]", p.Name(), err)
			continue
		}

		if doc.Type == connectors.TypeFile {
			log.Infof("processor [%s] extract file [%s]'s content", p.Name(), doc.Title)
			err = p.processLocalFile(ctx.Context, &doc)
			if err != nil {
				log.Errorf("processor [%s] failed to extract file [%s]'s content: %s", p.Name(), doc.Title, err)
				continue
			}
			// Update msg.Data with the new document content
			updatedDocBytes := util.MustToJSONBytes(doc)
			msg.Data = updatedDocBytes
		}

		if p.outputQueue != nil {
			if err := queue.Push(p.outputQueue, msg.Data); err != nil {
				log.Errorf("failed to push document to [%s]'s output queue: %v", p.Name(), err)
			}
		}
	}
	return nil
}

func (p *FileExtractionProcessor) processLocalFile(ctx context.Context, doc *core.Document) error {
	path := doc.URL
	ext := strings.ToLower(filepath.Ext(path))

	var pagesWithoutOcr []string
	var pagesWithOcr []string
	var error error

	switch ext {
	case ".pdf":
		pagesWithoutOcr, pagesWithOcr, error = p.processPdf(ctx, doc)
	case ".pptx", ".ppt", ".pptm":
		pagesWithoutOcr, pagesWithOcr, error = p.processPptx(ctx, doc)
	default:
		// Use the PDF implementation as a fallback, as it uses Tika for extracting
		// both text and attachment, which should work with many file types, though
		// it may not work well.
		pagesWithoutOcr, pagesWithOcr, error = p.processPdf(ctx, doc)
	}

	if error != nil {
		return error
	}

	doc.Chunks = SplitPagesToChunks(pagesWithoutOcr, p.config.ChunkSize)
	doc.ChunksWithImageContent = SplitPagesToChunks(pagesWithOcr, p.config.ChunkSize)

	return nil
}
