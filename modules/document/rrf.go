/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"sort"
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
)

// RRFConfig controls how the BM25 route and the kNN route are fused with
// Reciprocal Rank Fusion:
//
//	score(d) = text_weight/(k+rank_text(d)) + semantic_weight/(k+rank_semantic(d))
//
// Ranks are 1-based; a document recalled by only one route simply gets no
// contribution from the other. k smooths the influence of top ranks — the
// classic default is 60.
type RRFConfig struct {
	K              float64 `json:"k"`
	TextWeight     float64 `json:"text_weight"`
	SemanticWeight float64 `json:"semantic_weight"`
}

const (
	defaultRRFK      = 60.0
	defaultRRFWeight = 1.0
	// maxRRFWindow bounds how many candidates each route fetches before
	// fusion, so a hostile from/size cannot turn one search into two
	// unbounded scans.
	maxRRFWindow = 200
)

func (c RRFConfig) normalized() RRFConfig {
	if c.K < 1 {
		c.K = defaultRRFK
	}
	if c.TextWeight < 0 {
		c.TextWeight = defaultRRFWeight
	}
	if c.SemanticWeight < 0 {
		c.SemanticWeight = defaultRRFWeight
	}
	if c.TextWeight == 0 && c.SemanticWeight == 0 {
		c.TextWeight = defaultRRFWeight
		c.SemanticWeight = defaultRRFWeight
	}
	return c
}

// rrfBreakdown carries the per-route rank/score of one document plus its RRF
// components; shared by the production fused search and the studio endpoint
// so the numbers users tune with are the numbers production computes.
type rrfBreakdown struct {
	ID                   string  `json:"id"`
	TextRank             int     `json:"text_rank"`     // 1-based, 0 = not recalled by BM25
	SemanticRank         int     `json:"semantic_rank"` // 1-based, 0 = not recalled by kNN
	TextScore            float64 `json:"text_score"`
	SemanticScore        float64 `json:"semantic_score"`
	TextContribution     float64 `json:"text_contribution"`
	SemanticContribution float64 `json:"semantic_contribution"`
	Score                float64 `json:"score"`
}

// rrfFuse ranks the two routes' hit lists with RRF. The returned hits keep
// the _source of the better-ranked occurrence (text route wins ties), with
// _score replaced by the fused score; the breakdown slice is parallel and
// sorted identically.
func rrfFuse(textHits, semanticHits []elastic.DocumentWithMeta[core.Document], cfg RRFConfig) ([]elastic.DocumentWithMeta[core.Document], []rrfBreakdown) {
	cfg = cfg.normalized()

	type entry struct {
		breakdown rrfBreakdown
		hit       elastic.DocumentWithMeta[core.Document]
	}
	byID := make(map[string]*entry, len(textHits)+len(semanticHits))
	order := make([]*entry, 0, len(textHits)+len(semanticHits))

	get := func(hit elastic.DocumentWithMeta[core.Document]) *entry {
		if e, ok := byID[hit.ID]; ok {
			return e
		}
		e := &entry{}
		byID[hit.ID] = e
		order = append(order, e)
		return e
	}

	for i, hit := range textHits {
		e := get(hit)
		e.breakdown.ID = hit.ID
		e.breakdown.TextRank = i + 1
		e.breakdown.TextScore = float64(hit.Score)
		e.hit = hit
	}
	for i, hit := range semanticHits {
		e := get(hit)
		e.breakdown.ID = hit.ID
		e.breakdown.SemanticRank = i + 1
		e.breakdown.SemanticScore = float64(hit.Score)
		if e.hit.ID == "" {
			e.hit = hit
		}
	}

	for _, e := range order {
		if e.breakdown.TextRank > 0 {
			e.breakdown.TextContribution = cfg.TextWeight / (cfg.K + float64(e.breakdown.TextRank))
		}
		if e.breakdown.SemanticRank > 0 {
			e.breakdown.SemanticContribution = cfg.SemanticWeight / (cfg.K + float64(e.breakdown.SemanticRank))
		}
		e.breakdown.Score = e.breakdown.TextContribution + e.breakdown.SemanticContribution
		e.hit.Score = float32(e.breakdown.Score)
	}

	// Sort by fused score; prefer documents recalled by both routes, then
	// by the better text rank, then by ID for deterministic output.
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i].breakdown, order[j].breakdown
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		aBoth := a.TextRank > 0 && a.SemanticRank > 0
		bBoth := b.TextRank > 0 && b.SemanticRank > 0
		if aBoth != bBoth {
			return aBoth
		}
		if (a.TextRank > 0) != (b.TextRank > 0) {
			return a.TextRank > 0
		}
		if a.TextRank != b.TextRank {
			return a.TextRank < b.TextRank
		}
		return a.ID < b.ID
	})

	hits := make([]elastic.DocumentWithMeta[core.Document], len(order))
	breakdowns := make([]rrfBreakdown, len(order))
	for i, e := range order {
		hits[i] = e.hit
		breakdowns[i] = e.breakdown
	}
	return hits, breakdowns
}

func nowMilli() int64 {
	return time.Now().UnixMilli()
}
