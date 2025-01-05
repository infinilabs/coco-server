/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"context"
	log "github.com/cihub/seelog"
	"infini.sh/coco/lib/langchaingo/llms"
	"infini.sh/coco/lib/langchaingo/llms/ollama"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/api/websocket"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"net/http"
	"strings"
	"sync"
)

type Session struct {
	orm.ORMObjectBase
	Status  string `config:"status" json:"status,omitempty" elastic_mapping:"status:{type:keyword}"`
	Title   string `config:"title" json:"title,omitempty" elastic_mapping:"title:{type:keyword}"`
	Summary string `config:"summary" json:"summary,omitempty" elastic_mapping:"summary:{type:keyword}"`
}

type MessageRequest struct {
	Message string `json:"message"`
}

type ChatSession struct {
	ChatSessionID string
	WebsocketSessionID string
}

var sessions = map[string]ChatSession{} //chat_session_id => session_object

const MessageTypeUser string = "user"
const MessageTypeAssistant string = "assistant"
const MessageTypeSystem string = "system"

type ChatMessage struct {
	orm.ORMObjectBase
	MessageType string `json:"type"` // user, assistant, system
	SessionID   string `json:"session_id"`
	From        string `json:"from"`
	To          string `json:"to,omitempty"`
	Message     string `config:"message" json:"message,omitempty" elastic_mapping:"message:{type:keyword}"`
}

func (h APIHandler) getChatSessions(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	q := orm.Query{}
	q.From = h.GetIntOrDefault(req, "from", 0)
	q.Size = h.GetIntOrDefault(req, "size", 20)

	err, res := orm.Search(&Session{}, &q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) newChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	obj := Session{
		Status: "active",
	}

	err := orm.Create(nil, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.WriteJSON(w, util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"_source": obj,
	}, 200)

	sessions[obj.ID]=ChatSession{ChatSessionID: obj.ID}

	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) openChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("session_id")

	obj := Session{}
	obj.ID = id
	
	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	obj.Status = "active"
	err = orm.Update(nil, &obj)
	if err != nil {
		h.Error(w, err)
		return
	}

	err = h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) getChatHistoryBySession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	q := orm.Query{}
	q.Conds = orm.And(orm.Eq("session_id", ps.MustGetParameter("session_id")))
	q.From = h.GetIntOrDefault(req, "from", 0)
	q.Size = h.GetIntOrDefault(req, "size", 20)

	err, res := orm.Search(&ChatMessage{}, &q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

var inflightMessages=sync.Map{}

func (h APIHandler) cancelReplyMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	sessionID := ps.MustGetParameter("session_id")
	v,ok:=inflightMessages.Load(sessionID)
	if ok{
		task.StopTask(v.(string))
	}
	err:=h.WriteAckOKJSON(w)
	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) sendChatMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	webSocketID:=req.Header.Get("WEBSOCKET-SESSION-ID")

	log.Info(req.Header)

	sessionID := ps.MustGetParameter("session_id")
	var request MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	obj := ChatMessage{
		SessionID: sessionID,
		MessageType: MessageTypeUser,
		Message:   request.Message,
	}

	err := orm.Create(nil, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response:=[]util.MapStr{util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"_source": obj,
	}}

	if webSocketID!=""{
		//de-duplicate background task per-session, cancelable
		taskID:=task.RunWithinGroup("assistant-session", func(taskCtx context.Context) error {
			//timeout for 30 seconds

			log.Debugf("place a assistant background job for session: %v, websocket: %v ",sessionID,webSocketID)

			//TODO
			//1. retrieve related documents from background server
			//2. summary previous history chat as context
			//3. assemble with the agent's role setting
			//4. send to LLM

			ollamaConfig:=common.AppConfig().OllamaConfig
			llm, err := ollama.New(
				ollama.WithServerURL(ollamaConfig.Endpoint),
				ollama.WithModel(ollamaConfig.Model),
				ollama.WithKeepAlive(ollamaConfig.Keepalive))

			//TODO, more options exposed to config
			if err != nil {
				log.Error(err)
				return err
			}
			ctx := context.Background()

			content := []llms.MessageContent{
				//llms.TextParts(llms.ChatMessageTypeSystem, "You are a company branding design wizard."),
				//llms.TextParts(llms.ChatMessageTypeHuman, "What would be a good company name for a comapny that produces Go-backed LLM tools?"),
				llms.TextParts(llms.ChatMessageTypeHuman, request.Message),
			}

			chunkSeq:=0
			messageID:=util.GetUUID()
			requestMessageID:=obj.ID
			messageBuffer:=strings.Builder{}
			completion, err := llm.GenerateContent(ctx, content,
				llms.WithTemperature(0.8),
				llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
					chunkSeq+=1
					msg:=util.MustToJSON(util.MapStr{
						"session_id": sessionID,
						"message_id": messageID,
						"message_type": MessageTypeAssistant,
						"reply_to_message": requestMessageID,
						"chunk_sequence":chunkSeq,
						"message_chunk":string(chunk),
					})
					messageBuffer.Write(chunk)
					websocket.SendPrivateMessage(webSocketID,msg)
					return nil
				}))
			if err != nil {
				log.Error(err)
				return err
			}
			_ = completion

			chunkSeq+=1
			msg:=util.MustToJSON(util.MapStr{
				"session_id": sessionID,
				"message_id": messageID,
				"message_type": MessageTypeSystem,
				"reply_to_message": requestMessageID,
				"chunk_sequence":chunkSeq,
				"message_chunk":string("assistant finished output"),
			})
			websocket.SendPrivateMessage(webSocketID,msg)

			//save message to system
			if messageBuffer.Len()>0{
				obj = ChatMessage{
					SessionID: sessionID,
					MessageType: MessageTypeAssistant,
					Message:   messageBuffer.String(),
				}
				obj.ID=messageID

				err=orm.Save(nil,&obj)
				if err != nil {
					log.Error(err)
					return err
				}
			}
			return nil
		})
		inflightMessages.Store(sessionID,taskID)
	}else{
		log.Debugf("no websocket: %v found for session: %v ",webSocketID,sessionID)
	}

	err = h.WriteJSON(w, response, 200)

	if err != nil {
		h.Error(w, err)
	}
}

func (h APIHandler) closeChatSession(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	id := ps.MustGetParameter("session_id")
	obj := Session{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	obj.Status = "closed"
	err = orm.Update(nil, &obj)
	if err != nil {
		h.Error(w, err)
		return
	}

	err = h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
	if err != nil {
		h.Error(w, err)
	}

}
