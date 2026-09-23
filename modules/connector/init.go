/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
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
		api.Feature(core.FeatureMaskSensitiveField),
		api.MCPTool("search_connectors", "List or search the data connectors configured in Coco AI (GitHub, Confluence, local filesystem, ...). A connector defines where content is crawled from; use its ID with search_datasources to find the datasources it created."),
		api.Label(api.MCPToolInputSchema, common.MCPQueryEnvelopeSchema(util.MapStr{
			"query": util.MapStr{"type": "string", "description": "Optional keyword to filter connectors by name or description."},
			"size":  util.MapStr{"type": "integer", "description": "Page size, default 10."},
			"from":  util.MapStr{"type": "integer", "description": "Pagination offset."},
		}, nil)))
	api.HandleUIMethod(api.POST, "/connector/_search", handler.search, api.RequirePermission(searchPermission), api.Feature(core.FeatureCORS),
		api.Feature(core.FeatureMaskSensitiveField))

}
