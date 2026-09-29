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

// Route names in the multi-route fusion. Adding a route is: a name here, a
// weight in RRFConfig.Weights, and hits from the route runner — the math
// treats them all the same.
const (
	rrfRouteText     = "text"
	rrfRouteSemantic = "semantic"
	rrfRouteWiki     = "wiki"
	rrfRouteGraph    = "graph"
)

// rrfRouteNames is the canonical order routes are listed and rendered in.
var rrfRouteNames = []string{rrfRouteText, rrfRouteSemantic, rrfRouteWiki, rrfRouteGraph}

// RRFConfig controls how the recall routes are fused with Reciprocal Rank
// Fusion:
//
//	score(d) = Σ_route weight_route/(k+rank_route(d))
//
// Ranks are 1-based; a document recalled by only some routes simply gets no
// contribution from the others. k smooths the influence of top ranks — the
// classic default is 60. A weight of 0 deliberately mutes a route while
// still returning its documents (zero contribution).
type RRFConfig struct {
	K       float64            `json:"k"`
	Weights map[string]float64 `json:"weights"`
}

const (
	defaultRRFK      = 60.0
	defaultRRFWeight = 1.0
	// maxRRFWindow bounds how many candidates each route fetches before
	// fusion, so a hostile from/size cannot turn one search into several
	// unbounded scans.
	maxRRFWindow = 200
)

func (c RRFConfig) normalized() RRFConfig {
	if c.K < 1 {
		c.K = defaultRRFK
	}
	if c.Weights == nil {
		c.Weights = map[string]float64{}
	}
	for name, w := range c.Weights {
		if w < 0 {
			c.Weights[name] = defaultRRFWeight
		}
	}
	allZero := len(c.Weights) > 0
	for _, w := range c.Weights {
		if w != 0 {
			allZero = false
			break
		}
	}
	if len(c.Weights) == 0 || allZero {
		c.Weights = map[string]float64{}
		for _, name := range rrfRouteNames {
			c.Weights[name] = defaultRRFWeight
		}
	}
	return c
}

// weight returns the route's weight; routes absent from the map default to
// the neutral weight so new routes join without touching every caller.
func (c RRFConfig) weight(route string) float64 {
	if w, ok := c.Weights[route]; ok {
		return w
	}
	return defaultRRFWeight
}

// rrfRecallWindow returns how many candidates a fused route should fetch for
// a page: the page span, capped by maxRRFWindow.
func rrfRecallWindow(from, size int) int {
	if from < 0 {
		from = 0
	}
	if size < 1 {
		size = 10
	}
	window := from + size
	if window > maxRRFWindow {
		window = maxRRFWindow
	}
	return window
}

// rrfBreakdown carries one document's per-route rank, raw score and RRF
// contribution; shared by the production fused search and the studio
// endpoint so the numbers users tune with are the numbers production
// computes.
type rrfBreakdown struct {
	ID            string             `json:"id"`
	Ranks         map[string]int     `json:"ranks"`         // route → 1-based rank; absent = not recalled
	RouteScores   map[string]float64 `json:"route_scores"`  // route → the route's own score
	Contributions map[string]float64 `json:"contributions"` // route → weight/(k+rank)
	Score         float64            `json:"score"`
}

// rrfRouteHits is one route's ranked list.
type rrfRouteHits struct {
	Name string
	Hits []elastic.DocumentWithMeta[core.Document]
}

// rrfFuseMulti ranks the routes' hit lists with RRF. The returned hits keep
// the _source of the first route (in route order) that recalled the
// document, with _score replaced by the fused score; the breakdown slice is
// parallel and sorted identically. Ties break toward documents recalled by
// more routes, then by the best rank any route gave, then by ID for
// deterministic output.
func rrfFuseMulti(routes []rrfRouteHits, cfg RRFConfig) ([]elastic.DocumentWithMeta[core.Document], []rrfBreakdown) {
	cfg = cfg.normalized()

	type entry struct {
		breakdown rrfBreakdown
		hit       elastic.DocumentWithMeta[core.Document]
	}
	byID := map[string]*entry{}
	order := make([]*entry, 0)

	for _, route := range routes {
		for i, hit := range route.Hits {
			e, ok := byID[hit.ID]
			if !ok {
				e = &entry{}
				e.breakdown = rrfBreakdown{
					ID:            hit.ID,
					Ranks:         map[string]int{},
					RouteScores:   map[string]float64{},
					Contributions: map[string]float64{},
				}
				byID[hit.ID] = e
				order = append(order, e)
			}
			e.breakdown.Ranks[route.Name] = i + 1
			e.breakdown.RouteScores[route.Name] = float64(hit.Score)
			if e.hit.ID == "" {
				e.hit = hit
			}
		}
	}

	for _, e := range order {
		for route, rank := range e.breakdown.Ranks {
			w := cfg.weight(route)
			e.breakdown.Contributions[route] = w / (cfg.K + float64(rank))
		}
		e.breakdown.Score = 0
		for _, c := range e.breakdown.Contributions {
			e.breakdown.Score += c
		}
		e.hit.Score = float32(e.breakdown.Score)
	}

	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i].breakdown, order[j].breakdown
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if len(a.Ranks) != len(b.Ranks) {
			return len(a.Ranks) > len(b.Ranks) // recalled by more routes wins
		}
		bestRank := func(r map[string]int) int {
			best := 0
			for _, rank := range r {
				if best == 0 || rank < best {
					best = rank
				}
			}
			return best
		}
		if ba, bb := bestRank(a.Ranks), bestRank(b.Ranks); ba != bb {
			return ba < bb
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
