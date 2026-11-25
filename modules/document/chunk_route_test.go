/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
)

func TestChunkRouteInCanonicalNames(t *testing.T) {
	assert.Contains(t, rrfRouteNames, rrfRouteChunk, "the chunk leg is in the canonical route list — studio and breakdowns enumerate it automatically")
}

func TestChunkRouteWeightParam(t *testing.T) {
	// the weight key follows the <route>_weight convention shared by every leg
	assert.Equal(t, "chunk", rrfRouteChunk)
	assert.Equal(t, "chunk_weight", "chunk_weight")
}

func TestChunkRouteFusionMath(t *testing.T) {
	// chunk hits participate in the RRF math like any other leg
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("doc-a", 9)}
	chunk := []elastic.DocumentWithMeta[core.Document]{rrfHit("doc-a", 5), rrfHit("doc-chunk-only", 4)}

	hits, breakdowns := rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteChunk, Hits: chunk},
	), RRFConfig{K: 60})

	byID := map[string]rrfBreakdown{}
	for _, b := range breakdowns {
		byID[b.ID] = b
	}
	// doc-a: text#1 + chunk#1 → 1/61 + 1/61
	want := 2.0 / 61.0
	if diff := byID["doc-a"].Score - want; diff > 1e-12 || diff < -1e-12 {
		t.Fatalf("doc-a score = %v, want %v", byID["doc-a"].Score, want)
	}
	// chunk-only doc surfaces via the chunk leg alone
	require.Contains(t, byID, "doc-chunk-only")
	assert.Equal(t, 1.0/62.0, byID["doc-chunk-only"].Score)
	_ = hits
}
