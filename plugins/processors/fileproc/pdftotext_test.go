/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package fileproc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSplitPdftotextPages(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single page", "hello world", []string{"hello world"}},
		{"pages split on form feed", "page one\fpage two\fpage three", []string{"page one", "page two", "page three"}},
		{"blank pages dropped", "real page 1\f\f   \fpage 2", []string{"real page 1", "page 2"}},
		{"trailing feed from final page break", "only page\f", []string{"only page"}},
		{"whitespace trimmed", "  spaced  \f\n\nlined\n", []string{"spaced", "lined"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := splitPdftotextPages(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("got %d pages %q, want %d", len(got), got, len(c.want))
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("page %d: got %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestPdftotextPagesWithoutBinary(t *testing.T) {
	if PdftotextAvailable() {
		t.Skip("pdftotext installed; absence path not reachable")
	}
	_, err := PdftotextPages(context.Background(), 5, filepath.Join(t.TempDir(), "x.pdf"))
	if err == nil || !strings.Contains(err.Error(), "pdftotext not available") {
		t.Fatalf("expected not-available error, got %v", err)
	}
}

func TestPdftotextPagesTimeout(t *testing.T) {
	if !PdftotextAvailable() {
		t.Skip("pdftotext not installed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already-expired context must surface as an error
	_, err := PdftotextPages(ctx, 1, "/etc/hosts")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	_ = time.Second
}
