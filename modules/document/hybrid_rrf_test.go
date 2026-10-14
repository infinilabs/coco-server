/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
)

func rrfHit(id string, score float32) elastic.DocumentWithMeta[core.Document] {
	hit := elastic.DocumentWithMeta[core.Document]{}
	hit.ID = id
	hit.Score = score
	hit.Source.Title = "doc-" + id
	return hit
}

func rrfRoutes(pairs ...rrfRouteHits) []rrfRouteHits { return pairs }

func TestRRFConfigNormalized(t *testing.T) {
	got := RRFConfig{}.normalized()
	if got.K != defaultRRFK {
		t.Fatalf("k should default, got %v", got.K)
	}
	for _, name := range rrfRouteNames {
		if got.weight(name) != 1 {
			t.Fatalf("route %s should default to weight 1, got %+v", name, got.Weights)
		}
	}

	got = RRFConfig{K: 0.5, Weights: map[string]float64{rrfRouteText: -2, rrfRouteSemantic: 3}}.normalized()
	if got.K != defaultRRFK {
		t.Errorf("k below 1 should clamp to default, got %v", got.K)
	}
	if got.weight(rrfRouteText) != defaultRRFWeight {
		t.Errorf("negative weight should clamp to default, got %v", got.weight(rrfRouteText))
	}
	if got.weight(rrfRouteSemantic) != 3 {
		t.Errorf("valid weight should stay, got %v", got.weight(rrfRouteSemantic))
	}
	// unlisted routes default to 1
	if got.weight(rrfRouteWiki) != 1 {
		t.Errorf("unlisted route should default to 1, got %v", got.weight(rrfRouteWiki))
	}

	got = RRFConfig{K: 10, Weights: map[string]float64{rrfRouteText: 0, rrfRouteSemantic: 0}}.normalized()
	if got.weight(rrfRouteText) != 1 || got.weight(rrfRouteSemantic) != 1 {
		t.Fatalf("all-zero weights should reset to 1, got %+v", got.Weights)
	}

	got = RRFConfig{K: 10, Weights: map[string]float64{rrfRouteText: 2, rrfRouteSemantic: 0}}.normalized()
	if got.weight(rrfRouteText) != 2 || got.weight(rrfRouteSemantic) != 0 {
		t.Fatalf("deliberately muting one route must survive, got %+v", got.Weights)
	}
}

func TestRRFFuseMultiMath(t *testing.T) {
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 9), rrfHit("b", 8), rrfHit("c", 7)}
	semantic := []elastic.DocumentWithMeta[core.Document]{rrfHit("b", 0.9), rrfHit("d", 0.8), rrfHit("a", 0.7)}

	cfg := RRFConfig{K: 60, Weights: map[string]float64{rrfRouteText: 1, rrfRouteSemantic: 1}}
	hits, breakdowns := rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteSemantic, Hits: semantic},
	), cfg)

	if len(hits) != 4 || len(breakdowns) != 4 {
		t.Fatalf("expected 4 unique docs, got %d", len(hits))
	}

	byID := map[string]rrfBreakdown{}
	for _, b := range breakdowns {
		byID[b.ID] = b
	}

	// a: text rank 1 + semantic rank 3
	wantA := 1.0/61.0 + 1.0/63.0
	if diff := byID["a"].Score - wantA; diff > 1e-12 || diff < -1e-12 {
		t.Errorf("doc a score = %v, want %v", byID["a"].Score, wantA)
	}
	if byID["a"].Ranks[rrfRouteText] != 1 || byID["a"].Ranks[rrfRouteSemantic] != 3 {
		t.Errorf("doc a ranks wrong: %+v", byID["a"].Ranks)
	}

	// b: text rank 2 + semantic rank 1 — the winner
	wantB := 1.0/62.0 + 1.0/61.0
	if diff := byID["b"].Score - wantB; diff > 1e-12 || diff < -1e-12 {
		t.Errorf("doc b score = %v, want %v", byID["b"].Score, wantB)
	}

	// c: text only
	if _, ok := byID["c"].Ranks[rrfRouteSemantic]; ok || byID["c"].Contributions[rrfRouteSemantic] != 0 {
		t.Errorf("doc c should have no semantic contribution: %+v", byID["c"])
	}

	if hits[0].ID != "b" || hits[1].ID != "a" {
		t.Errorf("fused order should be b,a first, got %s,%s", hits[0].ID, hits[1].ID)
	}
	for i, hit := range hits {
		if hit.ID != breakdowns[i].ID {
			t.Fatalf("hits and breakdowns must stay parallel at %d", i)
		}
	}
}

func TestRRFFuseMultiThreeRoutes(t *testing.T) {
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 9), rrfHit("b", 8)}
	semantic := []elastic.DocumentWithMeta[core.Document]{rrfHit("b", 0.9), rrfHit("c", 0.8)}
	wiki := []elastic.DocumentWithMeta[core.Document]{rrfHit("c", 5), rrfHit("a", 4)}

	hits, breakdowns := rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteSemantic, Hits: semantic},
		rrfRouteHits{Name: rrfRouteWiki, Hits: wiki},
	), RRFConfig{K: 60})

	byID := map[string]rrfBreakdown{}
	for _, b := range breakdowns {
		byID[b.ID] = b
	}
	// a: text#1 + wiki#2; b: text#2 + semantic#1; c: semantic#2 + wiki#1 —
	// all three tie; map iteration order makes the addition order vary, so
	// compare with tolerance
	want := 1.0/61.0 + 1.0/62.0
	for _, id := range []string{"a", "b", "c"} {
		if diff := byID[id].Score - want; diff > 1e-12 || diff < -1e-12 {
			t.Fatalf("doc %s score = %v, want %v", id, byID[id].Score, want)
		}
	}
	// a, b, c all tie at the same fused score; a and b were recalled by two
	// routes like c — the best-rank tie-break decides (a has rank 1)
	winner, ok := byID[hits[0].ID]
	if !ok {
		t.Fatalf("winner %s missing breakdown", hits[0].ID)
	}
	if best := winner.breakdownBestRankForTest(); best != 1 {
		t.Fatalf("winner should hold a rank-1 from some route, got %d", best)
	}
}

func TestRRFFuseMultiWeightsFlipOrder(t *testing.T) {
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 9), rrfHit("b", 8)}
	semantic := []elastic.DocumentWithMeta[core.Document]{rrfHit("b", 0.9), rrfHit("a", 0.8)}

	hits, _ := rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteSemantic, Hits: semantic},
	), RRFConfig{K: 10, Weights: map[string]float64{rrfRouteText: 5, rrfRouteSemantic: 1}})
	if hits[0].ID != "a" {
		t.Errorf("text-heavy fusion should rank a first, got %s", hits[0].ID)
	}

	hits, _ = rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteSemantic, Hits: semantic},
	), RRFConfig{K: 10, Weights: map[string]float64{rrfRouteText: 1, rrfRouteSemantic: 5}})
	if hits[0].ID != "b" {
		t.Errorf("semantic-heavy fusion should rank b first, got %s", hits[0].ID)
	}

	// zero semantic weight: pure BM25 ranking, semantic-only docs still return
	semanticOnly := []elastic.DocumentWithMeta[core.Document]{rrfHit("z", 0.9)}
	hits, _ = rrfFuseMulti(rrfRoutes(
		rrfRouteHits{Name: rrfRouteText, Hits: text},
		rrfRouteHits{Name: rrfRouteSemantic, Hits: semanticOnly},
	), RRFConfig{K: 10, Weights: map[string]float64{rrfRouteText: 1, rrfRouteSemantic: 0}})
	if len(hits) != 3 {
		t.Fatalf("muted route's docs should still be returned (score 0), got %d hits", len(hits))
	}
	if hits[0].ID != "a" || hits[1].ID != "b" {
		t.Errorf("pure BM25 order wrong: %s,%s", hits[0].ID, hits[1].ID)
	}
}

func TestRRFFuseMultiEmptyRoutes(t *testing.T) {
	hits, breakdowns := rrfFuseMulti(nil, RRFConfig{})
	if len(hits) != 0 || len(breakdowns) != 0 {
		t.Fatalf("empty routes should fuse to empty, got %d hits", len(hits))
	}

	semantic := []elastic.DocumentWithMeta[core.Document]{rrfHit("s1", 0.9)}
	hits, _ = rrfFuseMulti(rrfRoutes(rrfRouteHits{Name: rrfRouteSemantic, Hits: semantic}), RRFConfig{K: 60})
	if len(hits) != 1 || hits[0].ID != "s1" {
		t.Fatalf("single-route fusion broken: %+v", hits)
	}
}

// breakdownBestRankForTest mirrors the fuse's internal tie-break helper.
func (b rrfBreakdown) breakdownBestRankForTest() int {
	best := 0
	for _, rank := range b.Ranks {
		if best == 0 || rank < best {
			best = rank
		}
	}
	return best
}
