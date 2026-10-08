/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func writeParseFile(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte("%PDF-1.4 fake bytes"), 0o644))
	return p
}

func TestParseRemoteJSONPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pages":["# 第一章","正文一","# 第二章","正文二"]}`))
	}))
	defer srv.Close()

	pages, err := parseRemoteFn(context.Background(), srv.URL, "book.pdf", writeParseFile(t, "book.pdf"), 5*time.Second)
	require.NoError(t, err)
	require.Len(t, pages, 4)
	assert.Equal(t, "# 第一章", pages[0])
}

func TestParseRemotePlainTextFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("  整篇 markdown 文本  "))
	}))
	defer srv.Close()

	pages, err := parseRemoteFn(context.Background(), srv.URL, "doc.pdf", writeParseFile(t, "doc.pdf"), 5*time.Second)
	require.NoError(t, err)
	require.Len(t, pages, 1)
	assert.Equal(t, "整篇 markdown 文本", pages[0])
}

func TestParseRemoteErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	_, err := parseRemoteFn(context.Background(), srv.URL, "x.pdf", writeParseFile(t, "x.pdf"), 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestParseRemoteEmptyAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"pages":["", "  "]}`))
	}))
	defer srv.Close()

	_, err := parseRemoteFn(context.Background(), srv.URL, "x.pdf", writeParseFile(t, "x.pdf"), 5*time.Second)
	require.Error(t, err, "an empty answer loses the first-parser race, the caller falls back")
}

func TestTryRemoteParseBackendGating(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"pages":["远程解析结果"]}`))
	}))
	defer srv.Close()

	p := &DocumentTextAttachmentExtractionProcessor{config: &DocumentConfig{
		RemoteParseURL:        srv.URL,
		RemoteParseTimeoutSec: 5,
		RemoteParseExts:       []string{".pdf"},
	}}
	doc := &core.Document{Title: "report.pdf"}

	// configured extension: remote wins
	pages, won := p.tryRemoteParseBackend(context.Background(), doc, ".pdf", writeParseFile(t, "report.pdf"))
	require.True(t, won)
	assert.Equal(t, []string{"远程解析结果"}, pages)

	// unconfigured extension: not even attempted
	_, won = p.tryRemoteParseBackend(context.Background(), doc, ".docx", writeParseFile(t, "a.docx"))
	assert.False(t, won)

	// no URL configured: chain disabled
	p2 := &DocumentTextAttachmentExtractionProcessor{config: &DocumentConfig{}}
	_, won = p2.tryRemoteParseBackend(context.Background(), doc, ".pdf", writeParseFile(t, "b.pdf"))
	assert.False(t, won)
}

func TestTryRemoteParseBackendFallsBackOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := &DocumentTextAttachmentExtractionProcessor{config: &DocumentConfig{
		RemoteParseURL:  srv.URL,
		RemoteParseExts: []string{".pdf"},
	}}
	doc := &core.Document{Title: "x.pdf"}
	_, won := p.tryRemoteParseBackend(context.Background(), doc, ".pdf", writeParseFile(t, "x.pdf"))
	assert.False(t, won, "a failing remote backend falls through to the built-in routes")
}

func TestExtInList(t *testing.T) {
	assert.True(t, extInList(".PDF", []string{".pdf"}))
	assert.False(t, extInList(".pdf", []string{".docx"}))
	assert.False(t, extInList(".pdf", nil))
}
