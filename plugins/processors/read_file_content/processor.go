package read_file_content

import (
	"fmt"
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

func init() {
	pipeline.RegisterProcessorPlugin("read_file_content", New)
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
	fmt.Printf("DBG: read_file_content.New invoked\n")

	cfg := Config{
		MessageField: "messages",
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
	return "read_file_content"
}

func (p *ReadFileContentProcessor) Process(ctx *pipeline.Context) error {
	fmt.Printf("DBG: read_file_content.Process invoked.\n")

	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		fmt.Printf("DBG: read_file_content.Process obj is nil for field: %s\n", p.config.MessageField)
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
