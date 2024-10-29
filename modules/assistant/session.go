/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"context"
	httprouter "infini.sh/framework/core/api/router"
	"net/http"
	"time"
)

// Session represents a single chat session
type Session struct {
	SessionID string    `json:"session_id"`
	Created   *time.Time `json:"created"`
	Updated   *time.Time `json:"updated,omitempty"`
	Status    string    `json:"status"`
	Title     string    `json:"title,omitempty"`
	Summary   string    `json:"summary,omitempty"`
}

// ChatSessions represents the response for retrieving all chat sessions
type Sessions struct {
	Sessions []Session `json:"sessions"`
}


// MessageRequest represents the request payload for sending a message
type MessageRequest struct {
	Message string `json:"message"`
}

// ChatMessage represents an individual message within a chat session history
type ChatMessage struct {
	SessionID  string     `json:"session_id"`
	SequenceID string        `json:"sequence_id"`
	Created    *time.Time `json:"created"`
	Message    string     `json:"message"`
	Response   string     `json:"response"`
}

// ChatSessionHistoryResponse represents the response for retrieving chat session history
type ChatSessionHistoryResponse struct {
	Messages []ChatMessage `json:"messages"`
}

func (h APIHandler) getChatSessions(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	t:= time.Now()
	response := Sessions{
		Sessions: []Session{
			{
				SessionID: "1",
				Created:  &t,
				Status:   "active",
				Title:    "Chat SessionID 1",
				Summary:  "This is a summary of the chat session",
			},
		},
	}

	err := h.WriteJSON(w, response, 200)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) newChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	t:= time.Now()
	response := Session{
		SessionID: "1",
		Created:  &t,
		Status:   "active",
	}

	err := h.WriteJSON(w, response, 200)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) openChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	sessionID:=ps.MustGetParameter("session_id")

	t:= time.Now()
	response := Session{
		SessionID: sessionID,
		Updated:  &t,
		Status:   "active",
	}

	err := h.WriteJSON(w, response, 200)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) getChatHistoryBySession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	sessionID:=ps.MustGetParameter("session_id")

	t:= time.Now()
	response := ChatSessionHistoryResponse{
		Messages: []ChatMessage{
			{
				SequenceID: sessionID,
				Created:    &t,
				Message:    "Hello",
				Response:   "Hi",
			},
		},
	}

	err := h.WriteJSON(w, response, 200)
	if err != nil {
		h.Error(w, err)
	}
}


type PromoteRequest struct {
	Prompt string  `json:"prompt"`
}

func (h APIHandler) sendChatMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	sessionID:=ps.MustGetParameter("session_id")

	var request MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	topk:=5

	rst, err := useRetriaver(getStore(), request.Message, topk)
	if err != nil {
		h.WriteError(w,err.Error(),500)
		return
	}

	llm:=getOllamaMistral()
	answer, err := GetAnswer(context.Background(), llm, rst, request.Message)
	if err != nil {
		h.WriteError(w,err.Error(),500)
		return
	}

	rst1, err1:= Translate(getOllamaLlama2(), answer)
	if err1 != nil {
		h.WriteError(w,err.Error(),500)
		return
	}

	// websocket.SendPrivateMessage(ps.MustGetParameter("session_id"),rst1)

	t:= time.Now()
	response:=ChatMessage{
		SequenceID: sessionID,
		Created:    &t,
		Message:    request.Message,
		Response:   rst1,
	}

	err = h.WriteJSON(w, response, 200)
	if err != nil {
		h.Error(w, err)
	}
}


func (h APIHandler) closeChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	sessionID :=ps.MustGetParameter("session_id")

	t:= time.Now()
	response := Session{
		SessionID: sessionID,
		Created:   &t,
		Status:    "closed",
	}

	err := h.WriteJSON(w, response, 200)
	if err != nil {
		h.Error(w, err)
	}
}