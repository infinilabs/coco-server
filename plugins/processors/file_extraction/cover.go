/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/go-fitz"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	_ "golang.org/x/image/webp"
)

// CoverWidth and CoverHeight define the standard cover dimensions
const (
	CoverWidth  = 320
	CoverHeight = 180
)

// CssStyle defines styling for Markdown rendering
const CssStyle = `
<style>
    body {
        font-family: "Microsoft YaHei", "Helvetica Neue", Helvetica, Arial, sans-serif;
        font-size: 16px;
        line-height: 1.6;
        color: #333;
        padding: 40px;
        margin: 0;
        background-color: #fff;
    }
    h1, h2, h3 { color: #2c3e50; margin-top: 24px; margin-bottom: 16px; }
    h1 { font-size: 32px; border-bottom: 1px solid #eaecef; padding-bottom: 0.3em; }
    h2 { font-size: 24px; border-bottom: 1px solid #eaecef; padding-bottom: 0.3em; }
    p { margin-bottom: 16px; }
    code { background-color: #f6f8fa; padding: 0.2em 0.4em; border-radius: 3px; font-family: monospace; }
    pre { background-color: #f6f8fa; padding: 16px; overflow: auto; border-radius: 6px; }
    blockquote { border-left: 4px solid #dfe2e5; color: #6a737d; padding-left: 1em; margin: 0; }
    img { max-width: 100%; }
</style>
`

// GenerateCover generates a cover image for the given file and saves it to outPath.
// The cover is automatically resized to 320x180.
func GenerateCover(file, outPath string) error {
	ext := strings.ToLower(filepath.Ext(file))

	var err error
	switch ext {
	case ".pdf":
		err = generatePdfCover(file, outPath)
	case ".md":
		err = generateMarkdownCover(file, outPath)
	case ".docx", ".pptx", ".xlsx":
		err = generateOfficeCover(file, outPath)
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff":
		err = generateImageCover(file, outPath)
	default:
		return fmt.Errorf("unsupported file type for cover generation: %s", ext)
	}

	if err != nil {
		return err
	}

	// Resize the generated cover to standard dimensions
	return resizeCover(outPath, CoverWidth, CoverHeight)
}

// resizeCover resizes an image to the specified dimensions
func resizeCover(imagePath string, width, height int) error {
	img, err := imaging.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image for resizing: %w", err)
	}

	// Resize using Lanczos filter for quality
	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	// Save back to the same path
	return imaging.Save(resized, imagePath)
}

// generateImageCover creates a cover from an image file
func generateImageCover(imagePath, outPath string) error {
	img, err := imaging.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Save as JPEG
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	return jpeg.Encode(out, img, &jpeg.Options{Quality: 85})
}

// generatePdfCover generates a cover from the first page of a PDF
func generatePdfCover(pdfPath, outPath string) error {
	doc, err := fitz.New(pdfPath)
	if err != nil {
		return fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	// Render first page (index 0)
	img, err := doc.Image(0)
	if err != nil {
		return fmt.Errorf("failed to render PDF page: %w", err)
	}

	// Save as JPEG
	return saveImageAsJpeg(img, outPath)
}

// generateOfficeCover generates a cover for Office documents by converting to PDF first
func generateOfficeCover(filePath, outPath string) error {
	// Create temporary directory for PDF conversion
	tmpDir, err := os.MkdirTemp("", "office-cover-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Convert to PDF using LibreOffice
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

	return generatePdfCover(pdfPath, outPath)
}

// generateMarkdownCover generates a cover by rendering Markdown to HTML and taking a screenshot
func generateMarkdownCover(mdPath, outPath string) error {
	// Read Markdown file
	mdData, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}

	// Configure Markdown parser with GitHub Flavored Markdown
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

	// Build full HTML document
	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8">`)
	buf.WriteString(CssStyle)
	buf.WriteString(`</head><body>`)

	if err := mdParser.Convert(mdData, &buf); err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}
	buf.WriteString(`</body></html>`)
	fullHTML := buf.String()

	// Take screenshot using headless Chrome
	return takeHTMLScreenshot(fullHTML, outPath)
}

// takeHTMLScreenshot renders HTML and takes a screenshot using headless Chrome
func takeHTMLScreenshot(htmlContent, outPath string) error {
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
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.EmulateViewport(1280, 720),
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
		chromedp.Sleep(500*time.Millisecond),
		chromedp.FullScreenshot(&screenshotBuf, 90),
	)

	if err != nil {
		return fmt.Errorf("chrome screenshot failed: %w", err)
	}

	// Decode and save
	img, err := imaging.Decode(bytes.NewReader(screenshotBuf))
	if err != nil {
		return fmt.Errorf("failed to decode screenshot: %w", err)
	}

	return saveImageAsJpeg(img, outPath)
}

// saveImageAsJpeg saves an image as JPEG with quality 85
func saveImageAsJpeg(img image.Image, outPath string) error {
	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	return jpeg.Encode(out, img, &jpeg.Options{Quality: 85})
}
