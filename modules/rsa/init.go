/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rsa

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

func init() {
	createPermission := security.GetSimplePermission(Category, Resource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Resource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Resource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Resource, string(security.Search))
	debugPermission := security.GetSimplePermission(Category, Resource, "debug")
	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission, debugPermission)

	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/rsa/", handler.createRemoteSource, api.RequirePermission(createPermission))
	api.HandleUIMethod(api.GET, "/rsa/:id", handler.getRemoteSource, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.PUT, "/rsa/:id", handler.updateRemoteSource, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/rsa/:id", handler.deleteRemoteSource, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/rsa/_search", handler.searchRemoteSource, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/rsa/_search", handler.searchRemoteSource, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/rsa/:id/_test", handler.testRemoteSource, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.GET, "/rsa/_stats", handler.statsHandler, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/rsa/_debug", handler.debugRemoteSource, api.RequirePermission(debugPermission))
}
