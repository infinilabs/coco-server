/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// fused-page cache: one query's fused+reranked list is reused across the
// page fetches of a scrolling session. The rerank model's scores jitter
// slightly between calls (and min-max normalization amplifies it), so
// re-fusing per page request shuffled the global order — adjacent scroll
// pages then repeated/skipped hits. A short-TTL snapshot guarantees every
// page of one query slices the SAME list.
type fusedCacheEntry struct {
	fused        []elastic.DocumentWithMeta[core.Document]
	total        int64
	took         int
	aggregations map[string]elastic.AggregationResponse
	note         string
	rwApplied    bool
	expires      time.Time
}

const (
	fusedCacheTTL  = 2 * time.Minute
	fusedCacheSize = 128
)

var (
	fusedCacheMu    sync.Mutex
	fusedCacheStore = map[string]fusedCacheEntry{}
	fusedCacheKeys  []string // FIFO eviction order
)

func fusedCacheGet(key string) *fusedCacheEntry {
	fusedCacheMu.Lock()
	defer fusedCacheMu.Unlock()
	e, ok := fusedCacheStore[key]
	if !ok {
		return nil
	}
	if time.Now().After(e.expires) {
		delete(fusedCacheStore, key)
		return nil
	}
	return &e
}

func fusedCachePut(key string, e fusedCacheEntry) {
	fusedCacheMu.Lock()
	defer fusedCacheMu.Unlock()
	if _, exists := fusedCacheStore[key]; !exists {
		fusedCacheKeys = append(fusedCacheKeys, key)
	}
	e.expires = time.Now().Add(fusedCacheTTL)
	fusedCacheStore[key] = e
	for len(fusedCacheKeys) > fusedCacheSize {
		oldest := fusedCacheKeys[0]
		fusedCacheKeys = fusedCacheKeys[1:]
		delete(fusedCacheStore, oldest)
	}
}

// fusedCacheKey fingerprints everything the fusion depends on, EXCLUDING
// from/size/skip_log (pagination must hit the same entry) and INCLUDING the
// URL filter params, the request body DSL, and the settings-driven knobs.
func fusedCacheKey(req *http.Request, query string, cfg RRFConfig) string {
	vals := req.URL.Query()
	volatile := map[string]bool{"from": true, "size": true, "skip_log": true}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		if !volatile[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+3)
	parts = append(parts, "q="+query)
	for _, k := range keys {
		parts = append(parts, k+"="+strings.Join(vals[k], ","))
	}
	parts = append(parts, fmt.Sprintf("k=%g", cfg.K))
	for _, name := range rrfRouteNames {
		parts = append(parts, fmt.Sprintf("w_%s=%g", name, cfg.weight(name)))
	}
	if body, err := util.ReadBody(req); err == nil && len(body) > 0 {
		parts = append(parts, fmt.Sprintf("body=%x", sha256.Sum256(body)))
	}
	return strings.Join(parts, "|")
}

// cloneFusedHits deep-copies hits (JSON round-trip) so the cache stays
// pristine: the search handler later mutates hits in place (icon refinement,
// content masking) and those mutations must not leak into cached pages.
func cloneFusedHits(hits []elastic.DocumentWithMeta[core.Document]) []elastic.DocumentWithMeta[core.Document] {
	out := make([]elastic.DocumentWithMeta[core.Document], len(hits))
	for i := range hits {
		b, err := util.ToJSONBytes(hits[i])
		if err != nil {
			out[i] = hits[i]
			continue
		}
		util.MustFromJSONBytes(b, &out[i])
	}
	return out
}

// parseRRFParams reads the fusion knobs from query parameters so the tuned
// values travel with the request, not with server state. A weight parameter
// set to 0 explicitly mutes its route (zero contribution, per RRFConfig);
// an absent parameter leaves the default.
func (h *APIHandler) parseRRFParams(req *http.Request) RRFConfig {
	cfg := RRFConfig{
		K: defaultRRFK,
		Weights: map[string]float64{
			rrfRouteText:     defaultRRFWeight,
			rrfRouteSemantic: defaultRRFWeight,
			rrfRouteWiki:     defaultWikiRRFWeight, // curated layer outranks single-leg raw hits
			rrfRouteGraph:    defaultRRFWeight,
			rrfRouteEntity:   defaultRRFWeight,
			rrfRouteChunk:    defaultRRFWeight,
			rrfRouteRewrite:  defaultRRFWeight,
			rrfRouteRSA:      defaultRRFWeight,
		},
	}
	// 搜索设置 · 检索策略: the operator's per-route weight defaults fill in
	// before the per-request overrides below — an explicit request weight
	// still wins, a non-positive stored value keeps the built-in default
	if ss := searchSettingsSnapshot(); ss != nil && ss.RouteWeights != nil {
		sw := ss.RouteWeights
		applyRouteWeight := func(route string, w float64) {
			if w > 0 {
				cfg.Weights[route] = w
			}
		}
		applyRouteWeight(rrfRouteText, sw.Text)
		applyRouteWeight(rrfRouteSemantic, sw.Semantic)
		applyRouteWeight(rrfRouteWiki, sw.Wiki)
		applyRouteWeight(rrfRouteGraph, sw.Graph)
		applyRouteWeight(rrfRouteRewrite, sw.Rewrite)
	}
	if v := h.GetParameterOrDefault(req, "rrf_k", ""); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			cfg.K = f
		}
	} else if k := searchSettingsSnapshot().EffectiveRRFRankConstant(); k > 0 {
		// 搜索设置 · 语义与 AI: the operator's fusion-K default fills in
		// when the request carries no explicit ?rrf_k=
		cfg.K = float64(k)
	}
	// rewrite=0/off mutes the rewrite leg without touching its weight —
	// same effect as rewrite_weight=0, friendlier to toggle per request
	if v := strings.ToLower(h.GetParameterOrDefault(req, "rewrite", "")); v == "0" || v == "off" || v == "false" {
		cfg.Weights[rrfRouteRewrite] = 0
	}
	setWeight := func(route, param string) {
		v := h.GetParameterOrDefault(req, param, "")
		if v == "" {
			return
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			cfg.Weights[route] = f
		}
	}
	setWeight(rrfRouteText, "text_weight")
	setWeight(rrfRouteSemantic, "semantic_weight")
	setWeight(rrfRouteWiki, "wiki_weight")
	setWeight(rrfRouteGraph, "graph_weight")
	setWeight(rrfRouteEntity, "entity_weight")
	setWeight(rrfRouteChunk, "chunk_weight")
	setWeight(rrfRouteRewrite, "rewrite_weight")
	setWeight(rrfRouteRSA, "rsa_weight")
	return cfg.normalized()
}

// queryWithRRF executes every recall route as an independent search (each
// carrying the full permission/document filters) and fuses the ranked lists
// client-side, so the fusion is transparent and its parameters tunable —
// unlike the engine-side hybrid query where the merge is opaque. Routes:
// BM25 over documents, the semantic leg (engine query or client rerank per
// the capability plan), the curated wiki layer, the ontology graph leg, and
// the model-rewritten query as an extra keyword route (D8). The returned
// note carries the rewrite/rerank verdicts for the Warning header, and the
// bool reports whether the rewrite leg fired (search-log telemetry).
// diag (nil-safe) collects the per-leg timings and the engine-side took for
// the search log.
func (h *APIHandler) queryWithRRF(req *http.Request, query, datasource, integrationID, category, subcategory, richCategory string, fuzziness int, diag *searchDiagnostics) (*elastic.SearchResponseWithMeta[core.Document], string, bool, error) {
	cfg := h.parseRRFParams(req)

	from := h.GetIntOrDefault(req, "from", 0)
	if from < 0 {
		from = 0
	}
	size := h.GetIntOrDefault(req, "size", 10)
	if size < 1 {
		size = 10
	}
	if size > maxSearchPageSize {
		size = maxSearchPageSize
	}
	// FIXED recall window, NOT from+size: the route lists and the RRF
	// fusion must be identical for every page, otherwise the fused order
	// shifts as the window grows and slicing [from:from+size] out of a
	// from-dependent fusion made scroll pages overlap/duplicate (same
	// root cause as the semanticRouteClient slice in search.go).
	window := maxRRFWindow

	// snapshot cache: pages of one scrolling session slice the same
	// fused+reranked list (see fusedCacheEntry)
	cacheKey := fusedCacheKey(req, query, cfg)
	if e := fusedCacheGet(cacheKey); e != nil {
		end := from + size
		if end > len(e.fused) {
			end = len(e.fused)
		}
		if from > len(e.fused) {
			from = len(e.fused)
		}
		out := &elastic.SearchResponseWithMeta[core.Document]{
			Took:         e.took,
			Aggregations: e.aggregations,
		}
		out.Hits.Total = map[string]interface{}{"value": e.total, "relation": "eq"}
		out.Hits.Hits = cloneFusedHits(e.fused[from:end])
		if len(out.Hits.Hits) > 0 {
			out.Hits.MaxScore = out.Hits.Hits[0].Score
		}
		return out, e.note, e.rwApplied, nil
	}

	// resolve the rewrite first: its text feeds an extra keyword route, and
	// cache hits return before any route runs
	rw := rewriteResult{Query: query}
	if cfg.weight(rrfRouteRewrite) > 0 {
		rwStart := time.Now()
		rw = rewriteSearchQuery(req.Context(), query)
		if diag != nil {
			diag.LegRewriteMS = time.Since(rwStart).Milliseconds()
		}
	}

	routes := []rrfRouteHits{}
	var lastErr error
	var total int64
	var took int
	var aggregations map[string]elastic.AggregationResponse

	addRoute := func(name string, resp *elastic.SearchResponseWithMeta[core.Document], err error) {
		if err != nil {
			log.Warnf("hybrid_rrf: %s route failed: %v", name, err)
			lastErr = err
			return
		}
		if resp == nil {
			return
		}
		if resp.GetTotal() > total {
			total = resp.GetTotal()
		}
		took += resp.Took
		if aggregations == nil && resp.Aggregations != nil {
			aggregations = resp.Aggregations
		}
		routes = append(routes, rrfRouteHits{Name: name, Hits: resp.Hits.Hits})
	}

	timedRoute := func(name, leg string) *elastic.SearchResponseWithMeta[core.Document] {
		start := time.Now()
		resp, err := h.rrfRoute(req, name, query, datasource, integrationID, category, subcategory, richCategory, fuzziness, window)
		if diag != nil {
			ms := time.Since(start).Milliseconds()
			switch leg {
			case "keyword":
				diag.LegKeywordMS += ms
			case "semantic":
				diag.LegSemanticMS += ms
			}
		}
		addRoute(name, resp, err)
		return resp
	}

	timedRoute("keyword", "keyword")
	timedRoute("semantic", "semantic")
	resp, err := h.wikiRoute(req, query, window)
	addRoute(rrfRouteWiki, resp, err)
	graphResp, _, err := h.graphRoute(req, query, window)
	addRoute(rrfRouteGraph, graphResp, err)
	entityResp, err := h.entityRoute(req, query, window)
	addRoute(rrfRouteEntity, entityResp, err)
	// W3 方案 c gray switch: the chunk-index leg runs alongside the
	// document-level text leg — both feed the fusion, chunk hits carry
	// mom/breadcrumb/quote for citations. Off = zero behavior change.
	if cfg := appConfigFn(); cfg.SearchSettings != nil && cfg.SearchSettings.ChunkRoute {
		chunkResp, cerr := h.chunkIndexRoute(req, query, window)
		if cerr != nil {
			log.Warnf("hybrid_rrf: chunk route failed: %v", cerr)
		} else if chunkResp != nil {
			addRoute(rrfRouteChunk, chunkResp, nil)
		}
	}
	// RSA federated leg (DESIGN-RSA.md): direct queries over enabled
	// remote sources, mapped to standard documents. Failures degrade to
	// fewer-route fusion like every other leg.
	if cfg := appConfigFn(); cfg.SearchSettings != nil && cfg.SearchSettings.RsaEnabled {
		rsaResp, rsaDiag, rerr := h.rsaRoute(req, query, window)
		if rerr != nil {
			log.Warnf("hybrid_rrf: rsa route failed: %v", rerr)
		} else if rsaResp != nil {
			addRoute(rrfRouteRSA, rsaResp, nil)
			if diag != nil {
				for _, sd := range rsaDiag {
					diag.LegRsaMS += sd.TookMS
				}
			}
		}
	}
	if rw.Applied {
		rewrittenStart := time.Now()
		resp, err = h.rrfRoute(req, "keyword", rw.Query, datasource, integrationID, category, subcategory, richCategory, fuzziness, window)
		if diag != nil {
			diag.LegKeywordMS += time.Since(rewrittenStart).Milliseconds()
		}
		addRoute(rrfRouteRewrite, resp, err)
	}

	if diag != nil {
		diag.EngineTookMS = int64(took)
	}

	// One failing route (e.g. no embedding service for the query text)
	// degrades to fewer-route fusion; only all failing is an error.
	if len(routes) == 0 {
		return nil, "", false, lastErr
	}

	fused, _ := rrfFuseMulti(routes, cfg)
	fused, rerankNote := applyRerank(req.Context(), query, fused)

	notes := []string{}
	if rw.Applied {
		notes = append(notes, fmt.Sprintf("query rewritten: %s", rw.Query))
	}
	if rerankNote != "" {
		notes = append(notes, rerankNote)
	}

	fusedCachePut(cacheKey, fusedCacheEntry{
		fused:        cloneFusedHits(fused),
		total:        total,
		took:         took,
		aggregations: aggregations,
		note:         strings.Join(notes, "; "),
		rwApplied:    rw.Applied,
	})

	out := &elastic.SearchResponseWithMeta[core.Document]{
		Took:         took,
		Aggregations: aggregations,
	}
	// the fused page is a union — it can carry more hits than any single
	// route's total counts, so the reported total must never be smaller
	// than the list beneath it (pagination keys off that number)
	if int64(len(fused)) > total {
		total = int64(len(fused))
	}
	// plain map: GetTotal's type switch does not recognize util.MapStr, a
	// named type would read as -1 and break the zero-hit telemetry
	out.Hits.Total = map[string]interface{}{"value": total, "relation": "eq"}

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
	return out, strings.Join(notes, "; "), rw.Applied, nil
}

// rrfRoute runs one recall route. util.ReadBody restores the request body, so
// the second NewQueryBuilderFromRequest call sees the same body DSL.
func (h *APIHandler) rrfRoute(req *http.Request, searchType, query, datasource, integrationID, category, subcategory, richCategory string, fuzziness, window int) (*elastic.SearchResponseWithMeta[core.Document], error) {
	builder, err := orm.NewQueryBuilderFromRequest(req)
	if err != nil {
		return nil, err
	}
	builder.EnableBodyBytes()
	// same pure-filter guard as the keyword/semantic leg: an empty query
	// turns a posted highlight section into mark-everything fragments
	stripHighlightWithoutQuery(builder, query)
	// ?tags= controlled-vocabulary facet filter (W4) applies to every
	// hybrid leg's recall — the routes filter identically to the UI facets
	applyTagsFilter(builder, req.URL.Query().Get("tags"))
	// Both routes must rank by relevance from the very top for RRF to be
	// meaningful; page after fusion, not per route.
	builder.From(0)
	builder.Size(window)

	// The semantic leg follows the capability plan: engine query when the
	// engine has an embedding service, client-side cosine rerank when only
	// Coco has one, skip when neither — one failing route still fuses.
	if searchType == "semantic" {
		switch plan := planSemantic(req.Context()); plan.Route {
		case semanticRouteClient:
			out, _, err := clientSemanticRecall(req.Context(), builder, query, datasource, integrationID, category, subcategory, richCategory, fuzziness)
			return out, err
		case semanticRouteNone:
			return nil, fmt.Errorf("semantic route skipped: %s", plan.Reason)
		}
	}

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
