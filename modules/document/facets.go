/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"net/http"
	"sort"
	"strings"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

// Aggregation facets over the fused hybrid_rrf window (W15): the left
// sidebar filters must describe EXACTLY what the search sees — so the
// buckets aggregate the same fused hits, under the same permission
// filters, from the same pipeline call. Client-side over the recall
// window (server-side per-route aggs would describe single routes, not
// the fusion). The response shape is deliberately flat: name → count,
// sorted by count desc, capped per facet.

const (
	aggFacetCap          = 15
	aggregationRecallCap = 200
)

// facetBucket is one {value, count} row.
type facetBucket struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// aggregationsHandler runs the hybrid_rrf fusion with from=0 and the
// recall window, then buckets the fused hits: datasource (built-in KB
// sources included), type (document|wiki_article|entity|faq…), tags,
// entity types (from entity hits' metadata) and a coarse time histogram.
// GET /query/_aggregations?query=...&search_type=hybrid_rrf...
func (h *APIHandler) aggregationsHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	query := strings.TrimSpace(h.GetParameterOrDefault(req, "query", ""))
	if query == "" {
		query = strings.TrimSpace(h.GetParameterOrDefault(req, "q", ""))
	}
	if query == "" {
		h.WriteError(w, "query is required", http.StatusBadRequest)
		return
	}

	datasource := h.GetParameterOrDefault(req, "datasource", "")
	integrationID := h.GetParameterOrDefault(req, "integration_id", "")
	category := h.GetParameterOrDefault(req, "category", "")
	subcategory := h.GetParameterOrDefault(req, "subcategory", "")
	richCategory := h.GetParameterOrDefault(req, "rich_category", "")
	fuzziness := h.GetIntOrDefault(req, "fuzziness", 3)

	// aggregate the head of the fusion, not the current page: facets
	// describe the whole recall space the filters narrow
	aggReq := req.Clone(req.Context())
	q := aggReq.URL.Query()
	q.Set("from", "0")
	q.Set("size", "10") // window is derived internally; page params only matter for latency
	aggReq.URL.RawQuery = q.Encode()

	resp, _, _, err := h.queryWithRRF(aggReq, query, datasource, integrationID, category, subcategory, richCategory, fuzziness)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hits := resp.Hits.Hits
	if len(hits) > aggregationRecallCap {
		hits = hits[:aggregationRecallCap]
	}

	out := util.MapStr{
		"total":  resp.GetTotal(),
		"window": len(hits),
	}
	out["datasources"] = bucketBy(hits, func(hit hitRef) (string, bool) {
		name := hit.Source.Source.Name
		if name == "" {
			name = hit.Source.Source.ID
		}
		return name, name != ""
	})
	out["types"] = bucketBy(hits, func(hit hitRef) (string, bool) {
		t := hit.Source.Type
		if t == "" {
			t = "document"
		}
		return t, true
	})
	out["tags"] = bucketByMany(hits, func(hit hitRef) ([]string, bool) {
		return hit.Source.Tags, len(hit.Source.Tags) > 0
	})
	out["entity_types"] = bucketBy(hits, func(hit hitRef) (string, bool) {
		if hit.Source.Metadata == nil {
			return "", false
		}
		et, _ := hit.Source.Metadata["entity_type"].(string)
		return et, et != ""
	})
	out["months"] = bucketBy(hits, func(hit hitRef) (string, bool) {
		if hit.Source.Updated == nil || hit.Source.Updated.IsZero() {
			return "", false
		}
		return hit.Source.Updated.Format("2006-01"), true
	})

	h.WriteOKJSON(w, out)
}

// hitRef is the fused-hit type the helpers walk.
type hitRef = elastic.DocumentWithMeta[core.Document]

// bucketBy counts one value per hit.
func bucketBy(hits []hitRef, pick func(hitRef) (string, bool)) []facetBucket {
	counts := map[string]int{}
	for i := range hits {
		if v, ok := pick(hits[i]); ok {
			counts[v]++
		}
	}
	return topBuckets(counts)
}

// bucketByMany counts multiple values per hit (tags).
func bucketByMany(hits []hitRef, pick func(hitRef) ([]string, bool)) []facetBucket {
	counts := map[string]int{}
	for i := range hits {
		if vs, ok := pick(hits[i]); ok {
			for _, v := range vs {
				counts[v]++
			}
		}
	}
	return topBuckets(counts)
}

func topBuckets(counts map[string]int) []facetBucket {
	out := make([]facetBucket, 0, len(counts))
	for v, c := range counts {
		out = append(out, facetBucket{Value: v, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Value < out[j].Value
	})
	if len(out) > aggFacetCap {
		out = out[:aggFacetCap]
	}
	return out
}
