/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

const Category = "coco"
const Datasource = "connector"

func init() {

	createPermission := security.GetSimplePermission(Category, Datasource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(security.Search))
	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission)
	security.AssignPermissionsToRoles(searchPermission, core.WidgetRole)

	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/connector/", handler.create, api.RequirePermission(createPermission))
	api.HandleUIMethod(api.GET, "/connector/:id", handler.get, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.PUT, "/connector/:id", handler.update, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/connector/:id", handler.delete, api.RequirePermission(deletePermission))

	api.HandleUIMethod(api.OPTIONS, "/connector/_search", handler.search, api.RequirePermission(searchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.GET, "/connector/_search", handler.search, api.RequirePermission(searchPermission), api.Feature(core.FeatureCORS),
		api.Feature(core.FeatureMaskSensitiveField))
	api.HandleUIMethod(api.POST, "/connector/_search", handler.search, api.RequirePermission(searchPermission), api.Feature(core.FeatureCORS),
		api.Feature(core.FeatureMaskSensitiveField))

}
