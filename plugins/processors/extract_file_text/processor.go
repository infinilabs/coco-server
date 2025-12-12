/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

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

			/*
				Extract document content, store it in "pages"
			*/
			var pages []string
			pagesSelection := docHTML.Find("div.page")
			// Find all div with class "page"
			for i = 0; i < pagesSelection.Length(); i++ {
				s := pagesSelection.Eq(i)
				appendPage(&pages, s)
			}

			// If no pages found (maybe not a PDF or Tika returned plain text
			// wrapped in body), try getting body text
			if len(pages) == 0 {
				s := docHTML.Find("body")
				appendPage(&pages, s)
			}

			doc.Chunks = SplitPagesToChunks(pages, p.config.ChunkSize)

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

// "s" is a selection of a page ("div.page"), extract and process the page content,
// then append it to "pages".
func appendPage(pages *[]string, s *goquery.Selection) {
	/*
		Replace <img> tag with "[[Images(fileName)]]"
	*/
	s.Find("img").Each(func(i int, img *goquery.Selection) {
		imageName, exists := img.Attr("src")
		if exists {
			// For the images embedded within the document, Tika typically generates
			// tags like "<img src="embedded:image3.png" alt="image3.png"/>". We need
			// to remove the "embedded:" prefix as it is useless.
			imageName = strings.TrimPrefix(imageName, "embedded:")
			img.ReplaceWithHtml(fmt.Sprintf("[[Images(%s)]]", imageName))
		}
	})

	pageContent := strings.TrimSpace(s.Text())
	if pageContent != "" {
		*pages = append(*pages, pageContent)
	}
}

// Splits page texts into chunks using character count as a token proxy
// and tracks the page range for each chunk.
func SplitPagesToChunks(pages []string, chunkSize int) []core.DocumentChunk {
	// Early return
	if chunkSize <= 0 {
		return nil
	}
	if len(pages) == 0 {
		return make([]core.DocumentChunk, 0)
	}

	var chunks []core.DocumentChunk

	buf := make([]rune, 0, chunkSize)
	// Value 0 means `startPage`` and `lastPage` are not initialized
	startPage := 0
	lastPage := 0

	for idx, page := range pages {
		pageNumber := idx + 1
		pageChars := []rune(page)

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

				chunks = append(chunks, core.DocumentChunk{
					Range: chunkRange,
					Text:  textChunk,
					// this field remain uninitialized
					// Embedding: core.Embedding{},
				})

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
		chunks = append(chunks, core.DocumentChunk{
			Range: chunkRange,
			Text:  textChunk,
			// this field remain uninitialized
			// Embedding: core.Embedding{},
		})
	}

	return chunks
}
