/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	log "github.com/cihub/seelog"
	_ "github.com/tmc/langchaingo/llms/ollama"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/api/websocket"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"net/http"
	"sync"
)

func (h APIHandler) getSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")

	obj := common.Session{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
}

func (h APIHandler) deleteSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")

	obj := common.Session{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	//deleting related documents
	query := util.MapStr{
		"query": util.MapStr{
			"term": util.MapStr{
				"session_id": id,
			},
		},
	}
	err = orm.DeleteBy(&common.ChatMessage{}, util.MustToJSONBytes(query))
	if err != nil {
		log.Errorf("delete related documents with chat session [%s], error: %v", id, err)
	}

	err = orm.Delete(nil, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "deleted",
	}, 200)
}

func (h APIHandler) updateSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")
	obj := common.Session{}
	var err error
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	previousObj := common.Session{}
	previousObj.ID = id
	exists, err := orm.Get(&previousObj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	var changed = false
	if obj.Context != nil {
		previousObj.Context = obj.Context
		changed = true
	}

	if obj.Title != "" {
		previousObj.Title = obj.Title
		previousObj.ManuallyRenamedTitle = true
		changed = true
	}

	if !changed {
		h.WriteError(w, "no changes found", 400)
		return
	}

	//protect
	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
	err = orm.Update(ctx, &previousObj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    id,
		"result": "updated",
	}, 200)
}

func (h APIHandler) getChatSessions(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	q := orm.Query{}
	q.From = h.GetIntOrDefault(req, "from", 0)
	q.Size = h.GetIntOrDefault(req, "size", 20)
	query := h.GetParameterOrDefault(req, "query", "")
	if query != "" {
		q.Conds = orm.Or(orm.Prefix("title", query), orm.QueryString("*", query))
	}

	q.AddSort("updated", orm.DESC)
	err, res := orm.Search(&common.Session{}, &q)
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

	var request common.MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		//h.WriteError(w, err.Error(), http.StatusInternalServerError)
		//TODO, should panic after v0.2
	}

	obj := common.Session{
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

	assistantID := h.GetParameterOrDefault(req, "assistant_id", DefaultAssistantID)
	var firstMessage *common.ChatMessage
	//save first message to history
	if request.Message != "" {
		firstMessage, err = h.handleMessage(req, obj.ID, assistantID, request.Message)
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

	obj := common.Session{}
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

func getChatHistoryBySessionInternal(sessionID string, size int) ([]common.ChatMessage, error) {
	q := orm.Query{}
	q.Conds = orm.And(orm.Eq("session_id", sessionID))
	q.From = 0
	q.Size = size
	q.AddSort("created", orm.DESC)
	docs := []common.ChatMessage{}
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

	err, res := orm.Search(&common.ChatMessage{}, &q)
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
	SessionID   string
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
	var request common.MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		log.Error(err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	assistantID := h.GetParameterOrDefault(req, "assistant_id", DefaultAssistantID)

	obj, err := h.handleMessage(req, sessionID, assistantID, request.Message)
	if err != nil {
		log.Error(err)
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
	obj := common.Session{}
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
