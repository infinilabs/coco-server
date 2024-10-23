/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package query

import (
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
)

// RecommendRequest represents the input for the recommend API
type RecommendRequest struct {
	UserID             string   `json:"user_id"`                       // ID of the user making the recommendation request
	RecentInteractions []string `json:"recent_interactions,omitempty"` // Optional, list of recent user interactions (e.g., clicks, views)
	Context            string   `json:"context,omitempty"`             // Optional, current context of the user (e.g., active category, search context)
	Filters            []string `json:"filters,omitempty"`             // Optional, filters applied to recommendations (e.g., category, tags)
	NumRecommendations int      `json:"num_recommendations,omitempty"` // Optional, number of recommendations requested
}

// EntityCard represents an individual recommendation returned by the API
type EntityCard struct {
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
	UserID          string       `json:"user_id"`         // ID of the user receiving recommendations
	Recommendations []EntityCard `json:"recommendations"` // List of recommended entity cards
	Total           int          `json:"total"`           // Total number of recommendations
}

func (h APIHandler) recommend(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// Parse the request body
	var recommendReq RecommendRequest
	if err := h.DecodeJSON(req, &recommendReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Placeholder: Generate some recommendations based on the request (in a real scenario, this would query a recommendation engine)
	recommendations := []EntityCard{
		{
			Title:       "Introduction to AI",
			Description: "A comprehensive guide to artificial intelligence.",
			Score:       0.98,
			URL:         "https://example.com/entity_101",
			Category:    "Technology",
			Breadcrumbs: []Link{
				{Name: "Tech", URL: "/category/tech"},
				{Name: "AI", URL: "/category/ai"},
			},
		},
		{
			Title:       "Data Science for Beginners",
			Description: "Learn the basics of data science with this beginner's guide.",
			Score:       0.93,
			URL:         "https://example.com/entity_202",
			Category:    "Science",
			Breadcrumbs: []Link{
				{Name: "Science", URL: "/category/science"},
				{Name: "Data Science", URL: "/category/data-science"},
			},
		},
	}

	// Limit the number of recommendations based on the request
	if recommendReq.NumRecommendations > 0 && len(recommendations) > recommendReq.NumRecommendations {
		recommendations = recommendations[:recommendReq.NumRecommendations]
	}

	// Create the response
	response := RecommendResponse{
		UserID:          recommendReq.UserID,
		Recommendations: recommendations,
		Total:           len(recommendations),
	}

	err := h.WriteJSON(w, response, 200)
	if err != nil {
		h.Error(w, err)
	}
}
