/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package chat

import (
	"context"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/api/websocket"
	"net/http"
)

type APIHandler struct {
	api.Handler
}


func init() {
	handler := APIHandler{}

	api.HandleAPIMethod(api.POST, "/chat/_history", handler.sendChatMessage)
	api.HandleAPIMethod(api.POST, "/chat/_new", handler.newChatSession)
	api.HandleAPIMethod(api.POST, "/chat/:id/_open", handler.sendChatMessage)
	api.HandleAPIMethod(api.POST, "/chat/:id/_send", handler.sendChatMessage)
	api.HandleAPIMethod(api.POST, "/chat/:id/_clear", handler.sendChatMessage)
	api.HandleAPIMethod(api.POST, "/chat/:id/_close", handler.sendChatMessage)
	api.HandleAPIMethod(api.POST, "/chat/:id/_history", handler.sendChatMessage)
}


type PromoteRequest struct {
	Prompt string  `json:"prompt"`
}

func (h APIHandler) sendChatMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	topk:=5

	prompt:=PromoteRequest{}
	if err:=h.DecodeJSON(req, &prompt);err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	rst, err := useRetriaver(getStore(), prompt.Prompt, topk)
	if err != nil {
		h.WriteError(w,err.Error(),500)
		return
	}

	llm:=getOllamaMistral()
	answer, err := GetAnswer(context.Background(), llm, rst, prompt.Prompt)
	if err != nil {
		h.WriteError(w,err.Error(),500)
		return
	}

	rst1, err1:= Translate(getOllamaLlama2(), answer)
	if err1 != nil {
		h.WriteError(w,err.Error(),500)
		return
	}

	websocket.SendPrivateMessage(ps.MustGetParameter("id"),rst1)

	h.WriteJSON(w,rst1,200)
}

func (h APIHandler) newChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {



}
