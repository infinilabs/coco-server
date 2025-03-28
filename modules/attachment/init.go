/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package attachment

import (
	"infini.sh/cloud/core/security/rbac"
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

const Category = "coco"
const Datasource = "attachment"

func init() {

	createPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(rbac.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(rbac.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(rbac.Search))
	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission)
	security.AssignPermissionsToRoles(searchPermission, core.WidgetRole)

	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/attachment/:file_id", handler.getAttachment, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.DELETE, "/attachment/:file_id", handler.deleteAttachment, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.HEAD, "/attachment/:file_id", handler.checkAttachment, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.GET, "/attachment/_search", handler.getAttachments, api.RequirePermission(searchPermission))
}
