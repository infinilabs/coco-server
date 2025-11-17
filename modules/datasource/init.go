/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package datasource

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/document"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

const Category = "coco"
const Datasource = "datasource"

func init() {

	createPermission := security.GetSimplePermission(Category, Datasource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(security.Search))

	createDocPermission := security.GetSimplePermission(Category, document.Resource, string(security.Create))

	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission, createDocPermission)
	security.RegisterPermissionsToRole(core.WidgetRole, searchPermission)

	handler := APIHandler{}

	api.HandleUIMethod(api.POST, "/datasource/", handler.createDatasource, api.RequirePermission(createPermission))
	api.HandleUIMethod(api.DELETE, "/datasource/:id", handler.deleteDatasource, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/datasource/:id", handler.getDatasource, api.RequirePermission(readPermission))
	api.HandleUIMethod(api.PUT, "/datasource/:id", handler.updateDatasource, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.OPTIONS, "/datasource/_search", handler.searchDatasource, api.RequirePermission(searchPermission), api.Feature(core.FeatureCORS))

	var secretKeys = map[string]bool{}
	secretKeys["config"] = true

	api.HandleUIMethod(api.GET, "/datasource/_search", handler.searchDatasource, api.RequirePermission(searchPermission),
		api.Feature(core.FeatureCORS), api.Feature(core.FeatureMaskSensitiveField), api.Feature(core.FeatureRemoveSensitiveField),
		api.Label(core.SensitiveFields, secretKeys))
	api.HandleUIMethod(api.POST, "/datasource/_search", handler.searchDatasource, api.RequirePermission(searchPermission),
		api.Feature(core.FeatureCORS),
		api.Feature(core.FeatureMaskSensitiveField),
		api.Feature(core.FeatureRemoveSensitiveField),
		api.Label(core.SensitiveFields, secretKeys))

	//shortcut to indexing docs into this datasource
	api.HandleUIMethod(api.POST, "/datasource/:id/_doc", handler.createDocInDatasource, api.RequirePermission(createPermission))
	api.HandleUIMethod(api.POST, "/datasource/:id/_doc/:doc_id", handler.createDocInDatasourceWithID, api.RequirePermission(createPermission))

}
