/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package text_attachment_extraction

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/processors/fileproc"
	"infini.sh/framework/core/util"
)

func (p *DocumentTextAttachmentExtractionProcessor) processPdf(ctx context.Context, doc *core.Document, localPath string) (fileproc.Extraction, error) {
	htmlReader, err := fileproc.TikaGetTextHtml(ctx, p.config.TikaEndpoint, p.config.TikaTimeoutInSeconds, localPath)
	if err != nil {
		return fileproc.Extraction{}, fmt.Errorf("failed to extract text for [%s] using tika: %w", localPath, err)
	}
	defer fileproc.DeferClose(htmlReader)

	docHTML, err := goquery.NewDocumentFromReader(htmlReader)
	if err != nil {
		return fileproc.Extraction{}, fmt.Errorf("failed to parse tika response for %s: %w", localPath, err)
	}

	nameToId := make(map[string]string)
	nameToText := make(map[string]string)
	nameToPageNums := make(map[string][]int)

	var attachmentDirPath string

	// Only extract attachments when enabled.
	if *p.config.ExtractAttachments {
		attachmentDirPath, err = os.MkdirTemp("", "attachment-temp-")
		if err != nil {
			return fileproc.Extraction{}, fmt.Errorf("failed to create temporary directory for attachments: %w", err)
		}
		defer os.RemoveAll(attachmentDirPath)

		if err := fileproc.TikaUnpackAllTo(ctx, p.config.TikaEndpoint, localPath, attachmentDirPath, p.config.TikaTimeoutInSeconds); err != nil {
			return fileproc.Extraction{}, fmt.Errorf("failed to extract document attachments: %w", err)
		}

		// Pre-process: assign UUIDs and OCR for images
		entries, err := os.ReadDir(attachmentDirPath)
		if err != nil {
			return fileproc.Extraction{}, fmt.Errorf("failed to read attachment directory: %w", err)
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() {
				continue
			}
			name := entry.Name()
			if name == "__METADATA__" || name == "__TEXT__" {
				continue
			}
			nameToId[name] = util.GetUUID()
			if fileproc.IsImage(name) {
				fullPath := filepath.Join(attachmentDirPath, name)
				text, err := fileproc.OCR(ctx, p.config.TikaEndpoint, p.config.TikaTimeoutInSeconds, fullPath)
				if err != nil {
					log.Warnf("failed to perform OCR for image [%s]: %v", name, err)
				} else {
					nameToText[name] = text
				}
			}
		}
	}

	// Extract page content
	var pages []string
	pagesSelection := docHTML.Find("div.page")
	for i := 0; i < pagesSelection.Length(); i++ {
		s := pagesSelection.Eq(i)
		p.appendPage(s, i+1, nameToId, nameToText, nameToPageNums, &pages)
	}
	if len(pages) == 0 {
		s := docHTML.Find("body")
		p.appendPage(s, 1, nameToId, nameToText, nameToPageNums, &pages)
	}

	if *p.config.ExtractAttachments && attachmentDirPath != "" {
		if err := fileproc.UploadAttachmentsToBlobStore(ctx, attachmentDirPath, doc, nameToId, nameToText, nameToPageNums); err != nil {
			return fileproc.Extraction{}, fmt.Errorf("failed to upload document attachments: %w", err)
		}
	}

	var attachmentIds []string
	for _, id := range nameToId {
		attachmentIds = append(attachmentIds, id)
	}

	return fileproc.Extraction{Pages: pages, Attachments: attachmentIds}, nil
}

// appendPage processes a goquery page selection and appends its text (with image
// markers) to pages.  Images are replaced with [[Image(UUID\tOCRText)]] tags.
// When attachment extraction is disabled (nameToId is empty), img tags are simply removed.
func (p *DocumentTextAttachmentExtractionProcessor) appendPage(s *goquery.Selection, pageNum int, nameToId map[string]string, nameToText map[string]string, nameToPageNums map[string][]int, pages *[]string) {
	s.Find("img").Each(func(i int, img *goquery.Selection) {
		imageName, exists := img.Attr("src")
		if exists {
			imageName = strings.TrimPrefix(imageName, "embedded:")
			uuid, ok := nameToId[imageName]
			if !ok {
				// Attachment extraction disabled or image not found; remove img tag.
				img.Remove()
				return
			}
			nameToPageNums[imageName] = append(nameToPageNums[imageName], pageNum)
			text := fileproc.Escape(nameToText[imageName], []rune{']', '\t'})
			img.ReplaceWithHtml(fmt.Sprintf("[[Image(%s\t%s)]]", uuid, text))
		}
	})
	*pages = append(*pages, strings.TrimSpace(s.Text()))
}
