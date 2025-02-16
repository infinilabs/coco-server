/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/lib/langchaingo/llms"
	"infini.sh/coco/lib/langchaingo/llms/ollama"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/search"
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
	ChatSessionID      string
	WebsocketSessionID string
}

var sessions = map[string]ChatSession{} //chat_session_id => session_object

const MessageTypeUser string = "user"
const MessageTypeAssistant string = "assistant"
const MessageTypeSystem string = "system"

type ChatMessage struct {
	orm.ORMObjectBase
	MessageType string      `json:"type"` // user, assistant, system
	SessionID   string      `json:"session_id"`
	Parameters  util.MapStr `json:"parameters,omitempty"`
	From        string      `json:"from"`
	To          string      `json:"to,omitempty"`
	Message     string      `config:"message" json:"message,omitempty" elastic_mapping:"message:{type:keyword}"`
}

func (h APIHandler) getChatSessions(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	q := orm.Query{}
	q.From = h.GetIntOrDefault(req, "from", 0)
	q.Size = h.GetIntOrDefault(req, "size", 20)
	q.AddSort("updated", orm.DESC)
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

	sessions[obj.ID] = ChatSession{ChatSessionID: obj.ID}

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
	q.AddSort("updated", orm.ASC)

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

var inflightMessages = sync.Map{}

func (h APIHandler) cancelReplyMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	sessionID := ps.MustGetParameter("session_id")
	v, ok := inflightMessages.Load(sessionID)
	if ok {
		task.StopTask(v.(string))
	}
	err := h.WriteAckOKJSON(w)
	if err != nil {
		h.Error(w, err)
	}
}

func formatDocumentReferences(docs []common.Document) string {
	var sb strings.Builder
	sb.WriteString("<REFERENCES>\n")
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("<Doc>"))
		sb.WriteString(fmt.Sprintf("ID #%d - %v\n", i+1, doc.ID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", doc.Title))
		sb.WriteString(fmt.Sprintf("Source: %s\n", doc.Source))
		sb.WriteString(fmt.Sprintf("Updated: %s\n", doc.Updated))
		sb.WriteString(fmt.Sprintf("Category: %s\n", doc.GetAllCategories()))
		sb.WriteString(fmt.Sprintf("Summary: %s\n", doc.Summary))
		sb.WriteString(fmt.Sprintf("Content: %s\n", doc.Content))
		sb.WriteString(fmt.Sprintf("</Doc>\n"))

	}
	sb.WriteString("</REFERENCES>")
	return sb.String()
}

func formatDocumentReferencesToDisplay(docs []common.Document) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<Source total=%v>\n", len(docs)))
	outDocs := []util.MapStr{}
	for _, doc := range docs {
		item := util.MapStr{}
		item["id"] = doc.ID
		item["title"] = doc.Title
		item["source"] = doc.Source
		item["updated"] = doc.Updated
		item["category"] = doc.Category
		item["summary"] = doc.Summary
		item["icon"] = doc.Icon
		item["size"] = doc.Size
		item["thumbnail"] = doc.Thumbnail
		item["url"] = doc.URL
		outDocs = append(outDocs, item)
	}
	sb.WriteString(util.MustToJSON(outDocs))
	sb.WriteString("</Source>")
	return sb.String()
}

func fetchDocuments(query *orm.Query) ([]common.Document, error) {
	var docs []common.Document
	err, _ := orm.SearchWithJSONMapper(&docs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}
	return docs, nil
}

func (h APIHandler) sendChatMessage(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	webSocketID := req.Header.Get("WEBSOCKET-SESSION-ID")

	log.Trace(req.Header)

	sessionID := ps.MustGetParameter("session_id")
	var request MessageRequest
	if err := h.DecodeJSON(req, &request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var (
		from         = h.GetIntOrDefault(req, "from", 0)
		size         = h.GetIntOrDefault(req, "size", 10)
		datasource   = h.GetParameterOrDefault(req, "datasource", "")
		category     = h.GetParameterOrDefault(req, "category", "")
		username     = h.GetParameterOrDefault(req, "username", "")
		userid       = h.GetParameterOrDefault(req, "userid", "")
		tags         = h.GetParameterOrDefault(req, "tags", "")
		subcategory  = h.GetParameterOrDefault(req, "subcategory", "")
		richCategory = h.GetParameterOrDefault(req, "rich_category", "")
		field        = h.GetParameterOrDefault(req, "search_field", "title")
		source       = h.GetParameterOrDefault(req, "source_fields", "*")
	)

	searchDB := h.GetBoolOrDefault(req, "search", true)

	obj := ChatMessage{
		SessionID:   sessionID,
		MessageType: MessageTypeUser,
		Message:     request.Message,
	}

	if searchDB {
		obj.Parameters = util.MapStr{}
		obj.Parameters["search"] = searchDB
	}

	err := orm.Create(nil, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := []util.MapStr{util.MapStr{
		"_id":     obj.ID,
		"result":  "created",
		"_source": obj,
	}}

	if webSocketID != "" {
		var query *orm.Query
		if searchDB {
			mustClauses := search.BuildMustClauses(datasource, category, subcategory, richCategory, username, userid)
			query = search.BuildTemplatedQuery(from, size, mustClauses, field, obj.Message, source, tags)

		}

		//de-duplicate background task per-session, cancelable
		taskID := task.RunWithinGroup("assistant-session", func(taskCtx context.Context) error {
			//timeout for 30 seconds

			log.Debugf("place a assistant background job for session: %v, websocket: %v ", sessionID, webSocketID)

			//1. expand and rewrite the query
			// use the title and summary to judge which document need to fetch in-depth, also the updated time to check the data is fresh or not
			// pick N related documents and combine with the memory and the near chat history as the chat context
			//2. summary previous history chat as context, update as memory
			//3. assemble with the agent's role setting
			//4. send to LLM

			ollamaConfig := common.AppConfig().OllamaConfig
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

			// Prepare the system message
			content := []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeSystem, "You are a personal AI assistant designed by Coco AI(https://coco.rs), the backend team is behind INFINI Labs(https://infinilabs.com)."),
			}

			var references string
			var simpliedReferences string
			//Retrieve related documents from background server
			if searchDB && query != nil {
				docs, err := fetchDocuments(query)
				if err != nil {
					log.Errorf("Failed to fetch documents from DB: %v", err)
					// Proceed without RAG
				} else if len(docs) > 0 {
					references = formatDocumentReferences(docs)
					simpliedReferences = formatDocumentReferencesToDisplay(docs)
				}
			}

			prompt := fmt.Sprintf(`You are a friendly assistant designed to help users access and understand their personal or company data. Your responses should be clear, concise, and based solely on the information provided below. If the information is insufficient, please indicate that you need more details to assist effectively.

Query: %s

Data:
%s`, request.Message, references)

			// Append the user's message
			content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, prompt))

			log.Debug(content)

			chunkSeq := 0
			messageID := util.GetUUID()
			requestMessageID := obj.ID
			messageBuffer := strings.Builder{}

			if simpliedReferences != "" {
				messageBuffer.WriteString(simpliedReferences)
			}

			sentSource := false

			completion, err := llm.GenerateContent(ctx, content,
				llms.WithTemperature(0.8),
				llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {

					var txtMsg string
					if !sentSource {
						if simpliedReferences != "" {
							txtMsg = simpliedReferences
						}
						sentSource = true
					}
					txtMsg += string(chunk)

					chunkSeq += 1
					msg := util.MustToJSON(util.MapStr{
						"session_id":       sessionID,
						"message_id":       messageID,
						"message_type":     MessageTypeAssistant,
						"reply_to_message": requestMessageID,
						"chunk_sequence":   chunkSeq,
						"message_chunk":    txtMsg,
					})
					messageBuffer.Write(chunk)
					websocket.SendPrivateMessage(webSocketID, msg)
					return nil
				}))
			if err != nil {
				log.Error(err)
				return err
			}
			_ = completion

			chunkSeq += 1
			msg := util.MustToJSON(util.MapStr{
				"session_id":       sessionID,
				"message_id":       messageID,
				"message_type":     MessageTypeSystem,
				"reply_to_message": requestMessageID,
				"chunk_sequence":   chunkSeq,
				"message_chunk":    string("assistant finished output"),
			})
			websocket.SendPrivateMessage(webSocketID, msg)

			//save message to system
			if messageBuffer.Len() > 0 {
				obj = ChatMessage{
					SessionID:   sessionID,
					MessageType: MessageTypeAssistant,
					Message:     messageBuffer.String(),
				}
				obj.ID = messageID

				err = orm.Save(nil, &obj)
				if err != nil {
					log.Error(err)
					return err
				}
			}
			return nil
		})
		inflightMessages.Store(sessionID, taskID)
	} else {
		log.Debugf("no websocket: %v found for session: %v ", webSocketID, sessionID)
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
