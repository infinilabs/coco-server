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

// Aggregation buckets over the fused hybrid_rrf window (W15): the left
// sidebar filters must describe EXACTLY what the search sees — so the
// buckets aggregate the same fused hits, under the same permission
// filters, from the same pipeline call. Client-side over the recall
// window (server-side per-route aggs would describe single routes, not
// the fusion). The response shape is deliberately flat: name → count,
// sorted by count desc, capped per aggregation.

const (
	defaultAggBucketCap  = 15
	aggregationRecallCap = 200
)

// bucketEntry is one {value, count} row.
type bucketEntry struct {
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

	// 搜索设置 · 聚合与筛选: the operator can switch the aggregations endpoint off
	// entirely (an empty but well-shaped response, mirroring suggest)
	searchCfg := searchSettingsSnapshot()
	if !searchCfg.AggregationsEnabledVal() {
		empty := []bucketEntry{}
		h.WriteOKJSON(w, util.MapStr{
			"total": 0, "window": 0,
			"datasources": empty, "types": empty, "tags": empty,
			"entity_types": empty, "months": empty,
		})
		return
	}

	datasource := h.GetParameterOrDefault(req, "datasource", "")
	integrationID := h.GetParameterOrDefault(req, "integration_id", "")
	category := h.GetParameterOrDefault(req, "category", "")
	subcategory := h.GetParameterOrDefault(req, "subcategory", "")
	richCategory := h.GetParameterOrDefault(req, "rich_category", "")
	fuzziness := h.GetIntOrDefault(req, "fuzziness", searchCfg.EffectiveFuzziness())

	// aggregate the head of the fusion, not the current page: buckets
	// describe the whole recall space the filters narrow
	aggReq := req.Clone(req.Context())
	q := aggReq.URL.Query()
	q.Set("from", "0")
	q.Set("size", "10") // window is derived internally; page params only matter for latency
	aggReq.URL.RawQuery = q.Encode()

	resp, _, _, err := h.queryWithRRF(aggReq, query, datasource, integrationID, category, subcategory, richCategory, fuzziness, nil)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hits := resp.Hits.Hits
	if len(hits) > aggregationRecallCap {
		hits = hits[:aggregationRecallCap]
	}

	// the per-aggregation bucket cap is operator-configurable (搜索设置 · 聚合与
	// 筛选): 0/unset keeps the historic 15; so is the bucket ordering
	aggBucketCap := searchCfg.EffectiveAggregationBucketCap()
	aggBucketSort := searchCfg.EffectiveAggregationBucketSort()

	// per-dimension gate (搜索设置 · 聚合与筛选): a disabled dimension returns
	// empty buckets so the response shape stays stable for clients. Semantic
	// keys map to the fused names here (source→datasources, time→months);
	// entity_types has no UI dimension and always renders.
	bucketFor := func(dim string, pick func(hitRef) (string, bool)) []bucketEntry {
		if searchCfg.AggregationFieldDisabled(dim) {
			return []bucketEntry{}
		}
		return bucketBy(hits, aggBucketCap, aggBucketSort, pick)
	}

	out := util.MapStr{
		"total":  resp.GetTotal(),
		"window": len(hits),
	}
	out["datasources"] = bucketFor("source", func(hit hitRef) (string, bool) {
		name := hit.Source.Source.Name
		if name == "" {
			name = hit.Source.Source.ID
		}
		return name, name != ""
	})
	out["types"] = bucketFor("type", func(hit hitRef) (string, bool) {
		t := hit.Source.Type
		if t == "" {
			t = "document"
		}
		return t, true
	})
	if searchCfg.AggregationFieldDisabled("tags") {
		out["tags"] = []bucketEntry{}
	} else {
		out["tags"] = bucketByMany(hits, aggBucketCap, aggBucketSort, func(hit hitRef) ([]string, bool) {
			return hit.Source.Tags, len(hit.Source.Tags) > 0
		})
	}
	out["entity_types"] = bucketBy(hits, aggBucketCap, aggBucketSort, func(hit hitRef) (string, bool) {
		if hit.Source.Metadata == nil {
			return "", false
		}
		et, _ := hit.Source.Metadata["entity_type"].(string)
		return et, et != ""
	})
	out["months"] = bucketFor("time", func(hit hitRef) (string, bool) {
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
func bucketBy(hits []hitRef, aggBucketCap int, aggBucketSort string, pick func(hitRef) (string, bool)) []bucketEntry {
	counts := map[string]int{}
	for i := range hits {
		if v, ok := pick(hits[i]); ok {
			counts[v]++
		}
	}
	return topBuckets(counts, aggBucketCap, aggBucketSort)
}

// bucketByMany counts multiple values per hit (tags).
func bucketByMany(hits []hitRef, aggBucketCap int, aggBucketSort string, pick func(hitRef) ([]string, bool)) []bucketEntry {
	counts := map[string]int{}
	for i := range hits {
		if vs, ok := pick(hits[i]); ok {
			for _, v := range vs {
				counts[v]++
			}
		}
	}
	return topBuckets(counts, aggBucketCap, aggBucketSort)
}

func topBuckets(counts map[string]int, aggBucketCap int, aggBucketSort string) []bucketEntry {
	out := make([]bucketEntry, 0, len(counts))
	for v, c := range counts {
		out = append(out, bucketEntry{Value: v, Count: c})
	}
	if aggBucketSort == "value" {
		sort.Slice(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	} else {
		sort.Slice(out, func(i, j int) bool {
			if out[i].Count != out[j].Count {
				return out[i].Count > out[j].Count
			}
			return out[i].Value < out[j].Value
		})
	}
	if aggBucketCap <= 0 {
		aggBucketCap = defaultAggBucketCap
	}
	if len(out) > aggBucketCap {
		out = out[:aggBucketCap]
	}
	return out
}
