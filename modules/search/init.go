/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package search

import (
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/security/filter"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

type APIHandler struct {
	api.Handler
}

const Category = "coco"
const Resource = "search"
const Assistant = "assistant"
const QuickAISearchAction = "quick_ai_access"

func init() {

	permission := security.GetSimplePermission(Category, Resource, string(security.Search))
	assistantSearchPermission := security.GetSimplePermission(Category, Assistant, string(QuickAISearchAction))

	security.GetOrInitPermissionKeys(assistantSearchPermission)
	security.AssignPermissionsToRoles(permission, core.WidgetRole)

	handler := APIHandler{}
	api.HandleUIMethod(api.GET, "/query/_suggest", handler.suggest, api.RequirePermission(permission))
	api.HandleUIMethod(api.GET, "/query/_recommend", handler.recommend, api.RequirePermission(permission))

	api.HandleUIMethod(api.OPTIONS, "/query/_search", handler.search, api.RequirePermission(permission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/query/_search", handler.search, api.RequirePermission(permission), api.Feature(filter.FeatureCORS))

}
