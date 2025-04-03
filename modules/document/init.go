/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

const Category = "coco"
const Resource = "document"

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
}
