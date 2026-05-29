package common

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/memory"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
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

	QueryIntent  *QueryIntent
	PickedDocIDS []string

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

	return params, nil
}

// MustGetAnsweringModel resolves and returns the effective answering model.
// If the assistant has no model configured, it falls back to the system default.
func (r RAGContext) MustGetAnsweringModel() *core.ModelConfig {
	if r.AssistantCfg == nil {
		panic("invalid AssistantCfg")
	}

	// Resolve on demand: assistant override -> settings default -> settings language model
	resolved := llmmodule.ResolveAssistantModel(core.AssistantModelUseAnswering, &core.ModelId{
		ProviderID: r.AssistantCfg.AnsweringModel.ProviderID,
		ID:         r.AssistantCfg.AnsweringModel.Name,
	})
	if resolved == nil {
		panic(fmt.Sprintf("assistant [%s] has no answering model configured and no default in settings", r.AssistantCfg.Name))
	}

	// Build a ModelConfig with the resolved identity and the assistant's settings
	model := r.AssistantCfg.AnsweringModel
	model.ProviderID = resolved.ProviderID
	model.Name = resolved.ID

	// Merge reasoning capability from provider's model definition
	modelProvider, err := common.GetModelProvider(resolved.ProviderID)
	if err == nil && modelProvider != nil {
		for _, v := range modelProvider.Models {
			if v.Name == resolved.ID {
				model.SupportReasoning = v.SupportReasoning
				break
			}
		}
	}

	return &model
}

// GetAnsweringProvider resolves and returns the model provider for the answering model.
func (r RAGContext) GetAnsweringProvider() *core.ModelProvider {
	model := r.MustGetAnsweringModel()

	modelProvider, err := common.GetModelProvider(model.ProviderID)
	if err != nil {
		panic(fmt.Errorf("failed to get model provider: %w", err))
	}

	if modelProvider == nil {
		panic("invalid modelProvider")
	}

	return modelProvider
}
