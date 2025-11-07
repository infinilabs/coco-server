/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"infini.sh/coco/core"
	"net/http"
	"sync"

	log "github.com/cihub/seelog"
	_ "github.com/tmc/langchaingo/llms/ollama"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

func (h APIHandler) getSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")

	obj := core.Session{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
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

	obj := core.Session{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	//deleting related documents
	builder, err := orm.NewQueryBuilderFromRequest(req)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	builder.Filter(orm.TermQuery("session_id", id))

	ctx1 := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx1, &core.ChatMessage{})

	_, err = orm.DeleteByQuery(ctx1, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx.Refresh = orm.WaitForRefresh
	err = orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteDeletedOKJSON(w, obj.ID)
}

func (h APIHandler) updateSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")
	obj := core.Session{}
	var err error
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	previousObj := core.Session{}
	previousObj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &previousObj)
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
	ctx.Refresh = orm.WaitForRefresh

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

	builder, err := orm.NewQueryBuilderFromRequest(req, "title", "summary")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if builder.SizeVal() == 0 {
		builder.Size(20)
	}

	builder.SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	builder.Not(orm.TermQuery("visible", false))

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.Session{})

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Payload.([]byte))
	if err != nil {
		h.Error(w, err)
	}

}

func (h APIHandler) createChatSession(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := h.GetParameterOrDefault(r, "assistant_id", DefaultAssistantID)

	assistant, exists, err := common.GetAssistant(r, id)
	if !exists || err != nil {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	//launch the LLM task
	//streaming output result to HTTP client

	var request core.MessageRequest
	if err := h.DecodeJSON(r, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session, err, reqMsg, finalResult := CreateAndSaveNewChatMessage(r, id, &request, true)
	if err != nil {
		h.Error(w, err)
		return
	}

	//return for create session only request
	if reqMsg == nil {
		h.WriteJSON(w, finalResult, 200)
		return
	}

	ctx := r.Context()
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.Error(w, errors.New("http.Flusher not supported"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	_ = enc.Encode(finalResult)
	flusher.Flush()

	params, err := h.getRAGContext(r, assistant)
	if err != nil {
		h.Error(w, err)
		return
	}
	params.SessionID = session.ID
	// Create a context with cancel to handle the message asynchronously
	ctx, cancel := context.WithCancel(r.Context())
	streamSender := &HTTPStreamSender{
		Enc:     enc,
		Flusher: flusher,
		Ctx:     ctx, // assuming this is in an HTTP handler
	}
	replyMsgTaskID := getReplyMessageTaskID(session.ID, reqMsg.ID)
	inflightMessages.Store(replyMsgTaskID, MessageTask{
		SessionID:  session.ID,
		CancelFunc: cancel,
	})
	_ = processMessageAsync(orm.NewContextWithParent(ctx), reqMsg, params, streamSender)
}

func CreateAndSaveNewChatMessage(request *http.Request, assistantID string, req *core.MessageRequest, visible bool) (core.Session, error, *core.ChatMessage, util.MapStr) {
	ctx := orm.NewContextWithParent(request.Context())
	return InternalCreateAndSaveNewChatMessage(ctx, assistantID, req, visible)
}

func InternalCreateAndSaveNewChatMessage(ctx *orm.Context, assistantID string, req *core.MessageRequest, visible bool) (core.Session, error, *core.ChatMessage, util.MapStr) {

	//if !rate.GetRateLimiterPerSecond("assistant_new_chat", clientIdentity, 10).Allow() {
	//	panic("too many requests")
	//}
	ctx.Refresh = orm.WaitForRefresh

	obj := core.Session{
		Status:  "active",
		Visible: visible,
	}

	if req != nil && req.Message != "" {
		obj.Title = util.SubString(req.Message, 0, 50)
	}

	//save session
	err := orm.Create(ctx, &obj)
	if err != nil {
		return core.Session{}, err, nil, nil
	}

	result := util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"_source": obj,
	}

	var firstMessage *core.ChatMessage
	//save first message to history
	if req != nil && !req.IsEmpty() {
		firstMessage, err = saveRequestMessage(ctx, obj.ID, assistantID, req)
		if err != nil {
			return core.Session{}, err, nil, nil
		}
		result["payload"] = firstMessage
	}

	return obj, err, firstMessage, result
}

func (h *APIHandler) askAssistant(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	assistant, exists, err := common.GetAssistant(r, id)
	if !exists || err != nil {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	//launch the LLM task
	//streaming output result to HTTP client

	var request core.MessageRequest
	if err := h.DecodeJSON(r, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if request.IsEmpty() {
		h.WriteError(w, "message is empty", 400)
		return
	}

	session, err, reqMsg, finalResult := CreateAndSaveNewChatMessage(r, id, &request, false)
	if err != nil || reqMsg == nil {
		h.Error(w, err)
		return
	}

	ctx := r.Context()
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.Error(w, errors.New("http.Flusher not supported"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	_ = enc.Encode(finalResult)
	flusher.Flush()

	params, err := h.getRAGContext(r, assistant)
	if err != nil {
		h.Error(w, err)
		return
	}
	params.SessionID = session.ID
	streamSender := &HTTPStreamSender{
		Enc:     enc,
		Flusher: flusher,
		Ctx:     r.Context(), // assuming this is in an HTTP handler
	}
	_ = processMessageAsync(orm.NewContextWithParent(ctx), reqMsg, params, streamSender)

}

func saveRequestMessage(ctx *orm.Context, sessionID, assistantID string, req *core.MessageRequest) (*core.ChatMessage, error) {

	if sessionID == "" || assistantID == "" || req.IsEmpty() {
		panic("invalid chat message")
	}

	msg := &core.ChatMessage{
		SessionID:   sessionID,
		AssistantID: assistantID,
		MessageType: core.MessageTypeUser,
		Message:     req.Message,
		Attachments: req.Attachments,
	}
	msg.ID = util.GetUUID()

	msg.Parameters = util.MapStr{}

	if err := orm.Create(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (h APIHandler) openChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")

	obj := core.Session{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	if !obj.Visible {
		obj.Status = "active"
		obj.Visible = true

		ctx.Refresh = orm.ImmediatelyRefresh

		err = orm.Update(ctx, &obj)
		if err != nil {
			h.Error(w, err)
			return
		}
	}

	h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)

}

func getChatHistoryBySessionInternal(sessionID string, size int) ([]core.ChatMessage, error) {
	q := orm.Query{}
	q.Conds = orm.And(orm.Eq("session_id", sessionID))
	q.From = 0
	q.Size = size
	q.AddSort("created", orm.DESC)
	docs := []core.ChatMessage{}
	err, _ := orm.SearchWithJSONMapper(&docs, &q)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (h APIHandler) getChatHistoryBySession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	builder, err := orm.NewQueryBuilderFromRequest(req, "message")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if builder.SizeVal() == 0 {
		builder.Size(20)
	}

	builder.SortBy(orm.Sort{Field: "created", SortType: orm.ASC})
	builder.Must(orm.TermQuery("session_id", ps.MustGetParameter("session_id")))

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.ChatMessage{})

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Payload.([]byte))
	if err != nil {
		h.Error(w, err)
	}
}

var inflightMessages = sync.Map{}

type MessageTask struct {
	SessionID string
	// Deprecated
	TaskID string

	CancelFunc func()
}

func stopMessageReplyTask(taskID string) {
	v, ok := inflightMessages.Load(taskID)
	if ok {
		v1, ok := v.(MessageTask)
		if ok {
			log.Debug("stop task:", v1)
			if v1.TaskID != "" {
				task.StopTask(v1.TaskID)
			} else if v1.CancelFunc != nil {
				v1.CancelFunc()
			}
		}
	} else {
		_ = log.Warnf("task id [%s] was not found", taskID)
	}
}

func (h APIHandler) cancelReplyMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	sessionID := ps.MustGetParameter("session_id")
	messageID := h.GetParameterOrDefault(req, "message_id", "")
	taskID := getReplyMessageTaskID(sessionID, messageID)
	stopMessageReplyTask(taskID)
	h.WriteAckOKJSON(w)
}

func (h APIHandler) sendChatMessageV2(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sessionID := ps.MustGetParameter("session_id")

	ormCtx := orm.NewContextWithParent(r.Context())
	ormCtx.Refresh = orm.WaitForRefresh

	id := h.GetParameterOrDefault(r, "assistant_id", DefaultAssistantID)

	assistant, exists, err := common.GetAssistant(r, id)
	if !exists || err != nil {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	//launch the LLM task
	//streaming output result to HTTP client

	var request core.MessageRequest
	if err := h.DecodeJSON(r, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reqMsg, err := saveRequestMessage(ormCtx, sessionID, id, &request)
	if err != nil {
		h.Error(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.Error(w, errors.New("http.Flusher not supported"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	response := []util.MapStr{util.MapStr{
		"_id":     reqMsg.ID,
		"result":  "created",
		"_source": reqMsg,
	}}

	enc := json.NewEncoder(w)
	_ = enc.Encode(response)
	flusher.Flush()

	params, err := h.getRAGContext(r, assistant)
	if err != nil {
		h.Error(w, err)
		return
	}
	params.SessionID = sessionID
	// Create a context with cancel to handle the message asynchronously
	ctx, cancel := context.WithCancel(r.Context())
	streamSender := &HTTPStreamSender{
		Enc:     enc,
		Flusher: flusher,
		Ctx:     ctx, // assuming this is in an HTTP handler
	}
	replyMsgTaskID := getReplyMessageTaskID(sessionID, reqMsg.ID)
	inflightMessages.Store(replyMsgTaskID, MessageTask{
		SessionID:  sessionID,
		CancelFunc: cancel,
	})
	_ = processMessageAsync(orm.NewContextWithParent(ctx), reqMsg, params, streamSender)

}

func getReplyMessageTaskID(sessionID, messageID string) string {
	if messageID == "" {
		return sessionID
	}
	return fmt.Sprintf("%s_%s", sessionID, messageID)
}

func (h APIHandler) closeChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	id := ps.MustGetParameter("session_id")
	obj := core.Session{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	//obj.Status = "closed"
	//err = orm.Update(&orm.Context{
	//	Refresh: "wait_for",
	//}, &obj)
	//if err != nil {
	//	h.Error(w, err)
	//	return
	//}

	h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)

}
