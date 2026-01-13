/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/connector"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
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
		integrationID := req.Header.Get(core.HeaderIntegrationID)

		teamsID, _ := reqUser.GetStringArray(orm.TeamsIDKey)

		result := elastic.SearchResponseWithMeta[core.Document]{}
		resp, err := QueryDocuments(req.Context(), reqUser.MustGetUserID(), teamsID, builder, query, datasource, integrationID, category, subcategory, richCategory, nil)
		if err != nil {
			panic(err)
		}
		util.MustFromJSONBytes(resp.Raw, &result)

		docsSize := len(result.Hits.Hits)
		//update icon
		if docsSize > 0 {
			for i := range result.Hits.Hits {
				RefineIcon(req.Context(), &result.Hits.Hits[i].Source)
			}
		}

		size := h.GetIntOrDefault(req, "size", 10)
		assistantSearchPermission := security.GetSimplePermission(Category, Assistant, string(QuickAISearchAction))
		perID := security.GetOrInitPermissionKey(assistantSearchPermission)

		//not for widget integration
		if datasource == "" && integrationID == "" && ((reqUser.Roles != nil && util.AnyInArrayEquals(reqUser.Roles, security.RoleAdmin)) || reqUser.UserAssignedPermission.ValidateFor(perID)) {
			assistantSize := 2
			if docsSize < 5 {
				assistantSize = size - (docsSize)
			}

			assistants := searchAssistant(req, query, assistantSize)
			if len(assistants) > 0 {
				newHits := make([]elastic.DocumentWithMeta[core.Document], 0, len(assistants))
				for i, assistant := range assistants {
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
					newHit := elastic.DocumentWithMeta[core.Document]{
						ID:     assistant.ID,
						Index:  "assistant",
						Source: doc,
						Score:  result.Hits.MaxScore + float32(size-i),
					}
					newHits = append(newHits, newHit)
				}
				result.Hits.Hits = append(newHits, result.Hits.Hits...)
			}
		}

		api.WriteJSON(w, result, 200)
	} else {
		h.WriteJSON(w, elastic.SearchResponse{Hits: elastic.Hits{Total: elastic.TotalHits{Value: 0, Relation: "eq"}}}, http.StatusOK)
	}
}

// ResolveIcon runs the icon fallback chain:
// 1. currentIcon
// 2. datasource.Icon
// 3. connector.Icon
func ResolveIcon(
	connectorConfig *core.Connector,
	datasourceConfig *core.DataSource,
	currentIcon string,
) string {

	// 1. Try current field's icon
	if icon := common.ParseAndGetIcon(connectorConfig, currentIcon); icon != "" {
		return icon
	}

	// 2. Try datasource icon
	if datasourceConfig.Icon != "" {
		if icon := common.ParseAndGetIcon(connectorConfig, datasourceConfig.Icon); icon != "" {
			return icon
		}
	}

	// 3. Try connector default icon
	if icon := common.ParseAndGetIcon(connectorConfig, connectorConfig.Icon); icon != "" {
		return icon
	}

	return ""
}

func RefineIcon(ctx context.Context, doc *core.Document) {
	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectReadAccess()
	ctx1.PermissionScope(security.PermissionScopePlatform)

	datasourceConfig, err := common.GetDatasourceConfig(ctx1, doc.Source.ID)
	if err != nil || datasourceConfig == nil || datasourceConfig.Connector.ConnectorID == "" {
		return
	}

	connectorConfig, err := connector.GetConnectorConfig(datasourceConfig.Connector.ConnectorID)
	if err != nil || connectorConfig == nil {
		return
	}

	// Update doc.Icon
	if icon := ResolveIcon(connectorConfig, datasourceConfig, doc.Icon); icon != "" {
		doc.Icon = icon
	}

	// Update doc.Source.Icon
	if icon := ResolveIcon(connectorConfig, datasourceConfig, doc.Source.Icon); icon != "" {
		doc.Source.Icon = icon
	}
}

func searchAssistant(req *http.Request, query string, size int) []core.Assistant {
	docs := []core.Assistant{}
	if size <= 0 {
		size = 2
	}

	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name^10", "name.pinyin^5", "combined_fulltext^1")
	if err != nil {
		return docs
	}
	builder.Query(query)
	builder.Must(orm.TermQuery("enabled", true))
	builder.Size(size)
	builder.Fuzziness(3)

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

func BuildDatasourceFilter(userID string, checkingScopeDatasources, directAccessDatasources []string, queryDatasourceIDs []string, integrationID string, filterDisabled bool) ([]string, []string, []string) {

	//merge user's own datasource, other shareable datasource, within user's query datasource, within integration's datasource

	//fist, merge the user's accessable datasource
	userOwnDatasourceIDs := common.GetUsersOwnDatasource(userID)
	directAccessDatasources = append(directAccessDatasources, userOwnDatasourceIDs...)

	log.Trace("userID:", userID, "user's own", userOwnDatasourceIDs, ",queryDatasource:", queryDatasourceIDs, ",integrationID:", integrationID, ",merged datasources:", directAccessDatasources)

	finalDatasourceIDs := directAccessDatasources
	if len(queryDatasourceIDs) > 0 && !util.ContainsAnyInArray("*", queryDatasourceIDs) {
		//only merge if the query are specify datasources
		finalDatasourceIDs = util.StringArrayIntersection(queryDatasourceIDs, finalDatasourceIDs)
		checkingScopeDatasources = util.StringArrayIntersection(queryDatasourceIDs, checkingScopeDatasources)
	}

	if integrationID != "" {
		// get queryDatasource by integration id
		datasourceIDs, hasAll, err := GetDatasourceByIntegration(integrationID)
		if err != nil {
			panic(err)
		}

		if len(datasourceIDs) == 0 {
			log.Warnf("empty datasource for integration: %v", integrationID)
		}

		log.Trace("integration:", integrationID, ", datasource:", datasourceIDs, ",has all:", hasAll)

		//finalDatasourceIDs = datasourceIDs
		if !hasAll {
			if len(datasourceIDs) > 0 {
				finalDatasourceIDs = util.StringArrayIntersection(datasourceIDs, finalDatasourceIDs)
				checkingScopeDatasources = util.StringArrayIntersection(datasourceIDs, checkingScopeDatasources)

				//log.Error("finalDatasourceIDs:", finalDatasourceIDs, ",checkingScopeDatasources", checkingScopeDatasources)

			}
		}
	}

	if len(finalDatasourceIDs) == 0 && len(checkingScopeDatasources) == 0 {
		panic("empty datasource")
	}

	log.Trace("userID:", userID, "user's own", userOwnDatasourceIDs, ",queryDatasource:", queryDatasourceIDs, ",integrationID:", integrationID, ",final merged directAccess datasources:", finalDatasourceIDs)

	if !filterDisabled {
		return checkingScopeDatasources, finalDatasourceIDs, []string{}
	}

	disabledIDs, err := common.GetDisabledDatasourceIDs()
	if err != nil {
		panic(err)
	}

	return checkingScopeDatasources, finalDatasourceIDs, disabledIDs
}
