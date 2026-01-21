/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

// RecommendRequest represents the input for the recommend API
type RecommendRequest struct {
	RecentInteractions []string `json:"recent_interactions,omitempty"` // Optional, list of recent user interactions (e.g., clicks, views)
	Context            string   `json:"context,omitempty"`             // Optional, current context of the user (e.g., active category, search context)
	Filters            []string `json:"filters,omitempty"`             // Optional, filters applied to recommendations (e.g., category, tags)
	NumRecommendations int      `json:"num_recommendations,omitempty"` // Optional, number of recommendations requested
}

// RecommendEntityCard represents an individual recommendation returned by the API
type RecommendEntityCard struct {
	Title       string      `json:"title"`                 // Title of the entity card
	Description string      `json:"description,omitempty"` // Optional, description of the entity
	Score       float64     `json:"score,omitempty"`       // Optional, relevance score of the recommendation
	Icon        string      `json:"icon,omitempty"`        // Optional, icon for the entity card
	Banner      *Link       `json:"banner,omitempty"`
	URL         string      `json:"url"`                   // URL to the entity
	Category    string      `json:"category,omitempty"`    // Optional, category of the entity
	Breadcrumbs []Link      `json:"breadcrumbs,omitempty"` // Optional, breadcrumb navigation links
	Context     interface{} `json:"context,omitempty"`     // Optional, additional context for the entity
}

// RecommendResponse represents the response structure for the recommend API
type RecommendResponse struct {
	Recommendations []RecommendEntityCard `json:"recommendations,omitempty"` // List of recommended entity cards
	Total           int                   `json:"total,omitempty"`           // Total number of recommendations
}
