/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"infini.sh/coco/plugins/security/core"
	"infini.sh/framework/core/api"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/chat/_history", core.RequireLogin(handler.getChatSessions))
	api.HandleUIMethod(api.POST, "/chat/_new", core.RequireLogin(handler.newChatSession))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_open", core.RequireLogin(handler.openChatSession))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_send", core.RequireLogin(handler.sendChatMessage))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_cancel", core.RequireLogin(handler.cancelReplyMessage))
	api.HandleUIMethod(api.POST, "/chat/:session_id/_close", core.RequireLogin(handler.closeChatSession))
	api.HandleUIMethod(api.GET, "/chat/:session_id/_history", core.RequireLogin(handler.getChatHistoryBySession))
}
