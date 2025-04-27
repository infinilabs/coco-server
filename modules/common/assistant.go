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

package common

import (
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"time"
)

const (
	AssistantTypeSimple           = "simple"
	AssistantTypeDeepThink        = "deep_think"
	AssistantTypeExternalWorkflow = "external_workflow"

	AssistantCachePrimary = "assistant"
)

type Assistant struct {
	CombinedFullText
	Name           string           `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`
	Description    string           `json:"description" elastic_mapping:"description:{type:keyword,copy_to:combined_fulltext}"`
	Icon           string           `json:"icon" elastic_mapping:"icon:{type:keyword}"`
	Type           string           `json:"type" elastic_mapping:"type:{type:keyword}"`                // assistant type, default value: "simple", possible values: "simple", "deep_think", "external_workflow"
	Config         interface{}      `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"` // Assistant-specific configuration settings with type
	AnsweringModel ModelConfig      `json:"answering_model" elastic_mapping:"answering_model:{type:object,enabled:false}"`
	Datasource     DatasourceConfig `json:"datasource" elastic_mapping:"datasource:{type:object,enabled:false}"`
	ToolsConfig    ToolsConfig      `json:"tools,omitempty" elastic_mapping:"tools:{type:object,enabled:false}"`
	MCPConfig      MCPConfig        `json:"mcp_servers,omitempty" elastic_mapping:"mcp_servers:{type:object,enabled:false}"`
	Keepalive      string           `json:"keepalive" elastic_mapping:"keepalive:{type:keyword}"`
	Enabled        bool             `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`
	ChatSettings   ChatSettings     `json:"chat_settings" elastic_mapping:"chat_settings:{type:object,enabled:false}"`
	Builtin        bool             `json:"builtin" elastic_mapping:"builtin:{type:keyword}"`         // Whether the model provider is builtin
	RolePrompt     string           `json:"role_prompt" elastic_mapping:"role_prompt:{type:keyword}"` // Role prompt for the assistant

	DeepThinkConfig *DeepThinkConfig `json:"-"`
}

type DeepThinkConfig struct {
	IntentAnalysisModel ModelConfig `json:"intent_analysis_model"`
	PickingDocModel     ModelConfig `json:"picking_doc_model"`

	PickDatasource          bool `json:"pick_datasource"`
	PickTools               bool `json:"pick_tools"`
	ToolsPromisedResultSize int  `json:"tools_promised_result_size"`

	Visible bool `json:"visible"` // Whether the deep think mode is visible to the user
}

type WorkflowConfig struct {
}

type DatasourceConfig struct {
	Enabled bool        `json:"enabled"`
	IDs     []string    `json:"ids,omitempty"`
	Visible bool        `json:"visible"`          // Whether the deep datasource is visible to the user
	Filter  interface{} `json:"filter,omitempty"` // Filter for the datasource
}

type MCPConfig struct {
	Enabled       bool         `json:"enabled"`
	IDs           []string     `json:"ids,omitempty"`
	Visible       bool         `json:"visible"` // Whether the deep datasource is visible to the user
	Model         *ModelConfig `json:"model"`   //if not specified, use the answering model
	MaxIterations int          `json:"max_iterations"`
}

type ToolsConfig struct {
	Enabled      bool               `json:"enabled"`
	BuiltinTools BuiltinToolsConfig `json:"builtin,omitempty" elastic_mapping:"builtin:{enabled:false}"`
}
type BuiltinToolsConfig struct {
	Calculator bool `json:"calculator"`
	Wikipedia  bool `json:"wikipedia"`
	Duckduckgo bool `json:"duckduckgo"`
	Scraper    bool `json:"scraper"`
}

type ModelConfig struct {
	ProviderID   string        `json:"provider_id,omitempty"`
	Name         string        `json:"name"`
	Settings     ModelSettings `json:"settings"`
	PromptConfig *PromptConfig `json:"prompt,omitempty"`
}

type PromptConfig struct {
	PromptTemplate string   `json:"template"`
	InputVars      []string `json:"input_vars"`
}

type ModelSettings struct {
	Reasoning        bool    `json:"reasoning"`
	Temperature      float64 `json:"temperature"`
	TopP             float64 `json:"top_p"`
	PresencePenalty  float64 `json:"presence_penalty"`
	FrequencyPenalty float64 `json:"frequency_penalty"`
	MaxTokens        int     `json:"max_tokens"`
	MaxLength        int     `json:"max_length"`
}

type ChatSettings struct {
	GreetingMessage string `json:"greeting_message"`
	Suggested       struct {
		Enabled   bool     `json:"enabled"`
		Questions []string `json:"questions"`
	} `json:"suggested"`
	InputPreprocessTemplate string `json:"input_preprocess_tpl"`
	HistoryMessage          struct {
		Number               int  `json:"number"`
		CompressionThreshold int  `json:"compression_threshold"`
		Summary              bool `json:"summary"`
	} `json:"history_message"`
}

// GetAssistant retrieves the assistant object from the cache or database.
func GetAssistant(assistantID string) (*Assistant, bool, error) {
	item := GeneralObjectCache.Get(AssistantCachePrimary, assistantID)
	var assistant *Assistant
	if item != nil && !item.Expired() {
		var ok bool
		if assistant, ok = item.Value().(*Assistant); ok {
			return assistant, true, nil
		}
	}
	assistant = &Assistant{}
	assistant.ID = assistantID
	exists, err := orm.Get(assistant)
	if err != nil {
		return nil, exists, err
	}

	//expand datasource is the datasource is `*`
	if util.ContainsAnyInArray("*", assistant.Datasource.IDs) {
		ids, err := GetAllEnabledDatasourceIDs()
		if err != nil {
			panic(err)
		}
		assistant.Datasource.IDs = ids
	}

	if util.ContainsAnyInArray("*", assistant.MCPConfig.IDs) {
		ids, err := GetAllEnabledMCPServerIDs()
		if err != nil {
			panic(err)
		}
		assistant.MCPConfig.IDs = ids
	}

	//set default value
	if assistant.MCPConfig.MaxIterations <= 1 {
		assistant.MCPConfig.MaxIterations = 5
	}

	if assistant.AnsweringModel.PromptConfig == nil {
		assistant.AnsweringModel.PromptConfig = &PromptConfig{PromptTemplate: GenerateAnswerPromptTemplate}
	} else if assistant.AnsweringModel.PromptConfig.PromptTemplate == "" {
		assistant.AnsweringModel.PromptConfig.PromptTemplate = GenerateAnswerPromptTemplate
	}

	if assistant.Type == AssistantTypeDeepThink {
		deepThinkCfg := DeepThinkConfig{}
		buf := util.MustToJSONBytes(assistant.Config)
		util.MustFromJSONBytes(buf, &deepThinkCfg)

		if deepThinkCfg.IntentAnalysisModel.PromptConfig == nil {
			deepThinkCfg.IntentAnalysisModel.PromptConfig = &PromptConfig{PromptTemplate: QueryIntentPromptTemplate}
		} else if deepThinkCfg.IntentAnalysisModel.PromptConfig.PromptTemplate == "" {
			deepThinkCfg.IntentAnalysisModel.PromptConfig.PromptTemplate = QueryIntentPromptTemplate
		}

		assistant.Config = deepThinkCfg
		assistant.DeepThinkConfig = &deepThinkCfg
	}

	if assistant.RolePrompt == "" {
		assistant.RolePrompt = "You are a personal AI assistant designed by Coco AI(https://coco.rs), the backend team is behind INFINI Labs(https://infinilabs.com)."
	}

	// Cache the assistant object
	GeneralObjectCache.Set(AssistantCachePrimary, assistantID, assistant, time.Duration(30)*time.Minute)
	return assistant, true, nil
}
