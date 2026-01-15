package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
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
}

func (s *HTTPStreamSender) SendMessage(msg *core.MessageChunk) error {

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
