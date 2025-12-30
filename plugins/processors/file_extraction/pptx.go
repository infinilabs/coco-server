package file_extraction

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
)

func (p *FileExtractionProcessor) processPptx(ctx context.Context, doc *core.Document) (Extraction, error) {
	// 1. Prepare Temp Directory for Attachments
	attachmentDirPath, err := os.MkdirTemp("", "attachment-pptx-")
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to create temporary directory for extraction: %w", err)
	}
	defer os.RemoveAll(attachmentDirPath)

	// 3. Open PPTX File
	r, err := zip.OpenReader(doc.URL)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to open pptx file [%s]: %w", doc.URL, err)
	}
	defer DeferClose(r)

	// 4. Extract Images to Temp Dir
	// We do this first so they are available on disk for the OCR calls later.
	if err := saveImagesToDisk(r, attachmentDirPath); err != nil {
		return Extraction{}, fmt.Errorf("failed to extract images from pptx: %w", err)
	}

	// 5. Identify and Sort Slides
	slideFiles, err := getSortedSlideFiles(r)
	if err != nil {
		// If strictly no slides found, it might not be a valid pptx or just empty.
		// Return empty arrays instead of error to allow flow to continue.
		log.Warnf("No slides found in PPTX %s: %v", doc.URL, err)
		return Extraction{
			Pages:       nil,
			Attachments: nil,
		}, nil
	}

	var pages []string
	// filename -> text extracted via OCR
	imageOCR := make(map[string]string)

	// 6. Process Slides
	for _, slideFile := range slideFiles {
		// A. Build Relationship Map for this slide (rId -> imageFilename)
		relsMap, err := getSlideRelationships(r, slideFile.Name)
		if err != nil {
			log.Warnf("Failed to get relationships for slide %s: %v", slideFile.Name, err)
			relsMap = make(map[string]string)
		}

		// B. Parse Content & Perform OCR
		text, err := p.parseSlideContent(ctx, doc.ID, slideFile, relsMap, attachmentDirPath, imageOCR)
		if err != nil {
			log.Warnf("Failed to parse content for slide %s: %v", slideFile.Name, err)
			// Append empty strings to keep page count consistent
			text = ""
		}

		pages = append(pages, text)
	}

	// 7. Upload Attachments
	attachmentIds, err := uploadAttachmentsToBlobStore(ctx, attachmentDirPath, doc, imageOCR)
	if err != nil {
		return Extraction{}, fmt.Errorf("failed to upload document attachments: %w", err)
	}

	return Extraction{
		Pages:       pages,
		Attachments: attachmentIds,
	}, nil
}

// --- XML Parsing & OCR Logic ---

// ocrTask holds information for parallel OCR processing
type ocrTask struct {
	rId      string
	filename string
	index    int
}

// parseSlideContent parses the XML of a slide and returns the text content.
// Images are replaced with [[Image(UUID\tOCRText)]] tags.
func (p *FileExtractionProcessor) parseSlideContent(ctx context.Context, docID string, f *zip.File, rels map[string]string, attachmentDir string, imageOCR map[string]string) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer DeferClose(rc)

	decoder := xml.NewDecoder(rc)
	var sb strings.Builder

	// Track processed image IDs per slide to prevent duplicate markers if XML repeats tags
	processedRels := make(map[string]bool)

	// Collect OCR tasks for concurrent processing
	var ocrTasks []ocrTask
	taskIndex := 0

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
			// Handle Paragraphs (Newlines)
			if t.Name.Local == "p" && strings.Contains(t.Name.Space, "drawingml") {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
			}

			// Handle Text Content
			if t.Name.Local == "t" {
				var textContent string
				if err := decoder.DecodeElement(&textContent, &t); err == nil {
					sb.WriteString(textContent)
				}
			}

			// Handle Images (Catch-All Strategy via Attributes)
			for _, attr := range t.Attr {
				rId := attr.Value
				if filename, exists := rels[rId]; exists {
					if !processedRels[rId] {
						// Collect OCR task for parallel processing
						ocrTasks = append(ocrTasks, ocrTask{
							rId:      rId,
							filename: filename,
							index:    taskIndex,
						})

						// Append placeholder in builder
						sb.WriteString(fmt.Sprintf("\n[[OCR_PLACEHOLDER_%d]]\n", taskIndex))
						taskIndex++

						processedRels[rId] = true
					}
				}
			}
		}
	}

	// Process all OCR tasks concurrently
	if len(ocrTasks) > 0 {
		results := make(map[int]string)
		var mu sync.Mutex
		var wg sync.WaitGroup

		for _, task := range ocrTasks {
			wg.Add(1)
			go func(t ocrTask) {
				defer wg.Done()

				fullPath := filepath.Join(attachmentDir, t.filename)
				extractedText, err := ocr(ctx, p.config.TikaEndpoint, p.config.TimeoutInSeconds, fullPath)
				if err != nil {
					log.Warnf("doing OCR failed for image %s: %v", t.filename, err)
					extractedText = ""
				}

				mu.Lock()
				results[t.index] = extractedText
				imageOCR[t.filename] = extractedText
				mu.Unlock()
			}(task)
		}

		wg.Wait()

		// Replace placeholders with actual [[Image(UUID\tOCRText)]] tags
		finalText := sb.String()
		for _, task := range ocrTasks {
			placeholder := fmt.Sprintf("[[OCR_PLACEHOLDER_%d]]", task.index)
			uuid := docID + task.filename
			tag := fmt.Sprintf("[[Image(%s\t%s)]]", uuid, results[task.index])
			finalText = strings.Replace(finalText, placeholder, tag, 1)
		}
		return finalText, nil
	}

	return sb.String(), nil
}

// saveImagesToDisk iterates the zip and saves files in ppt/media/ to outputDir
func saveImagesToDisk(r *zip.ReadCloser, outDir string) error {
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/media/") {
			fileName := filepath.Base(f.Name)
			outPath := filepath.Join(outDir, fileName)

			// Optimization: Check if exists
			if _, err := os.Stat(outPath); err == nil {
				continue
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}

			// Safe file creation
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
	}
	return nil
}

// getSortedSlideFiles finds slide XMLs and sorts them naturally (1, 2, 10)
func getSortedSlideFiles(r *zip.ReadCloser) ([]*zip.File, error) {
	var slides []*zip.File
	re := regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)

	for _, f := range r.File {
		if re.MatchString(f.Name) {
			slides = append(slides, f)
		}
	}

	sort.Slice(slides, func(i, j int) bool {
		numI, _ := strconv.Atoi(re.FindStringSubmatch(slides[i].Name)[1])
		numJ, _ := strconv.Atoi(re.FindStringSubmatch(slides[j].Name)[1])
		return numI < numJ
	})

	if len(slides) == 0 {
		return nil, fmt.Errorf("no slides found")
	}
	return slides, nil
}

// getSlideRelationships parses the .rels file for a specific slide
func getSlideRelationships(r *zip.ReadCloser, slidePath string) (map[string]string, error) {
	dir := filepath.Dir(slidePath)
	base := filepath.Base(slidePath)
	// slide1.xml -> _rels/slide1.xml.rels
	relsPath := filepath.Join(dir, "_rels", base+".rels")
	relsPath = strings.ReplaceAll(relsPath, "\\", "/") // Zip uses forward slashes

	relsMap := make(map[string]string)

	var relFile *zip.File
	for _, f := range r.File {
		if f.Name == relsPath {
			relFile = f
			break
		}
	}

	if relFile == nil {
		return relsMap, nil
	}

	rc, err := relFile.Open()
	if err != nil {
		return nil, err
	}
	defer DeferClose(rc)

	// Minimal XML structs
	type Relationship struct {
		Id     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	}
	type Relationships struct {
		List []Relationship `xml:"Relationship"`
	}

	var rels Relationships
	if err := xml.NewDecoder(rc).Decode(&rels); err != nil {
		return nil, err
	}

	for _, rel := range rels.List {
		if isImageFile(rel.Target) {
			relsMap[rel.Id] = filepath.Base(rel.Target)
		}
	}

	return relsMap, nil
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tiff", ".jfif":
		return true
	default:
		return false
	}
}
