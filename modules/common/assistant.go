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
	ccache "infini.sh/framework/lib/cache"
	"time"
)

const (
	AssistantTypeSimple           = "simple"
	AssistantTypeDeepThink        = "deep_think"
	AssistantTypeExternalWorkflow = "external_workflow"
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
	MCPServers     DatasourceConfig `json:"mcp_servers,omitempty" elastic_mapping:"mcp_servers:{type:object,enabled:false}"`
	Keepalive      string           `json:"keepalive" elastic_mapping:"keepalive:{type:keyword}"`
	Enabled        bool             `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`
	ChatSettings   ChatSettings     `json:"chat_settings" elastic_mapping:"chat_settings:{type:object,enabled:false}"`
	Builtin        bool             `json:"builtin" elastic_mapping:"builtin:{type:keyword}"`         // Whether the model provider is builtin
	RolePrompt     string           `json:"role_prompt" elastic_mapping:"role_prompt:{type:keyword}"` // Role prompt for the assistant
}

var AssistantCache = ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100))

const (
	AssistantCachePrimary = "assistant"
)

// GetAssistant retrieves the assistant object from the cache or database.
func GetAssistant(assistantID string) (*Assistant, error) {
	item := AssistantCache.Get(AssistantCachePrimary, assistantID)
	var assistant *Assistant
	if item != nil && !item.Expired() {
		var ok bool
		if assistant, ok = item.Value().(*Assistant); ok {
			return assistant, nil
		}
	}
	assistant = &Assistant{}
	assistant.ID = assistantID
	_, err := orm.Get(assistant)
	if err != nil {
		return nil, err
	}
	// Cache the assistant object
	AssistantCache.Set(AssistantCachePrimary, assistantID, assistant, time.Duration(30)*time.Minute)
	return assistant, nil
}

type DeepThinkConfig struct {
	IntentAnalysisModel ModelConfig `json:"intent_analysis_model"`
	PickingDocModel     ModelConfig `json:"picking_doc_model"`
	Visible             bool        `json:"visible"` // Whether the deep think mode is visible to the user
}

type WorkflowConfig struct {
}

type DatasourceConfig struct {
	Enabled bool     `json:"enabled"`
	IDs     []string `json:"ids,omitempty"`
	Visible bool     `json:"visible"` // Whether the deep datasource is visible to the user
}
type ModelConfig struct {
	ProviderID string        `json:"provider_id,omitempty"`
	Name       string        `json:"name"`
	Settings   ModelSettings `json:"settings"`
}

type ModelSettings struct {
	Temperature      float64 `json:"temperature"`
	TopP             float64 `json:"top_p"`
	PresencePenalty  int     `json:"presence_penalty"`
	FrequencyPenalty int     `json:"frequency_penalty"`
	MaxTokens        int     `json:"max_tokens"`
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
