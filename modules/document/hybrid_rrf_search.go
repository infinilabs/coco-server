/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

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
			rrfRouteWiki:     defaultRRFWeight,
			rrfRouteGraph:    defaultRRFWeight,
		},
	}
	if v := h.GetParameterOrDefault(req, "rrf_k", ""); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			cfg.K = f
		}
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
	return cfg.normalized()
}

// queryWithRRF executes every recall route as an independent search (each
// carrying the full permission/document filters) and fuses the ranked lists
// client-side, so the fusion is transparent and its parameters tunable —
// unlike the engine-side hybrid query where the merge is opaque. Routes:
// BM25 over documents, the semantic leg (engine query or client rerank per
// the capability plan), the curated wiki layer, and the ontology graph leg.
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
	window := rrfRecallWindow(from, size)

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

	resp, err := h.rrfRoute(req, "keyword", query, datasource, integrationID, category, subcategory, richCategory, fuzziness, window)
	addRoute(rrfRouteText, resp, err)
	resp, err = h.rrfRoute(req, "semantic", query, datasource, integrationID, category, subcategory, richCategory, fuzziness, window)
	addRoute(rrfRouteSemantic, resp, err)
	resp, err = h.wikiRoute(req, query, window)
	addRoute(rrfRouteWiki, resp, err)
	graphResp, _, err := h.graphRoute(req, query, window)
	addRoute(rrfRouteGraph, graphResp, err)

	// One failing route (e.g. no embedding service for the query text)
	// degrades to fewer-route fusion; only all failing is an error.
	if len(routes) == 0 {
		return nil, lastErr
	}

	fused, _ := rrfFuseMulti(routes, cfg)

	out := &elastic.SearchResponseWithMeta[core.Document]{
		Took:         took,
		Aggregations: aggregations,
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
