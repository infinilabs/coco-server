/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	const defaultTimeout = 10 * time.Minute
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

type searchResultContext struct {
	// UserMessage is persisted to Elasticsearch as the user's chat message.
	UserMessage string
	// ModelMessage is sent to the LLM as the prompt content.
	ModelMessage string
	// FetchSource is the selected search result context.
	FetchSource interface{}
}

type searchResultPayload struct {
	Query  string          `json:"query"`
	Result json.RawMessage `json:"result"`
}

// helper function to split a search-result payload into the history message,
// model prompt, and selected search sources.
func parseSearchResultPayload(message string) (*searchResultContext, error) {
	payload, err := decodeSearchResultPayload(message)
	if err != nil {
		return nil, err
	}
	if payload.Query == "" {
		return nil, errors.New("query is empty")
	}

	fetchSource, err := decodeSearchResultFetchSource(payload.Result)
	if err != nil {
		return nil, err
	}
	return &searchResultContext{UserMessage: payload.Query, ModelMessage: message, FetchSource: fetchSource}, nil
}

// helper function to decode the search-result payload sent by search clients.
func decodeSearchResultPayload(message string) (*searchResultPayload, error) {
	rawMessage := []byte(message)
	for range 2 {
		var payload searchResultPayload
		if err := json.Unmarshal(rawMessage, &payload); err == nil {
			return &payload, nil
		}

		var encodedMessage string
		if err := json.Unmarshal(rawMessage, &encodedMessage); err != nil {
			return nil, err
		}
		rawMessage = []byte(encodedMessage)
	}
	return nil, errors.New("search result payload is too deeply encoded")
}

// helper function to extract fetch sources from the search result payload.
func decodeSearchResultFetchSource(raw json.RawMessage) (interface{}, error) {
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
	if err != nil {
		h.WriteError(w, "failed to get assistant", http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteJSON(w, util.MapStr{
			"_id":  id,
			"open": true,
		}, http.StatusOK)
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

// askAssistant handles _ask requests.
//
// The message field accepts two forms:
//  1. A search-result payload (JSON with "query" and "result" fields) — the
//     query is persisted as the user message, the full payload is sent to the
//     model, and the selected results are stored as fetch_source on the reply.
//  2. Any other plain text — used as-is for both persistence and the model prompt.
func (h *APIHandler) askAssistant(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	assistant, exists, err := service.GetAssistant(r, id)
	if err != nil {
		h.WriteError(w, "failed to get assistant", http.StatusInternalServerError)
		return
	}
	if !exists {
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
	searchCtx, _ := parseSearchResultPayload(request.Message)
	if searchCtx != nil {
		request.Message = searchCtx.UserMessage
	}

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
	if searchCtx != nil {
		reqMsg.Message = searchCtx.ModelMessage
	}

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
	if searchCtx != nil && searchCtx.FetchSource != nil {
		replyMsg.Details = append(replyMsg.Details, core.ProcessingDetails{Order: 20, Type: common.FetchSource, Payload: searchCtx.FetchSource})
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

	rawBytes := res.Payload.([]byte)
	refined, err := refineAttachmentURLs(rawBytes)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, refined)
	if err != nil {
		h.Error(w, err)
	}
}

// helper function to expand relative attachment URLs in deep research report
// payloads to absolute URLs for frontend consumption.
// It works on the raw ES response bytes so that no fields are omitted by
// round-tripping through a typed struct.
func refineAttachmentURLs(raw []byte) ([]byte, error) {
	var response map[string]interface{}
	if err := util.FromJSONBytes(raw, &response); err != nil {
		return nil, err
	}

	appCfg := common.AppConfig()
	baseEndpoint := appCfg.ServerInfo.Endpoint
	if baseEndpoint == "" {
		return raw, nil
	}

	hits, ok := response["hits"].(map[string]interface{})
	if !ok {
		return raw, nil
	}
	hitList, ok := hits["hits"].([]interface{})
	if !ok {
		return raw, nil
	}

	for _, hit := range hitList {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		msgType, _ := source["type"].(string)
		if msgType != "assistant" {
			continue
		}

		payload, ok := source["payload"]
		if !ok || payload == nil {
			continue
		}
		p, ok := payload.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := p["attachment"].(string); !ok {
			continue
		}
		urlStr, ok := p["url"].(string)
		if !ok || !strings.HasPrefix(urlStr, "/") {
			continue
		}
		p["url"] = fmt.Sprintf("%s%s", baseEndpoint, urlStr)
	}

	return util.MustToJSONBytes(response), nil
}

func (h APIHandler) cancelReplyMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	sessionID := ps.MustGetParameter("session_id")
	messageID := h.GetParameterOrDefault(req, "message_id", "")
	log.Info("cancel reply to message: ", messageID, ", session: ", sessionID)
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
	if err != nil {
		h.WriteError(w, "failed to get assistant", http.StatusInternalServerError)
		return
	}
	if !exists {
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
