package extract_file_text

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/param"
	"infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

const ProcessorName = "extract_file_text"

func init() {
	pipeline.RegisterProcessorPlugin(ProcessorName, New)
}

type ExtractFileTextProcessor struct {
	config      *Config
	outputQueue *queue.QueueConfig
}

type Config struct {
	MessageField     param.ParaKey      `config:"message_field"`
	OutputQueue      *queue.QueueConfig `config:"output_queue"`
	TikaEndpoint     string             `config:"tika_endpoint"`
	TimeoutInSeconds int                `config:"timeout_in_seconds"`
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

	p := &ExtractFileTextProcessor{config: &cfg}

	if cfg.OutputQueue != nil {
		p.outputQueue = queue.SmartGetOrInitConfig(cfg.OutputQueue)
	}

	return p, nil
}

func (p *ExtractFileTextProcessor) Name() string {
	return ProcessorName
}

func (p *ExtractFileTextProcessor) Process(ctx *pipeline.Context) error {
	obj := ctx.Get(p.config.MessageField)
	if obj == nil {
		log.Warnf("processor [] receives an empty pipeline context", p.Name())
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
			log.Error("error on handle document:", err)
			continue
		}

		if doc.Type == connectors.TypeFile {
			// Use Tika Server to extract text
			// We use Accept: text/html to get page boundaries (<div class="page">)

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.config.TimeoutInSeconds)*time.Second)
			defer cancel()

			file, err := os.Open(doc.URL)
			if err != nil {
				log.Errorf("failed to open file %s: %v", doc.URL, err)
				continue
			}
			defer file.Close()

			url := fmt.Sprintf("%s/tika", p.config.TikaEndpoint)
			req, err := http.NewRequestWithContext(ctx, "PUT", url, file)
			if err != nil {
				log.Errorf("failed to create request for %s: %v", doc.URL, err)
				continue
			}
			req.Header.Set("Accept", "text/html")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Errorf("failed to request tika for %s: %v", doc.URL, err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				log.Errorf("tika returned status %d for %s: %s", resp.StatusCode, doc.URL, string(body))
				continue
			}

			// Parse HTML response
			docHTML, err := goquery.NewDocumentFromReader(resp.Body)
			if err != nil {
				log.Errorf("failed to parse tika response for %s: %v", doc.URL, err)
				continue
			}

			var pages []core.PageText

			// Find all div with class "page"
			docHTML.Find("div.page").Each(func(i int, s *goquery.Selection) {
				pageContent := strings.TrimSpace(s.Text())
				if pageContent != "" {
					pages = append(pages, core.PageText{
						PageNumber: i + 1,
						Content:    pageContent,
					})
				}
			})

			// If no pages found (maybe not a PDF or Tika returned plain text
			// wrapped in body), try getting body text
			if len(pages) == 0 {
				bodyText := strings.TrimSpace(docHTML.Find("body").Text())
				if bodyText != "" {
					pages = append(pages, core.PageText{
						PageNumber: 1,
						Content:    bodyText,
					})
				}
			}

			doc.Text = pages

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
