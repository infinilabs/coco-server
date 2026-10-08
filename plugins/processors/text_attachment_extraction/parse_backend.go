/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
)

// Pluggable parse backends (W2 L1): Tika stays the always-on baseline;
// a remote vision parser (mineru / docling class, PDF→markdown) can take
// over per pipeline for the extensions the operator lists. First-parser
// semantics: the remote backend runs FIRST for its extensions and wins on
// a non-empty answer; any failure (network, timeout, garbage) falls back
// to the Tika path — an optional accelerator must never become a
// single point of failure for ingestion.
//
// Protocol: POST the raw file bytes to <url>?filename=<name>, response is
// either JSON {"pages": ["...", ...]} or plain text (one page).

const remoteParseDefaultTimeoutSec = 300

// remoteParseResult accepts both response shapes.
type remoteParseResult struct {
	Pages   []string `json:"pages"`
	Content string   `json:"content"`
}

// parseRemote posts the file to the configured backend and returns pages.
var parseRemoteFn = parseRemote

func parseRemote(ctx context.Context, endpoint, filename, localPath string, timeout time.Duration) ([]string, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost,
		fmt.Sprintf("%s?filename=%s", strings.TrimSuffix(endpoint, "/"), url.QueryEscape(filename)), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20)) // 64MB answer cap
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("remote parser returned %s: %s", resp.Status, msg)
	}

	// JSON shape first; plain text is the single-page fallback
	parsed := remoteParseResult{}
	if err := json.Unmarshal(body, &parsed); err == nil {
		pages := make([]string, 0, len(parsed.Pages))
		for _, p := range parsed.Pages {
			if strings.TrimSpace(p) != "" {
				pages = append(pages, p)
			}
		}
		if len(pages) > 0 {
			return pages, nil
		}
		if strings.TrimSpace(parsed.Content) != "" {
			return []string{parsed.Content}, nil
		}
		return nil, fmt.Errorf("remote parser answered empty")
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return nil, fmt.Errorf("remote parser answered empty")
	}
	return []string{text}, nil
}

// tryRemoteParseBackend applies the first-parser chain for the routed
// extension: remote first when configured for it, Tika flow otherwise.
// Returns (pages, true) when the remote backend won, (nil, false) to
// fall through to the built-in routes.
func (p *DocumentTextAttachmentExtractionProcessor) tryRemoteParseBackend(ctx context.Context, doc *core.Document, routedExt, localPath string) ([]string, bool) {
	cfg := p.config
	if cfg.RemoteParseURL == "" || !extInList(routedExt, cfg.RemoteParseExts) {
		return nil, false
	}
	timeout := time.Duration(cfg.RemoteParseTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = remoteParseDefaultTimeoutSec * time.Second
	}
	pages, err := parseRemoteFn(ctx, cfg.RemoteParseURL, doc.Title, localPath, timeout)
	if err != nil {
		log.Warnf("processor [%s] remote parse backend failed for [%s] (%v), falling back to built-in", p.Name(), doc.Title, err)
		return nil, false
	}
	log.Infof("processor [%s] remote parse backend won for [%s]: %d pages", p.Name(), doc.Title, len(pages))
	return pages, true
}

func extInList(ext string, list []string) bool {
	for _, e := range list {
		if strings.EqualFold(strings.TrimSpace(e), ext) {
			return true
		}
	}
	return false
}
