/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

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

	createAssistantPermission := security.GetSimplePermission(Category, Session, string(security.Create))
	updateAssistantPermission := security.GetSimplePermission(Category, Session, string(security.Update))
	readAssistantPermission := security.GetSimplePermission(Category, Session, string(security.Read))
	deleteAssistantPermission := security.GetSimplePermission(Category, Session, string(security.Delete))
	searchAssistantPermission := security.GetSimplePermission(Category, Session, string(security.Search))
	askAssistantPermission := security.GetSimplePermission(Category, Assistant, string("ask"))

	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, askAssistantPermission, deletePermission, searchPermission, viewHistoryPermission, manageChatSessionPermission, cancelChatSessionAction)
	security.GetOrInitPermissionKeys(createAssistantPermission, updateAssistantPermission, readAssistantPermission, askAssistantPermission, deleteAssistantPermission, searchAssistantPermission)

	security.RegisterPermissionsToRole(core.WidgetRole, createPermission, searchPermission, viewSessionHistoryPermission, readAssistantPermission, searchAssistantPermission, askAssistantPermission, cancelChatSessionAction)

	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/chat/_history", handler.getChatSessions, api.RequirePermission(viewHistoryPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/_history", handler.getChatSessions, api.RequirePermission(viewHistoryPermission), api.Feature(filter.FeatureCORS))

	//deprecated, will be removed soon
	api.HandleUIMethod(api.POST, "/chat/_new", handler.newChatSession, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureFingerprintThrottle))
	api.HandleUIMethod(api.OPTIONS, "/chat/_new", handler.newChatSession, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS))

	//deprecated, will be removed soon
	api.HandleUIMethod(api.POST, "/chat/:session_id/_send", handler.sendChatMessage, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureFingerprintThrottle))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_send", handler.sendChatMessage, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/_create", handler.createChatSession, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureFingerprintThrottle))
	api.HandleUIMethod(api.OPTIONS, "/chat/_create", handler.createChatSession, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_chat", handler.sendChatMessageV2, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureFingerprintThrottle))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_chat", handler.sendChatMessageV2, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.GET, "/chat/:session_id", handler.getSession, api.RequirePermission(readPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.PUT, "/chat/:session_id", handler.updateSession, api.RequirePermission(updatePermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.DELETE, "/chat/:session_id", handler.deleteSession, api.RequirePermission(deletePermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_cancel", handler.cancelReplyMessage, api.RequirePermission(manageChatSessionPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_cancel", handler.cancelReplyMessage, api.RequirePermission(manageChatSessionPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_open", handler.openChatSession, api.RequirePermission(manageChatSessionPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_open", handler.openChatSession, api.RequirePermission(manageChatSessionPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_close", handler.closeChatSession, api.RequirePermission(manageChatSessionPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_close", handler.closeChatSession, api.RequirePermission(manageChatSessionPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.GET, "/chat/:session_id/_history", handler.getChatHistoryBySession, api.RequirePermission(viewSessionHistoryPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_history", handler.getChatHistoryBySession, api.RequirePermission(viewSessionHistoryPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.POST, "/assistant/", handler.createAssistant, api.RequirePermission(createAssistantPermission))
	api.HandleUIMethod(api.GET, "/assistant/:id", handler.getAssistant, api.RequirePermission(readAssistantPermission))

	api.HandleUIMethod(api.POST, "/assistant/:id/_ask", handler.askAssistant, api.RequirePermission(askAssistantPermission), api.Feature(filter.FeatureCORS), api.Feature(filter.FeatureFingerprintThrottle))
	api.HandleUIMethod(api.OPTIONS, "/assistant/:id/_ask", handler.askAssistant, api.RequirePermission(askAssistantPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.PUT, "/assistant/:id", handler.updateAssistant, api.RequirePermission(updateAssistantPermission))
	api.HandleUIMethod(api.DELETE, "/assistant/:id", handler.deleteAssistant, api.RequirePermission(deleteAssistantPermission))
	api.HandleUIMethod(api.GET, "/assistant/_search", handler.searchAssistant, api.RequirePermission(searchAssistantPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/assistant/_search", handler.searchAssistant, api.RequirePermission(searchAssistantPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/assistant/_search", handler.searchAssistant, api.RequirePermission(searchAssistantPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/assistant/:id/_clone", handler.cloneAssistant, api.RequirePermission(createAssistantPermission))
}
