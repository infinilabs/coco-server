/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
)

type Session struct {
	orm.ORMObjectBase
	Status  string `config:"status" json:"status,omitempty" elastic_mapping:"status:{type:keyword}"`
	Title   string `config:"title" json:"title,omitempty" elastic_mapping:"title:{type:keyword}"`
	Summary string `config:"summary" json:"summary,omitempty" elastic_mapping:"summary:{type:keyword}"`
}

type MessageRequest struct {
	Message string `json:"message"`
}

const MessageTypeUser string = "user"
const MessageTypeAssistant string = "assistant"
const MessageTypeSystem string = "system"

type ChatMessage struct {
	orm.ORMObjectBase
	MessageType string `json:"type"` // user, assistant, system
	SessionID   string `json:"session_id"`
	From        string `json:"from"`
	To          string `json:"to,omitempty"`
	Message     string `config:"message" json:"message,omitempty" elastic_mapping:"message:{type:keyword}"`
}

func (h APIHandler) getChatSessions(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	q := orm.Query{}
	q.From = h.GetIntOrDefault(req, "from", 0)
	q.Size = h.GetIntOrDefault(req, "size", 20)

	err, res := orm.Search(&Session{}, &q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) newChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	obj := Session{
		Status: "active",
	}

	err := orm.Create(nil, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.WriteJSON(w, util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"_source": obj,
	}, 200)

	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) openChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")

	obj := Session{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	obj.Status = "active"
	err = orm.Update(nil, &obj)
	if err != nil {
		h.Error(w, err)
		return
	}

	err = h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) getChatHistoryBySession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	q := orm.Query{}
	q.Conds = orm.And(orm.Eq("session_id", ps.MustGetParameter("session_id")))
	q.From = h.GetIntOrDefault(req, "from", 0)
	q.Size = h.GetIntOrDefault(req, "size", 20)

	err, res := orm.Search(&ChatMessage{}, &q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) sendChatMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	sessionID := ps.MustGetParameter("session_id")
	var request MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	obj := ChatMessage{
		SessionID: sessionID,
		MessageType: MessageTypeUser,
		Message:   request.Message,
	}

	err := orm.Create(nil, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.WriteJSON(w, util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"_source": obj,
	}, 200)

	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) closeChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	id := ps.MustGetParameter("session_id")
	obj := Session{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	obj.Status = "closed"
	err = orm.Update(nil, &obj)
	if err != nil {
		h.Error(w, err)
		return
	}

	err = h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
	if err != nil {
		h.Error(w, err)
	}

}
