/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rag

import (
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
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

// RangeQuery represents a range query in Elasticsearch
type RangeQuery struct {
	Field string `json:"field"`
	GTE   *int   `json:"gte,omitempty"` // Greater than or equal
	LTE   *int   `json:"lte,omitempty"` // Less than or equal
}

// BoolQuery supports combining multiple queries
type BoolQuery struct {
	Must     []interface{} `json:"must,omitempty"`
	Should   []interface{} `json:"should,omitempty"`
	MustNot  []interface{} `json:"must_not,omitempty"`
	Filter   []interface{} `json:"filter,omitempty"`
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

// SearchRequest mirrors the structure of Elasticsearch's search request
type SearchRequest struct {
	Query  interface{}     `json:"query,omitempty"`   // Query DSL for search
	From   int             `json:"from,omitempty"`    // Pagination: start offset
	Size   int             `json:"size,omitempty"`    // Pagination: number of results
	Sort   []SortOption    `json:"sort,omitempty"`    // Sorting options
	Source []string        `json:"_source,omitempty"` // Fields to include in response
	Aggs   map[string]Aggregation `json:"aggs,omitempty"` // Aggregations for analytics
}

// TotalHits represents the total number of hits in the search response
type TotalHits struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"` // "eq" (exact) or "gte" (greater than or equal)
}

// SearchResponse represents the response to a search query
type SearchResponse struct {
	Took     int                    `json:"took"`
	Hits     []map[string]interface{} `json:"hits"`
	Total    TotalHits              `json:"total"`
}


func (h APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var searchReq SearchRequest
	if err:=h.DecodeJSON(req, &searchReq);err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the presence of a query
	if searchReq.Query == nil {
		http.Error(w, "query must be provided", http.StatusBadRequest)
		return
	}

	// Simulate some search hits and response
	hits := []map[string]interface{}{
		{
			"title": "Sample Search Result 1",
			"content": "This is a sample content for result 1",
		},
		{
			"title": "Sample Search Result 2",
			"content": "This is a sample content for result 2",
		},
	}

	// Simulate total hits with approximate count
	totalHits := TotalHits{
		Value:    100,
		Relation: "gte", // Indicating an approximate result count
	}

	// Create the response
	response := SearchResponse{
		Took:  15,   // Placeholder for query time in milliseconds
		Hits:  hits, // Simulated search results
		Total: totalHits,
	}

	err := h.WriteJSON(w, response,200)
	if err != nil {
		h.Error(w,err)
	}
}
