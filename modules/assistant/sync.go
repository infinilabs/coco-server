/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"context"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/orm"
	"strings"
	"sync"
)

type MemoryMessageSender struct {
	mu       sync.Mutex
	messages []string
}

func (m *MemoryMessageSender) SendMessage(msg *common.MessageChunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Trace("got message:", msg.ChunkType, "=> ", msg.MessageChunk)

	if msg == nil || (msg.MessageType == common.Response && strings.TrimSpace(msg.MessageChunk) == "") {
		return nil
	}

	//only keep response type
	if msg.ChunkType == common.Response {
		m.messages = append(m.messages, msg.MessageChunk)
	}
	return nil
}

func (m *MemoryMessageSender) FinalResponse() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return strings.Join(m.messages, "")
}

// AskAssistantSync sends a message to an assistant and waits for a full response.
// This version is fully detached from APIHandler and HTTP context.
func AskAssistantSync(ctx context.Context, id string, message string, vars map[string]any) (string, error) {
	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectAccess()
	assistant, exists, err := common.InternalGetAssistant(ctx1, id)
	if !exists || err != nil {
		return "", fmt.Errorf("assistant %s not found: %w", id, err)
	}

	if message == "" {
		return "", errors.New("message is empty")
	}

	// Construct and save a new chat message
	request := common.MessageRequest{
		Message: message,
	}
	session, err, reqMsg, _ := InternalCreateAndSaveNewChatMessage(ctx1, id, &request, false)
	if err != nil || reqMsg == nil {
		return "", fmt.Errorf("failed to create chat message: %w", err)
	}

	ragCtx := &RAGContext{}
	ragCtx.AssistantCfg = assistant
	ragCtx.SessionID = session.ID
	ragCtx.InputValues = vars

	// Memory-based receiver for synchronous mode
	receiver := &MemoryMessageSender{}
	if err := processMessageAsync(ctx1, reqMsg, ragCtx, receiver); err != nil {
		return "", fmt.Errorf("process message failed: %w", err)
	}

	return receiver.FinalResponse(), nil
}
