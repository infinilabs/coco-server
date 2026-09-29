/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"
	"strings"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

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
		WikiWeight     float64 `json:"wiki_weight"`
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
	Name   string           `json:"name"`
	TookMS int64            `json:"took_ms"`
	Total  int64            `json:"total"`
	Hits   []studioRouteHit `json:"hits"`
	// Route says which semantic route actually ran: "engine" (engine-side
	// semantic query), "client" (BM25 recall + Coco-side cosine rerank) or
	// "skipped" (neither head available, see Error).
	Route string `json:"route,omitempty"`
	Note  string `json:"note,omitempty"`
	Error string `json:"error,omitempty"`
}

type studioFusedHit struct {
	rrfBreakdown
	Title      string `json:"title"`
	Datasource string `json:"datasource"`
}

func studioRouteResultFromHits(name string, hits []elastic.DocumentWithMeta[core.Document], total int64) *studioRouteResult {
	result := &studioRouteResult{Name: name, Total: total, Hits: make([]studioRouteHit, 0, len(hits))}
	for i, hit := range hits {
		result.Hits = append(result.Hits, studioRouteHit{
			ID:         hit.ID,
			Title:      hit.Source.Title,
			Datasource: hit.Source.Source.Name,
			Rank:       i + 1,
			Score:      float64(hit.Score),
		})
	}
	return result
}

// searchStudioTest is the live tuning surface for the multi-route recall:
// it runs every route (BM25 documents, the semantic leg, the curated wiki
// layer) on the caller's own permissions, then fuses them with the submitted
// RRF parameters and returns every hit's per-route rank, raw score and
// contribution — the exact numbers the production hybrid_rrf search computes.
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
	cfg := RRFConfig{K: body.RRF.K, Weights: map[string]float64{
		rrfRouteText:     body.RRF.TextWeight,
		rrfRouteSemantic: body.RRF.SemanticWeight,
		rrfRouteWiki:     body.RRF.WikiWeight,
	}}.normalized()

	routes := []rrfRouteHits{}
	routeResults := []*studioRouteResult{}
	sourceByID := map[string]elastic.DocumentWithMeta[core.Document]{}
	collect := func(name string, hits []elastic.DocumentWithMeta[core.Document], result *studioRouteResult) {
		routeResults = append(routeResults, result)
		if len(hits) == 0 {
			return
		}
		routes = append(routes, rrfRouteHits{Name: name, Hits: hits})
		for _, hit := range hits {
			if _, ok := sourceByID[hit.ID]; !ok {
				sourceByID[hit.ID] = hit
			}
		}
	}

	// --- text route ---
	{
		builder := orm.NewQuery().From(0).Size(size)
		result := &studioRouteResult{Name: rrfRouteText, Hits: []studioRouteHit{}}
		started := nowMilli()
		resp, err := QueryDocuments(req.Context(), builder, query, body.Datasource, "", body.Category, body.Subcategory, body.RichCategory, "keyword", fuzziness, nil)
		result.TookMS = nowMilli() - started
		if err != nil {
			result.Error = err.Error()
			collect(rrfRouteText, nil, result)
		} else {
			out := &elastic.SearchResponseWithMeta[core.Document]{}
			if len(resp.Raw) > 0 {
				util.MustFromJSONBytes(resp.Raw, out)
			}
			*result = *studioRouteResultFromHits(rrfRouteText, out.Hits.Hits, out.GetTotal())
			result.TookMS = nowMilli() - started
			collect(rrfRouteText, out.Hits.Hits, result)
		}
	}

	// --- semantic route (capability-plan aware) ---
	{
		builder := orm.NewQuery().From(0).Size(size)
		result := &studioRouteResult{Name: rrfRouteSemantic, Hits: []studioRouteHit{}}
		started := nowMilli()
		switch plan := planSemantic(req.Context()); plan.Route {
		case semanticRouteEngine:
			result.Route = "engine"
			resp, err := QueryDocuments(req.Context(), builder, query, body.Datasource, "", body.Category, body.Subcategory, body.RichCategory, "semantic", fuzziness, nil)
			result.TookMS = nowMilli() - started
			if err != nil {
				result.Error = err.Error()
				collect(rrfRouteSemantic, nil, result)
			} else {
				out := &elastic.SearchResponseWithMeta[core.Document]{}
				if len(resp.Raw) > 0 {
					util.MustFromJSONBytes(resp.Raw, out)
				}
				*result = *studioRouteResultFromHits(rrfRouteSemantic, out.Hits.Hits, out.GetTotal())
				result.Route = "engine"
				result.TookMS = nowMilli() - started
				collect(rrfRouteSemantic, out.Hits.Hits, result)
			}
		case semanticRouteClient:
			result.Route = "client"
			out, note, err := clientSemanticRecall(req.Context(), builder, query, body.Datasource, "", body.Category, body.Subcategory, body.RichCategory, fuzziness)
			result.TookMS = nowMilli() - started
			result.Note = note
			if err != nil {
				result.Error = err.Error()
				collect(rrfRouteSemantic, nil, result)
			} else {
				*result = *studioRouteResultFromHits(rrfRouteSemantic, out.Hits.Hits, out.GetTotal())
				result.Route = "client"
				result.Note = note
				result.TookMS = nowMilli() - started
				collect(rrfRouteSemantic, out.Hits.Hits, result)
			}
		default:
			result.Route = "skipped"
			result.Error = "skipped: " + plan.Reason
			result.TookMS = nowMilli() - started
			collect(rrfRouteSemantic, nil, result)
		}
	}

	// --- wiki route ---
	{
		result := &studioRouteResult{Name: rrfRouteWiki, Hits: []studioRouteHit{}}
		started := nowMilli()
		resp, err := h.wikiRoute(req, query, size)
		result.TookMS = nowMilli() - started
		if err != nil {
			result.Error = err.Error()
			collect(rrfRouteWiki, nil, result)
		} else {
			*result = *studioRouteResultFromHits(rrfRouteWiki, resp.Hits.Hits, resp.GetTotal())
			result.TookMS = nowMilli() - started
			collect(rrfRouteWiki, resp.Hits.Hits, result)
		}
	}

	// --- fuse ---
	started := nowMilli()
	_, breakdowns := rrfFuseMulti(routes, cfg)
	fusedHits := make([]studioFusedHit, 0, len(breakdowns))
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
		"routes":    routeResults,
		"fused": util.MapStr{
			"took_ms": tookMS,
			"total":   len(fusedHits),
			"hits":    fusedHits,
		},
	}, http.StatusOK)
}
