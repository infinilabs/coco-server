/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package llm

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

const Category = "coco"
const Resource = "model_provider"

type APIHandler struct {
	api.Handler
}

func init() {
	createPermission := security.GetSimplePermission(Category, Resource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Resource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Resource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Resource, string(security.Search))
	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission)
	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/model_provider/", handler.create, api.RequireLogin(), api.RequirePermission(createPermission))
	api.HandleUIMethod(api.GET, "/model_provider/:id", handler.get, api.RequireLogin(), api.RequirePermission(readPermission))
	api.HandleUIMethod(api.PUT, "/model_provider/:id", handler.update, api.RequireLogin(), api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/model_provider/:id", handler.delete, api.RequireLogin(), api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/model_provider/_search", handler.search, api.RequireLogin(), api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/model_provider/_search", handler.search, api.RequireLogin(), api.RequirePermission(searchPermission))
}
