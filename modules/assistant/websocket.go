/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"infini.sh/framework/core/api/websocket"
)

func (h APIHandler) sendChatMessageToAIbot(c *websocket.WebsocketConnection, array []string) {
	//t:= time.Now()
	//response := MessageResponse{
	//	SequenceID: 1,
	//	Created:  &t,
	//	Message:  "Hello",
	//	Response: "Hi",
	//}

	//err := c.WriteJSON(response)
	//if err != nil {
	//	h.Error(c, err)
	//}
}