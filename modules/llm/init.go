/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package llm

import (
	"infini.sh/coco/core"
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
	createLLMPermission := security.GetSimplePermission(Category, Resource, string(security.Create))
	updateLLMPermission := security.GetSimplePermission(Category, Resource, string(security.Update))
	readLLMPermission := security.GetSimplePermission(Category, Resource, string(security.Read))
	deleteLLMPermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
	searchLLMPermission := security.GetSimplePermission(Category, Resource, string(security.Search))
	security.GetOrInitPermissionKeys(createLLMPermission, updateLLMPermission, readLLMPermission, deleteLLMPermission, searchLLMPermission)

	createMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Create))
	updateMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Update))
	readMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Read))
	deleteMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Delete))
	searchMCPServerPermission := security.GetSimplePermission(Category, MCPServerResource, string(security.Search))
	security.GetOrInitPermissionKeys(createMCPServerPermission, updateMCPServerPermission, readMCPServerPermission, deleteMCPServerPermission, searchMCPServerPermission)
	security.RegisterPermissionsToRole(core.WidgetRole, searchMCPServerPermission, searchLLMPermission)

	handler := APIHandler{}

	var secretKeys = map[string]bool{}
	secretKeys["config"] = true
	secretKeys["api_key"] = true

	api.HandleUIMethod(api.POST, "/model_provider/", handler.create, api.RequireLogin(), api.RequirePermission(createLLMPermission))
	api.HandleUIMethod(api.GET, "/model_provider/:id", handler.get, api.RequireLogin(), api.RequirePermission(readLLMPermission))
	api.HandleUIMethod(api.PUT, "/model_provider/:id", handler.update, api.RequireLogin(), api.RequirePermission(updateLLMPermission))
	api.HandleUIMethod(api.DELETE, "/model_provider/:id", handler.delete, api.RequireLogin(), api.RequirePermission(deleteLLMPermission))
	api.HandleUIMethod(api.GET, "/model_provider/_search", handler.search, api.RequireLogin(), api.RequirePermission(searchLLMPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureRemoveSensitiveField), api.Label(filter.SensitiveFields, secretKeys))
	api.HandleUIMethod(api.POST, "/model_provider/_search", handler.search, api.RequireLogin(), api.RequirePermission(searchLLMPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureRemoveSensitiveField), api.Label(filter.SensitiveFields, secretKeys))

	api.HandleUIMethod(api.POST, "/mcp_server/", handler.createMCPServer, api.RequirePermission(createMCPServerPermission))
	api.HandleUIMethod(api.GET, "/mcp_server/:id", handler.getMCPServer, api.RequirePermission(readMCPServerPermission))
	api.HandleUIMethod(api.PUT, "/mcp_server/:id", handler.updateMCPServer, api.RequirePermission(updateMCPServerPermission))
	api.HandleUIMethod(api.DELETE, "/mcp_server/:id", handler.deleteMCPServer, api.RequirePermission(deleteMCPServerPermission))

	api.HandleUIMethod(api.GET, "/mcp_server/_search", handler.searchMCPServer, api.RequireLogin(), api.Feature(filter.FeatureCORS), api.RequirePermission(searchMCPServerPermission), api.Feature(filter.FeatureRemoveSensitiveField), api.Label(filter.SensitiveFields, secretKeys))
	api.HandleUIMethod(api.OPTIONS, "/mcp_server/_search", handler.searchMCPServer, api.RequireLogin(), api.Feature(filter.FeatureCORS), api.RequirePermission(searchMCPServerPermission))
	api.HandleUIMethod(api.POST, "/mcp_server/_search", handler.searchMCPServer, api.RequireLogin(), api.Feature(filter.FeatureCORS), api.RequirePermission(searchMCPServerPermission), api.Feature(filter.FeatureRemoveSensitiveField), api.Label(filter.SensitiveFields, secretKeys))
}
