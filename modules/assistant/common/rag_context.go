package common

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/memory"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	api1 "infini.sh/framework/core/api"
	"infini.sh/framework/core/util"
)

type RAGContext struct {
	SearchDB      bool
	DeepThink     bool
	MCP           bool
	From          int
	Size          int
	MCPServers    []string
	Datasource    string
	Category      string
	Tags          string
	Subcategory   string
	RichCategory  string
	IntegrationID string

	SessionID string

	//prepare for final response
	SourceDocsSummaryBlock string

	//history
	ChatHistory *memory.ChatMessageHistory

	QueryIntent  *langchain.QueryIntent
	PickedDocIDS []string

	//pickingDocModel *core.ModelConfig
	//answeringModel      *common.ModelConfig
	//intentModelProvider *core.ModelProvider
	//pickingDocProvider  *core.ModelProvider
	answeringProvider *core.ModelProvider

	AssistantCfg *core.Assistant

	//user input values
	InputValues map[string]any
}

func NewRagContext(req *http.Request, assistant *core.Assistant, sessionID string) (*RAGContext, error) {
	params := &RAGContext{
		SearchDB:     api1.GetBoolOrDefault(req, "search", false),
		DeepThink:    api1.GetBoolOrDefault(req, "deep_thinking", false),
		MCP:          api1.GetBoolOrDefault(req, "mcp", false),
		From:         api1.GetIntOrDefault(req, "from", 0),
		Size:         api1.GetIntOrDefault(req, "size", 10),
		Datasource:   api1.GetParameterOrDefault(req, "datasource", ""),
		Category:     api1.GetParameterOrDefault(req, "category", ""),
		Tags:         api1.GetParameterOrDefault(req, "tags", ""),
		Subcategory:  api1.GetParameterOrDefault(req, "subcategory", ""),
		RichCategory: api1.GetParameterOrDefault(req, "rich_category", ""),
	}

	params.SessionID = sessionID

	if v := api1.GetParameterOrDefault(req, "mcp_servers", ""); v != "" {
		params.MCPServers = strings.Split(v, ",")
	}

	params.IntegrationID = api1.GetHeader(req, core.HeaderIntegrationID, "")

	params.AssistantCfg = assistant

	if assistant.Datasource.Enabled && len(params.Datasource) > 0 && len(assistant.Datasource.GetIDs()) > 0 {
		if params.Datasource == "" {
			params.Datasource = strings.Join(assistant.Datasource.GetIDs(), ",")
		} else {
			// calc intersection with datasource and assistant datasourceIDs
			queryDatasource := strings.Split(params.Datasource, ",")
			queryDatasource = util.StringArrayIntersection(queryDatasource, assistant.Datasource.GetIDs())
			params.Datasource = strings.Join(queryDatasource, ",")
		}
	}

	log.Trace(assistant.MCPConfig.Enabled, assistant.MCPConfig.GetIDs(), ",", params.MCPServers)

	if params.MCP && assistant.MCPConfig.Enabled && len(params.MCPServers) > 0 && len(assistant.MCPConfig.GetIDs()) > 0 {
		if len(params.MCPServers) == 0 {
			params.MCPServers = assistant.MCPConfig.GetIDs()
		} else {
			// calc intersection with datasource and assistant datasourceIDs
			queryMcpServers := params.MCPServers
			queryMcpServers = util.StringArrayIntersection(queryMcpServers, assistant.MCPConfig.GetIDs())
			params.MCPServers = queryMcpServers
		}
	} else {
		params.MCPServers = make([]string, 0)
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

	params.AssistantCfg.DeepResearchConfig = core.DefaultDeepResearchConfig()

	return params, nil
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
