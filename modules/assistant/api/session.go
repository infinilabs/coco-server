/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/smallnest/langgraphgo/log"
	_ "github.com/tmc/langchaingo/llms/ollama"
	"infini.sh/coco/core"
	common2 "infini.sh/coco/modules/assistant/common"
	"infini.sh/coco/modules/assistant/service"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// helper function to choose the request timeout for an assistant.
func resolveTimeout(assistant *core.Assistant) time.Duration {
	const defaultTimeout = 5 * time.Minute
	const deepResearchDefault = 30 * time.Minute

	if assistant == nil || assistant.Type != core.AssistantTypeDeepResearch {
		return defaultTimeout
	}
	if assistant.DeepResearchConfig != nil && assistant.DeepResearchConfig.Timeout != "" {
		if d, err := time.ParseDuration(assistant.DeepResearchConfig.Timeout); err == nil && d > 0 {
			return d
		}
	}
	return deepResearchDefault
}

type askPayloadContext struct {
	// UserMessage is persisted to Elasticsearch as the user's chat message.
	UserMessage string
	// ModelMessage is sent to the LLM as the prompt content.
	ModelMessage string
	// FetchSource is the selected search result context.
	FetchSource interface{}
}

type askPayload struct {
	Query  string          `json:"query"`
	Result json.RawMessage `json:"result"`
}

// helper function to split an _ask payload into the history message, model
// prompt, and selected search sources.
func parseAskPayload(message string) (*askPayloadContext, error) {
	payload, err := decodeAskPayload(message)
	if err != nil {
		return nil, err
	}
	if payload.Query == "" {
		return nil, errors.New("query is empty")
	}

	fetchSource, err := decodeAskFetchSource(payload.Result)
	if err != nil {
		return nil, err
	}
	return &askPayloadContext{UserMessage: payload.Query, ModelMessage: message, FetchSource: fetchSource}, nil
}

// helper function to read the _ask payload shape sent by search clients.
func decodeAskPayload(message string) (*askPayload, error) {
	rawMessage := []byte(message)
	for range 2 {
		var payload askPayload
		if err := json.Unmarshal(rawMessage, &payload); err == nil {
			return &payload, nil
		}

		var encodedMessage string
		if err := json.Unmarshal(rawMessage, &encodedMessage); err != nil {
			return nil, err
		}
		rawMessage = []byte(encodedMessage)
	}
	return nil, errors.New("ask payload is too deeply encoded")
}

// helper function to align selected search results with assistant reply details.
func decodeAskFetchSource(raw json.RawMessage) (interface{}, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var result interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if resultMap, ok := result.(map[string]interface{}); ok {
		if hits, ok := resultMap["hits"]; ok {
			return hits, nil
		}
	}
	return result, nil
}

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
	ctx := context.WithoutCancel(r.Context())
	ctx, cancel := context.WithTimeout(ctx, resolveTimeout(assistant))

	replyMsg := service.CreateAssistantReplyMessage(params.SessionID, reqMsg.AssistantID, reqMsg.ID)

	streamSender := &common2.HTTPStreamSender{
		ReqMsg:   reqMsg,
		ReplyMsg: replyMsg,

		Enc:     enc,
		Flusher: flusher,
		Ctx:     ctx, // assuming this is in an HTTP handler
	}

	replyMsgTaskID := service.GetReplyMessageTaskID(session.ID, reqMsg.ID)
	service.InflightMessages.Store(replyMsgTaskID, common2.MessageTask{
		SessionID:  session.ID,
		CancelFunc: cancel,
	})

	_ = service.ProcessMessageAsync(ctx, userInfo.MustGetUserID(), reqMsg, replyMsg, params, streamSender)
}

// askAssistant handles _ask requests from search surfaces.
//
// The incoming user message carries the query, selected search results, and
// attachments as the prompt context. For chat history, this route persists only
// the query as the user message, then records the selected results on the
// assistant reply so the history reads like a search-backed answer even though
// the search already happened before _ask was called.
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
	askContext, err := parseAskPayload(request.Message)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	request.Message = askContext.UserMessage

	session, err, reqMsg, finalResult := service.CreateAndSaveNewChatMessage(r, id, &request, false)
	if err != nil || reqMsg == nil {
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

	enc := json.NewEncoder(w)
	_ = enc.Encode(finalResult)
	flusher.Flush()

	// Keep the model prompt aligned with the user's active search context.
	reqMsg.Message = askContext.ModelMessage

	params, err := common2.NewRagContext(r, assistant, session.ID)
	if err != nil {
		h.Error(w, err)
		return
	}

	ctx := context.WithoutCancel(r.Context())
	ctx, cancel := context.WithTimeout(ctx, resolveTimeout(assistant))

	sessionID := reqMsg.SessionID
	replyMsgTaskID := service.GetReplyMessageTaskID(sessionID, reqMsg.ID)
	service.InflightMessages.Store(replyMsgTaskID, common2.MessageTask{
		SessionID:  sessionID,
		CancelFunc: cancel,
	})
	replyMsg := service.CreateAssistantReplyMessage(params.SessionID, reqMsg.AssistantID, reqMsg.ID)
	if askContext.FetchSource != nil {
		replyMsg.Details = append(replyMsg.Details, core.ProcessingDetails{Order: 20, Type: common.FetchSource, Payload: askContext.FetchSource})
	}

	streamSender := &common2.HTTPStreamSender{
		ReqMsg:   reqMsg,
		ReplyMsg: replyMsg,
		Enc:      enc,
		Flusher:  flusher,
		Ctx:      ctx, // Use the timeout context for consistency
	}

	_ = service.ProcessMessageAsync(ctx, userInfo.MustGetUserID(), reqMsg, replyMsg, params, streamSender)

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
	lang := h.GetParameterOrDefault(req, "lang", "")
	log.Info("cancel reply to message: ", messageID, ", session: ", sessionID)
	taskID := service.GetReplyMessageTaskID(sessionID, messageID)
	service.StopMessageReplyTask(taskID, lang)
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
	ctx := context.WithoutCancel(r.Context())
	ctx, cancel := context.WithTimeout(ctx, resolveTimeout(assistant))

	replyMsg := service.CreateAssistantReplyMessage(params.SessionID, reqMsg.AssistantID, reqMsg.ID)

	streamSender := &common2.HTTPStreamSender{
		ReqMsg:   reqMsg,
		ReplyMsg: replyMsg,
		Enc:      enc,
		Flusher:  flusher,
		Ctx:      ctx, // assuming this is in an HTTP handler
	}
	replyMsgTaskID := service.GetReplyMessageTaskID(sessionID, reqMsg.ID)
	service.InflightMessages.Store(replyMsgTaskID, common2.MessageTask{
		SessionID:  sessionID,
		CancelFunc: cancel,
	})
	_ = service.ProcessMessageAsync(ctx, userInfo.MustGetUserID(), reqMsg, replyMsg, params, streamSender)

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
