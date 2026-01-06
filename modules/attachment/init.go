/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package attachment

import (
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

	createPermission := security.GetSimplePermission(Category, Datasource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(security.Search))

	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission)
	security.AssignPermissionsToRoles(searchPermission, core.WidgetRole)

	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/attachment/:file_id", handler.getAttachment, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.DELETE, "/attachment/:file_id", handler.deleteAttachment, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.HEAD, "/attachment/:file_id", handler.checkAttachment, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.GET, "/attachment/_search", handler.getAttachments, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/attachment/_search", handler.getAttachments, api.RequirePermission(searchPermission))
	api.HandleUIMethod(api.POST, "/attachment/_upload", handler.uploadAttachment, api.RequirePermission(createPermission))

	api.HandleUIMethod(api.GET, "/attachment/:file_id/stats", handler.getAttachmentStats, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.GET, "/attachment/_stats", handler.batchGetAttachmentStats, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.POST, "/attachment/_stats", handler.batchGetAttachmentStats, api.RequirePermission(readPermission))

}
