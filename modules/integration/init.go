/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"infini.sh/cloud/core/security/rbac"
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
	wrapperTemplate *fasttemplate.Template
}

func NewAPIHandler() *APIHandler {
	handler := APIHandler{}
	tpl, err := util.FileGetContent(util.JoinPath(global.Env().SystemConfig.PathConfig.Config, "widget/wrapper.js"))
	if err != nil {
		panic(err)
	}
	handler.wrapperTemplate, err = fasttemplate.NewTemplate(string(tpl), "$[[", "]]")

	return &handler
}

const Category = "coco"
const Datasource = "integration"

func init() {

	// register allow origin function
	filter.RegisterAllowOriginFunc("integration", IntegrationAllowOrigin)

	createPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(rbac.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(rbac.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Search))
	updateSuggestTopicsPermission := security.GetSimplePermission(Category, Datasource, string("update_suggest_topics"))
	viewSuggestTopicsPermission := security.GetSimplePermission(Category, Datasource, string("view_suggest_topics"))

	createDocPermission := security.GetSimplePermission(Category, document.Resource, string(rbac.Create))

	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission, createDocPermission, updateSuggestTopicsPermission, viewSuggestTopicsPermission)
	security.RegisterPermissionsToRole(core.WidgetRole, readPermission, viewSuggestTopicsPermission)

	handler := NewAPIHandler()
	api.HandleUIMethod(api.POST, "/integration/", handler.create, api.RequirePermission(createPermission))

	api.HandleUIMethod(api.GET, "/integration/_search", handler.search, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/integration/_search", handler.search, api.RequirePermission(searchPermission))

	api.HandleUIMethod(api.OPTIONS, "/integration/:id", handler.get, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/integration/:id", handler.get, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.PUT, "/integration/:id", handler.update, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/integration/:id", handler.delete, api.RequirePermission(deletePermission))

	api.HandleUIMethod(api.POST, "/integration/:id/chat/_suggest", handler.updateSuggestTopic, api.RequirePermission(viewSuggestTopicsPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/integration/:id/chat/_suggest", handler.viewSuggestTopic, api.RequirePermission(viewSuggestTopicsPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/integration/:id/chat/_suggest", handler.viewSuggestTopic, api.RequirePermission(viewSuggestTopicsPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.GET, "/integration/widget/wrapper", handler.widgetWrapper, api.AllowPublicAccess(),
		api.Feature(filter.FeatureCORS), api.Feature(core.FeatureByPassCORSCheck))
	api.HandleUIMethod(api.OPTIONS, "/integration/widget/wrapper", handler.widgetWrapper, api.AllowPublicAccess(),
		api.Feature(filter.FeatureCORS), api.Feature(core.FeatureByPassCORSCheck))
}
