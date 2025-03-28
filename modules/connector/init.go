/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"infini.sh/cloud/core/security/rbac"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

const Category = "coco"
const Datasource = "connector"

func init() {

	createPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(rbac.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(rbac.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Search))
	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission)

	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/connector/", handler.create, api.RequirePermission(createPermission))
	api.HandleUIMethod(api.GET, "/connector/:id", handler.get, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.PUT, "/connector/:id", handler.update, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/connector/:id", handler.delete, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/connector/_search", handler.search, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/connector/_search", handler.search, api.RequirePermission(searchPermission))

}
