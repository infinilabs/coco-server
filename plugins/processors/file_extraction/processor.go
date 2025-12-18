/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	"infini.sh/coco/plugins/connectors"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
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
	tikaRequestCtx, cancel := context.WithTimeout(ctx, time.Duration(p.config.TimeoutInSeconds)*time.Second)
	defer cancel()

	file, err := os.Open(doc.URL)
	if err != nil {
		return fmt.Errorf("failed to open file [%s]: %w", doc.URL, err)
	}
	defer file.Close()
	// Do not let client.Do close this file because we need it later, in the
	// second tika call
	fileNopClose := io.NopCloser(file)

	url := fmt.Sprintf("%s/tika", p.config.TikaEndpoint)
	req, err := http.NewRequestWithContext(tikaRequestCtx, "PUT", url, fileNopClose)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", doc.URL, err)
	}
	req.Header.Set("Accept", "text/html")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to request tika for %s: %w", doc.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tika returned status %d for %s: %s", resp.StatusCode, doc.URL, string(body))
	}

	// Parse HTML response
	docHTML, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse tika response for %s: %w", doc.URL, err)
	}

	/*
		Extract document content, store it in "pages"
	*/
	var pages []string
	pagesSelection := docHTML.Find("div.page")
	// Find all div with class "page"
	for i := 0; i < pagesSelection.Length(); i++ {
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

	/*
		Extract assignments and upload them to blob store
	*/
	// Rewind file cursor for the second tika call
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file %s: %w", doc.URL, err)
	}

	unpackUrl := fmt.Sprintf("%s/unpack/all", p.config.TikaEndpoint)
	unpackReq, err := http.NewRequestWithContext(tikaRequestCtx, "PUT", unpackUrl, file)
	if err != nil {
		return fmt.Errorf("failed to create unpack request for %s: %w", doc.URL, err)
	}
	unpackReq.Header.Set("X-Tika-PDFextractInlineImages", "true")

	unpackResp, err := client.Do(unpackReq)
	if err != nil {
		return fmt.Errorf("failed to request tika unpack for %s: %w", doc.URL, err)
	}
	defer unpackResp.Body.Close()

	if unpackResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(unpackResp.Body)
		return fmt.Errorf("tika unpack returned status %d for %s: %s", unpackResp.StatusCode, doc.URL, string(body))
	}

	// Save zip to temp file
	tmpDir := filepath.Join(global.Env().GetDataDir(), "extract_file_content", doc.ID)
	err = os.MkdirAll(tmpDir, 0755)
	// register os.RemoveAll before checking nil so that even though os.MkdirAll()
	// partially succeeds(which is an error, err != nil), we can still let golang
	// runtime clean the directories for us.
	defer os.RemoveAll(tmpDir)
	if err != nil {
		return fmt.Errorf("failed to create temp dir %s: %w", tmpDir, err)
	}

	zipPath := filepath.Join(tmpDir, "response.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file %s: %w", zipPath, err)
	}

	_, err = io.Copy(zipFile, unpackResp.Body)
	zipFile.Close()
	if err != nil {
		return fmt.Errorf("failed to save zip file %s: %w", zipPath, err)
	}

	// Unzip
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file %s: %w", zipPath, err)
	}
	defer r.Close()

	ormCtx := orm.NewContextWithParent(ctx)
	// Grant read/write access to the database, which is needed because this
	// is a background processor, which has no user token stored in ctx.
	ormCtx.DirectAccess()
	ownerID := doc.GetOwnerID()
	if ownerID == "" {
		panic("document has an empty owner ID")
	}

	for _, f := range r.File {
		parentDir, _ := filepath.Split(f.Name)
		// The attached files should be in a plain structure, do not process
		// files under a directory
		if parentDir != "" {
			continue
		}

		// We only process regular file
		if !f.Mode().IsRegular() {
			continue
		}

		// We only want attachments, not text and file metadata
		if f.Name == "__METADATA__" || f.Name == "__TEXT__" {
			continue
		}

		// Process image/attachment
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip %s: %w", f.Name, err)
		}
		defer rc.Close()

		extractedFilePath := filepath.Join(tmpDir, f.Name)
		outFile, err := os.Create(extractedFilePath)
		if err != nil {
			return fmt.Errorf("failed to create extracted file %s: %w", extractedFilePath, err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return fmt.Errorf("failed to write extracted file %s: %w", f.Name, err)
		}

		uploadFile, err := os.Open(extractedFilePath)
		if err != nil {
			return fmt.Errorf("failed to open extracted file for upload %s: %w", extractedFilePath, err)
		}

		fileID := doc.ID + f.Name

		_, err = attachment.UploadToBlobStore(ormCtx, fileID, uploadFile, f.Name, ownerID, true)
		if err != nil {
			return fmt.Errorf("failed to upload attachment %s: %w", f.Name, err)
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
