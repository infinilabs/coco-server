// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package assistant

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"infini.sh/coco/modules/assistant/rag"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/search"
	"infini.sh/framework/core/api/websocket"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"net/http"
	"strings"
)

// Helper types and methods
type processingParams struct {
	searchDB     bool
	deepThink    bool
	from         int
	size         int
	datasource   string
	category     string
	username     string
	userid       string
	tags         string
	subcategory  string
	richCategory string
	field        string
	source       string

	//
	websocketID      string
	sessionID        string
	replyToMessageID string

	//prepare for final response
	sourceDocsSummaryBlock string
	historyBlock           string
	queryIntent            *rag.QueryIntent
	queryIntentStr         string
	pickedDocIDS           []string
	references             string

	intentModel     string
	pickingDocModel string
	answeringModel  string
}

func (h APIHandler) extractParameters(req *http.Request) *processingParams {
	cfg := common.AppConfig()
	return &processingParams{
		searchDB:        h.GetBoolOrDefault(req, "search", false),
		deepThink:       h.GetBoolOrDefault(req, "deep_thinking", false),
		from:            h.GetIntOrDefault(req, "from", 0),
		size:            h.GetIntOrDefault(req, "size", 10),
		datasource:      h.GetParameterOrDefault(req, "datasource", ""),
		category:        h.GetParameterOrDefault(req, "category", ""),
		username:        h.GetParameterOrDefault(req, "username", ""),
		userid:          h.GetParameterOrDefault(req, "userid", ""),
		tags:            h.GetParameterOrDefault(req, "tags", ""),
		subcategory:     h.GetParameterOrDefault(req, "subcategory", ""),
		richCategory:    h.GetParameterOrDefault(req, "rich_category", ""),
		field:           h.GetParameterOrDefault(req, "search_field", "title"),
		source:          h.GetParameterOrDefault(req, "source_fields", "*"),
		intentModel:     cfg.LLMConfig.IntentAnalysisModel,
		pickingDocModel: cfg.LLMConfig.PickingDocModel,
		answeringModel:  cfg.LLMConfig.AnsweringModel,
	}

}

func (h APIHandler) createInitialUserRequestMessage(sessionID, message string, params *processingParams) *ChatMessage {
	msg := &ChatMessage{
		SessionID:   sessionID,
		MessageType: MessageTypeUser,
		Message:     message,
	}
	msg.ID = util.GetUUID()

	if params.searchDB {
		msg.Parameters = util.MapStr{"params": params}
	}
	return msg
}

func (h APIHandler) saveMessage(msg *ChatMessage) error {
	return orm.Create(nil, msg)
}

func (h APIHandler) launchBackgroundTask(msg *ChatMessage, params *processingParams) {

	//1. expand and rewrite the query
	// use the title and summary to judge which document need to fetch in-depth, also the updated time to check the data is fresh or not
	// pick N related documents and combine with the memory and the near chat history as the chat context
	//2. summary previous history chat as context, update as memory
	//3. assemble with the agent's role setting
	//4. send to LLM

	taskID := task.RunWithinGroup("assistant-session", func(taskCtx context.Context) error {
		return h.processMessageAsync(taskCtx, msg, params)
	})

	log.Debugf("place a assistant background job: %v, for session: %v, websocket: %v ",
		taskID, params.sessionID, params.websocketID)

	inflightMessages.Store(params.sessionID, MessageTask{
		TaskID:      taskID,
		WebsocketID: params.websocketID,
	})
	log.Infof("Saved taskID: %v for session: %v", taskID, params.sessionID)
}

func (h APIHandler) createAssistantMessage(sessionID, requestMessageID string) *ChatMessage {
	msg := &ChatMessage{
		SessionID:      sessionID,
		MessageType:    MessageTypeAssistant,
		ReplyMessageID: requestMessageID,
	}
	msg.ID = util.GetUUID()

	return msg
}

// WebSocket helper
func (h APIHandler) sendWebsocketMessage(wsID string, msg *MessageChunk) {
	if err := websocket.SendPrivateMessage(wsID, util.MustToJSON(msg)); err != nil {
		log.Warnf("WebSocket send error: %v", err)
	}
}

func (h APIHandler) finalizeProcessing(ctx context.Context, wsID, sessionID string, msg *ChatMessage) {
	if err := orm.Save(nil, msg); err != nil {
		log.Errorf("Failed to save assistant message: %v", err)
	}

	h.sendWebsocketMessage(wsID, NewMessageChunk(
		sessionID, msg.ID, MessageTypeSystem, msg.ReplyMessageID,
		ReplyEnd, "Processing completed", 0,
	))
}

func (h APIHandler) processMessageAsync(ctx context.Context, reqMsg *ChatMessage, params *processingParams) error {
	log.Debugf("Starting async processing for session: %v", params.sessionID)

	replyMsg := h.createAssistantMessage(params.sessionID, reqMsg.ID)
	defer h.finalizeProcessing(ctx, params.websocketID, params.sessionID, replyMsg)

	reqMsg.Details = make([]ProcessingDetails, 0)

	var docs []common.Document
	// Processing pipeline
	_ = h.fetchSessionHistory(ctx, reqMsg, replyMsg, params, 10)

	_ = h.processQueryIntent(ctx, reqMsg, replyMsg, params)

	if params.searchDB {
		var fetchSize = 10
		if params.deepThink {
			fetchSize = 50
		}
		docs, _ = h.processInitialDocumentSearch(ctx, reqMsg, replyMsg, params, fetchSize)

		if params.deepThink && len(docs) > 10 {
			//re-pick top docs
			docs, _ = h.processPickDocuments(ctx, reqMsg, replyMsg, params, docs)
			_ = h.fetchDocumentInDepth(ctx, reqMsg, replyMsg, params, docs)
		}
	}

	h.generateFinalResponse(ctx, reqMsg, replyMsg, params)
	log.Info("async reply task done for query:", reqMsg.Message)
	return nil
}

func (h APIHandler) fetchSessionHistory(ctx context.Context, reqMsg, replyMsg *ChatMessage, params *processingParams, size int) error {
	var historyStr = strings.Builder{}

	//get chat history
	history, err := getChatHistoryBySessionInternal(params.sessionID, size)
	if err != nil {
		return err
	}
	historyStr.WriteString("<conversation>")

	//<summary>
	//session history summary within 500 words TODO
	//</summary>

	historyStr.WriteString("<recent>")
	for _, v := range history {
		historyStr.WriteString(v.MessageType + ": " + util.SubStringWithSuffix(v.Message, 500, "...") + ", " + v.Created.String())
		if v.DownVote > 0 {
			historyStr.WriteString(fmt.Sprintf("(%v people up voted this answer)", v.UpVote))
		}
		if v.DownVote > 0 {
			historyStr.WriteString(fmt.Sprintf("(%v people down voted this answer)", v.DownVote))
		}
		historyStr.WriteString("\n")
	}
	historyStr.WriteString("</recent>")

	historyStr.WriteString("</conversation>")

	params.historyBlock = historyStr.String()

	return nil
}

func (h APIHandler) processQueryIntent(ctx context.Context, reqMsg, replyMsg *ChatMessage, params *processingParams) error {
	//query intent
	var err error

	var queryIntent *rag.QueryIntent
	{
		log.Info("start analysis user's intent")
		defer log.Info("end analysis user's intent")

		queryIntentBuffer := strings.Builder{}
		content := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant trained to understand and analyze user queries. The user has provided the following query:"),
			llms.TextParts(llms.ChatMessageTypeHuman, reqMsg.Message),
			llms.TextParts(llms.ChatMessageTypeSystem, "Please analyze the query and identify the user's primary intent. "+
				"Determine if they are looking for information, making a request, or seeking clarification. "+
				"Category the intent in </Category>, brief the </Intent>, and rephrase the query in several different forms to improve clarity. "+
				"Provide possible variations of the query in <Query/> and identify relevant keywords in </Keyword> in JSON array format. "+
				"Provide possible related of the query in <Suggestion/> and expand the related query for query suggestion. "+
				"Please make sure the output is concise, well-organized, and easy to process."+
				"Please present these possible query and keyword items in both English and Chinese."+
				"if the possible query is in English, keep the original English one, and translate it to Chinese and keep it as a new query, to be clear, you should output: [Apple, 苹果], neither just `Apple` nor just `苹果`."+
				"Wrap the JSON result in <JSON></JSON> tags. "+
				"Your output should look like this format:\n"+
				"<JSON>"+
				"{\n"+
				"  \"category\": \"<Intent's Category>\",\n"+
				"  \"intent\": \"<User's Intent>\",\n"+
				"  \"query\": [\n"+
				"    \"<新的查询 1>\",\n"+
				"    \"<Rephrased Query 2>\",\n"+
				"    \"<Rephrased Query 3>\"\n"+
				"  ],\n"+
				"  \"keyword\": [\n"+
				"    \"<关键字 1>\",\n"+
				"    \"<Keyword 2>\",\n"+
				"    \"<Keyword 3>\"\n"+
				"  ],\n"+
				"  \"suggestion\": [\n"+
				"    \"<Suggest Query 1>\",\n"+
				"    \"<Suggest Query 2>\",\n"+
				"    \"<Suggest Query 3>\"\n"+
				"  ]\n"+
				"}"+
				"</JSON>"),
		}

		llm := getLLM(params.intentModel)
		var chunkSeq = 0
		if _, err := llm.GenerateContent(ctx, content,
			llms.WithMaxTokens(1024),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				if len(chunk) > 0 {
					chunkSeq++
					queryIntentBuffer.Write(chunk)
					msg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, QueryIntent, string(chunk), chunkSeq)
					err := websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(msg))
					if err != nil {
						return err
					}
				}
				return nil
			})); err != nil {
			return err
		}

		if queryIntentBuffer.Len() > 0 {
			//extract the category and query
			params.queryIntentStr = queryIntentBuffer.String()
			queryIntent, err = rag.QueryAnalysisFromString(params.queryIntentStr)
			if err != nil {
				return err
			}
			replyMsg.Details = append(replyMsg.Details, ProcessingDetails{
				Order:   10,
				Type:    QueryIntent,
				Payload: queryIntent,
			})
		}
	}
	params.queryIntent = queryIntent
	return nil
}

func (h APIHandler) processInitialDocumentSearch(ctx context.Context, reqMsg, replyMsg *ChatMessage, params *processingParams, fechSize int) ([]common.Document, error) {
	var query *orm.Query
	mustClauses := search.BuildMustClauses(params.datasource, params.category, params.subcategory, params.richCategory, params.username, params.userid)
	var shouldClauses interface{}
	if params.queryIntent != nil && len(params.queryIntent.Query) > 0 {
		shouldClauses = search.BuildShouldClauses(params.queryIntent.Query, params.queryIntent.Keyword)
	}

	from := 0
	query = search.BuildTemplatedQuery(from, fechSize, mustClauses, shouldClauses, params.field, reqMsg.Message, params.source, params.tags)

	if query != nil {
		docs, err := fetchDocuments(query)
		if err != nil {
			return nil, err
		}

		simpliedReferences := formatDocumentReferencesToDisplay(docs)

		var chunkSeq = 0
		chunkMsg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID,
			FetchSource, string(simpliedReferences), chunkSeq)
		err = websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(chunkMsg))
		if err != nil {
			return nil, err
		}

		fetchedDocs := formatDocumentForPick(docs)
		{
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("<Payload total=%v>\n", len(docs)))
			sb.WriteString(util.MustToJSON(fetchedDocs))
			sb.WriteString("</Payload>")
			params.sourceDocsSummaryBlock = sb.String()
		}
		replyMsg.Details = append(replyMsg.Details, ProcessingDetails{Order: 20, Type: FetchSource, Payload: fetchedDocs})
		return docs, err
	}
	return nil, errors.Error("nothing found")
}

func (h APIHandler) processPickDocuments(ctx context.Context, reqMsg, replyMsg *ChatMessage, params *processingParams, docs []common.Document) ([]common.Document, error) {

	if len(docs) == 0 {
		return nil, nil
	}

	echoMsg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, PickSource, string(""), 0)
	websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(echoMsg))

	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant trained to understand and analyze user queries. "),

		//get history
		llms.TextParts(llms.ChatMessageTypeSystem, "You will be given a conversation below and a follow up question. "+
			"You need to rephrase the follow-up question if needed so it is a standalone question that can be used by the LLM to search the knowledge base for information.\n"+
			"Conversation: "),
		llms.TextParts(llms.ChatMessageTypeSystem, params.historyBlock),
		//end history

		llms.TextParts(llms.ChatMessageTypeSystem, "The user has provided the following query:"),
		llms.TextParts(llms.ChatMessageTypeHuman, reqMsg.Message),

		llms.TextParts(llms.ChatMessageTypeSystem, "The primary intent behind this query is:"),
		llms.TextParts(llms.ChatMessageTypeSystem, string(params.queryIntentStr)),

		llms.TextParts(llms.ChatMessageTypeSystem, "The following documents might be related to answering the user's query:"),

		llms.TextParts(llms.ChatMessageTypeSystem, params.sourceDocsSummaryBlock),

		llms.TextParts(llms.ChatMessageTypeSystem, "\nPlease review these documents and identify which ones best match the user's query. "+
			"Choose no more than 5 relevant documents. These documents may be entirely unrelated, so prioritize those that provide direct answers or valuable context."+
			"If the document is unrelated not certain, don't include it."+
			" For each document, provide a brief explanation of why it was selected, categorizing it in </Document> tags."+
			" Make sure the output is concise and easy to process."+
			" Wrap the JSON result in <JSON></JSON> tags."+
			" The expected output format is:\n"+
			"<JSON>\n"+
			"[\n"+
			" { \"id\": \"<id of Doc 1>\", \"title\": \"<title of Doc 1>\", \"explain\": \"<Explain for Doc 1>\"  },\n"+
			" { \"id\": \"<id of Doc 2>\", \"title\": \"<title of Doc 2>\", \"explain\": \"<Explain for Doc 2>\"  },\n"+
			"]"+
			"</JSON>"),
	}

	log.Info("start filtering documents")
	var pickedDocsBuffer = strings.Builder{}
	var chunkSeq = 0
	llm := getLLM(params.pickingDocModel)
	if _, err := llm.GenerateContent(ctx, content,
		llms.WithMaxTokens(32768),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if len(chunk) > 0 {
				chunkSeq++
				pickedDocsBuffer.Write(chunk)
				msg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, PickSource, string(chunk), chunkSeq)
				err := websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(msg))
				if err != nil {
					return err
				}
			}
			return nil
		})); err != nil {
		return nil, err
	}

	pickeDocs, err := rag.PickedDocumentFromString(pickedDocsBuffer.String())
	if err != nil {
		return nil, err
	}

	//log.Debug("filter document results:", pickedDocsBuffer.String())
	{
		detail := ProcessingDetails{Order: 30, Type: PickSource, Payload: pickeDocs}
		replyMsg.Details = append(replyMsg.Details, detail)
	}

	docsMap := map[string]common.Document{}
	for _, v := range docs {
		docsMap[v.ID] = v
	}

	var pickedDocIDS []string
	var pickedFullDoc = []common.Document{}
	for _, v := range pickeDocs {
		x, v1 := docsMap[v.ID]
		if v1 {
			pickedDocIDS = append(pickedDocIDS, v.ID)
			pickedFullDoc = append(pickedFullDoc, x)
			//log.Info("pick doc:", x.ID,",",x.Title)
		} else {
			log.Error("wrong doc id, doc is missing")
		}
	}
	params.pickedDocIDS = pickedDocIDS

	//replace to picked one
	docs = pickedFullDoc
	return docs, err
}

func (h APIHandler) fetchDocumentInDepth(ctx context.Context, reqMsg, replyMsg *ChatMessage, params *processingParams, docs []common.Document) error {
	if len(params.pickedDocIDS) > 0 {
		var query = orm.Query{}
		query.Conds = orm.And(orm.InStringArray("_id", params.pickedDocIDS))

		pickedFullDoc, err := fetchDocuments(&query)

		strBuilder := strings.Builder{}
		var chunkSeq = 0
		for _, v := range pickedFullDoc {
			str := "Obtaining and analyzing documents in depth:  " + string(v.Title) + "\n"
			strBuilder.WriteString(str)
			chunkMsg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, DeepRead, str, chunkSeq)
			err = websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(chunkMsg))
			if err != nil {
				return err
			}
		}

		detail := ProcessingDetails{Order: 40, Type: DeepRead, Description: strBuilder.String()}
		replyMsg.Details = append(replyMsg.Details, detail)

		params.references = formatDocumentForReplyReferences(pickedFullDoc)
	}
	return nil
}

func (h APIHandler) generateFinalResponse(taskCtx context.Context, reqMsg, replyMsg *ChatMessage, params *processingParams) error {
	//Retrieve related documents from background server

	prompt := fmt.Sprintf(`You are a friendly assistant designed to help users access and understand their personal or company data. 
Your responses should be clear, concise, and based solely on the information provided below. 
If the information is insufficient, please indicate that you need more details to assist effectively.

Conversation: 
%s

Query: 
%s

Data:
%s`, params.historyBlock, reqMsg.Message, params.references)

	// Prepare the system message
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, "You are a personal AI assistant designed by Coco AI(https://coco.rs), the backend team is behind INFINI Labs(https://infinilabs.com)."),
	}

	// Append the user's message
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, prompt))

	//response
	reasoningBuffer := strings.Builder{}
	messageBuffer := strings.Builder{}
	chunkSeq := 0
	var err error

	llm := getLLM(params.answeringModel) //deepseek-r1 /deepseek-v3
	appConfig := common.AppConfig()
	log.Info(params.answeringModel, ",", util.MustToJSON(appConfig))

	options := []llms.CallOption{}
	options = append(options, llms.WithMaxTokens(appConfig.LLMConfig.Parameters.MaxTokens))
	options = append(options, llms.WithMaxLength(appConfig.LLMConfig.Parameters.MaxLength))
	options = append(options, llms.WithTemperature(0.8))

	if appConfig.LLMConfig.Type == "deepseek" {
		llms.WithStreamingReasoningFunc(func(ctx context.Context, reasoningChunk []byte, chunk []byte) error {
			// Use taskCtx here to check for cancellation or other context-specific logic
			select {
			case <-ctx.Done(): // Check if the task has been canceled or has expired
				log.Warnf("Task for message %v canceled", reqMsg.ID)
				return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
			case <-taskCtx.Done(): // Check if the task has been canceled or has expired
				log.Warnf("Task for message %v canceled", reqMsg.ID)
				return taskCtx.Err() // Return the context error (canceled or deadline exceeded)
			default:

				//Handle the <Think> part
				if len(reasoningChunk) > 0 {
					chunkSeq += 1
					reasoningBuffer.Write(reasoningChunk)
					msg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, Think, string(reasoningChunk), chunkSeq)
					//log.Info(util.MustToJSON(msg))
					err = websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(msg))
					if err != nil {
						panic(err)
					}
					return nil
				}

				//Handle response
				if len(chunk) > 0 {
					chunkSeq += 1

					msg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, Response, string(chunk), chunkSeq)
					err = websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(msg))
					if err != nil {
						panic(err)
					}

					//log.Debug(msg)
					messageBuffer.Write(chunk)
				}

				return nil
			}

		})
	} else {
		//this part works for ollama
		options = append(options, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			chunkSeq += 1
			msg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, Response, string(chunk), chunkSeq)
			err = websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(msg))
			messageBuffer.Write(chunk)
			return nil
		}))
	}

	completion, err := llm.GenerateContent(taskCtx, content, options...)
	if err != nil {
		log.Error(err)
		return err
	}
	_ = completion

	chunkSeq += 1

	{
		detail := ProcessingDetails{Order: 50, Type: Think, Description: reasoningBuffer.String()}
		replyMsg.Details = append(replyMsg.Details, detail)
	}

	{
		detail := ProcessingDetails{Order: 60, Type: Response, Description: messageBuffer.String()}
		replyMsg.Details = append(replyMsg.Details, detail)
	}

	msg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeSystem, reqMsg.ID, ReplyEnd, "assistant finished output", chunkSeq)
	err = websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(msg))
	if err != nil {
		panic(err)
	}

	//log.Info(util.MustToJSON(replyMsg))

	//save response message to system
	if messageBuffer.Len() > 0 || len(replyMsg.Details) > 0 {
		replyMsg.Message = messageBuffer.String()
		err = orm.Save(nil, replyMsg)
		if err != nil {
			log.Error(err)
			return err
		}
	} else {
		log.Warnf("seems empty reply for query:", replyMsg)
	}
	return nil
}

func (h APIHandler) handleMessage(req *http.Request, sessionID, message string) (*ChatMessage, error) {
	if wsID, err := h.GetUserWebsocketID(req); err == nil && wsID != "" {
		params := h.extractParameters(req)
		reqMsg := h.createInitialUserRequestMessage(sessionID, message, params)
		params.sessionID = sessionID
		params.websocketID = wsID
		if err := h.saveMessage(reqMsg); err != nil {
			return nil, err
		}

		h.launchBackgroundTask(reqMsg, params)
		return reqMsg, nil
	} else {
		err := errors.Errorf("No websocket [%v] for session: %v", wsID, sessionID)
		log.Error(err)
		panic(err)
	}
}

func formatDocumentForReplyReferences(docs []common.Document) string {
	var sb strings.Builder
	sb.WriteString("<REFERENCES>\n")
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("<Doc>"))
		sb.WriteString(fmt.Sprintf("ID #%d - %v\n", i+1, doc.ID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", doc.Title))
		sb.WriteString(fmt.Sprintf("Source: %s\n", doc.Source))
		sb.WriteString(fmt.Sprintf("Updated: %s\n", doc.Updated))
		sb.WriteString(fmt.Sprintf("Category: %s\n", doc.GetAllCategories()))
		//sb.WriteString(fmt.Sprintf("Summary: %s\n", doc.Summary))
		sb.WriteString(fmt.Sprintf("Content: %s\n", doc.Content))
		sb.WriteString(fmt.Sprintf("</Doc>\n"))

	}
	sb.WriteString("</REFERENCES>")
	return sb.String()
}

func formatDocumentReferencesToDisplay(docs []common.Document) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<Payload total=%v>\n", len(docs)))
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
	sb.WriteString("</Payload>")
	return sb.String()
}

func formatDocumentForPick(docs []common.Document) []util.MapStr {
	outDocs := []util.MapStr{}
	for _, doc := range docs {
		item := util.MapStr{}
		item["id"] = doc.ID
		item["title"] = doc.Title
		item["updated"] = doc.Updated
		item["icon"] = doc.Icon
		item["size"] = doc.Size
		item["thumbnail"] = doc.Thumbnail
		item["category"] = doc.Category
		item["summary"] = util.SubString(doc.Summary, 0, 500)
		item["url"] = doc.URL
		outDocs = append(outDocs, item)
	}
	return outDocs
}

func fetchDocuments(query *orm.Query) ([]common.Document, error) {
	var docs []common.Document
	err, _ := orm.SearchWithJSONMapper(&docs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}
	return docs, nil
}

func getLLM(model string) llms.Model {
	cfg := common.AppConfig()
	if model == "" {
		model = cfg.LLMConfig.DefaultModel
	}

	log.Debug("use model:", model)

	if cfg.LLMConfig.Type == common.OLLAMA {
		llm, err := ollama.New(
			ollama.WithServerURL(cfg.LLMConfig.Endpoint),
			ollama.WithModel(model),
			ollama.WithKeepAlive(cfg.LLMConfig.Keepalive))
		if err != nil {
			panic(err)
		}
		return llm

	} else {
		llm, err := openai.New(
			openai.WithToken(cfg.LLMConfig.Token),
			openai.WithBaseURL(cfg.LLMConfig.Endpoint),
			openai.WithModel(model),
		)
		if err != nil {
			panic(err)
		}
		return llm
	}
}
