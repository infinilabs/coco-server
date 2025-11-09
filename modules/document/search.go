/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"fmt"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/connector"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"net/http"
	"strings"
)

func (h APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var (
		query        = h.GetParameterOrDefault(req, "query", "")
		datasource   = h.GetParameterOrDefault(req, "datasource", "")
		category     = h.GetParameterOrDefault(req, "category", "")
		subcategory  = h.GetParameterOrDefault(req, "subcategory", "")
		richCategory = h.GetParameterOrDefault(req, "rich_category", "")
	)

	query = util.CleanUserQuery(query)

	//try to collect assistants
	if query != "" {
		builder, err := orm.NewQueryBuilderFromRequest(req)

		if err != nil {
			panic(err)
		}
		builder.EnableBodyBytes()

		reqUser := security.MustGetUserFromRequest(req)
		//userID := reqUser.MustGetUserID()
		integrationID := req.Header.Get(core.HeaderIntegrationID)
		//log.Error("integrationID:", integrationID)

		docs1, resp, err := QueryDocumentsAndFilter(req.Context(), reqUser, builder, integrationID, query, datasource, category, subcategory, richCategory)
		//log.Error(docs1, total, err)
		if err != nil {
			panic(err)
		}

		size := h.GetIntOrDefault(req, "size", 10)
		assistantSearchPermission := security.GetSimplePermission(Category, Assistant, string(QuickAISearchAction))
		perID := security.GetOrInitPermissionKey(assistantSearchPermission)

		//not for widget integration
		if integrationID == "" && ((reqUser.Roles != nil && util.AnyInArrayEquals(reqUser.Roles, security.RoleAdmin)) || reqUser.UserAssignedPermission.ValidateFor(perID)) {
			assistantSize := 2
			if len(docs1) < 5 {
				assistantSize = size - (len(docs1))
			}

			assistants := searchAssistant(req, query, assistantSize)
			if len(assistants) > 0 {
				for _, assistant := range assistants {
					doc := core.Document{}
					doc.ID = assistant.ID
					doc.Type = "AI Assistant"
					doc.Icon = assistant.Icon
					doc.Title = assistant.Name
					doc.Summary = assistant.Description
					doc.URL = fmt.Sprintf("coco://extenstions/infinilabs/ask_assistant/%v", assistant.ID)
					doc.Source = core.DataSourceReference{
						ID:   "assistant",
						Name: "Assistant",
						Icon: "font_robot",
					}
					//newHit := IndexDocument{Index: "assistant", ID: assistant.ID, Source: doc, Score: v2.Hits.MaxScore + 500}
					newHit := elastic.DocumentWithMeta[core.Document]{
						ID:     assistant.ID,
						Index:  "assistant",
						Source: doc,
					}
					docs1 = append(docs1, newHit)
				}
			}
		}

		result := elastic.SearchResponseWithMeta[core.Document]{}
		util.FromJSONBytes(resp.Raw, &result)

		result.Hits.Hits = docs1
		result.Hits.Total = util.MapStr{
			"value":    resp.Total,
			"relation": "eq",
		}

		api.WriteJSON(w, result, 200)
	} else {
		h.WriteJSON(w, elastic.SearchResponse{}, http.StatusOK)
		//api.WriteError(w,"query is empty",400)
	}
}

func RefineIcon(ctx context.Context, doc *core.Document) {
	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectReadAccess()
	// Get the pointer to doc.Source to make sure you're modifying the original
	datasourceConfig, err := common.GetDatasourceConfig(ctx1, doc.Source.ID)
	if err == nil && datasourceConfig != nil && datasourceConfig.Connector.ConnectorID != "" {
		connectorConfig, err := connector.GetConnectorConfig(datasourceConfig.Connector.ConnectorID)

		if connectorConfig != nil && err == nil {
			icon := common.ParseAndGetIcon(connectorConfig, doc.Source.Icon)
			if icon != "" {
				doc.Source.Icon = icon
			}

			if doc.Source.Icon != "" {
				icon = common.ParseAndGetIcon(connectorConfig, doc.Source.Icon)
				if icon != "" {
					doc.Source.Icon = icon
				}
			} else {
				//try datasource's icon
				if datasourceConfig.Icon != "" {
					icon = common.ParseAndGetIcon(connectorConfig, datasourceConfig.Icon)
					if icon != "" {
						doc.Source.Icon = icon
					}
				} else {
					//try connector's icon
					icon = common.ParseAndGetIcon(connectorConfig, connectorConfig.Icon)
					if icon != "" {
						doc.Source.Icon = icon
					}
				}
			}
		}
	}
}

func searchAssistant(req *http.Request, query string, size int) []core.Assistant {
	docs := []core.Assistant{}
	if size <= 0 {
		size = 2
	}

	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name", "combined_fulltext")
	if err != nil {
		return docs
	}
	builder.Query(query)
	builder.Must(orm.TermQuery("enabled", true))
	builder.Size(size)

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.Assistant{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "assistant")
	err, _ = elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	if err != nil {
		return docs
	}

	return docs
}

func BuildTemplatedQuery(from int, size int, mustClauses []interface{}, shouldClauses interface{}, field string, query string, source string, tags string) *orm.Query {
	templatedQuery := orm.TemplatedQuery{}
	templatedQuery.TemplateID = "coco-query-string"
	if shouldClauses != nil {
		templatedQuery.TemplateID = "coco-query-string-extra-should"
	}

	templatedQuery.Parameters = util.MapStr{
		"from":                 from,
		"size":                 size,
		"must_clauses":         mustClauses,
		"extra_should_clauses": shouldClauses,
		"field":                field,
		"query":                query,
		"source":               strings.Split(source, ","),
		"tags":                 strings.Split(tags, ","),
	}
	q := orm.Query{}
	q.TemplatedQuery = &templatedQuery
	return &q
}

func BuildDatasourceClause(datasource string, filterDisabled bool) interface{} {
	var datasourceClause interface{}
	if datasource != "" {
		if strings.Contains(datasource, ",") {
			arr := strings.Split(datasource, ",")
			datasourceClause = map[string]interface{}{
				"terms": map[string]interface{}{
					"source.id": arr,
				},
			}
		} else {
			datasourceClause = map[string]interface{}{
				"term": map[string]interface{}{
					"source.id": datasource,
				},
			}
		}
	}
	if !filterDisabled {
		return datasourceClause
	}

	disabledIDs, err := common.GetDisabledDatasourceIDs()
	if err != nil {
		panic(err)
	}
	if len(disabledIDs) == 0 {
		return datasourceClause
	}
	mustNot := map[string]interface{}{
		"terms": map[string]interface{}{
			"source.id": disabledIDs,
		},
	}
	if datasourceClause == nil {
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": mustNot,
			},
		}
	}
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"must_not": mustNot,
			"must":     datasourceClause,
		},
	}
}

func BuildMustFilterClauses(category string, subcategory string, richCategory string, username string, userid string) []interface{} {
	mustClauses := []interface{}{}

	// Check and add conditions to mustClauses

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

func BuildFilters(category string, subcategory string, richCategory string) []*orm.Clause {
	mustClauses := []*orm.Clause{}

	if category != "" {
		mustClauses = append(mustClauses, orm.TermQuery("category", category))
	}

	if subcategory != "" {
		mustClauses = append(mustClauses, orm.TermQuery("subcategory", subcategory))
	}

	if richCategory != "" {
		mustClauses = append(mustClauses, orm.TermQuery("rich_categories.key", richCategory))
	}

	return mustClauses
}

// GetDatasourceByIntegration returns the datasource IDs that the integration is allowed to access
func GetDatasourceByIntegration(integrationID string) ([]string, bool, error) {
	var items = []core.Integration{}
	q := orm.Query{
		Size:  1,
		Conds: orm.And(orm.Eq("id", integrationID), orm.Eq("enabled", true)),
	}
	err, _ := orm.SearchWithJSONMapper(&items, &q)
	if err != nil {
		return nil, false, err
	}
	if len(items) == 0 {
		return nil, false, nil
	}
	var ret = make([]string, 0, len(items))
	for _, item := range items {
		for _, datasourceID := range item.EnabledModule.Search.Datasource {
			if datasourceID == "*" {
				return nil, true, nil
			}
			ret = append(ret, datasourceID)
		}
	}
	return ret, false, nil
}

func BuildDatasourceFilter(userID string, sharedDatasources []string, queryDatasourceIDs []string, integrationID string, filterDisabled bool) ([]string, []*orm.Clause) {

	//log.Error("userID:", userID, ",queryDatasource:", queryDatasourceIDs, ",integrationID:", integrationID)

	finalDatasourceIDs := []string{}
	if integrationID != "" {
		// get queryDatasource by integration id
		datasourceIDs, hasAll, err := GetDatasourceByIntegration(integrationID)
		if err != nil {
			panic(err)
		}
		finalDatasourceIDs = datasourceIDs
		if !hasAll {
			if len(datasourceIDs) == 0 {
				// return empty search result when no queryDatasource found
				panic("integration datasource is empty")
			}
			// update queryDatasource filter
			//if queryDatasource == "" {
			//	queryDatasource = strings.Join(datasourceIDs, ",")
			//} else {
			// calc intersection with queryDatasource and datasourceIDs
			//queryDatasource = util.StringArrayIntersection(queryDatasourceIDs, datasourceIDs)
			//if len(queryDatasource) == 0 {
			//	// return empty search result when intersection queryDatasource ids is empty
			//	panic("queryDatasource is empty")
			//}
			//finalDatasourceIDs = queryDatasource
			//queryDatasource = strings.Join(queryDatasource, ",")
			//}
		} else {
			//user is select all queryDatasource
			finalDatasourceIDs = common.GetUsersOwnDatasource(userID)
		}
	} else {
		//if queryDatasource == "" || util.ContainStr(queryDatasource, "*") {
		//user is select all queryDatasource
		finalDatasourceIDs = common.GetUsersOwnDatasource(userID)
		//} else {
		//	queryDatasource := strings.Split(queryDatasource, ",")
		//	finalDatasourceIDs = queryDatasource
		//}
	}

	if len(queryDatasourceIDs) > 0 {
		//only merge if the query are specify datasources
		finalDatasourceIDs = append(finalDatasourceIDs, sharedDatasources...)
		finalDatasourceIDs = util.StringArrayIntersection(queryDatasourceIDs, finalDatasourceIDs)
	}

	//datasourceClause := BuildDatasourceClause(queryDatasource, true)
	//if datasourceClause != nil {
	//	mustClauses = append(mustClauses, datasourceClause)
	//}

	mustClauses := []*orm.Clause{}
	if len(finalDatasourceIDs) > 0 {
		//log.Error("HIT finalDatasourceIDs:", finalDatasourceIDs)
		//mergedArray := []string{}
		//mergedArray = append(finalDatasourceIDs, sharedDatasources...)
		mustClauses = append(mustClauses, orm.TermsQuery("source.id", finalDatasourceIDs))
	}

	//if queryDatasource != "" {
	//	if strings.Contains(queryDatasource, ",") {
	//		arr := strings.Split(queryDatasource, ",")
	//		mustClauses = append(mustClauses, orm.TermsQuery("source.id", arr))
	//	} else {
	//		mustClauses = append(mustClauses, orm.TermQuery("source.id", queryDatasource))
	//	}
	//}
	if !filterDisabled {
		return finalDatasourceIDs, mustClauses
	}

	disabledIDs, err := common.GetDisabledDatasourceIDs()
	if err != nil {
		panic(err)
	}

	//TODO verify this filter
	if len(disabledIDs) > 0 {
		mustClauses = append(mustClauses, orm.MustNotQuery(orm.TermsQuery("source.id", disabledIDs)))
	}

	return finalDatasourceIDs, mustClauses
}
