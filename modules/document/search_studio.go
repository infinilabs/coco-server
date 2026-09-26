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
