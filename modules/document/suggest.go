/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"strings"
)

func (h APIHandler) suggest(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var (
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

	docs := []core.Document{}
	err, _ := orm.SearchWithJSONMapper(&docs, &q)

	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	suggestions := []core.Suggestion{}
	for _, item := range docs {
		suggestions = append(suggestions, core.Suggestion{Suggestion: item.Title, Score: 0.99, Source: item.Source.Name})
	}

	// Limit the number of suggestions based on the size parameter
	if len(suggestions) > size {
		suggestions = suggestions[:size]
	}

	// Create the response
	response := core.SuggestResponse{
		Query:       query,
		Suggestions: suggestions,
	}

	h.WriteJSON(w, response, 200)

}
