/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

func (h *APIHandler) suggest(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var (
		query = h.GetParameterOrDefault(req, "query", "")
		from  = h.GetIntOrDefault(req, "from", 0)
		size  = h.GetIntOrDefault(req, "size", 10)
	)

	var response interface{}
	tag := ps.ByName("tag")
	log.Trace("suggest tag:", tag)

	switch tag {
	case core.SuggestTagFieldNames:
		response = h.suggestFieldNames(w, req, query, from, size)
		break
	case core.SuggestTagFieldValues:
		response = h.suggestFieldValues(w, req, query, from, size)
		break
	default:
		response = h.suggestDocuments(w, req, query, from, size)
	}

	h.WriteJSON(w, response, 200)

}

type FieldMetadata struct {
	FieldName          string `json:"field_name"`
	FieldDataType      string `json:"field_data_type"`
	SupportMultiSelect bool   `json:"support_multi_select"`
}

func (h *APIHandler) suggestFieldNames(w http.ResponseWriter, req *http.Request, query string, from int, size int) *core.SuggestResponse[FieldMetadata] {
	response := &core.SuggestResponse[FieldMetadata]{}

	if h.documentMetadata != "" {
		err := util.FromJson(h.documentMetadata, &response)
		if err != nil {
			panic(err)
		}
	}

	out := []core.Suggestion[FieldMetadata]{}
	for _, v := range response.Suggestions {
		if util.PrefixStr(v.Suggestion, query) || util.ContainStr(v.Suggestion, query) || util.PrefixStr(v.Payload.FieldName, query) || util.ContainStr(v.Payload.FieldName, query) {
			out = append(out, v)
		}
	}

	response.Suggestions = out
	response.Query = query

	return response
}

type AggResult map[string]TermsAggResult

type TermsAggResult struct {
	Buckets []Bucket `json:"buckets"`
}

type Bucket struct {
	Key      string `json:"key"`
	DocCount int    `json:"doc_count"`
}

type ESResponse struct {
	Aggregations AggResult `json:"aggregations"`
}

func (h *APIHandler) suggestDocuments(w http.ResponseWriter, req *http.Request, query string, from int, size int) *core.SuggestResponse[interface{}] {

	var (
		datasource   = h.GetParameterOrDefault(req, "datasource", "")
		category     = h.GetParameterOrDefault(req, "category", "")
		subcategory  = h.GetParameterOrDefault(req, "subcategory", "")
		richCategory = h.GetParameterOrDefault(req, "rich_category", "")
	)

	query = util.CleanUserQuery(query)
	response := &core.SuggestResponse[interface{}]{}

	if query != "" {
		builder, err := orm.NewQueryBuilderFromRequest(req)
		if err != nil {
			panic(err)
		}

		builder.Include("title", "category", "icon", "source.name")

		//ctx := orm.WithCollapseFieldForContext(req.Context(), "title.keyword")
		ctx := req.Context()

		builder.Collapse("title.keyword")

		reqUser := security.MustGetUserFromRequest(req)
		integrationID := req.Header.Get(core.HeaderIntegrationID)

		teamsID, _ := reqUser.GetStringArray(orm.TeamsIDKey)

		result := elastic.SearchResponseWithMeta[core.Document]{}
		resp, err := QueryDocuments(ctx, reqUser.MustGetUserID(), teamsID, builder, query, datasource, integrationID, category, subcategory, richCategory, nil)
		if err != nil {
			panic(err)
		}
		util.MustFromJSONBytes(resp.Raw, &result)
		suggestions := []core.Suggestion[interface{}]{}
		for _, item := range result.Hits.Hits {
			suggestions = append(suggestions, core.Suggestion[interface{}]{
				Suggestion: item.Source.Title, Score: float64(item.Score), Icon: item.Source.Icon, Source: item.Source.Category,
			})
		}
		response.Suggestions = suggestions
	}

	response.Query = query
	return response
}

func (h *APIHandler) suggestFieldValues(w http.ResponseWriter, req *http.Request, query string, from int, size int) *core.SuggestResponse[interface{}] {

	fieldName := h.MustGetParameter(w, req, "field_name")

	builder := orm.NewQuery()
	builder.Query(query)
	builder.DefaultQueryField(fieldName)
	builder.Fuzziness(5)
	builder.Size(0)

	rootAggs := map[string]orm.Aggregation{
		"suggestions": (&orm.TermsAggregation{
			Field:   fieldName,
			Include: query + ".*",
			Size:    10,
		}),
	}
	builder.SetAggregations(rootAggs)

	ctx := orm.NewContextWithParent(req.Context())
	ctx.DirectReadAccess()
	ctx.PermissionScope(security.PermissionScopePlatform)

	orm.WithModel(ctx, core.Document{})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		panic(err)
	}
	out := []core.Suggestion[interface{}]{}

	bytes, ok := res.Payload.([]byte)
	if ok {
		var resp ESResponse
		util.MustFromJSONBytes(bytes, &resp)
		if v, ok := resp.Aggregations["suggestions"]; ok {
			for _, bucket := range v.Buckets {
				out = append(out, core.Suggestion[interface{}]{Suggestion: bucket.Key})
			}
		}
	}

	response := &core.SuggestResponse[interface{}]{}
	response.Suggestions = out
	response.Query = query

	return response
}
