/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/document"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/security"

	"infini.sh/coco/modules/assistant/rag"
	"infini.sh/coco/modules/datasource"
	"infini.sh/coco/modules/llm"

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
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
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
	tags         string
	subcategory  string
	richCategory string
	//field        string
	source        string
	integrationID string

	SessionID string

	//prepare for final response
	sourceDocsSummaryBlock string

	//history
	chatHistory *memory.ChatMessageHistory

	QueryIntent  *rag.QueryIntent
	pickedDocIDS []string

	intentModel     *core.ModelConfig
	pickingDocModel *core.ModelConfig
	//answeringModel      *common.ModelConfig
	intentModelProvider *core.ModelProvider
	pickingDocProvider  *core.ModelProvider
	answeringProvider   *core.ModelProvider

	AssistantCfg *core.Assistant

	//user input values
	InputValues map[string]any
}

func (r RAGContext) MustGetAnsweringModel() *core.ModelConfig {
	if r.AssistantCfg == nil {
		panic("invalid AssistantCfg")
	}

	//for background job only, no performance issue need to care right now
	for _, v := range r.answeringProvider.Models {
		if v.Name == r.AssistantCfg.AnsweringModel.Name {
			r.AssistantCfg.AnsweringModel.Settings.Reasoning = v.Settings.Reasoning
		}
	}

	return &r.AssistantCfg.AnsweringModel
}

func (r RAGContext) GetAnsweringProvider() *core.ModelProvider {

	if r.answeringProvider != nil {
		return r.answeringProvider
	}

	if r.AssistantCfg == nil {
		panic("invalid AssistantCfg")
	}

	modelProvider, err := common.GetModelProvider(r.AssistantCfg.AnsweringModel.ProviderID)
	if err != nil {
		panic(fmt.Errorf("failed to get model provider: %w", err))
	}

	if modelProvider == nil {
		panic("invalid modelProvider")
	}

	r.answeringProvider = modelProvider

	return modelProvider
}

const DefaultAssistantID = "default"

func (h APIHandler) getRAGContext(req *http.Request, assistant *core.Assistant) (*RAGContext, error) {

	params := &RAGContext{
		SearchDB:     h.GetBoolOrDefault(req, "search", false),
		DeepThink:    h.GetBoolOrDefault(req, "deep_thinking", false),
		MCP:          h.GetBoolOrDefault(req, "mcp", false),
		From:         h.GetIntOrDefault(req, "from", 0),
		Size:         h.GetIntOrDefault(req, "size", 10),
		datasource:   h.GetParameterOrDefault(req, "datasource", ""),
		category:     h.GetParameterOrDefault(req, "category", ""),
		tags:         h.GetParameterOrDefault(req, "tags", ""),
		subcategory:  h.GetParameterOrDefault(req, "subcategory", ""),
		richCategory: h.GetParameterOrDefault(req, "rich_category", ""),
		source:       h.GetParameterOrDefault(req, "source_fields", "*"),
	}

	if v := h.GetParameterOrDefault(req, "mcp_servers", ""); v != "" {
		params.mcpServers = strings.Split(v, ",")
	}

	params.integrationID = h.GetHeader(req, core.HeaderIntegrationID, "")
	params.AssistantCfg = assistant

	if assistant.Datasource.Enabled && len(params.datasource) > 0 && len(assistant.Datasource.GetIDs()) > 0 {
		if params.datasource == "" {
			params.datasource = strings.Join(assistant.Datasource.GetIDs(), ",")
		} else {
			// calc intersection with datasource and assistant datasourceIDs
			queryDatasource := strings.Split(params.datasource, ",")
			queryDatasource = util.StringArrayIntersection(queryDatasource, assistant.Datasource.GetIDs())
			params.datasource = strings.Join(queryDatasource, ",")
		}
	}

	log.Trace(assistant.MCPConfig.Enabled, assistant.MCPConfig.GetIDs(), ",", params.mcpServers)

	if params.MCP && assistant.MCPConfig.Enabled && len(params.mcpServers) > 0 && len(assistant.MCPConfig.GetIDs()) > 0 {
		if len(params.mcpServers) == 0 {
			params.mcpServers = assistant.MCPConfig.GetIDs()
		} else {
			// calc intersection with datasource and assistant datasourceIDs
			queryMcpServers := params.mcpServers
			queryMcpServers = util.StringArrayIntersection(queryMcpServers, assistant.MCPConfig.GetIDs())
			params.mcpServers = queryMcpServers
		}
	} else {
		params.mcpServers = make([]string, 0)
	}

	if params.DeepThink {
		if assistant.Type == common.AssistantTypeDeepThink {
			deepThinkCfg := core.DeepThinkConfig{}
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
	//params.answeringModel = &assistant.AnsweringModel
	params.answeringProvider = modelProvider

	return params, nil
}

func createAssistantMessage(sessionID, assistantID, requestMessageID string) *core.ChatMessage {
	msg := &core.ChatMessage{
		SessionID:      sessionID,
		MessageType:    core.MessageTypeAssistant,
		ReplyMessageID: requestMessageID,
		AssistantID:    assistantID,
	}
	now := time.Now()
	msg.Created = &now
	msg.ID = util.GetUUID()

	return msg
}

func finalizeProcessing(ctx *orm.Context, sessionID string, msg *core.ChatMessage, sender core.MessageSender) {
	if err := orm.Save(ctx, msg); err != nil {
		_ = log.Errorf("Failed to save assistant message: %v", err)
	}

	_ = sender.SendMessage(core.NewMessageChunk(
		sessionID, msg.ID, core.MessageTypeSystem, msg.ReplyMessageID,
		common.ReplyEnd, "Processing completed", 0,
	))
}

func processMessageAsync(ctx *orm.Context, userID string, reqMsg *core.ChatMessage, params *RAGContext, sender core.MessageSender) error {
	log.Debugf("Starting async processing for session: %v", params.SessionID)

	replyMsg := createAssistantMessage(params.SessionID, reqMsg.AssistantID, reqMsg.ID)

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
					_ = sender.SendMessage(core.NewMessageChunk(
						params.SessionID, replyMsg.ID, core.MessageTypeSystem, reqMsg.ID,
						common.Response, msg, 0,
					))
				}
				_ = log.Error(msg)
			}
		}
		finalizeProcessing(ctx, params.SessionID, replyMsg, sender)
		// clear the inflight message task
		taskID := getReplyMessageTaskID(params.SessionID, reqMsg.ID)
		inflightMessages.Delete(taskID)
	}()

	reqMsg.Details = make([]core.ProcessingDetails, 0)

	// Prepare input values
	if params.InputValues == nil {
		params.InputValues = map[string]any{}
	}

	params.InputValues["query"] = reqMsg.Message

	// Processing pipeline
	if params.AssistantCfg.ChatSettings.HistoryMessage.Number > 0 {
		history, _ := fetchSessionHistory(ctx, reqMsg, replyMsg, params, params.AssistantCfg.ChatSettings.HistoryMessage.Number, params.InputValues)
		params.InputValues["history"] = history
	} else {
		params.InputValues["history"] = "</empty>"
	}

	if params.DeepThink && params.intentModel != nil {

		//tool_list
		//network_sources

		if params.AssistantCfg.DeepThinkConfig == nil {
			panic("invalid deep think config")
		}

		if params.AssistantCfg.DeepThinkConfig.PickDatasource {
			var datasourceStr = strings.Builder{}
			if len(params.AssistantCfg.Datasource.GetIDs()) > 0 {
				ds, err := datasource.GetDatasourceByID(params.AssistantCfg.Datasource.GetIDs())
				if err == nil && ds != nil {
					for _, v := range ds {
						datasourceStr.WriteString(fmt.Sprintf("ID: %v, Name: %v, Description: %v \n", v.ID, v.Name, v.Description))
					}
				}
			}
			params.InputValues["network_sources"] = datasourceStr.String()
		}

		if params.AssistantCfg.DeepThinkConfig.PickTools {
			var mcpServers = strings.Builder{}
			if len(params.AssistantCfg.MCPConfig.GetIDs()) > 0 {
				ds, err := llm.GetMCPServersByID(params.AssistantCfg.MCPConfig.GetIDs())
				if err == nil && ds != nil {
					for _, v := range ds {
						mcpServers.WriteString(fmt.Sprintf("Name: %v, Desc: %v \n", v.Name, v.Description))
					}
				}
			}

			params.InputValues["tool_list"] = mcpServers.String()
		}

		queryIntent, err := rag.ProcessQueryIntent(ctx, params.SessionID, params.intentModelProvider, params.intentModel, reqMsg, replyMsg, params.AssistantCfg, params.InputValues, sender)
		if err != nil {
			log.Error("error on processing query intent analysis: ", err)
		}
		// Store the query intent in the processing parameters
		params.QueryIntent = queryIntent
	}

	var toolsMayHavePromisedResult = false
	if params.MCP && ((params.AssistantCfg.MCPConfig.Enabled && len(params.mcpServers) > 0) || params.AssistantCfg.ToolsConfig.Enabled) {
		//process LLM tools / functions
		answer, err := processLLMTools(ctx, reqMsg, replyMsg, params, params.InputValues, sender)
		if err != nil {
			log.Error(answer, err)
		}

		if answer != "" {
			if params.AssistantCfg.DeepThinkConfig != nil && params.AssistantCfg.DeepThinkConfig.ToolsPromisedResultSize > 0 && len(answer) > params.AssistantCfg.DeepThinkConfig.ToolsPromisedResultSize {
				toolsMayHavePromisedResult = true
			}
			params.InputValues["tools_output"] = answer
		}
	}

	if params.SearchDB && !toolsMayHavePromisedResult && params.AssistantCfg.Datasource.Enabled && len(params.AssistantCfg.Datasource.GetIDs()) > 0 {
		var fetchSize = 10
		if params.DeepThink {
			fetchSize = 50
		}
		docs, _ := processInitialDocumentSearch(ctx, userID, reqMsg, replyMsg, params, fetchSize, sender)
		params.InputValues["references"] = docs

		if params.DeepThink && len(docs) > 10 {
			//re-pick top docs
			docs, _ = processPickDocuments(ctx, reqMsg, replyMsg, params, docs, sender)
			_ = fetchDocumentInDepth(ctx, reqMsg, replyMsg, params, docs, params.InputValues, sender)
		}
	}

	err := generateFinalResponse(ctx, reqMsg, replyMsg, params, params.InputValues, sender)
	log.Info("async reply task done for query:", reqMsg.Message)
	return err
}

func fetchSessionHistory(ctx context.Context, reqMsg, replyMsg *core.ChatMessage, params *RAGContext, size int, inputValues map[string]any) (string, error) {
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
		msgText := util.SubStringWithSuffix(v.Message, 1000, "...")
		switch v.MessageType {
		case core.MessageTypeSystem:
			msg := llms.SystemChatMessage{Content: msgText}
			_ = chatHistory.AddMessage(context.Background(), msg)
			break
		case core.MessageTypeAssistant:
			msg := llms.AIChatMessage{Content: msgText}
			_ = chatHistory.AddMessage(context.Background(), msg)
			break
		case core.MessageTypeUser:
			msg := llms.HumanChatMessage{Content: msgText}
			_ = chatHistory.AddMessage(context.Background(), msg)
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

func processLLMTools(ctx context.Context, reqMsg *core.ChatMessage, replyMsg *core.ChatMessage, params *RAGContext, inputValues map[string]any, sender core.MessageSender) (string, error) {
	if params == nil || params.AssistantCfg == nil {
		//return nil
		panic("invalid assistant config, skip")
	}

	if params.intentModel != nil && (params.AssistantCfg.DeepThinkConfig != nil && params.AssistantCfg.DeepThinkConfig.PickTools) {
		if !params.QueryIntent.NeedCallTools {
			log.Info("intent analyzer decided to skip call LLM tools")
			return "", nil
		}
	}

	//get llm for mcp, use answering model if not mcp specified model
	providerID := params.MustGetAnsweringModel().ProviderID
	modelName := params.MustGetAnsweringModel().Name
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
		return "", err
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
			_ = f.Close()
		}
	}()

	log.Debug("found total ", len(params.mcpServers), " mcp servers")

	for _, id := range params.mcpServers {
		v, err := common.GetMPCServer(id)
		if err != nil || v == nil {
			log.Errorf("Failed to get MPC Server [%s]: %v", id, err)
			continue
		}

		log.Tracef("start init mcp server: %v, %v", v.Name, v.Type)

		if !v.Enabled {
			continue
		}

		var mcpClient *client.Client
		switch v.Type {
		case common.StreamableHTTP:
			bytes := util.MustToJSONBytes(v.Config)
			cfg := core.StreamableHttpConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("convert from json fail: %v", err)
				}
				continue
			}

			if !util.IsValidURL(cfg.URL) {
				if global.Env().IsDebug {
					log.Errorf("invalid url: %v", cfg.URL)
				}
				continue
			}

			mcpClient, err = client.NewStreamableHttpClient(cfg.URL)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("NewStreamableHttpClient fail: %v", err)
				}
				continue
			}
			break
		case common.SSE:
			bytes := util.MustToJSONBytes(v.Config)
			cfg := core.SSEConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("convert from json fail: %v", err)
				}
				continue
			}

			mcpClient, err = client.NewSSEMCPClient(cfg.URL)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("NewSSEMCPClient fail: %v", err)
				}
				continue
			}
			if err := mcpClient.Start(context.Background()); err != nil {
				if global.Env().IsDebug {
					log.Errorf("start client fail: %v", err)
				}
				continue
			}

			break
		case common.Stdio:
			bytes := util.MustToJSONBytes(v.Config)

			cfg := core.StdioConfig{}
			err := util.FromJSONBytes(bytes, &cfg)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("convert from json fail: %v", err)
				}
				continue
			}
			envs := []string{}
			if len(cfg.Env) > 0 {
				for k, v := range cfg.Env {
					envs = append(envs, fmt.Sprintf("%v=%v", k, v))
				}
			}
			mcpClient, err = client.NewStdioMCPClient(cfg.Command, envs, cfg.Args...)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("NewStdioMCPClient fail: %v", err)
				}
				continue
			}
			//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			//defer cancel()
			if err := mcpClient.Start(context.Background()); err != nil {
				if global.Env().IsDebug {
					log.Errorf("start client fail: %v", err)
				}
				continue
			}
			break
		default:
			if global.Env().IsDebug {
				log.Errorf("invalid type: %v", v.Type)
			}
			continue
		}

		if mcpClient != nil {
			mcpClients = append(mcpClients, mcpClient)
			mcpAdapter, err := langchain.New(mcpClient)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("error on new langchain client: %v", err)
				}
				continue
			}

			mcpTools, err := mcpAdapter.Tools()
			log.Tracef("get %v tools from mcp server: %v", v.Name)
			if err != nil {
				if global.Env().IsDebug {
					log.Errorf("error get %v tools from mcp server: %v", v.Name, err)
				}
				continue
			}
			agentTools = append(agentTools, mcpTools...)
		}

		log.Tracef("end init mcp server: %v", v.Name)
	}

	if len(agentTools) <= 0 {
		log.Debug("total get ", len(agentTools), " tools")
		return "", nil
	}

	buffer := memory.NewConversationBuffer()
	if params.chatHistory != nil {
		buffer.ChatHistory = params.chatHistory
	}

	answerBuffer := strings.Builder{}
	callback := langchain.LogHandler{}
	toolsSeq := 0
	callback.CustomWriteFunc = func(chunk string) {
		if chunk != "" {
			answerBuffer.WriteString(chunk)
			echoMsg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Tools, chunk, toolsSeq)
			_ = sender.SendMessage(echoMsg)
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
		return answerBuffer.String(), fmt.Errorf("error on executor: %w", err)
	}

	log.Debugf("start call LLM tools")
	answer, err := chains.Run(context.Background(), executor, reqMsg.Message)
	if err != nil {
		return answerBuffer.String(), fmt.Errorf("error running chains: %w", err)
	}

	log.Debug("MCP call answer:", answer)

	return answer, nil
}

func processInitialDocumentSearch(ctx *orm.Context, userID string, reqMsg, replyMsg *core.ChatMessage, params *RAGContext, fechSize int, sender core.MessageSender) ([]core.Document, error) {

	if params.intentModel != nil && (params.AssistantCfg.DeepThinkConfig != nil && params.AssistantCfg.DeepThinkConfig.PickDatasource) && params.QueryIntent != nil {
		if !params.QueryIntent.NeedNetworkSearch {
			log.Info("intent analyzer decided to skip fetch datasource")
			return []core.Document{}, nil
		}
	}

	builder := orm.NewQuery()
	builder.Size(fechSize)

	//merge the user defined query to filter
	if params.AssistantCfg.Datasource.Enabled && params.AssistantCfg.Datasource.Filter != nil {
		log.Debug("custom filter:", params.AssistantCfg.Datasource.Filter)
		q := util.MapStr{}
		q["query"] = params.AssistantCfg.Datasource.Filter
		builder.SetRequestBodyBytes(util.MustToJSONBytes(q))
		builder.EnableBodyBytes()
	}

	if params.QueryIntent != nil && len(params.QueryIntent.Query) > 0 {
		builder.Should(orm.TermsQuery("combined_fulltext", params.QueryIntent.Keyword))
		builder.Should(orm.TermsQuery("combined_fulltext", params.QueryIntent.Query))
	}

	teamsID := []string{}
	teamsID = GetTeamsIDByUserID(ctx, userID)

	docs := []core.Document{}
	_, err := document.QueryDocuments(ctx.Context, userID, teamsID, builder, reqMsg.Message, params.datasource, params.integrationID, params.category, params.subcategory, params.richCategory, "keyword", 3, &docs)
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

			chunkMsg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID,
				common.FetchSource, string(chunkData), chunkSeq)

			err = sender.SendMessage(chunkMsg)
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
	replyMsg.Details = append(replyMsg.Details, core.ProcessingDetails{Order: 20, Type: common.FetchSource, Payload: fetchedDocs})
	return docs, err
}

func GetTeamsIDByUserID(ctx *orm.Context, userID string) []string {
	if global.Env().SystemConfig.WebAppConfig.Security.Managed {

		sessionUser := security.MustGetUserFromContext(ctx.Context)

		profileKey := fmt.Sprintf("%v:%v", sessionUser.MustGetString(orm.TenantIDKey), userID)

		//get profile
		data, err := kv.GetValue(core.UserProfileBucketKey, []byte(profileKey))
		if err != nil {
			panic(err)
		}

		p := &security.UserProfile{}
		util.MustFromJSONBytes(data, p)
		v, ok := p.GetSystemValue(orm.TeamsIDKey)
		if ok {
			v, ok := v.([]interface{})
			if ok {
				out := []string{}
				for _, v1 := range v {
					x, ok := v1.(string)
					if ok {
						out = append(out, x)
					}
				}
				return out
			}
		}
	}
	return []string{}
}

func processPickDocuments(ctx context.Context, reqMsg, replyMsg *core.ChatMessage, params *RAGContext, docs []core.Document, sender core.MessageSender) ([]core.Document, error) {

	if len(docs) == 0 {
		return nil, nil
	}

	echoMsg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.PickSource, string(""), 0)
	_ = sender.SendMessage(echoMsg)

	promptTemplate := common.PickingDocPromptTemplate
	if params.pickingDocModel != nil && params.pickingDocModel.PromptConfig != nil && params.pickingDocModel.PromptConfig.PromptTemplate != "" {
		promptTemplate = params.pickingDocModel.PromptConfig.PromptTemplate
	}
	// Create the prompt template
	inputValues := map[string]any{
		"query":  reqMsg.Message,
		"intent": util.MustToJSON(params.QueryIntent),
		"docs":   params.sourceDocsSummaryBlock,
	}
	finalPrompt, err := rag.GetPromptStringByTemplateArgs(params.pickingDocModel, promptTemplate, []string{"query", "intent", "summary"}, inputValues)
	if err != nil {
		panic(err)
	}
	content := []llms.MessageContent{
		llms.TextParts(
			llms.ChatMessageTypeSystem,
			finalPrompt,
		),
	}

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
				msg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.PickSource, string(chunk), chunkSeq)
				err := sender.SendMessage(msg)
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

	docsMap := map[string]core.Document{}
	for _, v := range docs {
		docsMap[v.ID] = v
	}

	var pickedDocIDS []string
	var pickedFullDoc = []core.Document{}
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
		detail := core.ProcessingDetails{Order: 30, Type: common.PickSource, Payload: validPickedDocs}
		replyMsg.Details = append(replyMsg.Details, detail)
	}

	params.pickedDocIDS = pickedDocIDS

	log.Debug("valid picked document results:", validPickedDocs)

	//replace to picked one
	docs = pickedFullDoc
	return docs, err
}

func fetchDocumentInDepth(ctx *orm.Context, reqMsg, replyMsg *core.ChatMessage, params *RAGContext, docs []core.Document, inputValues map[string]any, sender core.MessageSender) error {
	if len(params.pickedDocIDS) > 0 {
		var query = orm.Query{}
		query.Conds = orm.And(orm.InStringArray("_id", params.pickedDocIDS))

		pickedFullDoc, err := fetchDocuments(&query)

		strBuilder := strings.Builder{}
		var chunkSeq = 0
		for _, v := range pickedFullDoc {
			str := "Obtaining and analyzing documents in depth:  " + string(v.Title) + "\n"
			strBuilder.WriteString(str)
			chunkMsg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.DeepRead, str, chunkSeq)
			err = sender.SendMessage(chunkMsg)
			if err != nil {
				return err
			}
		}

		detail := core.ProcessingDetails{Order: 40, Type: common.DeepRead, Description: strBuilder.String()}
		replyMsg.Details = append(replyMsg.Details, detail)

		inputValues["references"] = formatDocumentForReplyReferences(pickedFullDoc)
	}
	return nil
}

func generateFinalResponse(taskCtx context.Context, reqMsg, replyMsg *core.ChatMessage, params *RAGContext, inputValues map[string]any, sender core.MessageSender) error {

	echoMsg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Response, string(""), 0)
	_ = sender.SendMessage(echoMsg)
	replyMsg.Message += echoMsg.MessageChunk

	// Prepare the system message
	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, params.AssistantCfg.RolePrompt),
	}

	//response
	reasoningBuffer := strings.Builder{}
	messageBuffer := strings.Builder{}
	// note: we use defer to ensure that the response message is saved after processing
	// even if user cancels the task or if an error occurs
	defer func() {
		//save response message to system
		if messageBuffer.Len() > 0 {
			replyMsg.Message = messageBuffer.String()
		} else {
			log.Warnf("seems empty reply for query: %v", replyMsg)
		}
		if reasoningBuffer.Len() > 0 {
			detail := core.ProcessingDetails{Order: 50, Type: common.Think, Description: reasoningBuffer.String()}
			replyMsg.Details = append(replyMsg.Details, detail)
		}
	}()
	chunkSeq := 0
	var err error

	provider := params.GetAnsweringProvider()
	//log.Error(util.ToJson(provider, true))
	//log.Error(util.ToJson(params.AssistantCfg, true))
	//llm := langchain.GetLLM(params.answeringProvider.BaseURL, params.answeringProvider.APIType, params.answeringModel.Name, params.answeringProvider.APIKey, params.AssistantCfg.Keepalive) //deepseek-r1 /deepseek-v3

	llm := langchain.GetLLM(provider.BaseURL, provider.APIType, params.MustGetAnsweringModel().Name, provider.APIKey, params.AssistantCfg.Keepalive) //deepseek-r1 /deepseek-v3
	appConfig := common.AppConfig()

	log.Trace(params.MustGetAnsweringModel(), ",", util.MustToJSON(appConfig))

	options := []llms.CallOption{}
	maxTokens := langchain.GetMaxTokens(params.MustGetAnsweringModel(), params.GetAnsweringProvider(), 1024)
	temperature := langchain.GetTemperature(params.MustGetAnsweringModel(), params.GetAnsweringProvider(), 0.8)
	maxLength := langchain.GetMaxLength(params.MustGetAnsweringModel(), params.GetAnsweringProvider(), 0)
	options = append(options, llms.WithMaxTokens(maxTokens))
	options = append(options, llms.WithMaxLength(maxLength))
	options = append(options, llms.WithTemperature(temperature))

	if params.MustGetAnsweringModel().Settings.Reasoning {
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
					msg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Think, string(reasoningChunk), chunkSeq)
					//log.Info(util.MustToJSON(msg))
					err = sender.SendMessage(msg)
					if err != nil {
						panic(err)
					}
					return nil
				}

				//Handle response
				if len(chunk) > 0 {
					chunkSeq += 1

					msg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Response, string(chunk), chunkSeq)
					err = sender.SendMessage(msg)
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
			if len(chunk) > 0 {
				log.Trace(string(chunk))
				chunkSeq += 1
				msg := core.NewMessageChunk(params.SessionID, replyMsg.ID, core.MessageTypeAssistant, reqMsg.ID, common.Response, string(chunk), chunkSeq)
				err = sender.SendMessage(msg)
				messageBuffer.Write(chunk)
			}
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
		contextPrompt += util.SubString(fmt.Sprintf("\nReferences:\n%v\n", v), 0, 4096*2) //TODO
	}

	if v, ok := inputValues["tools_output"]; ok {
		contextPrompt += fmt.Sprintf("\nTools Output:\n%v\n", v)
	}

	inputValues["context"] = contextPrompt

	template := common.GenerateAnswerPromptTemplate
	if params.AssistantCfg.AnsweringModel.PromptConfig != nil && params.AssistantCfg.AnsweringModel.PromptConfig.PromptTemplate != "" {
		template = params.AssistantCfg.AnsweringModel.PromptConfig.PromptTemplate
	}

	// Create the prompt template
	finalPrompt, err := rag.GetPromptStringByTemplateArgs(params.MustGetAnsweringModel(), template, []string{"query", "context"}, inputValues)
	if err != nil {
		panic(err)
	}

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

	return nil
}

func formatDocumentForReplyReferences(docs []core.Document) string {
	var sb strings.Builder
	sb.WriteString("<REFERENCES>\n")
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("<Doc>"))
		sb.WriteString(fmt.Sprintf("ID #%d - %v\n", i+1, doc.ID))
		sb.WriteString(fmt.Sprintf("Title: %s\n", doc.Title))
		sb.WriteString(fmt.Sprintf("Source: %s\n", doc.Source))
		sb.WriteString(fmt.Sprintf("Updated: %s\n", doc.Updated))
		sb.WriteString(fmt.Sprintf("Category: %s\n", doc.GetAllCategories()))
		sb.WriteString(fmt.Sprintf("Content: %s\n", doc.Content))
		sb.WriteString(fmt.Sprintf("</Doc>\n"))

	}
	sb.WriteString("</REFERENCES>")
	return sb.String()
}

func formatDocumentReferencesToDisplay(docs []core.Document) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<Payload total=%v>\n", len(docs)))
	outDocs := []util.MapStr{}
	for _, doc := range docs {
		item := util.MapStr{}
		item["id"] = doc.ID
		item["title"] = doc.Title
		item["source"] = doc.Source
		item["icon"] = doc.Icon
		item["url"] = doc.URL
		outDocs = append(outDocs, item)
	}
	sb.WriteString(util.MustToJSON(outDocs))
	sb.WriteString("</Payload>")
	return sb.String()
}

func formatDocumentForPick(docs []core.Document) []util.MapStr {
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

func fetchDocuments(query *orm.Query) ([]core.Document, error) {
	var docs []core.Document
	err, _ := orm.SearchWithJSONMapper(&docs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}
	return docs, nil
}
