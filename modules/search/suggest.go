/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package search

import (
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
)

// Suggestion represents an individual suggestion returned by the API
type Suggestion struct {
	Suggestion            string      `json:"suggestion"`
	HighlightedSuggestion string      `json:"highlighted_suggestion,omitempty"`  // Optional, Highlighted Suggestion
	Score                 float64     `json:"score,omitempty"`                   // Optional, Score of the Suggestion
	Icon                  string      `json:"icon,omitempty"`                    // Optional, Icon of the Suggestion
	Source                string      `json:"source,omitempty"`                  // Optional, Source of the Suggestion
	Time                  int         `json:"time,omitempty"`                    // Optional, Time of the Suggestion
	LastAccessTime        int         `json:"last_access_time,omitempty"`        // Optional, Time of Last Access
	Breadcrumbs           []Link      `json:"breadcrumbs,omitempty"`             // Optional, breadcrumb navigation links
	Context               interface{} `json:"context,omitempty"`                 // Optional, Context of the Suggestion
	EstimateNumberOfHits  int         `json:"estimate_number_of_hits,omitempty"` // Optional, Estimate Number of Hits
	Payload               interface{} `json:"payload,omitempty"`                 // Optional, Payload of the Suggestion
	URL                   string      `json:"url,omitempty"`                     // URL to the entity
}

// SuggestResponse represents the response structure for the suggest API
type SuggestResponse struct {
	Query          string       `json:"query,omitempty"`
	RecentSearches []Suggestion `json:"recent_searches,omitempty"`
	Suggestions    []Suggestion `json:"suggestions,omitempty"`
	Banner         *Link        `json:"banner,omitempty"`
}

type Link struct {
	Icon        string `json:"icon,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

func (h APIHandler) suggest(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	query := h.MustGetParameter(w, req, "query")
	size := h.GetIntOrDefault(req, "size", 10)

	context := h.GetParameterOrDefault(req, "context", "")
	//sources := req.URL.Query()["sources"] // Optional slice of sources

	// If query is missing, return an error
	if query == "" {
		http.Error(w, "query parameter is required", http.StatusBadRequest)
		return
	}

	// Placeholder: Generate some suggestions (In practice, this would query your data source)
	suggestions := []Suggestion{
		{Suggestion: "search engine", Score: 0.99, Source: "auto-complete"},
		{Suggestion: "search suggest api", Score: 0.95, Source: "recent-search", Context: context},
	}

	// Limit the number of suggestions based on the size parameter
	if len(suggestions) > size {
		suggestions = suggestions[:size]
	}

	// Create the response
	response := SuggestResponse{
		Query:       query,
		Suggestions: suggestions,
	}

	err := h.WriteJSON(w, response,200)
	if err != nil {
		h.Error(w,err)
	}

}
