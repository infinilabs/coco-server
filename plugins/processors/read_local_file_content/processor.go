/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package read_local_file_content

import (
	"os"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "read_local_file_content"

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type ReadFileContentProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

type Config struct {
	MessageField param.ParaKey      `config:"message_field"`
	OutputQueue  *queue.QueueConfig `config:"output_queue"`
}

func New(c *config.Config) (pipeline.Processor, error) {
	cfg := Config{
		MessageField: core.PipelineContextDocuments,
	}
	if err := c.Unpack(&cfg); err != nil {
		return nil, err
	}

	p := &ReadFileContentProcessor{config: &cfg}

	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}

	return p, nil
}

func (p *ReadFileContentProcessor) Name() string {
	return ProcessorName
}

func (p *ReadFileContentProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		return nil
	}

	messages, ok := obj.([]queue.Message)
	if !ok {
		return nil
	}

	for _, msg := range messages {
		doc := core.Document{}

		docBytes := msg.Data
		err := util.FromJSONBytes(docBytes, &doc)
		if err != nil {
			log.Error("error on handle document:", err)
			continue
		}

		if doc.Type == connectors.TypeFile {
			content, err := os.ReadFile(doc.URL)
			if err != nil {
				log.Errorf("failed to read file content from %s: %v", doc.URL, err)
				continue
			}
			doc.Content = string(content)
			updatedDocBytes := util.MustToJSONBytes(doc)
			msg.Data = updatedDocBytes
		}

		if p.outputQueue != nil {
			if err := queue.Push(p.outputQueue, msg.Data); err != nil {
				log.Error("failed to push document to [%s]'s output queue: %v\n", p.Name(), err)
			}
		}
	}
	return nil
}
