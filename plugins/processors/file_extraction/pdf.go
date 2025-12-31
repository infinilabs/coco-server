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

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
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
		Pre-process attachments: assign UUIDs and perform OCR for images
	*/
	// nameToId maps the original attachment filename (e.g., "image1.png") to a
	// unique UUID.
	nameToId := make(map[string]string)
	// nameToText maps the original attachment filename to its OCR-extracted text
	// content, if it is an image
	nameToText := make(map[string]string)
	// nameToPageNums maps the original attachment filename to the page numbers
	// where it appears.
	nameToPageNums := make(map[string][]int)
	entries, err := os.ReadDir(attachmentDirPath)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to read attachment directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		if name == "__METADATA__" || name == "__TEXT__" {
			continue
		}

		// Assign a unique ID for each attachment
		id := util.GetUUID()
		nameToId[name] = id

		// If it's an image, perform OCR synchronously
		if isImage(name) {
			fullPath := filepath.Join(attachmentDirPath, name)
			text, err := ocr(ctx, p.config.TikaEndpoint, p.config.TimeoutInSeconds, fullPath)
			if err != nil {
				log.Warnf("failed to perform OCR for image [%s]: %v", name, err)
			} else {
				nameToText[name] = text
			}
		}
	}

	/*
		Extract document content
	*/
	var pages []string
	pagesSelection := docHTML.Find("div.page")
	// Find all div with class "page"
	for i := 0; i < pagesSelection.Length(); i++ {
		s := pagesSelection.Eq(i)
		p.appendPage(s, i+1, nameToId, nameToText, nameToPageNums, &pages)
	}

	// If no pages found (maybe not a PDF or Tika returned plain text
	// wrapped in body), try getting body text
	if len(pages) == 0 {
		s := docHTML.Find("body")
		p.appendPage(s, 1, nameToId, nameToText, nameToPageNums, &pages)
	}

	/*
		Upload attachments
	*/
	err = uploadAttachmentsToBlobStore(ctx, attachmentDirPath, doc, nameToId, nameToText, nameToPageNums)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to upload document attachments: %w", err)
	}

	// Collect all assigned attachment IDs
	var attachmentIds []string
	for _, id := range nameToId {
		attachmentIds = append(attachmentIds, id)
	}

	return Extraction{
		Pages:       pages,
		Attachments: attachmentIds,
	}, nil
}

// appendPage processes a page selection, generating the text content.
// Images are replaced with [[Image(UUID\tOCRText)]] tags.
func (p *FileExtractionProcessor) appendPage(s *goquery.Selection, pageNum int, nameToId map[string]string, nameToText map[string]string, nameToPageNums map[string][]int, pages *[]string) {
	s.Find("img").Each(func(i int, img *goquery.Selection) {
		imageName, exists := img.Attr("src")
		if exists {
			imageName = strings.TrimPrefix(imageName, "embedded:")
			uuid, ok := nameToId[imageName]
			if !ok {
				panic(fmt.Sprintf("unreachable: attachment ID not found for file %s; all files in the directory should have been pre-processed and assigned a UUID", imageName))
			}

			// Record the page number where this image appears
			nameToPageNums[imageName] = append(nameToPageNums[imageName], pageNum)

			// Escape these chars because:
			// `]`: It is used as the pattern terminator
			// `\t`: It is used as the separator between UUID and TEXT
			text := escape(nameToText[imageName], []rune{']', '\t'})
			img.ReplaceWithHtml(fmt.Sprintf("[[Image(%s\t%s)]]", uuid, text))
		}
	})

	pageContent := strings.TrimSpace(s.Text())
	// Still need to append this page even though its content empty, or the
	// pages will be out-of-order.
	*pages = append(*pages, pageContent)
}
