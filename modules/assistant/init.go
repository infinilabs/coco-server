/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}


func init() {
	handler := APIHandler{}

	api.HandleAPIMethod(api.GET, "/chat/_history", handler.getChatSessions)
	api.HandleAPIMethod(api.POST, "/chat/_new", handler.newChatSession)
	api.HandleAPIMethod(api.POST, "/chat/:session_id/_open", handler.openChatSession)
	api.HandleAPIMethod(api.POST, "/chat/:session_id/_send", handler.sendChatMessage)
	api.HandleAPIMethod(api.POST, "/chat/:session_id/_cancel", handler.cancelReplyMessage)
	api.HandleAPIMethod(api.POST, "/chat/:session_id/_close", handler.closeChatSession)
	api.HandleAPIMethod(api.GET, "/chat/:session_id/_history", handler.getChatHistoryBySession)
}
