/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
)

func (h APIHandler) recommend(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// Parse the request body
	var recommendReq core.RecommendRequest
	if err := h.DecodeJSON(req, &recommendReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tag := ps.ByName("tag") //eg: hot
	log.Trace("recommend tag:", tag)

	// Placeholder: Generate some recommendations based on the request (in a real scenario, this would query a recommendation engine)
	recommendations := []core.RecommendEntityCard{
		{
			Title:       "Introduction to AI",
			Description: "A comprehensive guide to artificial intelligence.",
			Score:       0.98,
			URL:         "https://example.com/entity_101",
			Category:    "Technology",
			Breadcrumbs: []core.Link{
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
			Breadcrumbs: []core.Link{
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
	response := core.RecommendResponse{
		Recommendations: recommendations,
		Total:           len(recommendations),
	}

	h.WriteJSON(w, response, 200)

}
