/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"fmt"
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
		GraphWeight    float64 `json:"graph_weight"`
		EntityWeight   float64 `json:"entity_weight"`
		ChunkWeight    float64 `json:"chunk_weight"`
		RewriteWeight  float64 `json:"rewrite_weight"`
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
// layer, the ontology graph leg) on the caller's own permissions, then fuses
// them with the submitted RRF parameters and returns every hit's per-route
// rank, raw score and contribution — the exact numbers the production
// hybrid_rrf search computes.
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
		rrfRouteGraph:    body.RRF.GraphWeight,
		rrfRouteEntity:   body.RRF.EntityWeight,
		rrfRouteChunk:    body.RRF.ChunkWeight,
		rrfRouteRewrite:  body.RRF.RewriteWeight,
	}}.normalized()

	// --- query rewrite (D8): resolve before the routes; when it produces a
	// distinct text, that text runs as one more keyword route ---
	rewriteInfo := util.MapStr{"applied": false}
	rw := rewriteResult{Query: query}
	if cfg.weight(rrfRouteRewrite) > 0 {
		started := nowMilli()
		rw = rewriteSearchQuery(req.Context(), query)
		rewriteInfo = util.MapStr{
			"applied": rw.Applied,
			"query":   rw.Query,
			"cached":  rw.Cached,
			"took_ms": nowMilli() - started,
		}
		if rw.Note != "" {
			rewriteInfo["note"] = rw.Note
		}
	} else {
		rewriteInfo["note"] = "rewrite weight is 0, leg muted"
	}

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

	// --- graph route (ontology expansion; note says which entities fired) ---
	{
		result := &studioRouteResult{Name: rrfRouteGraph, Hits: []studioRouteHit{}}
		started := nowMilli()
		resp, note, err := h.graphRoute(req, query, size)
		result.TookMS = nowMilli() - started
		if err != nil {
			result.Error = err.Error()
			collect(rrfRouteGraph, nil, result)
		} else {
			*result = *studioRouteResultFromHits(rrfRouteGraph, resp.Hits.Hits, resp.GetTotal())
			result.Note = note
			result.TookMS = nowMilli() - started
			collect(rrfRouteGraph, resp.Hits.Hits, result)
		}
	}

	// --- entity route (direct entity hit; W10) ---
	{
		result := &studioRouteResult{Name: rrfRouteEntity, Hits: []studioRouteHit{}}
		started := nowMilli()
		resp, err := h.entityRoute(req, query, size)
		result.TookMS = nowMilli() - started
		if err != nil {
			result.Error = err.Error()
			collect(rrfRouteEntity, nil, result)
		} else {
			*result = *studioRouteResultFromHits(rrfRouteEntity, resp.Hits.Hits, resp.GetTotal())
			result.TookMS = nowMilli() - started
			collect(rrfRouteEntity, resp.Hits.Hits, result)
		}
	}

	// --- chunk route (standalone KnowledgeChunk index; W3 方案 c) ---
	{
		result := &studioRouteResult{Name: rrfRouteChunk, Hits: []studioRouteHit{}}
		started := nowMilli()
		resp, err := h.chunkIndexRoute(req, query, size)
		result.TookMS = nowMilli() - started
		if err != nil {
			result.Error = err.Error()
			collect(rrfRouteChunk, nil, result)
		} else if resp == nil {
			result.Note = "no chunk rows for this query's scope — leg off or no projection yet"
			collect(rrfRouteChunk, nil, result)
		} else {
			*result = *studioRouteResultFromHits(rrfRouteChunk, resp.Hits.Hits, resp.GetTotal())
			result.TookMS = nowMilli() - started
			collect(rrfRouteChunk, resp.Hits.Hits, result)
		}
	}

	// --- rewrite route (the rewritten text as an extra keyword leg) ---
	if rw.Applied {
		builder := orm.NewQuery().From(0).Size(size)
		result := &studioRouteResult{Name: rrfRouteRewrite, Hits: []studioRouteHit{}}
		started := nowMilli()
		resp, err := QueryDocuments(req.Context(), builder, rw.Query, body.Datasource, "", body.Category, body.Subcategory, body.RichCategory, "keyword", fuzziness, nil)
		result.TookMS = nowMilli() - started
		if err != nil {
			result.Error = err.Error()
			collect(rrfRouteRewrite, nil, result)
		} else {
			out := &elastic.SearchResponseWithMeta[core.Document]{}
			if len(resp.Raw) > 0 {
				util.MustFromJSONBytes(resp.Raw, out)
			}
			*result = *studioRouteResultFromHits(rrfRouteRewrite, out.Hits.Hits, out.GetTotal())
			result.TookMS = nowMilli() - started
			collect(rrfRouteRewrite, out.Hits.Hits, result)
		}
	}

	// --- fuse ---
	started := nowMilli()
	fusedHitsList, breakdowns := rrfFuseMulti(routes, cfg)
	fusedHits := make([]studioFusedHit, 0, len(breakdowns))
	for _, b := range breakdowns {
		hit := sourceByID[b.ID]
		fusedHits = append(fusedHits, studioFusedHit{rrfBreakdown: b, Title: hit.Source.Title, Datasource: hit.Source.Source.Name})
	}
	tookMS := nowMilli() - started

	// --- rerank leg (applies to the fused order; reported separately so the
	// RRF card stays the pure-RRF baseline for comparison) ---
	rerankInfo := util.MapStr{"applied": false}
	if plan := planRerank(); plan.Model != nil {
		started := nowMilli()
		_, changes, err := rerankFusedHits(req.Context(), plan, query, loadCompileRules(req.Context()).SourcePriority, fusedHitsList)
		if err != nil {
			rerankInfo["note"] = fmt.Sprintf("rerank degraded: %v", err)
		} else {
			titleByID := map[string]string{}
			for i := range fusedHitsList {
				titleByID[fusedHitsList[i].ID] = fusedHitsList[i].Source.Title
			}
			rows := make([]util.MapStr, 0, len(changes))
			for _, c := range changes {
				rows = append(rows, util.MapStr{
					"id": c.ID, "title": titleByID[c.ID],
					"rrf_rank": c.RRFRank, "rerank_rank": c.RerankRank,
					"delta": c.RRFRank - c.RerankRank, "relevance_score": c.RelevanceScore,
				})
			}
			rerankInfo = util.MapStr{
				"applied": true,
				"model":   fmt.Sprintf("%s/%s", plan.Model.ProviderID, plan.Model.ID),
				"took_ms": nowMilli() - started,
				"hits":    rows,
			}
		}
	} else {
		rerankInfo["note"] = plan.Note
	}

	h.WriteJSON(w, util.MapStr{
		"query":     query,
		"size":      size,
		"fuzziness": fuzziness,
		"rrf":       cfg,
		"rewrite":   rewriteInfo,
		"routes":    routeResults,
		"rerank":    rerankInfo,
		"fused": util.MapStr{
			"took_ms": tookMS,
			"total":   len(fusedHits),
			"hits":    fusedHits,
		},
	}, http.StatusOK)
}
