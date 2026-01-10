/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
	documentMetadata string
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
	api.HandleUIMethod(api.PUT, "/document/:doc_id", handler.updateDoc, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/document/:doc_id", handler.deleteDoc, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/document/_search", handler.searchDocs, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.DELETE, "/document/", handler.batchDeleteDoc, api.RequirePermission(deletePermission))

	querySearchPermission := security.GetSimplePermission(Category, Search, string(security.Search))
	assistantSearchPermission := security.GetSimplePermission(Category, Assistant, string(QuickAISearchAction))
	security.GetOrInitPermissionKeys(querySearchPermission, assistantSearchPermission)
	security.AssignPermissionsToRoles(querySearchPermission, core.WidgetRole)

	api.HandleUIMethod(api.OPTIONS, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.GET, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.POST, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_suggest", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_suggest", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_suggest/:tag", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_suggest/:tag", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_recommend/:tag", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_recommend/:tag", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	global.RegisterFuncAfterSetup(func() {
		cfg := struct {
			DocumentMetadata string `json:"document_metadata" config:"document_metadata"`
		}{}
		ok, err := env.ParseConfig("suggest", &cfg)
		if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		}

		handler.documentMetadata = cfg.DocumentMetadata
		log.Trace("document metadata:", handler.documentMetadata)
	})

}
