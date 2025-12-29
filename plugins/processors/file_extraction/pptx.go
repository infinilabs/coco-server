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
			PagesWithoutOcr: make([]string, 0),
			PagesWithOcr:    make([]string, 0),
			Images:          make(map[int][]string),
		}, nil
	}

	var pagesWithoutOcr []string
	var pagesWithOcr []string
	images := make(map[int][]string)

	// 6. Process Slides
	for slideIdx, slideFile := range slideFiles {
		// A. Build Relationship Map for this slide (rId -> imageFilename)
		relsMap, err := getSlideRelationships(r, slideFile.Name)
		if err != nil {
			log.Warnf("Failed to get relationships for slide %s: %v", slideFile.Name, err)
			relsMap = make(map[string]string)
		}

		// We collect all images linked to this slide from the relsMap
		var slideImages []string
		for _, filename := range relsMap {
			slideImages = append(slideImages, filename)
		}
		images[slideIdx] = slideImages

		// B. Parse Content & Perform OCR (Dual Extraction)
		textNoOcr, textOcr, err := p.parseSlideContentDual(ctx, slideFile, relsMap, attachmentDirPath)
		if err != nil {
			log.Warnf("Failed to parse content for slide %s: %v", slideFile.Name, err)
			// Append empty strings to keep page count consistent
			textNoOcr = ""
			textOcr = ""
		}

		pagesWithoutOcr = append(pagesWithoutOcr, textNoOcr)
		pagesWithOcr = append(pagesWithOcr, textOcr)
	}

	// 7. Upload Attachments
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

// --- XML Parsing & OCR Logic ---

// ocrTask holds information for parallel OCR processing
type ocrTask struct {
	rId      string
	filename string
	index    int
}

// parseSlideContentDual parses the XML of a slide and returns two strings:
// 1. Text with [[Image(filename)]]
// 2. Text with [[ImageContentViaOCR(content)]] (performs HTTP calls to Tika concurrently)
func (p *FileExtractionProcessor) parseSlideContentDual(ctx context.Context, f *zip.File, rels map[string]string, attachmentDir string) (string, string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", "", err
	}
	defer DeferClose(rc)

	decoder := xml.NewDecoder(rc)
	var sbNoOcr strings.Builder
	var sbOcr strings.Builder

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
			return "", "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Handle Paragraphs (Newlines)
			if t.Name.Local == "p" && strings.Contains(t.Name.Space, "drawingml") {
				if sbNoOcr.Len() > 0 {
					sbNoOcr.WriteString("\n")
					sbOcr.WriteString("\n")
				}
			}

			// Handle Text Content
			if t.Name.Local == "t" {
				var textContent string
				if err := decoder.DecodeElement(&textContent, &t); err == nil {
					sbNoOcr.WriteString(textContent)
					sbOcr.WriteString(textContent)
				}
			}

			// Handle Images (Catch-All Strategy via Attributes)
			for _, attr := range t.Attr {
				rId := attr.Value
				if filename, exists := rels[rId]; exists {
					if !processedRels[rId] {
						// Append to No-OCR Builder
						sbNoOcr.WriteString(fmt.Sprintf("\n[[Image(%s)]]\n", filename))

						// Collect OCR task for parallel processing
						ocrTasks = append(ocrTasks, ocrTask{
							rId:      rId,
							filename: filename,
							index:    taskIndex,
						})

						// Append placeholder in OCR builder
						sbOcr.WriteString(fmt.Sprintf("\n[[OCR_PLACEHOLDER_%d]]\n", taskIndex))
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
				ocrReader, ocrErr := tikaGetTextPlain(ctx, p.config.TikaEndpoint, p.config.TimeoutInSeconds, fullPath)
				var extractedText string

				if ocrErr != nil {
					log.Warnf("OCR failed for image %s: %v", t.filename, ocrErr)
					extractedText = ""
				} else {
					defer DeferClose(ocrReader)

					var buf strings.Builder
					_, copyErr := io.Copy(&buf, ocrReader)

					if copyErr != nil {
						log.Warnf("Failed to read OCR response for %s: %v", t.filename, copyErr)
						extractedText = ""
					} else {
						extractedText = strings.TrimSpace(buf.String())
						log.Debugf("OCR result [%s]", extractedText)
					}
				}

				mu.Lock()
				results[t.index] = extractedText
				mu.Unlock()
			}(task)
		}

		wg.Wait()

		// Replace placeholders with actual OCR results
		ocrText := sbOcr.String()
		for i := 0; i < len(ocrTasks); i++ {
			placeholder := fmt.Sprintf("[[OCR_PLACEHOLDER_%d]]", i)
			replacement := fmt.Sprintf("[[ImageContentViaOCR(%s)]]", results[i])
			ocrText = strings.Replace(ocrText, placeholder, replacement, 1)
		}

		return strings.TrimSpace(sbNoOcr.String()), strings.TrimSpace(ocrText), nil
	}

	return strings.TrimSpace(sbNoOcr.String()), strings.TrimSpace(sbOcr.String()), nil
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
