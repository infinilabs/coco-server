/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
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

// parseRRFParams reads the fusion knobs from query parameters so the tuned
// values travel with the request, not with server state.
func (h *APIHandler) parseRRFParams(req *http.Request) RRFConfig {
	cfg := RRFConfig{K: defaultRRFK, TextWeight: defaultRRFWeight, SemanticWeight: defaultRRFWeight}
	parse := func(name string, dst *float64) {
		if v := h.GetParameterOrDefault(req, name, ""); v != "" {
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				*dst = f
			}
		}
	}
	parse("rrf_k", &cfg.K)
	parse("text_weight", &cfg.TextWeight)
	parse("semantic_weight", &cfg.SemanticWeight)
	return cfg.normalized()
}

// queryWithRRF executes the BM25 and kNN routes as two independent searches
// (each carrying the full permission/document filters) and fuses the ranked
// lists client-side, so the fusion is transparent and its parameters tunable
// — unlike the engine-side hybrid query where the merge is opaque.
func (h *APIHandler) queryWithRRF(req *http.Request, query, datasource, integrationID, category, subcategory, richCategory string, fuzziness int) (*elastic.SearchResponseWithMeta[core.Document], error) {
	cfg := h.parseRRFParams(req)

	from := h.GetIntOrDefault(req, "from", 0)
	if from < 0 {
		from = 0
	}
	size := h.GetIntOrDefault(req, "size", 10)
	if size < 1 {
		size = 10
	}
	window := from + size
	if window > maxRRFWindow {
		window = maxRRFWindow
	}

	textResp, err := h.rrfRoute(req, "keyword", query, datasource, integrationID, category, subcategory, richCategory, fuzziness, window)
	if err != nil {
		log.Warnf("hybrid_rrf: BM25 route failed: %v", err)
	}
	semanticResp, err := h.rrfRoute(req, "semantic", query, datasource, integrationID, category, subcategory, richCategory, fuzziness, window)
	if err != nil {
		log.Warnf("hybrid_rrf: kNN route failed: %v", err)
	}
	// One failing route (e.g. no embedding service for the query text)
	// degrades to single-route fusion; only both failing is an error.
	if textResp == nil && semanticResp == nil {
		return nil, err
	}
	if textResp == nil {
		textResp = &elastic.SearchResponseWithMeta[core.Document]{}
	}
	if semanticResp == nil {
		semanticResp = &elastic.SearchResponseWithMeta[core.Document]{}
	}

	fused, _ := rrfFuse(textResp.Hits.Hits, semanticResp.Hits.Hits, cfg)

	total := textResp.GetTotal()
	if semanticResp.GetTotal() > total {
		total = semanticResp.GetTotal()
	}

	out := &elastic.SearchResponseWithMeta[core.Document]{
		Took:         textResp.Took + semanticResp.Took,
		Aggregations: textResp.Aggregations,
	}
	out.Hits.Total = elastic.NewGeneralTotal(total)

	end := from + size
	if end > len(fused) {
		end = len(fused)
	}
	if from > len(fused) {
		from = len(fused)
	}
	out.Hits.Hits = fused[from:end]
	if len(out.Hits.Hits) > 0 {
		out.Hits.MaxScore = out.Hits.Hits[0].Score
	}
	return out, nil
}

// rrfRoute runs one recall route. util.ReadBody restores the request body, so
// the second NewQueryBuilderFromRequest call sees the same body DSL.
func (h *APIHandler) rrfRoute(req *http.Request, searchType, query, datasource, integrationID, category, subcategory, richCategory string, fuzziness, window int) (*elastic.SearchResponseWithMeta[core.Document], error) {
	builder, err := orm.NewQueryBuilderFromRequest(req)
	if err != nil {
		return nil, err
	}
	builder.EnableBodyBytes()
	// Both routes must rank by relevance from the very top for RRF to be
	// meaningful; page after fusion, not per route.
	builder.From(0)
	builder.Size(window)

	resp, err := QueryDocuments(req.Context(), builder, query, datasource, integrationID, category, subcategory, richCategory, searchType, fuzziness, nil)
	if err != nil {
		return nil, err
	}
	out := &elastic.SearchResponseWithMeta[core.Document]{}
	if len(resp.Raw) > 0 {
		util.MustFromJSONBytes(resp.Raw, out)
	}
	return out, nil
}

func nowMilli() int64 {
	return time.Now().UnixMilli()
}

type searchStudioBody struct {
	Query        string `json:"query"`
	Datasource   string `json:"datasource"`
	Category     string `json:"category"`
	Subcategory  string `json:"subcategory"`
	RichCategory string `json:"rich_category"`
	Fuzziness    int    `json:"fuzziness"`
	Size         int    `json:"size"`
	RRF          struct {
		K              float64 `json:"k"`
		TextWeight     float64 `json:"text_weight"`
		SemanticWeight float64 `json:"semantic_weight"`
	} `json:"rrf"`
}

type studioRouteHit struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Datasource string  `json:"datasource"`
	Rank       int     `json:"rank"`
	Score      float64 `json:"score"`
}

type studioRouteResult struct {
	TookMS int64            `json:"took_ms"`
	Total  int64            `json:"total"`
	Hits   []studioRouteHit `json:"hits"`
	Error  string           `json:"error,omitempty"`
}

type studioFusedHit struct {
	rrfBreakdown
	Title      string `json:"title"`
	Datasource string `json:"datasource"`
}

// searchStudioTest is the live tuning surface for the dual-engine recall:
// it runs the BM25 route and the kNN route on the caller's own permissions,
// then fuses them with the submitted RRF parameters and returns every
// document's rank, raw score and per-route contribution — the exact numbers
// the production hybrid_rrf search computes.
func (h *APIHandler) searchStudioTest(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body := searchStudioBody{Fuzziness: 3, Size: 10}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	query := strings.TrimSpace(body.Query)
	if query == "" {
		h.WriteError(w, "query is required", http.StatusBadRequest)
		return
	}
	query = util.CleanUserQuery(query)

	size := body.Size
	if size < 1 {
		size = 10
	}
	if size > maxRRFWindow {
		size = maxRRFWindow
	}
	fuzziness := body.Fuzziness
	if fuzziness < 0 || fuzziness > 5 {
		fuzziness = 3
	}
	cfg := RRFConfig{K: body.RRF.K, TextWeight: body.RRF.TextWeight, SemanticWeight: body.RRF.SemanticWeight}.normalized()

	runRoute := func(searchType string) (*elastic.SearchResponseWithMeta[core.Document], *studioRouteResult) {
		builder := orm.NewQuery()
		builder.From(0)
		builder.Size(size)
		result := &studioRouteResult{}
		started := nowMilli()
		resp, err := QueryDocuments(req.Context(), builder, query, body.Datasource, "", body.Category, body.Subcategory, body.RichCategory, searchType, fuzziness, nil)
		result.TookMS = nowMilli() - started
		if err != nil {
			result.Error = err.Error()
			result.Hits = []studioRouteHit{}
			return nil, result
		}
		out := &elastic.SearchResponseWithMeta[core.Document]{}
		if len(resp.Raw) > 0 {
			util.MustFromJSONBytes(resp.Raw, out)
		}
		result.Total = out.GetTotal()
		result.Hits = make([]studioRouteHit, 0, len(out.Hits.Hits))
		for i, hit := range out.Hits.Hits {
			result.Hits = append(result.Hits, studioRouteHit{
				ID:         hit.ID,
				Title:      hit.Source.Title,
				Datasource: hit.Source.Source.Name,
				Rank:       i + 1,
				Score:      float64(hit.Score),
			})
		}
		return out, result
	}

	textResp, textResult := runRoute("keyword")
	semanticResp, semanticResult := runRoute("semantic")

	var textHits, semanticHits []elastic.DocumentWithMeta[core.Document]
	if textResp != nil {
		textHits = textResp.Hits.Hits
	}
	if semanticResp != nil {
		semanticHits = semanticResp.Hits.Hits
	}
	_, breakdowns := rrfFuse(textHits, semanticHits, cfg)

	started := nowMilli()
	fusedHits := make([]studioFusedHit, 0, len(breakdowns))
	sourceByID := make(map[string]elastic.DocumentWithMeta[core.Document], len(textHits)+len(semanticHits))
	for _, hit := range textHits {
		sourceByID[hit.ID] = hit
	}
	for _, hit := range semanticHits {
		if _, ok := sourceByID[hit.ID]; !ok {
			sourceByID[hit.ID] = hit
		}
	}
	for _, b := range breakdowns {
		hit := sourceByID[b.ID]
		fusedHits = append(fusedHits, studioFusedHit{rrfBreakdown: b, Title: hit.Source.Title, Datasource: hit.Source.Source.Name})
	}
	tookMS := nowMilli() - started

	h.WriteJSON(w, util.MapStr{
		"query":     query,
		"size":      size,
		"fuzziness": fuzziness,
		"rrf":       cfg,
		"text":      textResult,
		"semantic":  semanticResult,
		"fused": util.MapStr{
			"took_ms": tookMS,
			"total":   len(fusedHits),
			"hits":    fusedHits,
		},
	}, http.StatusOK)
}
