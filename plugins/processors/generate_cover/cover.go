/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package generate_cover

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

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	log "github.com/cihub/seelog"
	"github.com/disintegration/imaging"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	_ "golang.org/x/image/webp"

	"infini.sh/coco/plugins/processors/fileproc"
)

const (
	CoverWidth    = 320
	CoverHeight   = 180
	ThumbMaxWidth = 640
)

const cssStyleForThumbnail = `
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

// GenerateCoverAndThumbnail generates a cover image for file and saves it to
// coverOutPath. For image files it also generates an aspect-ratio-preserving
// thumbnail at thumbnailOutPath.
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
			err = GenerateThumbnail(file, thumbnailOutPath)
		}
	default:
		return fmt.Errorf("unsupported file type for cover generation: %s", ext)
	}
	if err != nil {
		return err
	}

	log.Tracef("resizing cover to %dx%d: %s", CoverWidth, CoverHeight, coverOutPath)
	return resizeCover(coverOutPath, CoverWidth, CoverHeight)
}

func resizeCover(imagePath string, width, height int) error {
	img, err := imaging.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image for resizing: %w", err)
	}
	resized := imaging.Fill(img, width, height, imaging.Top, imaging.Lanczos)
	out, err := os.Create(imagePath)
	if err != nil {
		return err
	}
	defer fileproc.DeferClose(out)
	return png.Encode(out, resized)
}

// GenerateThumbnail generates an aspect-ratio-preserving thumbnail. If the
// image width exceeds ThumbMaxWidth it is scaled down proportionally.
func GenerateThumbnail(inputPath, outPath string) error {
	img, err := imaging.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open image for thumbnail: %w", err)
	}

	var finalImg image.Image
	if img.Bounds().Dx() > ThumbMaxWidth {
		finalImg = imaging.Resize(img, ThumbMaxWidth, 0, imaging.Lanczos)
	} else {
		finalImg = img
	}

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer fileproc.DeferClose(out)
	return png.Encode(out, finalImg)
}

func generateImageCover(imagePath, outPath string) error {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}

func generatePdfCover(pdfPath, outPath string) error {
	tmpDir, err := os.MkdirTemp("", "coco-pdf-cover-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPrefix := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-f", "1", "-l", "1", "-jpeg", "-r", "150", pdfPath, outputPrefix)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pdftoppm failed (output: %s): %w", string(output), err)
	}

	generatedFile := outputPrefix + "-1.jpg"
	if _, err := os.Stat(generatedFile); os.IsNotExist(err) {
		generatedFile = outputPrefix + "-01.jpg"
		if _, err := os.Stat(generatedFile); os.IsNotExist(err) {
			return fmt.Errorf("pdftoppm output file not found")
		}
	}
	return fileproc.CopyLocalFile(generatedFile, outPath)
}

func generateOfficeCover(filePath, outPath string) error {
	tmpDir, err := os.MkdirTemp("", "coco-office-cover-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("soffice", "--headless", "--convert-to", "pdf", filePath, "--outdir", tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("soffice conversion failed (output: %s): %w", string(output), err)
	}

	baseName := filepath.Base(filePath)
	pdfName := strings.TrimSuffix(baseName, filepath.Ext(baseName)) + ".pdf"
	pdfPath := filepath.Join(tmpDir, pdfName)
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return fmt.Errorf("converted PDF not found at %s", pdfPath)
	}
	return generatePdfCover(pdfPath, outPath)
}

func generateMarkdownCover(mdPath, outPath string) error {
	mdData, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}

	mdParser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)

	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8">`)
	buf.WriteString(cssStyleForThumbnail)
	buf.WriteString(`</head><body>`)
	if err := mdParser.Convert(mdData, &buf); err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}
	buf.WriteString(`</body></html>`)
	return takeHTMLScreenshot(buf.String(), outPath)
}

func takeHTMLScreenshot(htmlContent, outPath string) error {
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
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var screenshotBuf []byte
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
		chromedp.CaptureScreenshot(&screenshotBuf),
	)
	if err != nil {
		return fmt.Errorf("chrome screenshot failed: %w", err)
	}

	img, err := imaging.Decode(bytes.NewReader(screenshotBuf))
	if err != nil {
		return fmt.Errorf("failed to decode screenshot: %w", err)
	}
	resized := imaging.Resize(img, CoverWidth, CoverHeight, imaging.Lanczos)

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer fileproc.DeferClose(out)
	return png.Encode(out, resized)
}
