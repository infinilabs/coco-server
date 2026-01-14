/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"context"
	"encoding/json"
	"net/http"

	_ "github.com/tmc/langchaingo/llms/ollama"
	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/assistant/service"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
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

	builder, err := orm.NewQueryBuilderFromRequest(req, "title", "title.pinyin", "summary")
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
	id := h.GetParameterOrDefault(r, "assistant_id", common2.DefaultAssistantID)
	userInfo := security.MustGetUserFromRequest(r)

	assistant, exists, err := service.GetAssistant(r, id)
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
	session, err, reqMsg, finalResult := service.CreateAndSaveNewChatMessage(r, id, &request, true)
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

	params, err := common2.NewRagContext(r, assistant, session.ID)
	if err != nil {
		h.Error(w, err)
		return
	}
	// Create a context with cancel to handle the message asynchronously
	ctx, cancel := context.WithCancel(r.Context())
	streamSender := &common2.HTTPStreamSender{
		Enc:     enc,
		Flusher: flusher,
		Ctx:     ctx, // assuming this is in an HTTP handler
	}
	replyMsgTaskID := service.GetReplyMessageTaskID(session.ID, reqMsg.ID)
	service.InflightMessages.Store(replyMsgTaskID, common2.MessageTask{
		SessionID:  session.ID,
		CancelFunc: cancel,
	})
	_ = service.ProcessMessageAsync(orm.NewContextWithParent(ctx), userInfo.MustGetUserID(), reqMsg, params, streamSender)
}

func (h *APIHandler) askAssistant(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	assistant, exists, err := service.GetAssistant(r, id)
	if !exists || err != nil {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	userInfo := security.MustGetUserFromRequest(r)

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

	session, err, reqMsg, finalResult := service.CreateAndSaveNewChatMessage(r, id, &request, false)
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

	params, err := common2.NewRagContext(r, assistant, session.ID)
	if err != nil {
		h.Error(w, err)
		return
	}
	streamSender := &common2.HTTPStreamSender{
		Enc:     enc,
		Flusher: flusher,
		Ctx:     r.Context(), // assuming this is in an HTTP handler
	}
	_ = service.ProcessMessageAsync(orm.NewContextWithParent(ctx), userInfo.MustGetUserID(), reqMsg, params, streamSender)

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

func (h APIHandler) cancelReplyMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	sessionID := ps.MustGetParameter("session_id")
	messageID := h.GetParameterOrDefault(req, "message_id", "")
	taskID := service.GetReplyMessageTaskID(sessionID, messageID)
	service.StopMessageReplyTask(taskID)
	h.WriteAckOKJSON(w)
}

func (h APIHandler) sendChatMessageV2(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sessionID := ps.MustGetParameter("session_id")
	userInfo := security.MustGetUserFromRequest(r)

	ormCtx := orm.NewContextWithParent(r.Context())
	ormCtx.Refresh = orm.WaitForRefresh

	id := h.GetParameterOrDefault(r, "assistant_id", common2.DefaultAssistantID)

	assistant, exists, err := service.GetAssistant(r, id)
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

	reqMsg, err := service.SaveRequestMessage(ormCtx, sessionID, id, &request)
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

	params, err := common2.NewRagContext(r, assistant, sessionID)
	if err != nil {
		h.Error(w, err)
		return
	}
	// Create a context with cancel to handle the message asynchronously
	ctx, cancel := context.WithCancel(r.Context())
	streamSender := &common2.HTTPStreamSender{
		Enc:     enc,
		Flusher: flusher,
		Ctx:     ctx, // assuming this is in an HTTP handler
	}
	replyMsgTaskID := service.GetReplyMessageTaskID(sessionID, reqMsg.ID)
	service.InflightMessages.Store(replyMsgTaskID, common2.MessageTask{
		SessionID:  sessionID,
		CancelFunc: cancel,
	})
	_ = service.ProcessMessageAsync(orm.NewContextWithParent(ctx), userInfo.MustGetUserID(), reqMsg, params, streamSender)

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
