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
		query        = h.GetParameterOrDefault(req, "query", "")
		from         = h.GetIntOrDefault(req, "from", 0)
		size         = h.GetIntOrDefault(req, "size", 10)
		datasource   = h.GetParameterOrDefault(req, "datasource", "")
		category     = h.GetParameterOrDefault(req, "category", "")
		username     = h.GetParameterOrDefault(req, "username", "")
		userid       = h.GetParameterOrDefault(req, "userid", "")
		tags         = h.GetParameterOrDefault(req, "tags", "")
		subcategory  = h.GetParameterOrDefault(req, "subcategory", "")
		richCategory = h.GetParameterOrDefault(req, "rich_category", "")
		field        = h.GetParameterOrDefault(req, "search_field", "title")
		source       = h.GetParameterOrDefault(req, "source_fields", "*")
	)

	mustClauses := BuildMustClauses(datasource, category, subcategory, richCategory, username, userid)

	var q *orm.Query
	if query != "" || len(mustClauses) > 0 {
		q = BuildTemplatedQuery(from, size, mustClauses, field, query, source, tags)
	} else {
		body, err := h.GetRawBody(req)
		if err != nil {
			http.Error(w, "query must be provided", http.StatusBadRequest)
			return
		}
		if len(body) == 0 {
			//ignore empty query
			return
		}
		q = &orm.Query{}
		q.RawQuery = body
	}

	docs := []common.Document{}
	err, res := orm.SearchWithJSONMapper(&docs, q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

func BuildTemplatedQuery(from int, size int, mustClauses []interface{}, field string, query string, source string, tags string) *orm.Query {
	templatedQuery := orm.TemplatedQuery{}
	templatedQuery.TemplateID = "coco-query-string"

	templatedQuery.Parameters = util.MapStr{
		"from":         from,
		"size":         size,
		"must_clauses": mustClauses,
		"field":        field,
		"query":        query,
		"source":       strings.Split(source, ","),
		"tags":         strings.Split(tags, ","),
	}
	q := orm.Query{}
	q.TemplatedQuery = &templatedQuery
	return &q
}

func BuildMustClauses(datasource string, category string, subcategory string, richCategory string, username string, userid string) []interface{} {
	mustClauses := []interface{}{}

	// Check and add conditions to mustClauses
	if datasource != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"source.id": datasource,
			},
		})
	}

	if category != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"category": category,
			},
		})
	}

	if subcategory != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"subcategory": subcategory,
			},
		})
	}

	if richCategory != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"rich_categories.key": richCategory,
			},
		})
	}

	if username != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"owner.username": username,
			},
		})
	}

	if userid != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"owner.userid": userid,
			},
		})
	}
	return mustClauses
}
