/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

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
)

// extractSurroundingText extracts embedded images and their surrounding text from a document.
// Returns a map keyed by image filename.
func extractSurroundingText(ctx context.Context, processor *FileExtractionProcessor, localPath string, doc *core.Document, contentType string) (map[string]SurroundingText, error) {
	switch contentType {
	case "image":
		return extractSurroundingTextForImage(doc)
	case "pptx":
		return extractSurroundingTextForPptx(localPath)
	case "docx", "xlsx", "pdf":
		return extractSurroundingTextUsingTika(ctx, processor, localPath)
	default:
		return extractSurroundingTextUsingTika(ctx, processor, localPath)
	}
}

// extractSurroundingTextForImage uses the LLM vision description as surrounding text
func extractSurroundingTextForImage(doc *core.Document) (map[string]SurroundingText, error) {
	result := make(map[string]SurroundingText)

	// The image itself is the "embedded" picture
	// Use Chunks[0].Text (LLM vision description) as "After" text
	if len(doc.Chunks) > 0 && doc.Chunks[0].Text != "" {
		result[filepath.Base(doc.URL)] = SurroundingText{
			After: doc.Chunks[0].Text,
		}
	} else {
		result[filepath.Base(doc.URL)] = SurroundingText{}
	}

	return result, nil
}

// extractSurroundingTextForPptx extracts surrounding text for images in PowerPoint files
func extractSurroundingTextForPptx(localPath string) (map[string]SurroundingText, error) {
	r, err := zip.OpenReader(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pptx file: %w", err)
	}
	defer r.Close()

	result := make(map[string]SurroundingText)

	// Get all slide files
	slideFiles, err := getSortedSlideFiles(r)
	if err != nil {
		return result, nil
	}

	// Process each slide
	for _, slideFile := range slideFiles {
		// Get image relationships for this slide
		relsMap, err := getSlideRelationships(r, slideFile.Name)
		if err != nil {
			continue
		}

		// Extract text content from slide
		slideText, err := extractTextFromSlide(slideFile)
		if err != nil {
			continue
		}

		// For each image in the slide, use the slide text as context
		for _, filename := range relsMap {
			if _, exists := result[filename]; !exists {
				result[filename] = SurroundingText{
					After: slideText,
				}
			}
		}
	}

	return result, nil
}

// extractTextFromSlide extracts all text from a slide XML
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

// extractSurroundingTextUsingTika extracts surrounding text for images in Word documents using Tika HTML
func extractSurroundingTextUsingTika(ctx context.Context, processor *FileExtractionProcessor, localPath string) (map[string]SurroundingText, error) {
	// Get HTML from Tika
	htmlReader, err := tikaGetTextHtml(ctx, processor.config.TikaEndpoint, processor.config.TimeoutInSeconds, localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML from Tika: %w", err)
	}
	defer htmlReader.Close()

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(htmlReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := make(map[string]SurroundingText)

	// Find all embedded images
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

		// Extract text from siblings (before/after the image)
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

		// Fallback: check parent's siblings
		if beforeText == "" {
			if prevParent := parent.Prev(); prevParent.Length() > 0 {
				beforeText = strings.TrimSpace(prevParent.Text())
			}
		}

		if afterText == "" {
			if nextParent := parent.Next(); nextParent.Length() > 0 {
				afterText = strings.TrimSpace(nextParent.Text())
			}
		}

		result[filename] = SurroundingText{
			Before: beforeText,
			After:  afterText,
		}
	})

	return result, nil
}
