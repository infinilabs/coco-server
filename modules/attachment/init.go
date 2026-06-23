/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package attachment

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

// attachmentProcessingQueue is the queue that upload handlers push attachment
// IDs into for post-upload processing. It is initialised in an AfterSetup
// callback so that the KV store (required by SmartGetOrInitConfig) is
// guaranteed to be registered before this call executes, and before the HTTP
// server starts accepting requests.
var attachmentProcessingQueue *queue.QueueConfig

const Category = "coco"
const Datasource = "attachment"

func init() {
	// Depends on the KV module; run after setup to ensure it is ready.
	global.RegisterFuncAfterSetup(func() {
		attachmentProcessingQueue = queue.SmartGetOrInitConfig(&queue.QueueConfig{Name: core.AttachmentProcessingQueue})
	})

	createPermission := security.GetSimplePermission(Category, Datasource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(security.Search))

	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission)
	security.AssignPermissionsToRoles(searchPermission, core.WidgetRole)

	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/attachment/:file_id", handler.getAttachment, api.RequirePermission(readPermission), api.AllowOPTIONSS(), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.DELETE, "/attachment/:file_id", handler.deleteAttachment, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.HEAD, "/attachment/:file_id", handler.checkAttachment, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.GET, "/attachment/_search", handler.getAttachments, api.RequirePermission(searchPermission), api.AllowOPTIONSS(), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.POST, "/attachment/_search", handler.getAttachments, api.RequirePermission(searchPermission), api.AllowOPTIONSS(), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.POST, "/attachment/_upload", handler.uploadAttachment, api.RequirePermission(createPermission), api.AllowOPTIONSS(), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/attachment/:file_id/status", handler.getAttachmentStatus, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.GET, "/attachment/_status", handler.batchGetAttachmentStatus, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.POST, "/attachment/_status", handler.batchGetAttachmentStatus, api.RequirePermission(readPermission))

}
