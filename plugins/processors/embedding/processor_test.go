/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package embedding

import (
	"testing"

	"infini.sh/coco/core"
)

func TestSplitPagesToChunks_NonPositiveChunkSize(t *testing.T) {
	chunks, ranges := SplitPagesToChunks([]core.PageText{{PageNumber: 1, Content: "abc"}}, 0)

	if chunks != nil || ranges != nil {
		t.Fatalf("expected nil slices for non-positive chunk size, got chunks=%v ranges=%v", chunks, ranges)
	}
}

func TestSplitPagesToChunks_EmptyPages(t *testing.T) {
	chunks, ranges := SplitPagesToChunks([]core.PageText{}, 4)

	if len(chunks) != 0 || len(ranges) != 0 {
		t.Fatalf("expected empty results for empty input, got chunks=%v ranges=%v", chunks, ranges)
	}
}

func TestSplitPagesToChunks_SpansPages(t *testing.T) {
	pages := []core.PageText{
		{PageNumber: 1, Content: "abc"},
		{PageNumber: 2, Content: "def"},
		{PageNumber: 3, Content: "gh"},
	}

	chunks, ranges := SplitPagesToChunks(pages, 5)

	expectedChunks := []string{"abcde", "fgh"}
	expectedRanges := []core.ChunkRange{{Start: 1, End: 2}, {Start: 2, End: 3}}

	if len(chunks) != len(expectedChunks) {
		t.Fatalf("expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}
	for i := range expectedChunks {
		if chunks[i] != expectedChunks[i] {
			t.Fatalf("chunk %d mismatch: expected %q, got %q", i, expectedChunks[i], chunks[i])
		}
		if ranges[i] != expectedRanges[i] {
			t.Fatalf("range %d mismatch: expected %+v, got %+v", i, expectedRanges[i], ranges[i])
		}
	}
}

func TestSplitPagesToChunks_SinglePageMultipleChunks(t *testing.T) {
	pages := []core.PageText{{PageNumber: 1, Content: "abcdefgh"}}

	chunks, ranges := SplitPagesToChunks(pages, 4)

	expectedChunks := []string{"abcd", "efgh"}
	expectedRanges := []core.ChunkRange{{Start: 1, End: 1}, {Start: 1, End: 1}}

	if len(chunks) != len(expectedChunks) {
		t.Fatalf("expected %d chunks, got %d", len(expectedChunks), len(chunks))
	}
	for i := range expectedChunks {
		if chunks[i] != expectedChunks[i] {
			t.Fatalf("chunk %d mismatch: expected %q, got %q", i, expectedChunks[i], chunks[i])
		}
		if ranges[i] != expectedRanges[i] {
			t.Fatalf("range %d mismatch: expected %+v, got %+v", i, expectedRanges[i], ranges[i])
		}
	}
}
