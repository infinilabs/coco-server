package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/smallnest/langgraphgo/log"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/util"
)

// Heavily based on Kubernetes' (https://github.com/GoogleCloudPlatform/kubernetes) detection code.
var connectionUpgradeRegex = regexp.MustCompile("(^|.*,\\s*)upgrade($|\\s*,)")

func isWebsocketRequest(req *http.Request) bool {
	return connectionUpgradeRegex.MatchString(strings.ToLower(req.Header.Get("Connection"))) && strings.ToLower(req.Header.Get("Upgrade")) == "websocket"
}

type HTTPStreamSender struct {
	Enc     *json.Encoder
	Flusher http.Flusher
	Ctx     context.Context

	ReqMsg, ReplyMsg *core.ChatMessage
}

func NewHTTPStreamSender(reqMsg, replyMsg *core.ChatMessage) *HTTPStreamSender {
	sender := &HTTPStreamSender{
		ReqMsg:   reqMsg,
		ReplyMsg: replyMsg,
	}
	return sender
}

func (s *HTTPStreamSender) SendChunkMessage(messageType, chunkType, messageChunk string, chunkSequence int) error {
	msg := core.NewMessageChunk(s.ReqMsg.SessionID, s.ReplyMsg.ID, messageType, s.ReqMsg.ID, chunkType, messageChunk, chunkSequence)
	return s.SendMessage(msg)
}

func (s *HTTPStreamSender) SendMessage(msg *core.MessageChunk) error {
	log.Info(util.MustToJSON(msg))
	if msg == nil || (msg.MessageType == common.Response && strings.TrimSpace(msg.MessageChunk) == "") {
		return nil
	}

	select {
	case <-s.Ctx.Done():
		return fmt.Errorf("client disconnected")
	default:
		if err := s.Enc.Encode(msg); err != nil {
			return err
		}
		s.Flusher.Flush()
		return nil
	}
}
