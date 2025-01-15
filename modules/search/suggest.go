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

	var (
		//context = h.GetParameterOrDefault(req, "context", "")
		query  = h.GetParameterOrDefault(req, "query", "")
		from   = h.GetIntOrDefault(req, "from", 0)
		size   = h.GetIntOrDefault(req, "size", 10)
		field  = h.GetParameterOrDefault(req, "search_field", "title")
		source = h.GetParameterOrDefault(req, "source_fields", "title,source,url")
	)

	q := orm.Query{}
	if query != "" {
		templatedQuery := orm.TemplatedQuery{}
		templatedQuery.TemplateID = "coco-query-string"
		templatedQuery.Parameters = util.MapStr{
			"from":   from,
			"size":   size,
			"field":  field,
			"query":  query,
			"source": strings.Split(source, ","),
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

	suggestions := []Suggestion{}
	for _, item := range res.Result {
		i, ok := item.(map[string]interface{})
		if ok {
			v, ok := i["title"]
			if ok {
				x, _ := i["source"]
				suggestions = append(suggestions, Suggestion{Suggestion: v.(string), Score: 0.99, Source: x.(string)})
			}
		}
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

	err = h.WriteJSON(w, response, 200)
	if err != nil {
		h.Error(w, err)
	}

}
