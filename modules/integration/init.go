/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/document"
	"infini.sh/coco/plugins/security/filter"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasttemplate"
)

type APIHandler struct {
	api.Handler
	searchBoxWrapperTemplate  *fasttemplate.Template
	searchPageWrapperTemplate *fasttemplate.Template
}

func NewAPIHandler() *APIHandler {
	handler := APIHandler{}
	tpl, err := util.FileGetContent(util.JoinPath(global.Env().SystemConfig.PathConfig.Config, "widget/searchbox/wrapper.js"))
	if err != nil {
		panic(err)
	}
	handler.searchBoxWrapperTemplate, err = fasttemplate.NewTemplate(string(tpl), "$[[", "]]")

	tpl, err = util.FileGetContent(util.JoinPath(global.Env().SystemConfig.PathConfig.Config, "widget/searchpage/wrapper.js"))
	if err != nil {
		panic(err)
	}
	handler.searchPageWrapperTemplate, err = fasttemplate.NewTemplate(string(tpl), "$[[", "]]")

	return &handler
}

const Category = "coco"
const Datasource = "integration"

func init() {

	// register allow origin function
	filter.RegisterAllowOriginFunc("integration", IntegrationAllowOrigin)

	createPermission := security.GetSimplePermission(Category, Datasource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(security.Search))
	updateSuggestTopicsPermission := security.GetSimplePermission(Category, Datasource, string("update_suggest_topics"))
	viewSuggestTopicsPermission := security.GetSimplePermission(Category, Datasource, string("view_suggest_topics"))

	createDocPermission := security.GetSimplePermission(Category, document.Resource, string(security.Create))

	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission, createDocPermission, updateSuggestTopicsPermission, viewSuggestTopicsPermission)
	security.RegisterPermissionsToRole(core.WidgetRole, readPermission, viewSuggestTopicsPermission)

	handler := NewAPIHandler()
	api.HandleUIMethod(api.POST, "/integration/", handler.create, api.RequirePermission(createPermission))

	api.HandleUIMethod(api.GET, "/integration/_search", handler.search, api.RequirePermission(searchPermission), api.Feature(filter.FeatureRemoveSensitiveField))
	api.HandleUIMethod(api.POST, "/integration/_search", handler.search, api.RequirePermission(searchPermission), api.Feature(filter.FeatureRemoveSensitiveField))

	api.HandleUIMethod(api.OPTIONS, "/integration/:id", handler.get, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/integration/:id", handler.get, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.PUT, "/integration/:id", handler.update, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/integration/:id", handler.delete, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.POST, "/integration/:id/_renew_token", handler.renewAPIToken, api.RequirePermission(updatePermission))

	api.HandleUIMethod(api.POST, "/integration/:id/chat/_suggest", handler.updateSuggestTopic, api.RequirePermission(viewSuggestTopicsPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/integration/:id/chat/_suggest", handler.viewSuggestTopic, api.RequirePermission(viewSuggestTopicsPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/integration/:id/chat/_suggest", handler.viewSuggestTopic, api.RequirePermission(viewSuggestTopicsPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.GET, "/integration/:id/widget", handler.widgetWrapper, api.AllowPublicAccess(),
		api.Feature(filter.FeatureCORS), api.Feature(core.FeatureByPassCORSCheck))
	api.HandleUIMethod(api.OPTIONS, "/integration/:id/widget", handler.widgetWrapper, api.AllowPublicAccess(),
		api.Feature(filter.FeatureCORS), api.Feature(core.FeatureByPassCORSCheck))
}
