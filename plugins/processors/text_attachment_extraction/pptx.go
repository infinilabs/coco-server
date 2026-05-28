/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package text_attachment_extraction

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/processors/fileproc"
	"infini.sh/framework/core/util"
)

func (p *TextAttachmentExtractionProcessor) processPptx(ctx context.Context, doc *core.Document, localPath string) (fileproc.Extraction, error) {
	r, err := zip.OpenReader(localPath)
	if err != nil {
		return fileproc.Extraction{}, fmt.Errorf("failed to open pptx file [%s]: %w", localPath, err)
	}
	defer fileproc.DeferClose(r)

	nameToId := make(map[string]string)
	nameToText := make(map[string]string)
	nameToPageNums := make(map[string][]int)

	var attachmentDirPath string

	// Only extract attachments when enabled.
	if *p.config.ExtractAttachments {
		attachmentDirPath, err = os.MkdirTemp("", "attachment-pptx-")
		if err != nil {
			return fileproc.Extraction{}, fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer os.RemoveAll(attachmentDirPath)

		if err := saveImagesToDisk(r, attachmentDirPath); err != nil {
			return fileproc.Extraction{}, fmt.Errorf("failed to extract images from pptx: %w", err)
		}

		entries, err := os.ReadDir(attachmentDirPath)
		if err != nil {
			return fileproc.Extraction{}, fmt.Errorf("failed to read attachment directory: %w", err)
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() {
				continue
			}
			name := entry.Name()
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

	slideFiles, err := fileproc.GetSortedSlideFiles(r)
	if err != nil {
		log.Warnf("no slides found in PPTX %s: %v", localPath, err)
		return fileproc.Extraction{}, nil
	}

	var pages []string
	for i, slideFile := range slideFiles {
		relsMap, err := fileproc.GetSlideRelationships(r, slideFile.Name)
		if err != nil {
			log.Warnf("failed to get relationships for slide %s: %v", slideFile.Name, err)
			relsMap = make(map[string]string)
		}

		text, err := p.parseSlideContent(slideFile, i+1, relsMap, nameToId, nameToText, nameToPageNums)
		if err != nil {
			log.Warnf("failed to parse content for slide %s: %v", slideFile.Name, err)
			text = ""
		}
		pages = append(pages, text)
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

// parseSlideContent parses a slide's XML and returns text with [[Image(...)]] markers.
// When attachment extraction is disabled (nameToId is empty), image markers are skipped.
func (p *TextAttachmentExtractionProcessor) parseSlideContent(f *zip.File, pageNum int, rels map[string]string, nameToId map[string]string, nameToText map[string]string, nameToPageNums map[string][]int) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer fileproc.DeferClose(rc)

	decoder := xml.NewDecoder(rc)
	var sb strings.Builder
	processedRels := make(map[string]bool)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "p" && strings.Contains(t.Name.Space, "drawingml") {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
			}
			if t.Name.Local == "t" {
				var textContent string
				if err := decoder.DecodeElement(&textContent, &t); err == nil {
					sb.WriteString(textContent)
				}
			}
			for _, attr := range t.Attr {
				rId := attr.Value
				if filename, exists := rels[rId]; exists && !processedRels[rId] {
					uuid, ok := nameToId[filename]
					if !ok {
						// Attachment extraction disabled or image not found; skip marker.
						processedRels[rId] = true
						continue
					}
					nameToPageNums[filename] = append(nameToPageNums[filename], pageNum)
					text := fileproc.Escape(nameToText[filename], []rune{']', '\t'})
					sb.WriteString(fmt.Sprintf("\n[[Image(%s\t%s)]]\n", uuid, text))
					processedRels[rId] = true
				}
			}
		}
	}
	return sb.String(), nil
}

// saveImagesToDisk writes all ppt/media/ files from the zip to outDir.
func saveImagesToDisk(r *zip.ReadCloser, outDir string) error {
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "ppt/media/") {
			continue
		}
		outPath := filepath.Join(outDir, filepath.Base(f.Name))
		if _, err := os.Stat(outPath); err == nil {
			continue // already extracted
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		outFile, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
