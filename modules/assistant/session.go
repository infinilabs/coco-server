/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	// Use the new enhanced task storage function
	storeMessageTask(session.ID, reqMsg.ID, "", "", cancel)
	_ = h.processMessageAsync(ctx, reqMsg, params, streamSender)
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
	streamSender := &HTTPStreamSender{
		Enc:     enc,
		Flusher: flusher,
		Ctx:     r.Context(), // assuming this is in an HTTP handler
	}
	_ = h.processMessageAsync(ctx, reqMsg, params, streamSender)

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
	q.AddSort("created", orm.ASC)

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

// Configuration constants for task cleanup
const (
	// Default TTL for message tasks (1 hour)
	DefaultTaskTTL = time.Hour

	// Cleanup interval (every 5 minutes)
	CleanupInterval = 5 * time.Minute

	// Maximum number of tasks to keep (safety limit)
	MaxActiveTasks = 10000

	// Cleanup batch size
	CleanupBatchSize = 100
)

type MessageTask struct {
	SessionID    string
	MessageID    string // Message ID for consistent key generation
	WebsocketID  string // WebSocket ID for disconnect handling
	TaskID       string // Deprecated but kept for backward compatibility
	CancelFunc   func()
	CreatedAt    time.Time // Creation timestamp for TTL cleanup
	LastAccessAt time.Time // Last access time for activity tracking
}

// TaskCleanupManager manages the lifecycle of inflight message tasks
type TaskCleanupManager struct {
	ctx      context.Context
	cancel   context.CancelFunc
	ttl      time.Duration
	interval time.Duration
	maxTasks int
	running  bool
	mu       sync.RWMutex
}

var cleanupManager *TaskCleanupManager
var cleanupOnce sync.Once

// Helper functions for consistent key management
func generateTaskKey(sessionID, messageID string) string {
	if messageID == "" {
		return sessionID
	}
	return fmt.Sprintf("%s_%s", sessionID, messageID)
}

func parseTaskKey(key string) (sessionID, messageID string) {
	parts := strings.SplitN(key, "_", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// getTaskTTL returns the configured TTL for tasks
func getTaskTTL() time.Duration {
	if ttlStr := os.Getenv("ASSISTANT_TASK_TTL"); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			return ttl
		}
	}
	return DefaultTaskTTL
}

// getCleanupInterval returns the configured cleanup interval
func getCleanupInterval() time.Duration {
	if intervalStr := os.Getenv("ASSISTANT_CLEANUP_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			return interval
		}
	}
	return CleanupInterval
}

// getMaxActiveTasks returns the configured maximum number of active tasks
func getMaxActiveTasks() int {
	if maxStr := os.Getenv("ASSISTANT_MAX_TASKS"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil && max > 0 {
			return max
		}
	}
	return MaxActiveTasks
}

// NewTaskCleanupManager creates a new cleanup manager instance
func NewTaskCleanupManager() *TaskCleanupManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskCleanupManager{
		ctx:      ctx,
		cancel:   cancel,
		ttl:      getTaskTTL(),
		interval: getCleanupInterval(),
		maxTasks: getMaxActiveTasks(),
		running:  false,
	}
}

// Start begins the cleanup manager's background operations
func (tcm *TaskCleanupManager) Start() {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	if tcm.running {
		return
	}

	tcm.running = true
	go tcm.cleanupLoop()
	log.Infof("Task cleanup manager started with TTL=%v, interval=%v, maxTasks=%d",
		tcm.ttl, tcm.interval, tcm.maxTasks)
}

// Stop gracefully stops the cleanup manager
func (tcm *TaskCleanupManager) Stop() {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	if !tcm.running {
		return
	}

	tcm.cancel()
	tcm.running = false
	log.Info("Task cleanup manager stopped")
}

// cleanupLoop runs the periodic cleanup process
func (tcm *TaskCleanupManager) cleanupLoop() {
	ticker := time.NewTicker(tcm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-tcm.ctx.Done():
			return
		case <-ticker.C:
			tcm.performCleanup()
		}
	}
}

// performCleanup performs the actual cleanup of expired and orphaned tasks
func (tcm *TaskCleanupManager) performCleanup() {
	now := time.Now()
	var cleanedCount int
	var totalCount int
	var expiredKeys []interface{}

	// First pass: identify expired tasks
	inflightMessages.Range(func(key, value interface{}) bool {
		totalCount++

		task, ok := value.(MessageTask)
		if !ok {
			// Invalid task type, mark for removal
			expiredKeys = append(expiredKeys, key)
			return true
		}

		// Check if task has expired
		if now.Sub(task.CreatedAt) > tcm.ttl {
			expiredKeys = append(expiredKeys, key)
			return true
		}

		// Check if we've exceeded the maximum task limit
		if totalCount > tcm.maxTasks {
			expiredKeys = append(expiredKeys, key)
			return true
		}

		return true
	})

	// Second pass: clean up expired tasks
	for _, key := range expiredKeys {
		if value, loaded := inflightMessages.LoadAndDelete(key); loaded {
			cleanedCount++

			// Try to cancel the task if possible
			if msgTask, ok := value.(MessageTask); ok {
				if msgTask.CancelFunc != nil {
					msgTask.CancelFunc()
				}
				if msgTask.TaskID != "" {
					task.StopTask(msgTask.TaskID)
				}
			}
		}

		// Process in batches to avoid blocking too long
		if cleanedCount%CleanupBatchSize == 0 {
			select {
			case <-tcm.ctx.Done():
				return
			default:
				// Continue processing
			}
		}
	}

	if cleanedCount > 0 {
		log.Warnf("Cleanup manager removed %d expired/orphaned tasks out of %d total tasks",
			cleanedCount, totalCount)
	} else if totalCount > 0 {
		log.Debugf("Cleanup manager checked %d active tasks, none expired", totalCount)
	}
}

// getOrCreateCleanupManager returns the singleton cleanup manager instance
func getOrCreateCleanupManager() *TaskCleanupManager {
	cleanupOnce.Do(func() {
		cleanupManager = NewTaskCleanupManager()
		cleanupManager.Start()
	})
	return cleanupManager
}

// ShutdownCleanupManager gracefully shuts down the cleanup manager
// This function should be called during application shutdown
func ShutdownCleanupManager() {
	if cleanupManager != nil {
		cleanupManager.Stop()
	}
}

// GetActiveTaskCount returns the current number of active tasks
// This is useful for monitoring and debugging
func GetActiveTaskCount() int {
	count := 0
	inflightMessages.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Enhanced task storage function with proper cleanup tracking
func storeMessageTask(sessionID, messageID, websocketID string, taskID string, cancelFunc func()) {
	key := generateTaskKey(sessionID, messageID)
	now := time.Now()

	task := MessageTask{
		SessionID:    sessionID,
		MessageID:    messageID,
		WebsocketID:  websocketID,
		TaskID:       taskID,
		CancelFunc:   cancelFunc,
		CreatedAt:    now,
		LastAccessAt: now,
	}

	inflightMessages.Store(key, task)

	// Ensure cleanup manager is running
	getOrCreateCleanupManager()

	log.Debugf("Stored message task with key=%s, sessionID=%s, messageID=%s",
		key, sessionID, messageID)
}

// Enhanced task cleanup function with proper error handling
func cleanupMessageTask(sessionID, messageID string) {
	key := generateTaskKey(sessionID, messageID)

	if value, loaded := inflightMessages.LoadAndDelete(key); loaded {
		if msgTask, ok := value.(MessageTask); ok {
			// Cancel the task if it has a cancel function
			if msgTask.CancelFunc != nil {
				msgTask.CancelFunc()
			}
			// Stop the task if it has a task ID
			if msgTask.TaskID != "" {
				task.StopTask(msgTask.TaskID)
			}
		}
		log.Debugf("Cleaned up message task with key=%s", key)
	} else {
		log.Debugf("Message task with key=%s not found for cleanup", key)
	}
}

func init() {
	// Enhanced WebSocket disconnect callback with better error handling
	websocket.RegisterDisconnectCallback(func(websocketID string) {
		log.Debugf("Cleaning up tasks for disconnected websocket: %v", websocketID)

		var tasksToCleanup []interface{}

		// First pass: identify tasks to cleanup
		inflightMessages.Range(func(key, value any) bool {
			if msgTask, ok := value.(MessageTask); ok && msgTask.WebsocketID == websocketID {
				tasksToCleanup = append(tasksToCleanup, key)
			}
			return true
		})

		// Second pass: cleanup identified tasks
		for _, key := range tasksToCleanup {
			if value, loaded := inflightMessages.LoadAndDelete(key); loaded {
				if msgTask, ok := value.(MessageTask); ok {
					log.Infof("Stopping task for disconnected websocket: %v, sessionID: %v",
						websocketID, msgTask.SessionID)

					// Cancel the task
					if msgTask.CancelFunc != nil {
						msgTask.CancelFunc()
					}
					if msgTask.TaskID != "" {
						task.StopTask(msgTask.TaskID)
					}
				}
			}
		}

		if len(tasksToCleanup) > 0 {
			log.Infof("Cleaned up %d tasks for disconnected websocket: %v",
				len(tasksToCleanup), websocketID)
		}
	})
}

func stopMessageReplyTask(taskID string) {
	// Try to load and cleanup the task using the provided taskID
	if value, loaded := inflightMessages.LoadAndDelete(taskID); loaded {
		if msgTask, ok := value.(MessageTask); ok {
			log.Debug("stop task:", msgTask)
			if msgTask.TaskID != "" {
				task.StopTask(msgTask.TaskID)
			}
			if msgTask.CancelFunc != nil {
				msgTask.CancelFunc()
			}
		}
	} else {
		// For backward compatibility, also try parsing as session_message format
		sessionID, messageID := parseTaskKey(taskID)
		if messageID != "" {
			cleanupMessageTask(sessionID, messageID)
		} else {
			_ = log.Warnf("task id [%s] was not found", taskID)
		}
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
	// Use the new enhanced task storage function
	storeMessageTask(sessionID, reqMsg.ID, "", "", cancel)
	_ = h.processMessageAsync(ctx, reqMsg, params, streamSender)

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
