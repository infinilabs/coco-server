/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package search

import (
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"strings"
)

// MatchQuery represents a match query in Elasticsearch
type MatchQuery struct {
	Field string `json:"field"`
	Query string `json:"query"`
}

// TermQuery represents a term query in Elasticsearch
type TermQuery struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type QueryStringQuery struct {
	Field string `json:"field"`
	Value string `json:"query"`
}

// RangeQuery represents a range query in Elasticsearch
type RangeQuery struct {
	Field string `json:"field"`
	GTE   *int   `json:"gte,omitempty"` // Greater than or equal
	LTE   *int   `json:"lte,omitempty"` // Less than or equal
}

// BoolQuery supports combining multiple queries
type BoolQuery struct {
	Must    []interface{} `json:"must,omitempty"`
	Should  []interface{} `json:"should,omitempty"`
	MustNot []interface{} `json:"must_not,omitempty"`
	Filter  []interface{} `json:"filter,omitempty"`
}

// Aggregation represents a basic metric aggregation
type Aggregation struct {
	Field string `json:"field"`
}

// SortOption defines sorting for search results
type SortOption struct {
	Field string `json:"field"`
	Order string `json:"order"` // "asc" or "desc"
}

// TotalHits represents the total number of hits in the search response
type TotalHits struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"` // "eq" (exact) or "gte" (greater than or equal)
}

// SearchResponse represents the response to a search query
type SearchResponse struct {
	Took  int                      `json:"took"`
	Hits  []map[string]interface{} `json:"hits"`
	Total TotalHits                `json:"total"`
}

func (h APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		query = h.GetParameterOrDefault(req, "query", "")
		from  = h.GetIntOrDefault(req, "from", 0)
		size  = h.GetIntOrDefault(req, "size", 20)
		field = h.GetParameterOrDefault(req, "search_field", "title")
		source = h.GetParameterOrDefault(req, "source_fields", "*")
	)

	q := orm.Query{}
	if query != "" {
		templatedQuery := orm.TemplatedQuery{}
		templatedQuery.TemplateID = "coco-query-string"
		templatedQuery.Parameters = util.MapStr{
			"from":  from,
			"size":  size,
			"field": field,
			"query": query,
			"source": strings.Split(source,","),
		}
		q.TemplatedQuery = &templatedQuery
	} else {
		body, err := h.GetRawBody(req)
		if err != nil {
			http.Error(w, "query must be provided", http.StatusBadRequest)
			return
		}
		q.RawQuery = body
	}

	err, res := orm.Search(&common.Document{}, &q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_,err=h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}
