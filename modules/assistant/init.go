/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"infini.sh/coco/plugins/security/filter"
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/chat/_history", handler.getChatSessions, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/_history", handler.getChatSessions, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/chat/_new", handler.newChatSession, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/_new", handler.newChatSession, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_open", handler.openChatSession, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_open", handler.openChatSession, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_send", handler.sendChatMessage, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_send", handler.sendChatMessage, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_cancel", handler.cancelReplyMessage, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_cancel", handler.cancelReplyMessage, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_close", handler.closeChatSession, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_close", handler.closeChatSession, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.GET, "/chat/:session_id/_history", handler.getChatHistoryBySession, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/chat/:session_id/_history", handler.getChatHistoryBySession, api.RequireLogin(), api.Feature(filter.FeatureCORS))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_upload", handler.uploadAttachment, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/attachment/:file_id", handler.getAttachment, api.RequireLogin())
	api.HandleUIMethod(api.DELETE, "/attachment/:file_id", handler.deleteAttachment, api.RequireLogin())
	api.HandleUIMethod(api.HEAD, "/attachment/:file_id", handler.checkAttachment, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/attachment/_search", handler.getAttachments, api.RequireLogin())
}
