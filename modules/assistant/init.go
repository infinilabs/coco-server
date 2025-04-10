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
const Datasource = "assistant"

const ViewHistoryAction = "view_all_session_history"
const ViewSingleSessionHistoryAction = "view_single_session_history"
const manageChatSessionAction = "view_single_session_history"

func init() {
	createPermission := security.GetSimplePermission(Category, Datasource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Datasource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Datasource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Datasource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Datasource, string(security.Search))
	manageChatSessionPermission := security.GetSimplePermission(Category, Datasource, manageChatSessionAction)
	viewHistoryPermission := security.GetSimplePermission(Category, Datasource, ViewHistoryAction)
	viewSessionHistoryPermission := security.GetSimplePermission(Category, Datasource, ViewSingleSessionHistoryAction)
	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission, viewHistoryPermission, manageChatSessionPermission)

	security.RegisterPermissionsToRole(core.WidgetRole, searchPermission, createPermission, viewSessionHistoryPermission)

	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/chat/_history", handler.getChatSessions, api.RequirePermission(viewHistoryPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/_history", handler.getChatSessions, api.RequirePermission(viewHistoryPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/_new", handler.newChatSession, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/_new", handler.newChatSession, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS))

	api.HandleUIMethod(api.POST, "/chat/:session_id/_send", handler.sendChatMessage, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_send", handler.sendChatMessage, api.RequirePermission(createPermission), api.Feature(filter.FeatureCORS))

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

	api.HandleUIMethod(api.POST, "/assistant/", handler.createAssistant)
	api.HandleUIMethod(api.GET, "/assistant/:id", handler.getAssistant)
	api.HandleUIMethod(api.PUT, "/assistant/:id", handler.updateAssistant)
	api.HandleUIMethod(api.DELETE, "/assistant/:id", handler.deleteAssistant)
	api.HandleUIMethod(api.GET, "/assistant/_search", handler.searchAssistant)
	api.HandleUIMethod(api.POST, "/assistant/_search", handler.searchAssistant)
}
