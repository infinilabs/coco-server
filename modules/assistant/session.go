/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	_ "github.com/tmc/langchaingo/llms/ollama"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/api/websocket"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

func (h APIHandler) getSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")

	obj := common.Session{}
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

	obj := common.Session{}
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
	orm.WithModel(ctx, &common.Session{})

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

// TODO to be removed
func (h APIHandler) newChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	assistantID := h.GetParameterOrDefault(req, "assistant_id", DefaultAssistantID)
	var request common.MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, err, firstMessage, finalResult := CreateAndSaveNewChatMessage(assistantID, &request, true)
	if err != nil {
		h.Error(w, err)
		return
	}

	//try to handle the message request
	if firstMessage != nil {
		err = h.handleMessage(w, req, session.ID, assistantID, firstMessage)
		if err != nil {
			h.Error(w, err)
			return
		}
	}

	h.WriteJSON(w, finalResult, 200)

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

	var request common.MessageRequest
	if err := h.DecodeJSON(r, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	session, err, reqMsg, finalResult := CreateAndSaveNewChatMessage(id, &request, true)
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
		CreatedAt:  time.Now(),
	})

	// Process message asynchronously and cleanup on completion
	go func() {
		defer func() {
			// Always cleanup the task when processing completes (success or failure)
			inflightMessages.Delete(replyMsgTaskID)
		}()
		_ = h.processMessageAsync(ctx, reqMsg, params, streamSender)
	}()
}

func CreateAndSaveNewChatMessage(assistantID string, req *common.MessageRequest, visible bool) (common.Session, error, *common.ChatMessage, util.MapStr) {

	//if !rate.GetRateLimiterPerSecond("assistant_new_chat", clientIdentity, 10).Allow() {
	//	panic("too many requests")
	//}

	obj := common.Session{
		Status:  "active",
		Visible: visible,
	}

	if req != nil && req.Message != "" {
		obj.Title = util.SubString(req.Message, 0, 50)
	}

	//save session
	err := orm.Create(nil, &obj)
	if err != nil {
		return common.Session{}, err, nil, nil
	}

	result := util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"_source": obj,
	}

	var firstMessage *common.ChatMessage
	//save first message to history
	if req != nil && !req.IsEmpty() {
		firstMessage, err = saveRequestMessage(obj.ID, assistantID, req)
		if err != nil {
			return common.Session{}, err, nil, nil
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

	var request common.MessageRequest
	if err := h.DecodeJSON(r, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if request.IsEmpty() {
		h.WriteError(w, "message is empty", 400)
		return
	}

	session, err, reqMsg, finalResult := CreateAndSaveNewChatMessage(id, &request, false)
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
		CreatedAt:  time.Now(),
	})

	// Process message asynchronously and cleanup on completion
	go func() {
		defer func() {
			// Always cleanup the task when processing completes (success or failure)
			inflightMessages.Delete(replyMsgTaskID)
		}()
		_ = h.processMessageAsync(ctx, reqMsg, params, streamSender)
	}()

}

func saveRequestMessage(sessionID, assistantID string, req *common.MessageRequest) (*common.ChatMessage, error) {

	if sessionID == "" || assistantID == "" || req.IsEmpty() {
		panic("invalid chat message")
	}

	msg := &common.ChatMessage{
		SessionID:   sessionID,
		AssistantID: assistantID,
		MessageType: common.MessageTypeUser,
		Message:     req.Message,
		Attachments: req.Attachments,
	}
	msg.ID = util.GetUUID()

	msg.Parameters = util.MapStr{}

	if err := orm.Create(nil, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (h APIHandler) handleMessage(w http.ResponseWriter, req *http.Request, sessionID, assistantID string, reqMsg *common.ChatMessage) error {

	//TODO, check if session and assistant exists

	assistant, _, err := common.GetAssistant(req, assistantID)
	if err != nil {
		return fmt.Errorf("failed to get assistant with id [%v]: %w", assistantID, err)
	}
	if assistant == nil {
		return fmt.Errorf("assistant [%s] is not found", assistantID)
	}
	if !assistant.Enabled {
		return fmt.Errorf("assistant [%s] is not enabled", assistant.Name)
	}

	if wsID, err := h.GetUserWebsocketID(req); err == nil && wsID != "" {
		params, err := h.getRAGContext(req, assistant)
		if err != nil {
			return err
		}

		params.SessionID = sessionID

		h.launchBackgroundTask(reqMsg, params, wsID)
		return nil
	} else {
		err := errors.Errorf("No websocket [%v] for session: %v", wsID, sessionID)
		log.Error(err)
		panic(err)
		//h.processMessageAsStreaming(req, sessionID, assistantID, message)
	}

	return nil
}

func (h APIHandler) openChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")

	obj := common.Session{}
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

func getChatHistoryBySessionInternal(sessionID string, size int) ([]common.ChatMessage, error) {
	builder := orm.NewQuery()
	builder.Must(orm.TermQuery("session_id", sessionID))
	builder.From(0).Size(size)
	builder.SortBy(orm.Sort{Field: "created", SortType: orm.DESC})

	// Use projection to exclude heavy fields for performance
	// Only include essential fields for chat history listing
	builder.Include("id", "created", "updated", "type", "session_id", "from", "to", "assistant_id", "reply_to_message", "up_vote", "down_vote")
	// Exclude heavy fields: message content, attachments, details, parameters
	builder.Exclude("message", "attachments", "details", "parameters")

	ctx := orm.NewContext()
	orm.WithModel(ctx, &common.ChatMessage{})

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, err
	}

	docs := []common.ChatMessage{}
	err = util.FromJSONBytes(res.Payload.([]byte), &docs)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (h APIHandler) getChatHistoryBySession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	builder := orm.NewQuery()
	builder.Must(orm.TermQuery("session_id", ps.MustGetParameter("session_id")))
	builder.From(h.GetIntOrDefault(req, "from", 0))
	builder.Size(h.GetIntOrDefault(req, "size", 20))
	builder.SortBy(orm.Sort{Field: "created", SortType: orm.ASC})

	// Check if full content is requested via query parameter
	fullContent := h.GetParameterOrDefault(req, "full_content", "false") == "true"

	if !fullContent {
		// Use projection for lighter queries by default
		// Include only essential fields for chat message listing
		builder.Include("id", "created", "updated", "type", "session_id", "from", "to", "assistant_id", "reply_to_message", "up_vote", "down_vote")
		// Exclude heavy content fields
		builder.Exclude("message", "attachments", "details", "parameters")
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &common.ChatMessage{})

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

// MessageTask represents an active message processing task
type MessageTask struct {
	SessionID   string
	TaskID      string // Deprecated
	WebsocketID string // Deprecated
	CancelFunc  func()
	CreatedAt   time.Time // For TTL-based cleanup
}

// TaskManager handles cleanup of inflight message tasks
type TaskManager struct {
	cleanupInterval time.Duration
	taskTTL         time.Duration
	stopChan        chan struct{}
}

var taskManager *TaskManager

func init() {
	// Initialize task manager with reasonable defaults
	taskManager = &TaskManager{
		cleanupInterval: 5 * time.Minute,  // Run cleanup every 5 minutes
		taskTTL:         30 * time.Minute, // Tasks expire after 30 minutes
		stopChan:        make(chan struct{}),
	}

	// Start background cleanup goroutine
	go taskManager.startCleanup()

	websocket.RegisterDisconnectCallback(func(websocketID string) {
		log.Debugf("stop task for websocket: %v after websocket disconnected", websocketID)
		inflightMessages.Range(func(key, value any) bool {
			v1, ok := value.(MessageTask)
			if ok {
				if v1.WebsocketID == websocketID {
					log.Info("stop task:", v1)
					task.StopTask(v1.TaskID)
					// Clean up the task from memory
					inflightMessages.Delete(key)
				}
			}
			return true
		})
	})
}

// startCleanup runs a background goroutine to clean up expired tasks
func (tm *TaskManager) startCleanup() {
	ticker := time.NewTicker(tm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.cleanupExpiredTasks()
		case <-tm.stopChan:
			return
		}
	}
}

// cleanupExpiredTasks removes tasks that have exceeded their TTL
func (tm *TaskManager) cleanupExpiredTasks() {
	now := time.Now()
	expiredTasks := make([]interface{}, 0)

	inflightMessages.Range(func(key, value any) bool {
		if messageTask, ok := value.(MessageTask); ok {
			if now.Sub(messageTask.CreatedAt) > tm.taskTTL {
				expiredTasks = append(expiredTasks, key)
				log.Debugf("Cleaning up expired task: %v (age: %v)", key, now.Sub(messageTask.CreatedAt))

				// Cancel the task if it has a cancel function
				if messageTask.CancelFunc != nil {
					messageTask.CancelFunc()
				}
				// Stop deprecated task
				if messageTask.TaskID != "" {
					task.StopTask(messageTask.TaskID)
				}
			}
		}
		return true
	})

	// Remove expired tasks from the map
	for _, key := range expiredTasks {
		inflightMessages.Delete(key)
	}

	if len(expiredTasks) > 0 {
		log.Infof("Cleaned up %d expired message tasks", len(expiredTasks))
	}
}

// stopCleanup stops the background cleanup goroutine
func (tm *TaskManager) stopCleanup() {
	close(tm.stopChan)
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
			// Remove task from memory after stopping
			inflightMessages.Delete(taskID)
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

func (h APIHandler) sendChatMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	sessionID := ps.MustGetParameter("session_id")

	var request common.MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		log.Error(err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	assistantID := h.GetParameterOrDefault(req, "assistant_id", DefaultAssistantID)

	reqMsg, err := saveRequestMessage(sessionID, assistantID, &request)
	if err != nil {
		h.Error(w, err)
		return
	}

	err = h.handleMessage(w, req, sessionID, assistantID, reqMsg)
	if err != nil {
		_ = log.Error(err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := []util.MapStr{util.MapStr{
		"_id":     reqMsg.ID,
		"result":  "created",
		"_source": reqMsg,
	}}

	h.WriteJSON(w, response, 200)
}

func (h APIHandler) sendChatMessageV2(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	sessionID := ps.MustGetParameter("session_id")

	id := h.GetParameterOrDefault(r, "assistant_id", DefaultAssistantID)

	assistant, exists, err := common.GetAssistant(r, id)
	if !exists || err != nil {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	//launch the LLM task
	//streaming output result to HTTP client

	var request common.MessageRequest
	if err := h.DecodeJSON(r, &request); err != nil {
		//error can be ignored, since older app version didn't have this option
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reqMsg, err := saveRequestMessage(sessionID, id, &request)
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
		CreatedAt:  time.Now(),
	})

	// Process message asynchronously and cleanup on completion
	go func() {
		defer func() {
			// Always cleanup the task when processing completes (success or failure)
			inflightMessages.Delete(replyMsgTaskID)
		}()
		_ = h.processMessageAsync(ctx, reqMsg, params, streamSender)
	}()

}

func getReplyMessageTaskID(sessionID, messageID string) string {
	if messageID == "" {
		return sessionID
	}
	return fmt.Sprintf("%s_%s", sessionID, messageID)
}

func (h APIHandler) closeChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	id := ps.MustGetParameter("session_id")
	obj := common.Session{}
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
