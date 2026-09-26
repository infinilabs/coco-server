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

func TestRRFConfigNormalized(t *testing.T) {
	got := RRFConfig{}.normalized()
	if got.K != defaultRRFK || got.TextWeight != 1 || got.SemanticWeight != 1 {
		t.Fatalf("zero config should become defaults, got %+v", got)
	}

	got = RRFConfig{K: 0.5, TextWeight: -2, SemanticWeight: 3}.normalized()
	if got.K != defaultRRFK {
		t.Errorf("k below 1 should clamp to default, got %v", got.K)
	}
	if got.TextWeight != defaultRRFWeight {
		t.Errorf("negative text weight should clamp to default, got %v", got.TextWeight)
	}
	if got.SemanticWeight != 3 {
		t.Errorf("valid semantic weight should stay, got %v", got.SemanticWeight)
	}

	got = RRFConfig{K: 10, TextWeight: 0, SemanticWeight: 0}.normalized()
	if got.TextWeight != 1 || got.SemanticWeight != 1 {
		t.Fatalf("both weights zero should reset to 1/1, got %+v", got)
	}

	got = RRFConfig{K: 10, TextWeight: 2, SemanticWeight: 0}.normalized()
	if got.TextWeight != 2 || got.SemanticWeight != 0 {
		t.Fatalf("deliberately disabling one route must survive, got %+v", got)
	}
}

func TestRRFFuseMath(t *testing.T) {
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 9), rrfHit("b", 8), rrfHit("c", 7)}
	semantic := []elastic.DocumentWithMeta[core.Document]{rrfHit("b", 0.9), rrfHit("d", 0.8), rrfHit("a", 0.7)}

	cfg := RRFConfig{K: 60, TextWeight: 1, SemanticWeight: 1}
	hits, breakdowns := rrfFuse(text, semantic, cfg)

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
	if byID["a"].TextRank != 1 || byID["a"].SemanticRank != 3 {
		t.Errorf("doc a ranks wrong: %+v", byID["a"])
	}

	// b: text rank 2 + semantic rank 1 — the winner
	wantB := 1.0/62.0 + 1.0/61.0
	if diff := byID["b"].Score - wantB; diff > 1e-12 || diff < -1e-12 {
		t.Errorf("doc b score = %v, want %v", byID["b"].Score, wantB)
	}

	// c: text only
	if byID["c"].SemanticRank != 0 || byID["c"].SemanticContribution != 0 {
		t.Errorf("doc c should have no semantic contribution: %+v", byID["c"])
	}

	// d: semantic only
	if byID["d"].TextRank != 0 || byID["d"].TextContribution != 0 {
		t.Errorf("doc d should have no text contribution: %+v", byID["d"])
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

func TestRRFFuseWeightsFlipOrder(t *testing.T) {
	text := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 9), rrfHit("b", 8)}
	semantic := []elastic.DocumentWithMeta[core.Document]{rrfHit("b", 0.9), rrfHit("a", 0.8)}

	// heavy text weight: a (text#1) wins
	hits, _ := rrfFuse(text, semantic, RRFConfig{K: 10, TextWeight: 5, SemanticWeight: 1})
	if hits[0].ID != "a" {
		t.Errorf("text-heavy fusion should rank a first, got %s", hits[0].ID)
	}

	// heavy semantic weight: b (semantic#1) wins
	hits, _ = rrfFuse(text, semantic, RRFConfig{K: 10, TextWeight: 1, SemanticWeight: 5})
	if hits[0].ID != "b" {
		t.Errorf("semantic-heavy fusion should rank b first, got %s", hits[0].ID)
	}

	// zero semantic weight: pure BM25 ranking, semantic-only docs drop out
	semanticOnly := []elastic.DocumentWithMeta[core.Document]{rrfHit("z", 0.9)}
	hits, _ = rrfFuse(text, semanticOnly, RRFConfig{K: 10, TextWeight: 1, SemanticWeight: 0})
	if len(hits) != 3 {
		t.Fatalf("semantic-only docs should still be returned (score 0), got %d hits", len(hits))
	}
	if hits[0].ID != "a" || hits[1].ID != "b" {
		t.Errorf("pure BM25 order wrong: %s,%s", hits[0].ID, hits[1].ID)
	}
}

func TestRRFFuseEmptyRoutes(t *testing.T) {
	hits, breakdowns := rrfFuse(nil, nil, RRFConfig{})
	if len(hits) != 0 || len(breakdowns) != 0 {
		t.Fatalf("empty routes should fuse to empty, got %d", len(hits))
	}

	semantic := []elastic.DocumentWithMeta[core.Document]{rrfHit("s1", 0.9)}
	hits, _ = rrfFuse(nil, semantic, RRFConfig{K: 60, TextWeight: 1, SemanticWeight: 1})
	if len(hits) != 1 || hits[0].ID != "s1" {
		t.Fatalf("single-route fusion broken: %+v", hits)
	}
}
