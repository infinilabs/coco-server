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
	// structured splitter (W3): 25 runes at cap 10 with the derived 15%
	// overlap → 3 chunks of 10/10/7 runes; every chunk carries the whole
	// content modulo the one-rune overlap prefixes on continuations
	assert.Len(t, doc.Chunks, 3)
	assert.Equal(t, 10, len([]rune(doc.Chunks[0].Text)))
	assert.Equal(t, 7, len([]rune(doc.Chunks[2].Text)))
	assert.Equal(t, core.ChunkRange{Start: 1, End: 1}, doc.Chunks[0].Range)
	var rebuilt strings.Builder
	for i, c := range doc.Chunks {
		runes := []rune(c.Text)
		if i > 0 && len(runes) > 0 {
			runes = runes[1:] // strip the single-rune overlap prefix
		}
		rebuilt.WriteString(string(runes))
	}
	assert.Equal(t, doc.Content, rebuilt.String(), "chunks minus overlap prefixes must reassemble to the content")
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
