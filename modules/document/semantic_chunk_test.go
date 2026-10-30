/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

func chunkHit(id string, score float32) elastic.DocumentWithMeta[core.Document] {
	return elastic.DocumentWithMeta[core.Document]{ID: id, Score: score}
}

func chunkVec(v []float32, text, breadcrumb string, start, end int) semanticChunkVec {
	return semanticChunkVec{Vector: v, Text: text, Breadcrumb: breadcrumb, StartPage: start, EndPage: end}
}

func TestRefineByBestChunkPicksMaxCosine(t *testing.T) {
	query := []float32{1, 0}
	// chunk 1 is the query's direction; chunk 0 is orthogonal, chunk 2 opposite
	chunks := map[string][]semanticChunkVec{
		"a": {
			chunkVec([]float32{0, 1}, "正交块", "根", 1, 1),
			chunkVec([]float32{1, 0}, "命中块", "根 > 财务", 2, 3),
			chunkVec([]float32{-1, 0}, "反向块", "根", 4, 4),
		},
	}
	hits := []elastic.DocumentWithMeta[core.Document]{chunkHit("a", 1)}

	refined := refineHitsByBestChunk(chunks, query, hits)
	require.Equal(t, 1, refined)
	// cosine(query, best)=1 → score = 1+1 = 2
	assert.InDelta(t, float32(2.0), hits[0].Score, 1e-6)

	sc, ok := hits[0].Source.Metadata["semantic_chunk"].(util.MapStr)
	require.True(t, ok, "hit must carry the block locator")
	assert.Equal(t, 1, sc["index"], "the best-cosine chunk wins, not the first")
	assert.Equal(t, "根 > 财务", sc["breadcrumb"])
	assert.Equal(t, "命中块", sc["quote"])
	pages := sc["pages"].(util.MapStr)
	assert.Equal(t, 2, pages["start"])
	assert.Equal(t, 3, pages["end"])
}

func TestRefineByBestChunkReranksHead(t *testing.T) {
	query := []float32{1, 0}
	// doc "b" ranks below "a" on doc-level scores, but its single chunk
	// aligns with the query — after refinement b must overtake a
	chunks := map[string][]semanticChunkVec{
		"b": {chunkVec([]float32{1, 0}, "b 命中", "B", 1, 1)},
	}
	hits := []elastic.DocumentWithMeta[core.Document]{chunkHit("a", 1.8), chunkHit("b", 1.2)}

	refineHitsByBestChunk(chunks, query, hits)
	assert.Equal(t, "b", hits[0].ID, "best-chunk score must reorder the head")
	assert.Equal(t, "a", hits[1].ID)
}

func TestRefineByBestChunkLeavesVectorlessHits(t *testing.T) {
	query := []float32{1, 0}
	hits := []elastic.DocumentWithMeta[core.Document]{chunkHit("plain", 1.5), chunkHit("none", 1.4)}

	refined := refineHitsByBestChunk(map[string][]semanticChunkVec{}, query, hits)
	assert.Zero(t, refined, "no chunk vectors → nothing refined")
	assert.Equal(t, "plain", hits[0].ID, "doc-level order and scores stay untouched")
	assert.InDelta(t, float32(1.5), hits[0].Score, 1e-6)
	assert.Nil(t, hits[0].Source.Metadata["semantic_chunk"])
}

func TestRefineByBestChunkRespectsCap(t *testing.T) {
	query := []float32{1, 0}
	chunks := map[string][]semanticChunkVec{}
	hits := make([]elastic.DocumentWithMeta[core.Document], 0, chunkRefineCap+10)
	for i := 0; i < chunkRefineCap+10; i++ {
		id := "doc-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		hits = append(hits, chunkHit(id, float32(2-i))) // descending doc-level order
		chunks[id] = []semanticChunkVec{chunkVec([]float32{1, 0}, "q", "", 1, 1)}
	}

	refined := refineHitsByBestChunk(chunks, query, hits)
	assert.Equal(t, chunkRefineCap, refined, "only the head gets refined")
	// the cap+10-th doc keeps its original doc-level score
	last := hits[len(hits)-1]
	assert.Nil(t, last.Source.Metadata["semantic_chunk"])
}

func TestQuoteTruncation(t *testing.T) {
	long := strings.Repeat("长", 400)
	assert.Equal(t, semanticQuoteRunes, len([]rune(truncateRunes(long, semanticQuoteRunes))))
	assert.Equal(t, "short", truncateRunes("short", semanticQuoteRunes))
}

func TestExpandNeighbors(t *testing.T) {
	cs := []semanticChunkVec{
		{Text: strings.Repeat("前", 500)},
		{Text: strings.Repeat("中", 100)}, // short best chunk triggers expansion
		{Text: strings.Repeat("后", 500)},
	}
	before, after := expandNeighbors(cs, 1)
	assert.NotEmpty(t, before, "short chunk pulls the previous chunk's tail")
	assert.NotEmpty(t, after, "and the next chunk's head")
	total := 100 + len([]rune(before)) + len([]rune(after))
	assert.LessOrEqual(t, total, neighborExpandTarget, "expansion respects the target budget")
	assert.Equal(t, strings.Repeat("前", len([]rune(before))), before)
	assert.Equal(t, strings.Repeat("后", len([]rune(after))), after)

	// a long-enough chunk does not expand
	before, after = expandNeighbors([]semanticChunkVec{{Text: strings.Repeat("长", 400)}}, 0)
	assert.Empty(t, before)
	assert.Empty(t, after)

	// boundary chunks only pull the existing side
	before, after = expandNeighbors([]semanticChunkVec{{Text: "独"}, {Text: strings.Repeat("下", 500)}}, 0)
	assert.Empty(t, before)
	assert.NotEmpty(t, after)
}
