/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package fileproc

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PdftotextAvailable reports whether poppler's pdftotext is installed and so
// the PDF fallback extraction route exists on this host.
func PdftotextAvailable() bool {
	_, err := exec.LookPath("pdftotext")
	return err == nil
}

// PdftotextPages extracts text from a PDF with poppler's pdftotext and
// returns one entry per page. It exists as the fallback for PDFs Tika cannot
// parse: PDFBox still crashes with a NullPointerException on the system-font
// substitution path for non-embedded CID fonts (PDFBOX-3515/2842 class,
// reproduced on tika-server 3.2.3 and 3.3.3), which kills the whole
// extraction; poppler handles these fonts via its built-in CMap resources.
// pdftotext separates pages with form feeds, which maps directly onto the
// per-page model the document pipeline uses.
func PdftotextPages(ctx context.Context, timeout int, path string) ([]string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return nil, fmt.Errorf("pdftotext not available on this host: %w", err)
	}

	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	// -q keeps copyright banners out of the text; output to stdout so nothing
	// is written next to the source file.
	cmd := exec.CommandContext(cctx, "pdftotext", "-q", "-enc", "UTF-8", path, "-")
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		snippet := strings.TrimSpace(stderr.String())
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("pdftotext failed for %s: %w: %s", path, err, snippet)
	}

	pages := splitPdftotextPages(stdout.String())
	if len(pages) == 0 {
		return nil, fmt.Errorf("pdftotext produced no text for %s", path)
	}
	return pages, nil
}

// splitPdftotextPages converts pdftotext's form-feed-separated output into
// the per-page list, dropping blank pages (scans with no text layer produce
// a run of empty feeds).
func splitPdftotextPages(text string) []string {
	var pages []string
	for _, page := range strings.Split(text, "\f") {
		if page = strings.TrimSpace(page); page != "" {
			pages = append(pages, page)
		}
	}
	return pages
}
