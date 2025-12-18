/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package file_extraction

import (
	"testing"

	"infini.sh/coco/core"
)

func TestSplitPagesToChunks_NonPositiveChunkSize(t *testing.T) {
	chunks := SplitPagesToChunks([]string{"abc"}, 0)

	if chunks != nil {
		t.Fatalf("expected nil slices for non-positive chunk size, got chunks=%+v", chunks)
	}
}

func TestSplitPagesToChunks_EmptyPages(t *testing.T) {
	chunks := SplitPagesToChunks([]string{}, 4)

	if len(chunks) != 0 {
		t.Fatalf("expected empty results for empty input, got chunks=%+v", chunks)
	}
}

func TestSplitPagesToChunks_SpansPages(t *testing.T) {
	pages := []string{"abc", "def", "gh"}

	chunks := SplitPagesToChunks(pages, 5)

	expectedChunks := []core.DocumentChunk{
		{Range: core.ChunkRange{Start: 1, End: 2}, Text: "abcde"},
		{Range: core.ChunkRange{Start: 2, End: 3}, Text: "fgh"},
	}

	if len(chunks) != len(expectedChunks) {
		t.Fatalf("expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}

	for i, expected := range expectedChunks {
		if chunks[i].Text != expected.Text {
			t.Fatalf("chunk %d text mismatch: expected %q, got %q", i, expected.Text, chunks[i].Text)
		}
		if chunks[i].Range != expected.Range {
			t.Fatalf("chunk %d range mismatch: expected %+v, got %+v", i, expected.Range, chunks[i].Range)
		}
	}
}

func TestSplitPagesToChunks_SinglePageMultipleChunks(t *testing.T) {
	pages := []string{"abcdefgh"}

	chunks := SplitPagesToChunks(pages, 4)

	expectedChunks := []core.DocumentChunk{
		{Range: core.ChunkRange{Start: 1, End: 1}, Text: "abcd"},
		{Range: core.ChunkRange{Start: 1, End: 1}, Text: "efgh"},
	}

	if len(chunks) != len(expectedChunks) {
		t.Fatalf("expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}

	for i, expected := range expectedChunks {
		if chunks[i].Text != expected.Text {
			t.Fatalf("chunk %d text mismatch: expected %q, got %q", i, expected.Text, chunks[i].Text)
		}
		if chunks[i].Range != expected.Range {
			t.Fatalf("chunk %d range mismatch: expected %+v, got %+v", i, expected.Range, chunks[i].Range)
		}
	}
}
