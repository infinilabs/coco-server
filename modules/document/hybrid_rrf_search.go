/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
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
