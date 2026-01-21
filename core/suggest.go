/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

// Suggestion represents an individual suggestion returned by the API
type Suggestion[T any] struct {
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
	Payload               T           `json:"payload,omitempty"`                 // Optional, Payload of the Suggestion
	URL                   string      `json:"url,omitempty"`                     // URL to the entity
}

// SuggestResponse represents the response structure for the suggest API
type SuggestResponse[T any] struct {
	Query          string          `json:"query,omitempty"`
	RecentSearches []Suggestion[T] `json:"recent_searches,omitempty"`
	Suggestions    []Suggestion[T] `json:"suggestions,omitempty"`
	Banner         *Link           `json:"banner,omitempty"`
}

type Link struct {
	Icon        string `json:"icon,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}
