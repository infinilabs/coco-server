/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"errors"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/kv"
	"net/http"
)

type APIHandler struct {
	api.Handler
}

func init() {
	handler := APIHandler{}

	api.HandleUIMethod(api.GET, "/chat/_history", handler.getChatSessions, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/chat/_new", handler.newChatSession, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/chat/:session_id/_open", handler.openChatSession, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/chat/:session_id/_send", handler.sendChatMessage, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/chat/:session_id/_cancel", handler.cancelReplyMessage, api.RequireLogin())
	api.HandleUIMethod(api.POST, "/chat/:session_id/_close", handler.closeChatSession, api.RequireLogin())
	api.HandleUIMethod(api.GET, "/chat/:session_id/_history", handler.getChatHistoryBySession, api.RequireLogin())

}

func (h APIHandler) GetUserWebsocketID(req *http.Request) (string, error) {
	//get websocket by user's id
	claims, err := core.ValidateLogin(req)
	if err != nil {
		return "", err
	}
	if claims != nil {
		if claims.UserId != "" {
			v, err := kv.GetValue(common.WEBSOCKET_USER_SESSION, []byte(claims.UserId))
			if err != nil {
				return "", err
			}
			return string(v), nil
		}
	}
	return "", errors.New("not found")
}
