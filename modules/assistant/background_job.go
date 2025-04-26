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
	"infini.sh/coco/modules/assistant/rag"
	"infini.sh/coco/modules/assistant/websocket"
	"net/http"
	"runtime"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	langchaingoTools "github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/duckduckgo"
	"github.com/tmc/langchaingo/tools/scraper"
	"github.com/tmc/langchaingo/tools/wikipedia"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/search"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

// Helper types and methods
type RAGContext struct {
	SearchDB     bool
	DeepThink    bool
	MCP          bool
	From         int
	Size         int
	mcpServers   []string
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
	WebsocketID string
	SessionID   string

	//prepare for final response
	sourceDocsSummaryBlock string

	//history
	HistoryBlock string
	chatHistory  *memory.ChatMessageHistory

	QueryIntent  *rag.QueryIntent
	pickedDocIDS []string
	references   string

	intentModel         *common.ModelConfig
	pickingDocModel     *common.ModelConfig
	answeringModel      *common.ModelConfig
	assistantID         string
	intentModelProvider *common.ModelProvider
	pickingDocProvider  *common.ModelProvider
	answeringProvider   *common.ModelProvider
	AssistantCfg        *common.Assistant

	toolsCallResponse string
}

const DefaultAssistantID = "default"

func (h APIHandler) extractParameters(req *http.Request) (*RAGContext, error) {
	params := &RAGContext{
		SearchDB:     h.GetBoolOrDefault(req, "search", false),
		DeepThink:    h.GetBoolOrDefault(req, "deep_thinking", false),
		MCP:          h.GetBoolOrDefault(req, "mcp", false),
		From:         h.GetIntOrDefault(req, "from", 0),
		Size:         h.GetIntOrDefault(req, "size", 10),
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

	if v := h.GetParameterOrDefault(req, "mcp_servers", ""); v != "" {
		params.mcpServers = strings.Split(v, ",")
	}

	assistantID := h.GetParameterOrDefault(req, "assistant_id", DefaultAssistantID)
	params.assistantID = assistantID

	assistant, err := common.GetAssistant(assistantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant with id [%v]: %w", assistantID, err)
	}
	if assistant == nil {
		return nil, fmt.Errorf("assistant [%s] is not found", assistantID)
	}
	if !assistant.Enabled {
		return nil, fmt.Errorf("assistant [%s] is not enabled", assistant.Name)
	}

	params.AssistantCfg = assistant

	if assistant.Datasource.Enabled && len(assistant.Datasource.IDs) > 0 {
		if params.datasource == "" {
			params.datasource = strings.Join(assistant.Datasource.IDs, ",")
		} else {
			// calc intersection with datasource and assistant datasourceIDs
			queryDatasource := strings.Split(params.datasource, ",")
			queryDatasource = util.StringArrayIntersection(queryDatasource, assistant.Datasource.IDs)
			params.datasource = strings.Join(queryDatasource, ",")
		}
	}

	log.Trace(assistant.MCPConfig.Enabled, assistant.MCPConfig.IDs, ",", params.mcpServers)

	if params.MCP && assistant.MCPConfig.Enabled && len(assistant.MCPConfig.IDs) > 0 {
		if len(params.mcpServers) == 0 {
			params.mcpServers = assistant.MCPConfig.IDs
		} else {
			// calc intersection with datasource and assistant datasourceIDs
			queryMcpServers := params.mcpServers
			queryMcpServers = util.StringArrayIntersection(queryMcpServers, assistant.MCPConfig.IDs)
			params.mcpServers = queryMcpServers
		}
	} else {
		params.mcpServers = make([]string, 0)
	}

	if params.DeepThink {
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
			// reset DeepThink to false if assistant is not deep think type
			params.DeepThink = false
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

func (h APIHandler) createInitialUserRequestMessage(sessionID, assistantID, message string, params *RAGContext) *common.ChatMessage {

	msg := &common.ChatMessage{
		SessionID:   sessionID,
		AssistantID: assistantID,
		MessageType: common.MessageTypeUser,
		Message:     message,
	}
	msg.ID = util.GetUUID()

	if params.SearchDB {
		msg.Parameters = util.MapStr{"params": params}
	}
	return msg
}

func (h APIHandler) saveMessage(msg *common.ChatMessage) error {
	return orm.Create(nil, msg)
}

func (h APIHandler) launchBackgroundTask(msg *common.ChatMessage, params *RAGContext) {

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
		taskID, params.SessionID, params.WebsocketID)

	inflightMessages.Store(params.SessionID, MessageTask{
		SessionID:   params.SessionID,
		TaskID:      taskID,
		WebsocketID: params.WebsocketID,
	})
	log.Infof("Saved taskID: %v for session: %v", taskID, params.SessionID)
}

func (h APIHandler) createAssistantMessage(sessionID, assistantID, requestMessageID string) *common.ChatMessage {
	msg := &common.ChatMessage{
		SessionID:      sessionID,
		MessageType:    common.MessageTypeAssistant,
		ReplyMessageID: requestMessageID,
		AssistantID:    assistantID,
	}
	msg.ID = util.GetUUID()

	return msg
}

// WebSocket helper
func (h APIHandler) sendWebsocketMessage(wsID string, msg *common.MessageChunk) {
	if err := websocket.SendMessageToWebsocket(wsID, util.MustToJSON(msg)); err != nil {
		log.Warnf("WebSocket send error: %v", err)
	}
}

func (h APIHandler) finalizeProcessing(ctx context.Context, wsID, sessionID string, msg *common.ChatMessage) {
	if err := orm.Save(nil, msg); err != nil {
		log.Errorf("Failed to save assistant message: %v", err)
	}

	h.sendWebsocketMessage(wsID, common.NewMessageChunk(
		sessionID, msg.ID, common.MessageTypeSystem, msg.ReplyMessageID,
		common.ReplyEnd, "Processing completed", 0,
	))
}

func (h APIHandler) processMessageAsync(ctx context.Context, reqMsg *common.ChatMessage, params *RAGContext) error {
	log.Debugf("Starting async processing for session: %v", params.SessionID)

	replyMsg := h.createAssistantMessage(params.SessionID, reqMsg.AssistantID, reqMsg.ID)

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
					h.sendWebsocketMessage(params.WebsocketID, common.NewMessageChunk(
						params.SessionID, replyMsg.ID, common.MessageTypeSystem, reqMsg.ID,
						common.Response, msg, 0,
					))
				}
				log.Error(msg)
			}
		}
		h.finalizeProcessing(ctx, params.WebsocketID, params.SessionID, replyMsg)
	}()

	reqMsg.Details = make([]common.ProcessingDetails, 0)

	var docs []common.Document

	// Prepare input values
	inputValues := map[string]any{
		"query": reqMsg.Message,
	}

	// Processing pipeline
	//log.Error("num of history: ", params.AssistantCfg.ChatSettings.HistoryMessage.Number)
	if params.AssistantCfg.ChatSettings.HistoryMessage.Number > 0 {
		history, _ := h.fetchSessionHistory(ctx, reqMsg, replyMsg, params, params.AssistantCfg.ChatSettings.HistoryMessage.Number, inputValues)
		inputValues["history"] = history
	} else {
		inputValues["history"] = "</empty>"
	}

	if params.DeepThink && params.intentModel != nil {
		queryIntent, err := rag.ProcessQueryIntent(ctx, params.SessionID, params.WebsocketID, params.intentModelProvider, params.intentModel, reqMsg, replyMsg, params.AssistantCfg, inputValues)
		if err != nil {
			log.Error("error on processing query intent analysis: ", err)
		}
		// Store the query intent in the processing parameters
		params.QueryIntent = queryIntent
	}

	if (params.AssistantCfg.MCPConfig.Enabled && len(params.mcpServers) > 0) || params.AssistantCfg.ToolsConfig.Enabled {
		//process LLM tools / functions
		err := h.processLLMTools(ctx, reqMsg, replyMsg, params, inputValues)
		if err != nil {
			log.Error(err)
		}
	}

	if params.SearchDB {
		var fetchSize = 10
		if params.DeepThink {
			fetchSize = 50
		}
		docs, _ = h.processInitialDocumentSearch(ctx, reqMsg, replyMsg, params, fetchSize)

		if params.DeepThink && len(docs) > 10 {
			//re-pick top docs
			docs, _ = h.processPickDocuments(ctx, reqMsg, replyMsg, params, docs)
			_ = h.fetchDocumentInDepth(ctx, reqMsg, replyMsg, params, docs, inputValues)
		}
	}

	h.generateFinalResponse(ctx, reqMsg, replyMsg, params, inputValues)
	log.Info("async reply task done for query:", reqMsg.Message)
	return nil
}

func (h APIHandler) fetchSessionHistory(ctx context.Context, reqMsg, replyMsg *common.ChatMessage, params *RAGContext, size int, inputValues map[string]any) (string, error) {
	var historyStr = strings.Builder{}

	chatHistory := memory.NewChatMessageHistory(memory.WithPreviousMessages([]llms.ChatMessage{}))

	//get chat history
	history, err := getChatHistoryBySessionInternal(params.SessionID, size)
	if err != nil {
		return "", err
	}

	if len(history) <= 1 {
		return "", nil
	}

	historyStr.WriteString("<conversation>\n")

	for i := len(history) - 1; i >= 0; i-- {
		v := history[i]
		msgText := util.SubStringWithSuffix(v.Message, 500, "...")
		switch v.MessageType {
		case common.MessageTypeSystem:
			msg := llms.SystemChatMessage{Content: msgText}
			chatHistory.AddMessage(context.Background(), msg)
			break
		case common.MessageTypeAssistant:
			msg := llms.AIChatMessage{Content: msgText}
			chatHistory.AddMessage(context.Background(), msg)
			break
		case common.MessageTypeUser:
			msg := llms.HumanChatMessage{Content: msgText}
			chatHistory.AddMessage(context.Background(), msg)
			break
		}

		historyStr.WriteString(v.MessageType + ": " + msgText)
		if v.DownVote > 0 {
			historyStr.WriteString(fmt.Sprintf("(%v people up voted this answer)", v.UpVote))
		}
		if v.DownVote > 0 {
			historyStr.WriteString(fmt.Sprintf("(%v people down voted this answer)", v.DownVote))
		}
		historyStr.WriteString("\n\n")
	}
	historyStr.WriteString("</conversation>")

	params.chatHistory = chatHistory

	return historyStr.String(), nil
}

func (h *APIHandler) processLLMTools(ctx context.Context, reqMsg *common.ChatMessage, replyMsg *common.ChatMessage, params *RAGContext, inputValues map[string]any) error {
	if params == nil || params.AssistantCfg == nil {
		//return nil
		panic("invalid assistant config, skip")
	}

	//get llm for mcp, use answering model if not mcp specified model
	providerID := params.answeringModel.ProviderID
	modelName := params.answeringModel.Name
	if params.AssistantCfg.MCPConfig.Enabled {
		if params.AssistantCfg.MCPConfig.Model != nil {
			if params.AssistantCfg.MCPConfig.Model.Name != "" {
				modelName = params.AssistantCfg.MCPConfig.Model.Name
				providerID = params.AssistantCfg.MCPConfig.Model.ProviderID
			}
		}
	}

	modelProvider, err := common.GetModelProvider(providerID)
	if err != nil {
		return err
	}

	llm := langchain.GetLLM(modelProvider.BaseURL, modelProvider.APIType, modelName, modelProvider.APIKey, params.AssistantCfg.Keepalive)
	agentTools := []langchaingoTools.Tool{}

	if params.AssistantCfg.ToolsConfig.Enabled {
		webAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

		if params.AssistantCfg.ToolsConfig.BuiltinTools.Calculator {
			agentTools = append(agentTools, langchaingoTools.Calculator{})
		}

		if params.AssistantCfg.ToolsConfig.BuiltinTools.Wikipedia {
			wp := wikipedia.New(webAgent)
			agentTools = append(agentTools, wp)
		}

		if params.AssistantCfg.ToolsConfig.BuiltinTools.Duckduckgo {
			ddg, err := duckduckgo.New(50, webAgent)
			if err == nil && ddg != nil {
				agentTools = append(agentTools, ddg)
			}
		}

		if params.AssistantCfg.ToolsConfig.BuiltinTools.Scraper {
			scr, err := scraper.New()
			if err == nil && scr != nil {
				agentTools = append(agentTools, scr)
			}
		}
	}

	mcpClients := []*client.Client{}
	defer func() {
		for _, f := range mcpClients {
			f.Close()
		}
	}()

	log.Debug("found total ", len(params.mcpServers), " mcp servers")

	for _, id := range params.mcpServers {
		v, err := common.GetMPCServer(id)
		if err != nil {
			panic(err)
		}
		if v == nil {
			panic("invalid mcp server")
		}

		log.Tracef("start init mcp server: %v, %v", v.Name, v.Type)

		if !v.Enabled {
			continue
		}

		var mcpClient *client.Client
		switch v.Type {
		case common.StreamableHTTP:
			bytes := util.MustToJSONBytes(v.Config)
			cfg := common.StreamableHttpConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				return err
			}

			mcpClient, err = client.NewStreamableHttpClient(cfg.URL)
			if err != nil {
				return fmt.Errorf("new mcp adapter: %w", err)
			}
			break
		case common.SSE:
			bytes := util.MustToJSONBytes(v.Config)
			cfg := common.SSEConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				return err
			}

			mcpClient, err = client.NewSSEMCPClient(cfg.URL)
			if err != nil {
				return fmt.Errorf("new mcp adapter: %w", err)
			}
			if err := mcpClient.Start(context.Background()); err != nil {
				return fmt.Errorf("new mcp adapter: %w", err)
			}

			break
		case common.Stdio:
			bytes := util.MustToJSONBytes(v.Config)

			cfg := common.StdioConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				return err
			}
			envs := []string{}
			if len(cfg.Env) > 0 {
				for k, v := range cfg.Env {
					envs = append(envs, fmt.Sprintf("%v=%v", k, v))
				}
			}
			mcpClient, err = client.NewStdioMCPClient(cfg.Command, envs, cfg.Args...)
			if err != nil {
				return fmt.Errorf("error on new stdio client: %w", err)
			}
			//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			//defer cancel()
			if err := mcpClient.Start(context.Background()); err != nil {
				return fmt.Errorf("error on start stdio client: %w", err)
			}
			break
		default:
			panic("unknown type")
		}

		if mcpClient != nil {
			mcpClients = append(mcpClients, mcpClient)
			mcpAdapter, err := langchain.New(mcpClient)
			if err != nil {
				return fmt.Errorf("new mcp adapter: %w", err)
			}

			mcpTools, err := mcpAdapter.Tools()
			log.Tracef("get %v tools from mcp server: %v", v.Name)
			if err != nil {
				return fmt.Errorf("append tools: %w", err)
			}
			agentTools = append(agentTools, mcpTools...)
		}

		log.Tracef("end init mcp server: %v", v.Name)
	}

	if len(agentTools) <= 0 {
		log.Debug("total get ", len(agentTools), " tools")
		return nil
	}

	buffer := memory.NewConversationBuffer()
	if params.chatHistory != nil {
		buffer.ChatHistory = params.chatHistory
	}

	callback := langchain.LogHandler{}
	toolsSeq := 0
	callback.CustomWriteFunc = func(chunk string) {
		if chunk != "" {
			echoMsg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.Tools, chunk, toolsSeq)
			websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(echoMsg))
		}
		toolsSeq++
	}

	executor, err := agents.Initialize(
		llm,
		agentTools,
		agents.ConversationalReactDescription,
		//agents.WithReturnIntermediateSteps(),
		agents.WithMaxIterations(params.AssistantCfg.MCPConfig.MaxIterations),
		agents.WithCallbacksHandler(&callback),
		agents.WithMemory(buffer),
	)
	if err != nil {
		return fmt.Errorf("error on executor: %w", err)
	}

	log.Debugf("start call LLM tools")
	answer, err := chains.Run(context.Background(), executor, reqMsg.Message)
	if err != nil {
		log.Error(answer, err)
		return fmt.Errorf("error running chains: %w", err)
	}

	log.Debugf("end call LLM tools")

	inputValues["tools_output"] = answer

	log.Debug("MCP call answer:", answer)

	return nil
}

func (h APIHandler) processInitialDocumentSearch(ctx context.Context, reqMsg, replyMsg *common.ChatMessage, params *RAGContext, fechSize int) ([]common.Document, error) {
	var query *orm.Query
	mustClauses := search.BuildMustClauses(params.category, params.subcategory, params.richCategory, params.username, params.userid)
	datasourceClause := search.BuildDatasourceClause(params.datasource, true)
	if datasourceClause != nil {
		mustClauses = append(mustClauses, datasourceClause)
	}
	var shouldClauses interface{}
	if params.QueryIntent != nil && len(params.QueryIntent.Query) > 0 {
		shouldClauses = search.BuildShouldClauses(params.QueryIntent.Query, params.QueryIntent.Keyword)
	}

	from := 0
	query = search.BuildTemplatedQuery(from, fechSize, mustClauses, shouldClauses, params.field, reqMsg.Message, params.source, params.tags)

	if query != nil {
		docs, err := fetchDocuments(query)
		if err != nil {
			log.Error(err)
			return nil, err
		}

		{
			simplifiedReferences := formatDocumentReferencesToDisplay(docs)
			const chunkSize = 512
			totalLen := len(simplifiedReferences)

			for chunkSeq := 0; chunkSeq*chunkSize < totalLen; chunkSeq++ {
				start := chunkSeq * chunkSize
				end := start + chunkSize
				if end > totalLen {
					end = totalLen
				}

				chunkData := simplifiedReferences[start:end]

				chunkMsg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID,
					common.FetchSource, string(chunkData), chunkSeq)

				err = websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(chunkMsg))
				if err != nil {
					log.Error(err)
					return nil, err
				}
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
		replyMsg.Details = append(replyMsg.Details, common.ProcessingDetails{Order: 20, Type: common.FetchSource, Payload: fetchedDocs})
		return docs, err
	}
	return nil, errors.Error("nothing found")
}

func (h APIHandler) processPickDocuments(ctx context.Context, reqMsg, replyMsg *common.ChatMessage, params *RAGContext, docs []common.Document) ([]common.Document, error) {

	if len(docs) == 0 {
		return nil, nil
	}

	echoMsg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.PickSource, string(""), 0)
	websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(echoMsg))

	content := []llms.MessageContent{
		llms.TextParts(
			llms.ChatMessageTypeSystem,
			`You are an AI assistant trained to select the most relevant documents for further processing and to answer user queries.
We have already queried the backend database and retrieved a list of documents that may help answer the user's query. And also invoke some external tools provided by MCP servers. 
Your task is to choose the best documents for further processing.`,
		),
	}

	content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, "The user has provided the following query:\n"))
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, reqMsg.Message))

	if params.QueryIntent != nil {
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, "The primary intent behind this query is:\n"))
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, util.MustToJSON(params.QueryIntent)))
	}

	if params.sourceDocsSummaryBlock != "" {
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, "The following documents are fetched from database:\n"))
		content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, params.sourceDocsSummaryBlock))
	}

	content = append(content, llms.TextParts(llms.ChatMessageTypeSystem, "\nPlease review these documents and identify which ones best related to user's query. "+
		"\nChoose no more than 5 relevant documents. These documents may be entirely unrelated, so prioritize those that provide direct answers or valuable context."+
		"\nIf the document is unrelated not certain, don't include it."+
		"\nFor each document, provide a brief explanation of why it was selected."+
		"\nYour decision should based solely on the information provided below. \nIf the information is insufficient, please indicate that you need more details to assist effectively. "+
		"\nDon't make anything up, which means if you can't identify which document best match the user's query, you should output nothing."+
		"\nMake sure the output is concise and easy to process."+
		"\nWrap the JSON result in <JSON></JSON> tags."+
		"\nThe expected output format is:\n"+
		"<JSON>\n"+
		"[\n"+
		" { \"id\": \"<id of Doc 1>\", \"title\": \"<title of Doc 1>\", \"explain\": \"<Explain for Doc 1>\"  },\n"+
		" { \"id\": \"<id of Doc 2>\", \"title\": \"<title of Doc 2>\", \"explain\": \"<Explain for Doc 2>\"  },\n"+
		"]"+
		"</JSON>"))

	log.Debug("start filtering documents")
	var pickedDocsBuffer = strings.Builder{}
	var chunkSeq = 0
	llm := langchain.GetLLM(params.pickingDocProvider.BaseURL, params.pickingDocProvider.APIType, params.pickingDocModel.Name, params.pickingDocProvider.APIKey, params.AssistantCfg.Keepalive)
	log.Trace(content)
	if _, err := llm.GenerateContent(ctx, content,
		llms.WithMaxLength(langchain.GetMaxLength(params.pickingDocModel, params.pickingDocProvider, 32768)),
		llms.WithMaxTokens(langchain.GetMaxTokens(params.pickingDocModel, params.pickingDocProvider, 32768)),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			if len(chunk) > 0 {
				chunkSeq++
				pickedDocsBuffer.Write(chunk)
				msg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.PickSource, string(chunk), chunkSeq)
				err := websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(msg))
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
		detail := common.ProcessingDetails{Order: 30, Type: common.PickSource, Payload: validPickedDocs}
		replyMsg.Details = append(replyMsg.Details, detail)
	}

	params.pickedDocIDS = pickedDocIDS

	log.Debug("valid picked document results:", validPickedDocs)

	//replace to picked one
	docs = pickedFullDoc
	return docs, err
}

func (h APIHandler) fetchDocumentInDepth(ctx context.Context, reqMsg, replyMsg *common.ChatMessage, params *RAGContext, docs []common.Document, inputValues map[string]any) error {
	if len(params.pickedDocIDS) > 0 {
		var query = orm.Query{}
		query.Conds = orm.And(orm.InStringArray("_id", params.pickedDocIDS))

		pickedFullDoc, err := fetchDocuments(&query)

		strBuilder := strings.Builder{}
		var chunkSeq = 0
		for _, v := range pickedFullDoc {
			str := "Obtaining and analyzing documents in depth:  " + string(v.Title) + "\n"
			strBuilder.WriteString(str)
			chunkMsg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.DeepRead, str, chunkSeq)
			err = websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(chunkMsg))
			if err != nil {
				return err
			}
		}

		detail := common.ProcessingDetails{Order: 40, Type: common.DeepRead, Description: strBuilder.String()}
		replyMsg.Details = append(replyMsg.Details, detail)

		inputValues["references"] = formatDocumentForReplyReferences(pickedFullDoc)
	}
	return nil
}

func (h APIHandler) generateFinalResponse(taskCtx context.Context, reqMsg, replyMsg *common.ChatMessage, params *RAGContext, inputValues map[string]any) error {

	echoMsg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.Response, string(""), 0)
	websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(echoMsg))

	// Prepare the system message
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, params.AssistantCfg.RolePrompt),
	}

	//response
	reasoningBuffer := strings.Builder{}
	messageBuffer := strings.Builder{}
	chunkSeq := 0
	var err error
	if params.answeringProvider == nil {
		return errors.Errorf("no answering provider with assistant: %v", params.assistantID)
	}

	llm := langchain.GetLLM(params.answeringProvider.BaseURL, params.answeringProvider.APIType, params.answeringModel.Name, params.answeringProvider.APIKey, params.AssistantCfg.Keepalive) //deepseek-r1 /deepseek-v3
	appConfig := common.AppConfig()

	log.Trace(params.answeringModel, ",", util.MustToJSON(appConfig))

	options := []llms.CallOption{}
	maxTokens := langchain.GetMaxTokens(params.answeringModel, params.answeringProvider, 1024)
	temperature := langchain.GetTemperature(params.answeringModel, params.answeringProvider, 0.8)
	maxLength := langchain.GetMaxLength(params.answeringModel, params.answeringProvider, 0)
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
					msg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.Think, string(reasoningChunk), chunkSeq)
					//log.Info(util.MustToJSON(msg))
					err = websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(msg))
					if err != nil {
						panic(err)
					}
					return nil
				}

				//Handle response
				if len(chunk) > 0 {
					chunkSeq += 1

					msg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.Response, string(chunk), chunkSeq)
					err = websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(msg))
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
			msg := common.NewMessageChunk(params.SessionID, replyMsg.ID, common.MessageTypeAssistant, reqMsg.ID, common.Response, string(chunk), chunkSeq)
			err = websocket.SendMessageToWebsocket(params.WebsocketID, util.MustToJSON(msg))
			messageBuffer.Write(chunk)
			return nil
		}))
	}

	contextPrompt := ``

	if v, ok := inputValues["history"]; ok {
		text, ok := v.(string)
		if ok {
			if params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold > 0 && len(text) > params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold {
				//log.Error("history is too large: %v, compressing, target size: %v", len(text), params.AssistantCfg.ChatSettings.HistoryMessage.CompressionThreshold)
				//TODO compress history
			}
			contextPrompt += fmt.Sprintf("\nConversation:\n%v\n", text)
		}
	}

	if v, ok := inputValues["references"]; ok {
		contextPrompt += fmt.Sprintf("\nReferences:\n%v\n", v)
	}

	if v, ok := inputValues["tools_output"]; ok {
		contextPrompt += fmt.Sprintf("\nTools Output:\n%v\n", v)
	}

	inputValues["context"] = contextPrompt

	template := rag.GenerateAnswerPromptTemplate
	if params.AssistantCfg.AnsweringModel.PromptConfig != nil && params.AssistantCfg.AnsweringModel.PromptConfig.PromptTemplate != "" {
		template = params.AssistantCfg.AnsweringModel.PromptConfig.PromptTemplate
	}

	// Create the prompt template
	promptTemplate, err := rag.GetPromptByTemplateArgs(params.answeringModel, template, []string{"query", "context"}, inputValues)
	if err != nil {
		panic(err)
	}

	promptValues, err := promptTemplate.FormatPrompt(inputValues)
	if err != nil {
		panic(err)
	}

	finalPrompt := promptValues.String()

	// Append the user's message
	content = append(content, llms.TextParts(llms.ChatMessageTypeHuman, finalPrompt))

	log.Info(content)

	completion, err := llm.GenerateContent(taskCtx, content, options...)
	if err != nil {
		log.Error(err)
		return err
	}
	_ = completion

	chunkSeq += 1

	{
		detail := common.ProcessingDetails{Order: 50, Type: common.Think, Description: reasoningBuffer.String()}
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
