/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
)

func (p *FileExtractionProcessor) processPdf(ctx context.Context, doc *core.Document) (Extraction, error) {
	path := doc.URL
	htmlReader, err := tikaGetTextHtml(ctx, p.config.TikaEndpoint, p.config.TimeoutInSeconds, path)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to extract text for [%s] using tika: %w", path, err)
	}
	defer DeferClose(htmlReader)

	// Parse HTML response
	docHTML, err := goquery.NewDocumentFromReader(htmlReader)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to parse tika response for %s: %w", path, err)
	}

	attachmentDirPath, err := os.MkdirTemp("", "attachment-temp-")
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to create temporary directory for extracting document attachments: %w", err)
	}
	defer os.RemoveAll(attachmentDirPath)

	err = tikaUnpackAllTo(ctx, p.config.TikaEndpoint, path, attachmentDirPath, p.config.TimeoutInSeconds)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to extract document attachments: %w", err)
	}

	/*
		Extract document content
	*/
	var pages []string
	imageOCR := make(map[string]string)
	pagesSelection := docHTML.Find("div.page")
	// Find all div with class "page"
	for i := 0; i < pagesSelection.Length(); i++ {
		s := pagesSelection.Eq(i)
		p.appendPage(ctx, p.config.TimeoutInSeconds, doc.ID, s, attachmentDirPath, &pages, imageOCR)
	}

	// If no pages found (maybe not a PDF or Tika returned plain text
	// wrapped in body), try getting body text
	if len(pages) == 0 {
		s := docHTML.Find("body")
		p.appendPage(ctx, p.config.TimeoutInSeconds, doc.ID, s, attachmentDirPath, &pages, imageOCR)
	}

	/*
		Upload attachments
	*/
	err = uploadAttachmentsToBlobStore(ctx, attachmentDirPath, doc, imageOCR)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to upload document attachments: %w", err)
	}

	return Extraction{
		Pages:    pages,
		ImageOCR: imageOCR,
	}, nil
}

// appendPage processes a page selection, generating the text content.
// Images are replaced with [[Image(UUID\tOCRText)]] tags.
func (p *FileExtractionProcessor) appendPage(ctx context.Context, timeout int, docID string, s *goquery.Selection, attachmentDir string, pages *[]string, imageOCR map[string]string) {
	// First pass: collect image paths in order
	imagePaths := make(map[int]string)
	imageNames := make(map[int]string)
	s.Find("img").Each(func(i int, img *goquery.Selection) {
		imageName, exists := img.Attr("src")
		if exists {
			imageName = strings.TrimPrefix(imageName, "embedded:")
			fullImagePath := filepath.Join(attachmentDir, imageName)
			imagePaths[i] = fullImagePath
			imageNames[i] = imageName
		}
	})

	// Process OCR concurrently for all images
	ocrResults := make(map[int]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for idx, imagePath := range imagePaths {
		wg.Add(1)
		go func(index int, path string, name string) {
			defer wg.Done()

			extractedText, err := ocr(ctx, p.config.TikaEndpoint, timeout, path)
			if err != nil {
				log.Warnf("doing OCR failed for image %s (index %d): %v", path, index, err)
				extractedText = ""
			} else {
				log.Debugf("OCR result for image %s (index %d): [%s]", path, index, extractedText)
			}

			mu.Lock()
			ocrResults[index] = extractedText
			imageOCR[name] = extractedText
			mu.Unlock()
		}(idx, imagePath, imageNames[idx])
	}

	wg.Wait()

	// Second pass: replace images with [[Image(UUID\tOCRText)]] results in order
	s.Find("img").Each(func(i int, img *goquery.Selection) {
		imageName, exists := img.Attr("src")
		if exists {
			imageName = strings.TrimPrefix(imageName, "embedded:")
			extractedText := ocrResults[i]
			uuid := docID + imageName
			img.ReplaceWithHtml(fmt.Sprintf("[[Image(%s\t%s)]]", uuid, extractedText))
		}
	})

	pageContent := strings.TrimSpace(s.Text())
	// Still need to append this page even though its content empty, or the
	// pages will be out-of-order.
	*pages = append(*pages, pageContent)
}
