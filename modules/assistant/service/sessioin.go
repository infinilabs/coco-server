package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/common"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

var InflightMessages = sync.Map{}

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
		firstMessage, err = SaveRequestMessage(ctx, obj.ID, assistantID, req)
		if err != nil {
			return core.Session{}, err, nil, nil
		}
		result["payload"] = firstMessage
	}

	return obj, err, firstMessage, result
}

func SaveRequestMessage(ctx *orm.Context, sessionID, assistantID string, req *core.MessageRequest) (*core.ChatMessage, error) {

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

func GetChatHistoryBySessionInternal(sessionID string, size int) ([]core.ChatMessage, error) {
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

func StopMessageReplyTask(taskID string) {
	v, ok := InflightMessages.Load(taskID)
	if ok {
		v1, ok := v.(common.MessageTask)
		if ok {
			seelog.Debug("stop task:", v1)
			if v1.TaskID != "" {
				task.StopTask(v1.TaskID)
			} else if v1.CancelFunc != nil {
				v1.CancelFunc()
			}
		}
	} else {
		_ = seelog.Warnf("task id [%s] was not found", taskID)
	}
}

func GetReplyMessageTaskID(sessionID, messageID string) string {
	if messageID == "" {
		return sessionID
	}
	return fmt.Sprintf("%s_%s", sessionID, messageID)
}

// AskAssistantSync sends a message to an assistant and waits for a full response.
// This version is fully detached from APIHandler and HTTP context.
func AskAssistantSync(ctx context.Context, userID string, id string, message string, vars map[string]any) (string, error) {
	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectAccess()
	ctx1.PermissionScope(security.PermissionScopePlatform)

	assistant, exists, err := InternalGetAssistant(ctx1, id)
	if !exists || err != nil {
		return "", fmt.Errorf("assistant %s not found: %w", id, err)
	}

	if message == "" {
		return "", errors.New("message is empty")
	}

	// Construct and save a new chat message
	request := core.MessageRequest{
		Message: message,
	}
	session, err, reqMsg, _ := InternalCreateAndSaveNewChatMessage(ctx1, id, &request, false)
	if err != nil || reqMsg == nil {
		return "", fmt.Errorf("failed to create chat message: %w", err)
	}

	ragCtx := &common.RAGContext{}
	ragCtx.AssistantCfg = assistant
	ragCtx.SessionID = session.ID
	ragCtx.InputValues = vars

	// Memory-based receiver for synchronous mode
	receiver := &common.MemoryMessageSender{}
	if err := ProcessMessageAsync(ctx1, userID, reqMsg, ragCtx, receiver); err != nil {
		return "", fmt.Errorf("process message failed: %w", err)
	}

	return receiver.FinalResponse(), nil
}

func CreateAndSaveNewChatMessage(request *http.Request, assistantID string, req *core.MessageRequest, visible bool) (core.Session, error, *core.ChatMessage, util.MapStr) {
	ctx := orm.NewContextWithParent(request.Context())
	return InternalCreateAndSaveNewChatMessage(ctx, assistantID, req, visible)
}
