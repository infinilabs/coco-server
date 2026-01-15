/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"strings"
	"sync"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
)

type MemoryMessageSender struct {
	mu       sync.Mutex
	messages []string
}

func (m *MemoryMessageSender) SendMessage(msg *core.MessageChunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if msg == nil || (msg.MessageType == common.Response && strings.TrimSpace(msg.MessageChunk) == "") {
		return nil
	}

	log.Trace("got message:", msg.ChunkType, "=> ", msg.MessageChunk)

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
