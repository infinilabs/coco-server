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
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

const Category = "coco"
const Datasource = "integration"

func init() {
	createPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(rbac.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(rbac.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Search))

	createDocPermission := security.GetSimplePermission(Category, document.Resource, string(rbac.Create))

	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission, createDocPermission)
	security.RegisterPermissionsToRole(core.WidgetRole, readPermission)

	handler := APIHandler{}
	api.HandleUIMethod(api.POST, "/integration/", handler.create, api.RequirePermission(createPermission))
	api.HandleUIMethod(api.OPTIONS, "/integration/:id", handler.get, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/integration/:id", handler.get, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.PUT, "/integration/:id", handler.update, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/integration/:id", handler.delete, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/integration/_search", handler.search, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/integration/_search", handler.search, api.RequirePermission(searchPermission))
	// register allow origin function
	filter.RegisterAllowOriginFunc("integration", IntegrationAllowOrigin)
}
