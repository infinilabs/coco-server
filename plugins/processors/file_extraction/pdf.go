/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
)

func (p *FileExtractionProcessor) processPdf(ctx context.Context, doc *core.Document) (Extraction, error) {
	tikaRequestCtx, cancel := context.WithTimeout(ctx, time.Duration(p.config.TimeoutInSeconds)*time.Second)
	defer cancel()

	path := doc.URL
	htmlReader, err := tikaGetTextHtml(tikaRequestCtx, p.config.TikaEndpoint, p.config.TimeoutInSeconds, path)
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

	err = tikaUnpackAllTo(tikaRequestCtx, p.config.TikaEndpoint, path, attachmentDirPath, p.config.TimeoutInSeconds)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to extract document attachments: %w", err)
	}

	/*
		Extract document content
	*/
	var pagesWithoutOcr []string
	var pagesWithOcr []string
	images := make(map[int][]string)
	pagesSelection := docHTML.Find("div.page")
	// Find all div with class "page"
	for i := 0; i < pagesSelection.Length(); i++ {
		s := pagesSelection.Eq(i)
		pageNum := i + 1
		imagesOfThisPage := make([]string, 0)

		p.appendPage(tikaRequestCtx, s, attachmentDirPath, &pagesWithoutOcr, &pagesWithOcr, &imagesOfThisPage)
		images[pageNum] = imagesOfThisPage
	}

	// If no pages found (maybe not a PDF or Tika returned plain text
	// wrapped in body), try getting body text
	if len(pagesWithoutOcr) == 0 {
		imagesOfThisPage := make([]string, 0)
		s := docHTML.Find("body")
		p.appendPage(tikaRequestCtx, s, attachmentDirPath, &pagesWithoutOcr, &pagesWithOcr, &imagesOfThisPage)
		images[1] = imagesOfThisPage
	}

	/*
		Upload attachments
	*/
	err = uploadAttachmentsToBlobStore(ctx, attachmentDirPath, doc)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to upload document attachments: %w", err)
	}

	return Extraction{
		PagesWithoutOcr: pagesWithoutOcr,
		PagesWithOcr:    pagesWithOcr,
		Images:          images,
	}, nil
}

// appendPage processes a page selection, generating two versions of the text:
// 1. pagesWithoutOcr: Images are replaced with filenames [[Image(name)]]
// 2. pagesWithOcr: Images are replaced with their OCR text content [[ImageContentViaOCR(text)]]
func (p *FileExtractionProcessor) appendPage(tikaRequestCtx context.Context, s *goquery.Selection, attachmentDir string, pagesWithoutOcr, pagesWithOcr *[]string, imagesOfThisPage *[]string) {
	// Clone s because goquery modifies nodes in-place.
	sClone := s.Clone()

	/*
	 * append to [pagesWithoutOcr] and collect the images that appear in this page
	 */
	s.Find("img").Each(func(i int, img *goquery.Selection) {
		imageName, exists := img.Attr("src")
		if exists {
			// For the images embedded within the document, Tika typically generates
			// tags like "<img src="embedded:image3.png" alt="image3.png"/>". We need
			// to remove the "embedded:" prefix as it is useless.
			imageName = strings.TrimPrefix(imageName, "embedded:")
			*imagesOfThisPage = append(*imagesOfThisPage, imageName)
			img.ReplaceWithHtml(fmt.Sprintf("[[Image(%s)]]", imageName))
		}
	})

	pageContent := strings.TrimSpace(s.Text())
	// Still need to append this page even though its content empty, or the
	// pages will be out-of-order.
	*pagesWithoutOcr = append(*pagesWithoutOcr, pageContent)

	/*
	 * append to [pagesWithOcr]
	 */
	// First pass: collect image paths in order
	imagePaths := make(map[int]string)
	sClone.Find("img").Each(func(i int, img *goquery.Selection) {
		imageName, exists := img.Attr("src")
		if exists {
			imageName = strings.TrimPrefix(imageName, "embedded:")
			fullImagePath := filepath.Join(attachmentDir, imageName)
			imagePaths[i] = fullImagePath
		}
	})

	// Process OCR concurrently for all images
	ocrResults := make(map[int]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for idx, imagePath := range imagePaths {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()

			rc, err := tikaGetTextPlain(tikaRequestCtx, p.config.TikaEndpoint, p.config.TimeoutInSeconds, path)
			var extractedText string

			if err != nil {
				log.Warnf("doing OCR failed with: %v ", err)
				extractedText = ""
			} else {
				defer DeferClose(rc)
				var buf strings.Builder
				_, err := io.Copy(&buf, rc)
				if err != nil {
					extractedText = ""
				} else {
					extractedText = strings.TrimSpace(buf.String())
				}
			}

			mu.Lock()
			ocrResults[index] = extractedText
			mu.Unlock()
		}(idx, imagePath)
	}

	wg.Wait()

	// Second pass: replace images with OCR results in order
	sClone.Find("img").Each(func(i int, img *goquery.Selection) {
		_, exists := img.Attr("src")
		if exists {
			extractedText := ocrResults[i]
			img.ReplaceWithHtml(fmt.Sprintf("[[ImageContentViaOCR(%s)]]", extractedText))
		}
	})

	pageContentOcr := strings.TrimSpace(sClone.Text())
	// Still need to append this page even though its content empty, or the
	// pages will be out-of-order.
	*pagesWithOcr = append(*pagesWithOcr, pageContentOcr)
}
