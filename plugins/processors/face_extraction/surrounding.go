/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package face_extraction

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/processors/fileproc"
)

// extractSurroundingText returns a map from image filename to surrounding text
// context.  Strategy depends on contentType.
func extractSurroundingText(ctx context.Context, tikaEndpoint string, tikaTimeout int, localPath string, doc *core.Document, contentType string) (map[string]SurroundingText, error) {
	switch contentType {
	case "image":
		return extractSurroundingTextForImage(doc)
	case "pptx":
		return extractSurroundingTextForPptx(localPath)
	default:
		return extractSurroundingTextUsingTika(ctx, tikaEndpoint, tikaTimeout, localPath)
	}
}

// extractSurroundingTextForImage uses doc.Chunks[0] (LLM vision description)
// as the "After" context for the image itself.
func extractSurroundingTextForImage(doc *core.Document) (map[string]SurroundingText, error) {
	result := make(map[string]SurroundingText)
	st := SurroundingText{}
	if len(doc.Chunks) > 0 && doc.Chunks[0].Text != "" {
		st.After = doc.Chunks[0].Text
	}
	result[filepath.Base(doc.URL)] = st
	return result, nil
}

// extractSurroundingTextForPptx extracts per-slide text from a PPTX zip and
// maps each embedded image filename to the text of the slide it appears on.
func extractSurroundingTextForPptx(localPath string) (map[string]SurroundingText, error) {
	r, err := zip.OpenReader(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pptx file: %w", err)
	}
	defer r.Close()

	result := make(map[string]SurroundingText)

	slideFiles, err := fileproc.GetSortedSlideFiles(r)
	if err != nil {
		return result, nil
	}

	for _, slideFile := range slideFiles {
		relsMap, err := fileproc.GetSlideRelationships(r, slideFile.Name)
		if err != nil {
			continue
		}
		slideText, err := extractTextFromSlide(slideFile)
		if err != nil {
			continue
		}
		for _, filename := range relsMap {
			if _, exists := result[filename]; !exists {
				result[filename] = SurroundingText{After: slideText}
			}
		}
	}
	return result, nil
}

// extractTextFromSlide reads all <a:t> text nodes from a slide XML.
func extractTextFromSlide(slideFile *zip.File) (string, error) {
	rc, err := slideFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var textParts []string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if t, ok := token.(xml.StartElement); ok && t.Name.Local == "t" {
			var textContent string
			if err := decoder.DecodeElement(&textContent, &t); err == nil {
				textParts = append(textParts, textContent)
			}
		}
	}
	return strings.Join(textParts, " "), nil
}

// extractSurroundingTextUsingTika uses Tika HTML output to find the text
// immediately before and after each embedded image.
func extractSurroundingTextUsingTika(ctx context.Context, tikaEndpoint string, tikaTimeout int, localPath string) (map[string]SurroundingText, error) {
	htmlReader, err := fileproc.TikaGetTextHtml(ctx, tikaEndpoint, tikaTimeout, localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML from Tika: %w", err)
	}
	defer htmlReader.Close()

	doc, err := goquery.NewDocumentFromReader(htmlReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := make(map[string]SurroundingText)

	doc.Find("img[src^=\"embedded:\"]").Each(func(_ int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}
		filename := strings.TrimPrefix(src, "embedded:")

		parent := s.Parent()
		if parent.Length() == 0 {
			return
		}

		allSiblings := parent.Contents()
		imgNode := s.Get(0)
		imgIndex := -1
		allSiblings.EachWithBreak(func(i int, child *goquery.Selection) bool {
			if child.Get(0) == imgNode {
				imgIndex = i
				return false
			}
			return true
		})
		if imgIndex == -1 {
			return
		}

		beforeText := strings.TrimSpace(allSiblings.Slice(0, imgIndex).Text())
		afterText := strings.TrimSpace(allSiblings.Slice(imgIndex+1, goquery.ToEnd).Text())

		if beforeText == "" {
			if prev := parent.Prev(); prev.Length() > 0 {
				beforeText = strings.TrimSpace(prev.Text())
			}
		}
		if afterText == "" {
			if next := parent.Next(); next.Length() > 0 {
				afterText = strings.TrimSpace(next.Text())
			}
		}

		result[filename] = SurroundingText{Before: beforeText, After: afterText}
	})

	return result, nil
}
