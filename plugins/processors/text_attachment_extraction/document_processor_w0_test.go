/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package text_attachment_extraction

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func newChunkTestProcessor(chunkSize int) *DocumentTextAttachmentExtractionProcessor {
	return &DocumentTextAttachmentExtractionProcessor{config: &DocumentConfig{ChunkSize: chunkSize}}
}

func TestChunkContentOnlySplitsContent(t *testing.T) {
	p := newChunkTestProcessor(10)
	doc := &core.Document{Content: strings.Repeat("a", 25)}

	require.True(t, p.chunkContentOnly(doc))
	require.NotEmpty(t, doc.Chunks)
	// 25 runes at chunk_size 10 → 3 chunks (10+10+5); content-only input
	// is a single "page", so every chunk's page range is {1,1}
	assert.Len(t, doc.Chunks, 3)
	assert.Equal(t, 10, len([]rune(doc.Chunks[0].Text)))
	assert.Equal(t, 5, len([]rune(doc.Chunks[2].Text)))
	assert.Equal(t, core.ChunkRange{Start: 1, End: 1}, doc.Chunks[0].Range)
	var rebuilt strings.Builder
	for _, c := range doc.Chunks {
		rebuilt.WriteString(c.Text)
	}
	assert.Equal(t, doc.Content, rebuilt.String(), "chunks must reassemble to the content losslessly")
}

func TestChunkContentOnlyRecomputesOverStaleChunks(t *testing.T) {
	p := newChunkTestProcessor(4)
	doc := &core.Document{Content: "abcdef"}
	// a previous run's chunks must never survive an edit
	doc.Chunks = []core.DocumentChunk{{Text: "stale stale stale"}}

	require.True(t, p.chunkContentOnly(doc))
	require.Len(t, doc.Chunks, 2)
	assert.Equal(t, "abcd", doc.Chunks[0].Text)
}

func TestChunkContentOnlySkipsBlank(t *testing.T) {
	p := newChunkTestProcessor(10)
	for _, content := range []string{"", "   ", "\n\t"} {
		doc := &core.Document{Content: content}
		assert.False(t, p.chunkContentOnly(doc), "blank content %.2q must not be chunked", content)
		assert.Empty(t, doc.Chunks)
	}
	assert.False(t, p.chunkContentOnly(nil))
}

func TestChunkContentOnlyShortContentStillChunks(t *testing.T) {
	// unlike the fingerprint floor (which guards dedup hashes), chunking
	// has no minimum: even a one-line API doc gets its one chunk
	p := newChunkTestProcessor(100)
	doc := &core.Document{Content: "short but real"}
	require.True(t, p.chunkContentOnly(doc))
	require.Len(t, doc.Chunks, 1)
	assert.Equal(t, "short but real", doc.Chunks[0].Text)
}
