package file_extraction

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
)

func (p *FileExtractionProcessor) processPdf(ctx context.Context, doc *core.Document) (Extraction, error) {
	tikaRequestCtx, cancel := context.WithTimeout(ctx, time.Duration(p.config.TimeoutInSeconds)*time.Second)
	defer cancel()

	path := doc.URL
	htmlReader, err := tikaGetTextHtml(tikaRequestCtx, p.config.TikaEndpoint, path)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to extract text for [%s] using tika: %w", path, err)
	}
	defer htmlReader.Close()

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

	err = tikaUnpackAllTo(tikaRequestCtx, p.config.TikaEndpoint, path, attachmentDirPath)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to extracte document attachments: %w", err)
	}

	/*
		Extract document content
	*/
	var pagesWithoutOcr []string
	var pagesWithOcr []string
	var images map[int][]string
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
		Images:          make(map[int][]string),
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
	sClone.Find("img").Each(func(i int, img *goquery.Selection) {
		imageName, exists := img.Attr("src")
		if exists {
			imageName = strings.TrimPrefix(imageName, "embedded:")
			// Construct full path to the extracted image file
			fullImagePath := filepath.Join(attachmentDir, imageName)
			rc, err := tikaGetTextPlain(tikaRequestCtx, p.config.TikaEndpoint, fullImagePath)

			var extractedText string
			if err != nil {
				log.Warnf("doing OCR failed with: %w ", err)
				extractedText = ""
			} else {
				var buf strings.Builder
				_, err := io.Copy(&buf, rc)
				if err != nil {
					extractedText = ""
				} else {
					extractedText = buf.String()
				}
			}

			extractedText = strings.TrimSpace(extractedText)
			img.ReplaceWithHtml(fmt.Sprintf("[[ImageContentViaOCR(%s)]]", extractedText))
		}
	})

	pageContentOcr := strings.TrimSpace(sClone.Text())
	// Still need to append this page even though its content empty, or the
	// pages will be out-of-order.
	*pagesWithOcr = append(*pagesWithOcr, pageContentOcr)
}
