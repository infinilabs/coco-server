/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/disintegration/imaging"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	_ "golang.org/x/image/webp"
)

// CoverWidth and CoverHeight define the standard cover dimensions
const (
	CoverWidth    = 320
	CoverHeight   = 180
	ThumbMaxWidth = 640 // Max width for thumbnail (aspect-ratio-preserving)
)

// CssStyleForThumbnail is optimized for thumbnail generation
// Features: larger fonts, reduced padding, hidden images for cleaner text rendering
const CssStyleForThumbnail = `
<style>
    body {
        font-family: system-ui, -apple-system, "Microsoft YaHei", sans-serif;
        font-size: 28px;
        line-height: 1.4;
        color: #24292e;
        padding: 24px;
        margin: 0;
        background-color: #fff;
        overflow: hidden;
    }
    h1 { 
        font-size: 48px;
        color: #0366d6; 
        border-bottom: 4px solid #eaecef;
        margin-top: 0;
        margin-bottom: 24px;
        padding-bottom: 12px;
    }
    h2 { font-size: 36px; border-bottom: 3px solid #eaecef; margin-top: 24px; }
    h3 { font-size: 32px; margin-top: 20px; }
    p, li { margin-bottom: 16px; }
    code { background-color: #f6f8fa; padding: 4px 8px; font-family: monospace; font-weight: bold; }
    pre { background-color: #f6f8fa; padding: 16px; overflow: hidden; border-radius: 6px; }
    blockquote { border-left: 4px solid #dfe2e5; color: #6a737d; padding-left: 1em; margin: 0; }
    img { display: none; }
</style>
`

// GenerateCoverAndThumbnail generates a cover image for the given
// file and saves it to outPath.  The cover is automatically resized
// to 320x180.
//
// If file is an image, we also generate a thumbnail for it.
func GenerateCoverAndThumbnail(file, coverOutPath, thumbnailOutPath string) error {
	ext := strings.ToLower(filepath.Ext(file))
	log.Tracef("generating cover/thumbnail for file type %s: %s", ext, file)

	var err error
	switch ext {
	case ".pdf":
		err = generatePdfCover(file, coverOutPath)
	case ".md":
		err = generateMarkdownCover(file, coverOutPath)
	case ".docx", ".pptx", ".xlsx":
		err = generateOfficeCover(file, coverOutPath)
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff":
		err = generateImageCover(file, coverOutPath)
		if err == nil && thumbnailOutPath != "" {
			// Generate thumbnail separately (aspect-ratio-preserving)
			// Propagate thumbnail error to caller
			err = GenerateThumbnail(file, thumbnailOutPath)
		}
	default:
		return fmt.Errorf("unsupported file type for cover generation: %s", ext)
	}

	if err != nil {
		return err
	}

	// Resize the generated cover to standard dimensions
	log.Tracef("resizing cover to %dx%d: %s", CoverWidth, CoverHeight, coverOutPath)
	return resizeCover(coverOutPath, CoverWidth, CoverHeight)
}

// resizeCover resizes and crops an image to exact dimensions using Fill
func resizeCover(imagePath string, width, height int) error {
	log.Tracef("opening image for resizing: %s", imagePath)
	img, err := imaging.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image for resizing: %w", err)
	}

	// Fill crops to exact dimensions using Top anchor (good for docs with titles)
	log.Tracef("cropping image to %dx%d using Top anchor", width, height)
	resized := imaging.Fill(img, width, height, imaging.Top, imaging.Lanczos)

	// Save back to the same path as PNG
	log.Tracef("saving resized image as PNG: %s", imagePath)
	out, err := os.Create(imagePath)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, resized)
}

// GenerateThumbnail generates an aspect-ratio-preserving thumbnail from an image.
// If width > ThumbMaxWidth, scales down proportionally. Otherwise uses original.
// Output is always PNG (metadata stripped, format unified).
func GenerateThumbnail(inputPath, outPath string) error {
	log.Tracef("generating thumbnail: %s -> %s", inputPath, outPath)
	img, err := imaging.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open image for thumbnail: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	log.Tracef("original image width: %d, max thumbnail width: %d", width, ThumbMaxWidth)

	var finalImg image.Image

	// If width exceeds limit, resize proportionally (height=0 preserves aspect ratio)
	if width > ThumbMaxWidth {
		log.Tracef("resizing image to width %d (preserving aspect ratio)", ThumbMaxWidth)
		finalImg = imaging.Resize(img, ThumbMaxWidth, 0, imaging.Lanczos)
	} else {
		log.Tracef("image width within limit, using original")
		finalImg = img
	}

	// Save as PNG
	log.Tracef("saving thumbnail as PNG: %s", outPath)
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, finalImg)
}

func generateImageCover(imagePath, outPath string) error {
	log.Tracef("generating image cover by copying: %s -> %s", imagePath, outPath)
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}

// generatePdfCover generates a cover from the first page of a PDF using pdftoppm
func generatePdfCover(pdfPath, outPath string) error {
	log.Tracef("generating PDF cover: %s", pdfPath)
	// Create temp directory for pdftoppm output
	tmpDir, err := os.MkdirTemp("", "coco-pdf-cover-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Output prefix for pdftoppm (it will create prefix-1.jpg)
	outputPrefix := filepath.Join(tmpDir, "page")

	// Run pdftoppm to convert first page to JPEG
	// -f 1 -l 1: first page only
	// -jpeg: output as JPEG
	// -r 150: 150 DPI (good balance of quality and size)
	log.Tracef("running pdftoppm to extract first page at 150 DPI")
	cmd := exec.Command("pdftoppm", "-f", "1", "-l", "1", "-jpeg", "-r", "150", pdfPath, outputPrefix)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pdftoppm failed (output: %s): %w", string(output), err)
	}

	// pdftoppm creates files like page-1.jpg
	generatedFile := outputPrefix + "-1.jpg"
	if _, err := os.Stat(generatedFile); os.IsNotExist(err) {
		// Try alternative naming (page-01.jpg for multi-digit padding)
		log.Tracef("trying alternative file naming: page-01.jpg")
		generatedFile = outputPrefix + "-01.jpg"
		if _, err := os.Stat(generatedFile); os.IsNotExist(err) {
			return fmt.Errorf("pdftoppm output file not found")
		}
	}

	// Copy to output path
	log.Tracef("copying generated PDF cover to: %s", outPath)
	return copyLocalFile(generatedFile, outPath)
}

// generateOfficeCover generates a cover for Office documents by converting to PDF first
func generateOfficeCover(filePath, outPath string) error {
	log.Tracef("generating Office cover: %s", filePath)
	// Create temporary directory for PDF conversion
	tmpDir, err := os.MkdirTemp("", "coco-office-cover-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Convert to PDF using LibreOffice
	log.Tracef("converting Office document to PDF using LibreOffice")
	cmd := exec.Command("soffice", "--headless", "--convert-to", "pdf", filePath, "--outdir", tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("soffice conversion failed (output: %s): %w", string(output), err)
	}

	// Find the converted PDF
	baseName := filepath.Base(filePath)
	pdfName := strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ".pdf"
	pdfPath := filepath.Join(tmpDir, pdfName)

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return fmt.Errorf("converted PDF not found at %s", pdfPath)
	}

	log.Tracef("extracting cover from converted PDF")
	return generatePdfCover(pdfPath, outPath)
}

// generateMarkdownCover generates a cover by rendering Markdown to HTML and taking a screenshot
func generateMarkdownCover(mdPath, outPath string) error {
	log.Tracef("generating Markdown cover: %s", mdPath)
	// Read Markdown file
	mdData, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}

	// Configure Markdown parser with GitHub Flavored Markdown
	log.Tracef("parsing Markdown with GitHub Flavored Markdown")
	mdParser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	// Build full HTML document with thumbnail-optimized CSS
	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8">`)
	buf.WriteString(CssStyleForThumbnail)
	buf.WriteString(`</head><body>`)

	if err := mdParser.Convert(mdData, &buf); err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}
	buf.WriteString(`</body></html>`)
	fullHTML := buf.String()

	// Take screenshot using headless Chrome
	log.Tracef("taking screenshot of rendered Markdown")
	return takeHTMLScreenshot(fullHTML, outPath)
}

// takeHTMLScreenshot renders HTML and takes a screenshot using headless Chrome
// Optimized for thumbnail generation: uses small viewport with 2x scale for crisp text
func takeHTMLScreenshot(htmlContent, outPath string) error {
	log.Tracef("taking HTML screenshot with headless Chrome")
	// Configure Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var screenshotBuf []byte

	// Execute screenshot task
	// Use 640x360 viewport with 2x scale to get 1280x720 output with large, readable text
	// Then resize to 320x180 - the 2:1 ratio preserves sharpness
	log.Tracef("rendering HTML with 640x360 viewport at 2x scale")
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.EmulateViewport(640, 360, chromedp.EmulateScale(2.0)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			js := fmt.Sprintf(`document.open(); document.write(%q); document.close();`, htmlContent)
			_, exp, err := runtime.Evaluate(js).Do(ctx)
			if err != nil {
				return err
			}
			if exp != nil {
				return exp
			}
			return nil
		}),
		chromedp.Sleep(200*time.Millisecond),
		// Use CaptureScreenshot (viewport only) instead of FullScreenshot (entire page)
		// This avoids compressing a long document into a tiny thumbnail
		chromedp.CaptureScreenshot(&screenshotBuf),
	)

	if err != nil {
		return fmt.Errorf("chrome screenshot failed: %w", err)
	}

	// Decode the screenshot
	log.Tracef("decoding screenshot and resizing to %dx%d", CoverWidth, CoverHeight)
	img, err := imaging.Decode(bytes.NewReader(screenshotBuf))
	if err != nil {
		return fmt.Errorf("failed to decode screenshot: %w", err)
	}

	// Resize from 1280x720 to 320x180 using Lanczos for sharp downsampling
	resized := imaging.Resize(img, CoverWidth, CoverHeight, imaging.Lanczos)

	// Save as PNG - much better for text than JPEG (no compression artifacts)
	log.Tracef("saving screenshot as PNG: %s", outPath)
	return saveImageAsPng(resized, outPath)
}

// saveImageAsPng saves an image as PNG (better for text/line art than JPEG)
func saveImageAsPng(img image.Image, outPath string) error {
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer DeferClose(out)

	return png.Encode(out, img)
}
