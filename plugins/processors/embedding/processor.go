/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package embedding

import (
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
)

const ProcessorName = "document_embedding"

type Config struct {
	MessageField param.ParaKey `config:"message_field"`
	OutputQueue  struct {
		Name   string                 `config:"name"`
		Labels map[string]interface{} `config:"label" json:"label,omitempty"`
	} `config:"output_queue"`
	Model           string `config:"model"`
	VectorDimension uint32 `config:"vector_dimension"`
}

type DocumentEmbeddingProcessor struct {
	config *Config
	outCfg *queue.QueueConfig
}

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

func New(c *config.Config) (pipeline.Processor, error) {
	panic("todo")
}

func (processor *DocumentEmbeddingProcessor) Name() string {
	return ProcessorName
}

func (processor *DocumentEmbeddingProcessor) Process(ctx *pipeline.Context) error {
	panic("todo")
}
