/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

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
const Session = "session"
const Assistant = "assistant"

const ViewHistoryAction = "view_all_session_history"
const ViewSingleSessionHistoryAction = "view_single_session_history"
const manageChatSessionAction = "view_single_session_history"
const cancelChatSessionAction = "cancel_session"

func init() {
	createPermission := security.GetSimplePermission(Category, Session, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Session, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Session, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Session, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Session, string(security.Search))
	manageChatSessionPermission := security.GetSimplePermission(Category, Session, manageChatSessionAction)
	viewHistoryPermission := security.GetSimplePermission(Category, Session, ViewHistoryAction)
	viewSessionHistoryPermission := security.GetSimplePermission(Category, Session, ViewSingleSessionHistoryAction)

	createAssistantPermission := security.GetSimplePermission(Category, Assistant, string(security.Create))
	updateAssistantPermission := security.GetSimplePermission(Category, Assistant, string(security.Update))
	readAssistantPermission := security.GetSimplePermission(Category, Assistant, string(security.Read))
	deleteAssistantPermission := security.GetSimplePermission(Category, Assistant, string(security.Delete))
	searchAssistantPermission := security.GetSimplePermission(Category, Assistant, string(security.Search))
	askAssistantPermission := security.GetSimplePermission(Category, Assistant, string("ask"))

	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, askAssistantPermission, deletePermission, searchPermission, viewHistoryPermission, manageChatSessionPermission, cancelChatSessionAction)
	security.GetOrInitPermissionKeys(createAssistantPermission, updateAssistantPermission, readAssistantPermission, askAssistantPermission, deleteAssistantPermission, searchAssistantPermission)

	security.RegisterPermissionsToRole(core.WidgetRole, createPermission, searchPermission, viewSessionHistoryPermission, readAssistantPermission, searchAssistantPermission, askAssistantPermission, cancelChatSessionAction)

	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/chat/_history", handler.getChatSessions, api.RequirePermission(viewHistoryPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/_history", handler.getChatSessions, api.RequirePermission(viewHistoryPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/_create", handler.createChatSession, api.RequirePermission(createPermission), api.Feature(core.FeatureCORS), api.Feature(core.FeatureFingerprintThrottle))
	api.HandleUIMethod(api.OPTIONS, "/chat/_create", handler.createChatSession, api.RequirePermission(createPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_chat", handler.sendChatMessageV2, api.RequirePermission(createPermission), api.Feature(core.FeatureCORS), api.Feature(core.FeatureFingerprintThrottle))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_chat", handler.sendChatMessageV2, api.RequirePermission(createPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/chat/:session_id", handler.getSession, api.RequirePermission(readPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.PUT, "/chat/:session_id", handler.updateSession, api.RequirePermission(updatePermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.DELETE, "/chat/:session_id", handler.deleteSession, api.RequirePermission(deletePermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_cancel", handler.cancelReplyMessage, api.RequirePermission(manageChatSessionPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_cancel", handler.cancelReplyMessage, api.RequirePermission(manageChatSessionPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_open", handler.openChatSession, api.RequirePermission(manageChatSessionPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_open", handler.openChatSession, api.RequirePermission(manageChatSessionPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_close", handler.closeChatSession, api.RequirePermission(manageChatSessionPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_close", handler.closeChatSession, api.RequirePermission(manageChatSessionPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/chat/:session_id/_history", handler.getChatHistoryBySession, api.RequirePermission(viewSessionHistoryPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_history", handler.getChatHistoryBySession, api.RequirePermission(viewSessionHistoryPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.POST, "/assistant/", handler.createAssistant, api.RequirePermission(createAssistantPermission))
	api.HandleUIMethod(api.GET, "/assistant/:id", handler.getAssistant, api.RequirePermission(readAssistantPermission))

	api.HandleUIMethod(api.POST, "/assistant/:id/_ask", handler.askAssistant, api.RequirePermission(askAssistantPermission), api.Feature(core.FeatureCORS), api.Feature(core.FeatureFingerprintThrottle))
	api.HandleUIMethod(api.OPTIONS, "/assistant/:id/_ask", handler.askAssistant, api.RequirePermission(askAssistantPermission), api.Feature(core.FeatureCORS))

	// non-streaming twin of _ask, exposed as the ask_assistant MCP tool
	api.HandleUIMethod(api.POST, "/assistant/:id/_ask_sync", handler.askAssistantSync, api.RequirePermission(askAssistantPermission), api.Feature(core.FeatureCORS),
		api.MCPTool("ask_assistant", "Ask a Coco AI assistant a question and get its final answer in one response. The assistant runs the full RAG pipeline over the connected enterprise content. Find assistant IDs with search_assistants."),
		api.Label(api.MCPToolInputSchema, util.ToJson(util.MapStr{
			"type": "object",
			"properties": util.MapStr{
				"path_params": util.MapStr{
					"type":        "object",
					"description": "Route parameters.",
					"properties":  util.MapStr{"id": util.MapStr{"type": "string", "description": "Assistant ID (see search_assistants)."}},
					"required":    []string{"id"},
				},
				"body": util.MapStr{
					"type":       "object",
					"properties": util.MapStr{"message": util.MapStr{"type": "string", "description": "The question to ask."}},
					"required":   []string{"message"},
				},
			},
			"required":             []string{"path_params", "body"},
			"additionalProperties": false,
		}, false)))

	api.HandleUIMethod(api.PUT, "/assistant/:id", handler.updateAssistant, api.RequirePermission(updateAssistantPermission))
	api.HandleUIMethod(api.DELETE, "/assistant/:id", handler.deleteAssistant, api.RequirePermission(deleteAssistantPermission))
	api.HandleUIMethod(api.GET, "/assistant/_search", handler.searchAssistant, api.RequirePermission(searchAssistantPermission), api.Feature(core.FeatureCORS),
		api.MCPTool("search_assistants", "List or search the AI assistants configured in Coco AI. Each assistant has its own model, data scope and behavior; use its ID with ask_assistant."),
		api.Label(api.MCPToolInputSchema, common.MCPQueryEnvelopeSchema(util.MapStr{
			"query": util.MapStr{"type": "string", "description": "Optional keyword to filter assistants by name or description."},
			"size":  util.MapStr{"type": "integer", "description": "Page size, default 10."},
			"from":  util.MapStr{"type": "integer", "description": "Pagination offset."},
		}, nil)))
	api.HandleUIMethod(api.OPTIONS, "/assistant/_search", handler.searchAssistant, api.RequirePermission(searchAssistantPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.POST, "/assistant/_search", handler.searchAssistant, api.RequirePermission(searchAssistantPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.POST, "/assistant/:id/_clone", handler.cloneAssistant, api.RequirePermission(createAssistantPermission))

	// scenario templates: read/search + one-click instantiate (copies into a
	// real assistant; optional kb_id binds it to a wiki knowledge base)
	registerTemplateCRUD()
	scheduleSeedTemplates()
	api.HandleUIMethod(api.POST, "/assistant-template/:id/_instantiate", handler.instantiateTemplate, api.RequirePermission(createAssistantPermission),
		api.MCPTool("instantiate_assistant_template", "Create a working assistant from a scenario template (support/sales/HR/IT presets); optionally bind it to a wiki knowledge base by kb_id. Returns the new assistant id."))
}
