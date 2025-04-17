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
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
	"net/http"
	"runtime"
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
	pickedDocIDS           []string
	references             string

	intentModel         *common.ModelConfig
	pickingDocModel     *common.ModelConfig
	answeringModel      *common.ModelConfig
	assistantID         string
	keepalive           string
	intentModelProvider *common.ModelProvider
	pickingDocProvider  *common.ModelProvider
	answeringProvider   *common.ModelProvider
}

const DefaultAssistantID = "default"

func (h APIHandler) extractParameters(req *http.Request) (*processingParams, error) {
	params := &processingParams{
		searchDB:     h.GetBoolOrDefault(req, "search", false),
		deepThink:    h.GetBoolOrDefault(req, "deep_thinking", false),
		from:         h.GetIntOrDefault(req, "from", 0),
		size:         h.GetIntOrDefault(req, "size", 10),
		datasource:   h.GetParameterOrDefault(req, "datasource", ""),
		category:     h.GetParameterOrDefault(req, "category", ""),
		username:     h.GetParameterOrDefault(req, "username", ""),
		userid:       h.GetParameterOrDefault(req, "userid", ""),
		tags:         h.GetParameterOrDefault(req, "tags", ""),
		subcategory:  h.GetParameterOrDefault(req, "subcategory", ""),
		richCategory: h.GetParameterOrDefault(req, "rich_category", ""),
		field:        h.GetParameterOrDefault(req, "search_field", "title"),
		source:       h.GetParameterOrDefault(req, "source_fields", "*"),
	}

	assistantID := h.GetParameterOrDefault(req, "assistant_id", DefaultAssistantID)
	params.assistantID = assistantID

	assistant, err := common.GetAssistant(assistantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant: %w", err)
	}
	if assistant == nil {
		return nil, fmt.Errorf("assistant [%s] is not found", assistantID)
	}
	if !assistant.Enabled {
		return nil, fmt.Errorf("assistant [%s] is not enabled", assistant.Name)
	}

	if assistant.Datasource.Enabled && len(assistant.Datasource.IDs) > 0 {
		if params.datasource == "" {
			params.datasource = strings.Join(assistant.Datasource.IDs, ",")
		} else {
			// calc intersection with datasource and assistant datasourceIDs
			queryDatasource := strings.Split(params.datasource, ",")
			queryDatasource = util.StringArrayIntersection(queryDatasource, assistant.Datasource.IDs)
			if len(queryDatasource) == 0 {
				//TODO: handle logic of empty datasource
			}
			params.datasource = strings.Join(queryDatasource, ",")
		}
	}
	params.keepalive = assistant.Keepalive
	if params.deepThink {
		if assistant.Type == common.AssistantTypeDeepThink {
			deepThinkCfg := common.DeepThinkConfig{}
			buf := util.MustToJSONBytes(assistant.Config)
			util.MustFromJSONBytes(buf, &deepThinkCfg)

			// set intent analysis model params
			params.pickingDocModel = &deepThinkCfg.PickingDocModel
			modelProvider, err := common.GetModelProvider(deepThinkCfg.PickingDocModel.ProviderID)
			if err != nil {
				return nil, fmt.Errorf("failed to get picking doc model provider: %w", err)
			}
			params.pickingDocProvider = modelProvider

			// set picking doc model params
			params.intentModel = &deepThinkCfg.IntentAnalysisModel
			modelProvider, err = common.GetModelProvider(deepThinkCfg.IntentAnalysisModel.ProviderID)
			if err != nil {
				return nil, fmt.Errorf("failed to get intent model provider: %w", err)
			}
			params.intentModelProvider = modelProvider
		} else {
			// reset deepThink to false if assistant is not deep think type
			params.deepThink = false
		}
	}
	if assistant.AnsweringModel.ProviderID == "" {
		return nil, fmt.Errorf("assistant [%s] has no answering model configured. Please set it up first", assistant.Name)
	}
	modelProvider, err := common.GetModelProvider(assistant.AnsweringModel.ProviderID)
	if err != nil {
		return params, fmt.Errorf("failed to get model provider: %w", err)
	}
	params.answeringModel = &assistant.AnsweringModel
	params.answeringProvider = modelProvider

	return params, nil
}

func (h APIHandler) createInitialUserRequestMessage(sessionID, assistantID, message string, params *processingParams) *ChatMessage {
	msg := &ChatMessage{
		SessionID:   sessionID,
		AssistantID: assistantID,
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

func (h APIHandler) createAssistantMessage(sessionID, assistantID, requestMessageID string) *ChatMessage {
	msg := &ChatMessage{
		SessionID:      sessionID,
		MessageType:    MessageTypeAssistant,
		ReplyMessageID: requestMessageID,
		AssistantID:    assistantID,
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

	replyMsg := h.createAssistantMessage(params.sessionID, reqMsg.AssistantID, reqMsg.ID)

	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}
				msg := fmt.Sprintf("⚠️ error in async processing message reply, %v", v)
				if replyMsg.Message == "" {
					replyMsg.Message = msg
					h.sendWebsocketMessage(params.websocketID, NewMessageChunk(
						params.sessionID, replyMsg.ID, MessageTypeSystem, reqMsg.ID,
						Response, msg, 0,
					))
				}
				log.Error(msg)
			}
		}
		h.finalizeProcessing(ctx, params.websocketID, params.sessionID, replyMsg)
	}()

	reqMsg.Details = make([]ProcessingDetails, 0)

	var docs []common.Document
	// Processing pipeline
	_ = h.fetchSessionHistory(ctx, reqMsg, replyMsg, params, 10)

	if params.deepThink && params.intentModel != nil {
		_ = h.processQueryIntent(ctx, reqMsg, replyMsg, params)
	}

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
		historyStr.WriteString(v.MessageType + ": " + util.SubStringWithSuffix(v.Message, 250, "..."))
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
		log.Debug("start analysis user's intent")
		defer log.Debug("end analysis user's intent")

		queryIntentBuffer := strings.Builder{}
		content := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, "You are an AI assistant trained to understand and analyze user queries.\n"),
			llms.TextParts(llms.ChatMessageTypeSystem, "You will be given a conversation below and a follow up question. "+
				"You need to rephrase the follow-up question if needed so it is a standalone question that can be used by the LLM to search the knowledge base for information.\n"+
				"Conversation: \n"),
			llms.TextParts(llms.ChatMessageTypeSystem, params.historyBlock),
			llms.TextParts(llms.ChatMessageTypeSystem, "The user has provided the following query:"),
			llms.TextParts(llms.ChatMessageTypeHuman, reqMsg.Message),
			llms.TextParts(llms.ChatMessageTypeSystem, "\nPlease analyze the query and identify the user's primary intent. "+
				"Determine if they are looking for information, making a request, or seeking clarification. "+
				"Category the intent in </Category>, brief the </Intent>, and rephrase the query in several different forms to improve clarity. "+
				"Provide possible variations of the query in <Query/> and identify relevant keywords in </Keyword> in JSON array format. "+
				"Provide possible related of the query in <Suggestion/> and expand the related query for query suggestion. "+
				"Please make sure the output is concise, well-organized, and easy to process."+
				"Please present these possible query and keyword items in both English and Chinese."+
				"if the possible query is in English, keep the original English one, and translate it to Chinese and keep it as a new query, to be clear, you should output: [Apple, 苹果], neither just `Apple` nor just `苹果`."+
				"Wrap the valid JSON result in <JSON></JSON> tags. "+
				"Your output should look like this format:\n"+
				"<JSON>"+
				"{\n"+
				"  \"category\": \"<Intent's Category>\",\n"+
				"  \"intent\": \"<User's Intent>\",\n"+
				"  \"query\": [\n"+
				"    \"<新的查询 1>\",\n"+
				"    \"<Rephrased Query 2>\",\n"+
				"    \"<Rephrased Query 3>\"\n"+
				"    \"<Rephrased Query N>\"\n"+
				"  ],\n"+
				"  \"keyword\": [\n"+
				"    \"<关键字 1>\",\n"+
				"    \"<Keyword 2>\",\n"+
				"    \"<Keyword 3>\"\n"+
				"    \"<Keyword N>\"\n"+
				"  ],\n"+
				"  \"suggestion\": [\n"+
				"    \"<Suggest Query 1>\",\n"+
				"    \"<Suggest Query 2>\",\n"+
				"    \"<Suggest Query 3>\"\n"+
				"    \"<Suggest Query N>\"\n"+
				"  ]\n"+
				"}"+
				"</JSON>"),
		}

		llm := getLLM(params.intentModelProvider.BaseURL, params.intentModelProvider.APIType, params.intentModel.Name, params.intentModelProvider.APIKey, params.keepalive)
		log.Trace(content)

		var chunkSeq = 0
		temperature := getTemperature(params.intentModel, params.intentModelProvider, 0.8)
		maxTokens := getMaxTokens(params.intentModel, params.intentModelProvider, 1024)
		if _, err := llm.GenerateContent(ctx, content,
			llms.WithTemperature(temperature),
			llms.WithMaxTokens(maxTokens),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				if len(chunk) > 0 {
					chunkSeq++
					queryIntentBuffer.Write(chunk)
					msg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, QueryIntent, string(chunk), chunkSeq)
					err := websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(msg))
					if err != nil {
						log.Error(err)
						return err
					}
				}
				return nil
			})); err != nil {
			return err
		}

		if queryIntentBuffer.Len() > 0 {
			//extract the category and query
			str := queryIntentBuffer.String()
			log.Trace("query intent: ", str)
			queryIntent, err = rag.QueryAnalysisFromString(str)
			if err != nil {
				log.Error(err)
				return err
			}
			log.Debug("queryIntent:", util.MustToJSON(queryIntent))
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

func getTemperature(model *common.ModelConfig, modelProvider *common.ModelProvider, defaultValue float64) float64 {
	temperature := 0.0
	if model.Settings.Temperature > 0 {
		temperature = model.Settings.Temperature
	}
	if temperature == 0 {
		for _, m := range modelProvider.Models {
			if m.Name == model.Name {
				if m.Settings.Temperature > 0 {
					temperature = m.Settings.Temperature
				}
				break
			}
		}
	}
	if temperature == 0 {
		temperature = defaultValue
	}
	return temperature
}

func getMaxLength(model *common.ModelConfig, modelProvider *common.ModelProvider, defaultValue int) int {
	maxLength := 0
	if model.Settings.MaxLength > 0 {
		maxLength = model.Settings.MaxLength
	}
	if maxLength == 0 {
		for _, m := range modelProvider.Models {
			if m.Name == model.Name {
				if m.Settings.MaxLength > 0 {
					maxLength = m.Settings.MaxLength
				}
				break
			}
		}
	}
	if maxLength == 0 {
		maxLength = defaultValue
	}
	return maxLength
}

func getMaxTokens(model *common.ModelConfig, modelProvider *common.ModelProvider, defaultValue int) int {
	var maxTokens int = 0
	if model.Settings.MaxTokens > 0 {
		maxTokens = model.Settings.MaxTokens
	}
	if maxTokens == 0 {
		for _, m := range modelProvider.Models {
			if m.Name == model.Name {
				if m.Settings.MaxTokens > 0 {
					maxTokens = m.Settings.MaxTokens
				}
				break
			}
		}
	}
	if maxTokens == 0 {
		maxTokens = defaultValue
	}
	return maxTokens
}

func (h APIHandler) processInitialDocumentSearch(ctx context.Context, reqMsg, replyMsg *ChatMessage, params *processingParams, fechSize int) ([]common.Document, error) {
	var query *orm.Query
	mustClauses := search.BuildMustClauses(params.category, params.subcategory, params.richCategory, params.username, params.userid)
	datasourceClause := search.BuildDatasourceClause(params.datasource, true)
	if datasourceClause != nil {
		mustClauses = append(mustClauses, datasourceClause)
	}
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

		{
			simpliedReferences := formatDocumentReferencesToDisplay(docs)
			var chunkSeq = 0
			chunkMsg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID,
				FetchSource, string(simpliedReferences), chunkSeq)

			err = websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(chunkMsg))
			if err != nil {
				return nil, err
			}
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
		llms.TextParts(
			llms.ChatMessageTypeSystem,
			`You are an AI assistant trained to select the most relevant documents for further processing and to answer user queries.
We have already queried the backend database and retrieved a list of documents that may help answer the user's query. 
Your task is to choose the best documents for further processing.`,
		),
	}

	content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, "The user has provided the following query:\n"))
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, reqMsg.Message))

	if params.queryIntent != nil {
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, "The primary intent behind this query is:\n"))
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, util.MustToJSON(params.queryIntent)))
	}

	if params.sourceDocsSummaryBlock != "" {
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, "The following documents are fetched from database:\n"))
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, params.sourceDocsSummaryBlock))
	}

	content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, "\nPlease review these documents and identify which ones best related to user's query. "+
		"Choose no more than 5 relevant documents. These documents may be entirely unrelated, so prioritize those that provide direct answers or valuable context."+
		"If the document is unrelated not certain, don't include it."+
		" For each document, provide a brief explanation of why it was selected."+
		" Your decision should based solely on the information provided below. \nIf the information is insufficient, please indicate that you need more details to assist effectively. "+
		" Don't make anything up, which means if you can't identify which document best match the user's query, you should output nothing."+
		" Make sure the output is concise and easy to process."+
		" Wrap the JSON result in <JSON></JSON> tags."+
		" The expected output format is:\n"+
		"<JSON>\n"+
		"[\n"+
		" { \"id\": \"<id of Doc 1>\", \"title\": \"<title of Doc 1>\", \"explain\": \"<Explain for Doc 1>\"  },\n"+
		" { \"id\": \"<id of Doc 2>\", \"title\": \"<title of Doc 2>\", \"explain\": \"<Explain for Doc 2>\"  },\n"+
		"]"+
		"</JSON>"))

	log.Debug("start filtering documents")
	var pickedDocsBuffer = strings.Builder{}
	var chunkSeq = 0
	llm := getLLM(params.pickingDocProvider.BaseURL, params.pickingDocProvider.APIType, params.pickingDocModel.Name, params.pickingDocProvider.APIKey, params.keepalive)
	log.Trace(content)
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

	log.Debug(pickedDocsBuffer.String())

	pickeDocs, err := rag.PickedDocumentFromString(pickedDocsBuffer.String())
	if err != nil {
		return nil, err
	}

	log.Debug("filter document results:", pickeDocs)

	docsMap := map[string]common.Document{}
	for _, v := range docs {
		docsMap[v.ID] = v
	}

	var pickedDocIDS []string
	var pickedFullDoc = []common.Document{}
	var validPickedDocs = []rag.PickedDocument{}
	for _, v := range pickeDocs {
		x, v1 := docsMap[v.ID]
		if v1 {
			pickedDocIDS = append(pickedDocIDS, v.ID)
			pickedFullDoc = append(pickedFullDoc, x)
			validPickedDocs = append(validPickedDocs, v)
			log.Debug("pick doc:", x.ID, ",", x.Title)
		} else {
			log.Error("wrong doc id, doc is missing")
		}
	}

	{
		detail := ProcessingDetails{Order: 30, Type: PickSource, Payload: validPickedDocs}
		replyMsg.Details = append(replyMsg.Details, detail)
	}

	params.pickedDocIDS = pickedDocIDS

	log.Debug("valid picked document results:", validPickedDocs)

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

	echoMsg := NewMessageChunk(params.sessionID, replyMsg.ID, MessageTypeAssistant, reqMsg.ID, Response, string(""), 0)
	websocket.SendPrivateMessage(params.websocketID, util.MustToJSON(echoMsg))

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
	if params.answeringProvider == nil {
		return errors.Errorf("no answering provider with assistant: %v", params.assistantID)
	}

	llm := getLLM(params.answeringProvider.BaseURL, params.answeringProvider.APIType, params.answeringModel.Name, params.answeringProvider.APIKey, params.keepalive) //deepseek-r1 /deepseek-v3
	appConfig := common.AppConfig()

	log.Trace(params.answeringModel, ",", util.MustToJSON(appConfig))

	options := []llms.CallOption{}
	maxTokens := getMaxTokens(params.answeringModel, params.answeringProvider, 1024)
	temperature := getTemperature(params.answeringModel, params.answeringProvider, 0.8)
	maxLength := getMaxLength(params.answeringModel, params.answeringProvider, 0)
	options = append(options, llms.WithMaxTokens(maxTokens))
	options = append(options, llms.WithMaxLength(maxLength))
	options = append(options, llms.WithTemperature(temperature))

	if params.answeringProvider.APIType == common.DEEPSEEK {
		options = append(options, llms.WithStreamingReasoningFunc(func(ctx context.Context, reasoningChunk []byte, chunk []byte) error {
			log.Trace(string(reasoningChunk), ",", string(chunk))
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

		}))
	} else {
		//this part works for ollama
		options = append(options, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {

			log.Trace(string(chunk))

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

	//save response message to system
	if messageBuffer.Len() > 0 || len(replyMsg.Details) > 0 {
		replyMsg.Message = messageBuffer.String()
	} else {
		log.Warnf("seems empty reply for query:", replyMsg)
	}
	return nil
}

func (h APIHandler) handleMessage(req *http.Request, sessionID, assistantID, message string) (*ChatMessage, error) {
	if wsID, err := h.GetUserWebsocketID(req); err == nil && wsID != "" {
		params, err := h.extractParameters(req)
		if err != nil {
			return nil, err
		}
		reqMsg := h.createInitialUserRequestMessage(sessionID, assistantID, message, params)
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
		//item["updated"] = doc.Updated
		//item["category"] = doc.Category
		//item["summary"] = doc.Summary
		item["icon"] = doc.Icon
		//item["size"] = doc.Size
		//item["thumbnail"] = doc.Thumbnail
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

func getLLM(endpoint, apiType, model, token string, keepalive string) llms.Model {
	if model == "" {
		panic("model is empty")
	}

	log.Debug("use model:", model, ",type:", apiType)

	if apiType == common.OLLAMA {
		llm, err := ollama.New(
			ollama.WithServerURL(endpoint),
			ollama.WithModel(model),
			ollama.WithKeepAlive(keepalive))
		if err != nil {
			panic(err)
		}
		return llm

	} else {
		llm, err := openai.New(
			openai.WithToken(token),
			openai.WithBaseURL(endpoint),
			openai.WithModel(model),
		)
		if err != nil {
			panic(err)
		}
		return llm
	}
}
