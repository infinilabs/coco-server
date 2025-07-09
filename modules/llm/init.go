/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package llm

import (
	"infini.sh/coco/plugins/security/filter"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

const Category = "coco"
const Resource = "model_provider"
const MCPServerResource = "mcp_server"

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

	createMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Create))
	updateMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Update))
	readMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Read))
	deleteMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Delete))
	searchMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Search))
	security.GetOrInitPermissionKeys(createMCPServerPermission, updateMCPServerPermission, readMCPServerPermission, deleteMCPServerPermission, searchMCPServerPermission)
	handler := APIHandler{}

	var secretKeys = map[string]bool{}
	secretKeys["config"] = true
	secretKeys["api_key"] = true

	api.HandleUIMethod(api.POST, "/model_provider/", handler.create, api.RequireLogin(), api.RequirePermission(createPermission))
	api.HandleUIMethod(api.GET, "/model_provider/:id", handler.get, api.RequireLogin(), api.RequirePermission(readPermission))
	api.HandleUIMethod(api.PUT, "/model_provider/:id", handler.update, api.RequireLogin(), api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/model_provider/:id", handler.delete, api.RequireLogin(), api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/model_provider/_search", handler.search, api.RequireLogin(), api.RequirePermission(searchPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureRemoveSensitiveField), api.Label(filter.SensitiveFields, secretKeys))
	api.HandleUIMethod(api.POST, "/model_provider/_search", handler.search, api.RequireLogin(), api.RequirePermission(searchPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureRemoveSensitiveField), api.Label(filter.SensitiveFields, secretKeys))

	api.HandleUIMethod(api.POST, "/mcp_server/", handler.createMCPServer)
	api.HandleUIMethod(api.GET, "/mcp_server/:id", handler.getMCPServer)
	api.HandleUIMethod(api.PUT, "/mcp_server/:id", handler.updateMCPServer)
	api.HandleUIMethod(api.DELETE, "/mcp_server/:id", handler.deleteMCPServer)
	api.HandleUIMethod(api.GET, "/mcp_server/_search", handler.searchMCPServer, api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureRemoveSensitiveField), api.Label(filter.SensitiveFields, secretKeys))
	api.HandleUIMethod(api.OPTIONS, "/mcp_server/_search", handler.searchMCPServer, api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/mcp_server/_search", handler.searchMCPServer, api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureRemoveSensitiveField), api.Label(filter.SensitiveFields, secretKeys))
}
