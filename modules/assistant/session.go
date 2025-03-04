/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	log "github.com/cihub/seelog"
	_ "github.com/tmc/langchaingo/llms/ollama"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/api/websocket"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"net/http"
	"sync"
)

type Session struct {
	orm.ORMObjectBase
	Status               string `config:"status" json:"status,omitempty" elastic_mapping:"status:{type:keyword}"`
	Title                string `config:"title" json:"title,omitempty" elastic_mapping:"title:{type:keyword}"`
	Summary              string `config:"summary" json:"summary,omitempty" elastic_mapping:"summary:{type:keyword}"`
	ManuallyRenamedTitle bool   `config:"manually_renamed_title" json:"manually_renamed_title,omitempty" elastic_mapping:"manually_renamed_title:{type:boolean}"`
}

func (h APIHandler) getChatSessions(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	q := orm.Query{}
	q.From = h.GetIntOrDefault(req, "from", 0)
	q.Size = h.GetIntOrDefault(req, "size", 20)
	q.AddSort("updated", orm.DESC)
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

	var request MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		//h.WriteError(w, err.Error(), http.StatusInternalServerError)
		//TODO, should panic after v0.2
	}

	obj := Session{
		Status: "active",
	}

	if request.Message != "" {
		obj.Title = util.SubString(request.Message, 0, 50)
	}

	err := orm.Create(nil, &obj)
	if err != nil {
		h.Error(w, err)
		return
	}

	var firstMessage *ChatMessage
	//save first message to history
	if request.Message != "" {
		firstMessage, err = h.handleMessage(req, obj.ID, request.Message)
		if err != nil {
			h.Error(w, err)
			return
		}
	}

	err = h.WriteJSON(w, util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"payload": firstMessage,
		"_source": obj,
	}, 200)
	if err != nil {
		h.Error(w, err)
		return
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

func getChatHistoryBySessionInternal(sessionID string) ([]ChatMessage, error) {
	q := orm.Query{}
	q.Conds = orm.And(orm.Eq("session_id", sessionID))
	q.From = 0
	q.Size = 5
	q.AddSort("created", orm.DESC)
	docs := []ChatMessage{}
	err, _ := orm.SearchWithJSONMapper(&docs, &q)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (h APIHandler) getChatHistoryBySession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	q := orm.Query{}
	q.Conds = orm.And(orm.Eq("session_id", ps.MustGetParameter("session_id")))
	q.From = h.GetIntOrDefault(req, "from", 0)
	q.Size = h.GetIntOrDefault(req, "size", 20)
	q.AddSort("updated", orm.ASC)

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

var inflightMessages = sync.Map{}

type MessageTask struct {
	TaskID      string
	WebsocketID string
}

func init() {
	websocket.RegisterDisconnectCallback(func(websocketID string) {
		log.Debugf("stop task for websocket: %v after websocket disconnected", websocketID)
		inflightMessages.Range(func(key, value any) bool {
			v1, ok := value.(MessageTask)
			if ok {
				if v1.WebsocketID == websocketID {
					log.Info("stop task:", v1)
					task.StopTask(v1.TaskID)
				}
			}
			return true
		})
	})
}

func stopSessionTask(sessionID string) {
	v, ok := inflightMessages.Load(sessionID)
	if ok {
		v1, ok := v.(MessageTask)
		if ok {
			log.Debug("stop task:", v1)
			task.StopTask(v1.TaskID)
		}
	} else {
		log.Warn("task id not found for session:", sessionID)
	}
}

func (h APIHandler) cancelReplyMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	sessionID := ps.MustGetParameter("session_id")
	stopSessionTask(sessionID)
	err := h.WriteAckOKJSON(w)
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

	obj, err := h.handleMessage(req, sessionID, request.Message)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := []util.MapStr{util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"_source": obj,
	}}

	err = h.WriteJSON(w, response, 200)
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
