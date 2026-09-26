/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	log "github.com/cihub/seelog"
	"github.com/emirpasic/gods/maps/treemap"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

type APIHandler struct {
	api.Handler
	recommendConfigs map[string]core.RecommendResponse
	fieldMetadata    *treemap.Map
}

const Category = "coco"
const Resource = "document"
const Search = "search"
const Assistant = "assistant"
const QuickAISearchAction = "quick_ai_access"

func init() {
	handler := APIHandler{}

	createPermission := security.GetSimplePermission(Category, Resource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Resource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Resource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Resource, string(security.Search))
	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission)

	//for internal document management, security should be enabled
	api.HandleUIMethod(api.POST, "/document/", handler.createDoc, api.RequirePermission(createPermission))
	api.HandleUIMethod(api.GET, "/document/:doc_id", handler.getDoc, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.GET, "/document/:doc_id/raw_content/:hint", handler.getDocRawContent, api.RequirePermission(readPermission), api.AllowOPTIONSS(), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.PUT, "/document/:doc_id", handler.updateDoc, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/document/:doc_id", handler.deleteDoc, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/document/_search", handler.searchDocs, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.DELETE, "/document/", handler.batchDeleteDoc, api.RequirePermission(deletePermission))

	querySearchPermission := security.GetSimplePermission(Category, Search, string(security.Search))
	assistantSearchPermission := security.GetSimplePermission(Category, Assistant, QuickAISearchAction)
	searchStudioPermission := security.GetSimplePermission(Category, Search, "studio")
	security.GetOrInitPermissionKeys(querySearchPermission, assistantSearchPermission, searchStudioPermission)
	security.AssignPermissionsToRoles(querySearchPermission, core.WidgetRole)

	//live tuning surface for the dual-engine recall: runs both routes and
	//returns the RRF fusion math behind the hybrid_rrf search mode
	api.HandleUIMethod(api.POST, "/search/studio/test", handler.searchStudioTest, api.RequirePermission(searchStudioPermission))

	api.HandleUIMethod(api.OPTIONS, "/field_meta/:field_name", handler.getFieldMeta, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.GET, "/field_meta/:field_name", handler.getFieldMeta, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.POST, "/field_meta/:field_name", handler.getFieldMeta, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.OPTIONS, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.GET, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS),
		api.MCPTool("search_documents", "Search the enterprise content indexed by Coco AI — files, wiki pages, chat messages, web pages and more, across all connected data sources. Returns matched documents with title, summary, source and deep link."),
		api.Label(api.MCPToolInputSchema, common.MCPQueryEnvelopeSchema(util.MapStr{
			"query":      util.MapStr{"type": "string", "description": "Keywords to search for; Lucene query_string syntax is supported."},
			"datasource": util.MapStr{"type": "string", "description": "Restrict the search to one datasource ID (list IDs with search_datasources)."},
			"category":   util.MapStr{"type": "string", "description": "Document category filter, e.g. file, page, message."},
			"size":       util.MapStr{"type": "integer", "description": "Page size, default 10."},
			"from":       util.MapStr{"type": "integer", "description": "Pagination offset."},
		}, []string{"query"})))
	api.HandleUIMethod(api.POST, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_suggest", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_suggest", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_suggest/:tag", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_suggest/:tag", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_recommend", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_recommend", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_recommend/:tag", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_recommend/:tag", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	global.RegisterFuncAfterSetup(func() {

		fieldMetadataMap := map[string]FieldMetadata{}
		ok, err := env.ParseConfig("field_metadata", &fieldMetadataMap)
		if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		}
		// Convert to TreeMap for sorted iteration by key
		handler.fieldMetadata = treemap.NewWithStringComparator()
		for k, v := range fieldMetadataMap {
			handler.fieldMetadata.Put(k, v)
		}
		log.Trace(util.ToJson(fieldMetadataMap, true))

		cfg1 := map[string]string{}
		ok, err = env.ParseConfig("recommend", &cfg1)
		if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		}

		recommends := map[string]core.RecommendResponse{}
		for k, v := range cfg1 {
			recommend := core.RecommendResponse{}
			err := util.FromJson(v, &recommend)
			if err != nil {
				panic(err)
			}
			recommends[k] = recommend
		}
		handler.recommendConfigs = recommends
	})

}
