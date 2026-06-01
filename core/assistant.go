/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"fmt"
	"slices"
	"time"

	"golang.org/x/text/language"
)

// AssistantModelUse identifies a place in the assistant chat flow that needs
// an LLM. Each value maps to a default model in Settings.DefaultModel; if that
// is not configured, callers fall back to Settings.DefaultModel.LanguageModel.
//
// Unlike LLMType (which categorizes a model itself: language / vision /
// embedding), AssistantModelUse describes a *use case* within the chat flow.
type AssistantModelUse int

const (
	AssistantModelUseAnswering AssistantModelUse = iota
	AssistantModelUseIntentAnalysis
	AssistantModelUsePickingDoc
	AssistantModelUsePickingTool
)

type Assistant struct {
	CombinedFullText
	Name           string           `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"`
	Description    string           `json:"description" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon           string           `json:"icon" elastic_mapping:"icon:{enabled:false}"`
	Type           string           `json:"type" elastic_mapping:"type:{type:keyword}"` // assistant type, default value: "simple", possible values: "simple", "deep_think", "external_workflow", "deep_research"
	Category       string           `json:"category,omitempty" elastic_mapping:"category:{type:keyword}"`
	Tags           []string         `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword}"`
	// Assistant-specific configuration settings
	//
	// * simple:        not used; this field is always nil.
	// * deep_think:    split between two places — common fields (AnsweringModel,
	//                  Datasource, ToolsConfig, …) stay on Assistant, while
	//                  deep-think-specific fields live in DeepThinkConfig.
	// * deep_research: all configuration is contained entirely in
	//                  DeepResearchConfig; no fields on Assistant are used.
	//
	// After loading, Config is decoded into the corresponding typed field:
	// DeepThinkConfig or DeepResearchConfig (both tagged json:"-").
	Config         interface{}      `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"`
	AnsweringModel ModelConfig      `json:"answering_model" elastic_mapping:"answering_model:{type:object,enabled:false}"`
	Datasource     DatasourceConfig `json:"datasource" elastic_mapping:"datasource:{type:object,enabled:false}"`
	ToolsConfig    ToolsConfig      `json:"tools,omitempty" elastic_mapping:"tools:{type:object,enabled:false}"`
	MCPConfig      MCPConfig        `json:"mcp_servers,omitempty" elastic_mapping:"mcp_servers:{type:object,enabled:false}"`
	UploadConfig   UploadConfig     `json:"upload,omitempty" elastic_mapping:"upload:{type:object,enabled:false}"`
	Keepalive      string           `json:"keepalive" elastic_mapping:"keepalive:{type:keyword}"`
	Enabled        bool             `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`
	ChatSettings   ChatSettings     `json:"chat_settings" elastic_mapping:"chat_settings:{type:object,enabled:false}"`
	Builtin        bool             `json:"builtin" elastic_mapping:"builtin:{type:boolean}"`          // Whether the model provider is builtin
	RolePrompt     string           `json:"role_prompt" elastic_mapping:"role_prompt:{enabled:false}"` // Role prompt for the assistant

	// DeepThinkConfig and DeepResearchConfig are populated at load time by
	// decoding Config into the appropriate type (based on Type). They are not
	// persisted; json:"-" keeps them out of serialization.
	DeepThinkConfig    *DeepThinkConfig    `json:"-"`
	DeepResearchConfig *DeepResearchConfig `json:"-"`
}

type DeepThinkConfig struct {
	IntentAnalysisModel ModelConfig `json:"intent_analysis_model"`
	PickingDocModel     ModelConfig `json:"picking_doc_model"`

	PickDatasource          bool `json:"pick_datasource"`
	PickTools               bool `json:"pick_tools"`
	ToolsPromisedResultSize int  `json:"tools_promised_result_size"`

	Visible bool `json:"visible"` // Whether the deep think mode is visible to the user
}

// DeepResearchInternalSearchConfig controls enterprise (internal) search behaviour.
type DeepResearchInternalSearchConfig struct {
	DatasourceIDs []string `json:"datasource_ids,omitempty"` // Restrict to these IDs; empty = all accessible.
}

// DeepResearchExternalSearchConfig controls external web search behaviour.
type DeepResearchExternalSearchConfig struct {
	Enabled bool   `json:"enabled"`           // Enable external web search.
	Engine  string `json:"engine"`            // One of: "duckduckgo", "wikipedia", "tavily".
	APIKey  string `json:"api_key,omitempty"` // Required when Engine is "tavily".
}

type DeepResearchConfig struct {
	// Models — one per pipeline stage; each may point to a different provider/model.
	PlanningModel  ModelConfig `json:"planning_model"`  // Decomposes the query into a step-by-step research plan.
	ResearchModel  ModelConfig `json:"research_model"`  // Analyzes search results for each individual research step.
	SynthesisModel ModelConfig `json:"synthesis_model"` // Synthesizes findings across sources within a step.
	ReportModel    ModelConfig `json:"report_model"`    // Writes the final structured report.

	// Execution limits
	MaxSteps                   int    `json:"max_steps"`                     // Max steps the planner may generate.
	MaxResearcherIterations    int    `json:"max_researcher_iterations"`     // Max researcher loop iterations over the plan.
	MaxConcurrentResearchUnits int    `json:"max_concurrent_research_units"` // Max parallel research workers (v1 only).
	MaxResults                 int    `json:"max_results"`                   // Max search results fetched per query.
	Timeout                    string `json:"timeout"`                       // Total research deadline; Go duration string, e.g. "30m", "1h".
	ResearchDepth              string `json:"research_depth"`                // Effort level: "basic", "comprehensive", or "exhaustive".

	// Output
	IncludeSources bool   `json:"include_sources"` // Append a citations section to the report.
	SourceFormat   string `json:"source_format"`   // Citation style: "APA", "MLA", or empty for plain Markdown links.
	ReportFormat   string `json:"report_format"`   // Rendered format: "markdown" (default) or "html".
	ReportLang     string `json:"report_lang"`     // Report language as a BCP 47 tag, e.g. "en-US", "zh-CN".

	// Search
	InternalSearch DeepResearchInternalSearchConfig `json:"internal_search"` // Internal enterprise search settings.
	ExternalSearch DeepResearchExternalSearchConfig `json:"external_search"` // External web search settings.
}

type UploadConfig struct {
	Enabled               bool     `json:"enabled"`
	AllowedFileExtensions []string `json:"allowed_file_extensions"`
	MaxFileSizeInBytes    int      `json:"max_file_size_in_bytes"`
	MaxFileCount          int      `json:"max_file_count"`
}

type DatasourceConfig struct {
	Enabled bool `json:"enabled"`

	IDs       []string `json:"ids,omitempty"`
	parsedIDs []string `json:"-"`

	Visible          bool        `json:"visible"`            // Whether the deep datasource is visible to the user
	Filter           interface{} `json:"filter,omitempty"`   // Filter for the datasource
	EnabledByDefault bool        `json:"enabled_by_default"` // Whether the datasource is enabled by default
}

type MCPConfig struct {
	Enabled bool `json:"enabled"`

	IDs       []string `json:"ids,omitempty"`
	parsedIDs []string `json:"-"`

	Visible          bool         `json:"visible"` // Whether the deep datasource is visible to the user
	Model            *ModelConfig `json:"model"`   //if not specified, use the answering model
	MaxIterations    int          `json:"max_iterations"`
	EnabledByDefault bool         `json:"enabled_by_default"` // Whether the MCP server is enabled by default
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

// ModelConfig is a runtime reference to a model, specifying which model to use
// and how to use it. This is stored in assistant configurations.
//
// This is distinct from Model (in llm_provider.go), which describes a model's
// static, immutable capabilities. ModelConfig references a Model by ProviderID
// and Name, and adds runtime settings that control inference behavior.
type ModelConfig struct {
	// --- Reference fields: identify the model ---

	ProviderID string `json:"provider_id"` // references the ModelProvider
	Name       string `json:"name"`        // references Model.Name within the provider

	// --- Runtime fields: per-invocation behavior ---

	Settings     ModelSettings `json:"settings"`
	PromptConfig *PromptConfig `json:"prompt,omitempty"`
	Keepalive    string        `json:"keepalive"`
}

type PromptConfig struct {
	PromptTemplate string   `json:"template"`
	InputVars      []string `json:"input_vars"`
}

type ModelSettings struct {
	// Reasoning controls whether reasoning mode is requested at inference time.
	// This field is only meaningful when the model's SupportReasoning is true;
	// if SupportReasoning is false, the backend will not read or act on this
	// field even if it is set to true.
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
	PlaceHolder             string `json:"placeholder"`
	HistoryMessage          struct {
		Number               int  `json:"number"`
		CompressionThreshold int  `json:"compression_threshold"`
		Summary              bool `json:"summary"`
	} `json:"history_message"`
}

func (cfg *DatasourceConfig) SetIDs(ids []string) {
	cfg.parsedIDs = ids
}

func (cfg *DatasourceConfig) GetIDs() []string {
	if cfg.parsedIDs != nil {
		return cfg.parsedIDs
	}
	return cfg.IDs
}

func (cfg *MCPConfig) SetIDs(ids []string) {
	cfg.parsedIDs = ids
}
func (cfg *MCPConfig) GetIDs() []string {
	if cfg.parsedIDs != nil {
		return cfg.parsedIDs
	}
	return cfg.IDs
}

// DefaultDeepResearchConfig returns default deep research configuration
func DefaultDeepResearchConfig() *DeepResearchConfig {
	return &DeepResearchConfig{
		PlanningModel: ModelConfig{
			ProviderID: "qianwen", // Default provider
			Name:       "qwq-plus",
			Settings: ModelSettings{
				Temperature: 0.7,
				TopP:        0.95,
				MaxTokens:   2000,
			},
		},
		ResearchModel: ModelConfig{
			ProviderID: "qianwen",
			Name:       "qwq-plus",
			Settings: ModelSettings{
				Temperature: 0.6,
				TopP:        0.9,
				MaxTokens:   1500,
			},
		},
		SynthesisModel: ModelConfig{
			ProviderID: "qianwen",
			Name:       "qwq-plus",
			Settings: ModelSettings{
				Temperature: 0.5,
				TopP:        0.95,
				MaxTokens:   4000,
			},
		},
		ReportModel: ModelConfig{
			ProviderID: "qianwen",
			Name:       "qwq-plus",
			Settings: ModelSettings{
				Temperature: 0.7,
				TopP:        0.9,
				MaxTokens:   10000,
			},
		},
		MaxSteps:                50,
		MaxResearcherIterations: 10,
		MaxResults:              100,
		Timeout:                 "1h",
		ResearchDepth:           "comprehensive",
		IncludeSources:          true,
		SourceFormat:            "APA",
		InternalSearch: DeepResearchInternalSearchConfig{},
		ExternalSearch: DeepResearchExternalSearchConfig{Enabled: true, Engine: "duckduckgo"},
		ReportLang:    "en-US",
		ReportFormat:  "markdown",
	}
}

// Validate validates the deep research configuration
func (cfg *DeepResearchConfig) Validate() error {
	// Validate required models
	if cfg.PlanningModel.Name == "" {
		return fmt.Errorf("planning model name is required")
	}
	if cfg.ResearchModel.Name == "" {
		return fmt.Errorf("research model name is required")
	}
	if cfg.SynthesisModel.Name == "" {
		return fmt.Errorf("synthesis model name is required")
	}
	if cfg.ReportModel.Name == "" {
		return fmt.Errorf("report model name is required")
	}

	// Validate report_lang if it is set
	if cfg.ReportLang != "" {
		_, err := language.Parse(cfg.ReportLang)
		if err != nil {
			return fmt.Errorf("report_lang is invalid: [%s]", err)
		}
	}

	// Validate research limits
	if cfg.MaxSteps <= 0 {
		return fmt.Errorf("max_steps must be positive")
	}
	if cfg.MaxResults <= 0 {
		return fmt.Errorf("max_results must be positive")
	}
	// At least external search must be enabled
	if !cfg.ExternalSearch.Enabled {
		return fmt.Errorf("external_search must be enabled")
	}

	// Validate external search engine
	if cfg.ExternalSearch.Enabled {
		validEngines := []string{"duckduckgo", "wikipedia", "tavily"}
		if !slices.Contains(validEngines, cfg.ExternalSearch.Engine) {
			return fmt.Errorf("external_search.engine must be one of: %v", validEngines)
		}
		if cfg.ExternalSearch.Engine == "tavily" && cfg.ExternalSearch.APIKey == "" {
			return fmt.Errorf("external_search.api_key is required when engine is \"tavily\"")
		}
	}

	// Validate research depth
	validDepths := []string{"basic", "comprehensive", "exhaustive"}
	if !slices.Contains(validDepths, cfg.ResearchDepth) {
		return fmt.Errorf("research_depth must be one of: %v", validDepths)
	}

	// Validate report format
	validFormats := []string{"markdown", "html"}
	if !slices.Contains(validFormats, cfg.ReportFormat) {
		return fmt.Errorf("report_format must be one of: %v", validFormats)
	}

	// Validate Timeout
	if cfg.Timeout != "" {
		if _, err := time.ParseDuration(cfg.Timeout); err != nil {
			return fmt.Errorf("timeout is invalid: %s", err)
		}
	}

	return nil
}

// MergeDeepResearchConfig merges user config with defaults
func MergeDeepResearchConfig(userConfig, defaultConfig *DeepResearchConfig) *DeepResearchConfig {
	if userConfig == nil {
		return defaultConfig
	}

	if userConfig.PlanningModel.Name == "" {
		userConfig.PlanningModel = defaultConfig.PlanningModel
	}
	if userConfig.ResearchModel.Name == "" {
		userConfig.ResearchModel = defaultConfig.ResearchModel
	}
	if userConfig.SynthesisModel.Name == "" {
		userConfig.SynthesisModel = defaultConfig.SynthesisModel
	}
	if userConfig.ReportModel.Name == "" {
		userConfig.ReportModel = defaultConfig.ReportModel
	}
	if userConfig.MaxSteps == 0 {
		userConfig.MaxSteps = defaultConfig.MaxSteps
	}
	if userConfig.MaxResults == 0 {
		userConfig.MaxResults = defaultConfig.MaxResults
	}
	if userConfig.Timeout == "" {
		userConfig.Timeout = defaultConfig.Timeout
	}
	if userConfig.ResearchDepth == "" {
		userConfig.ResearchDepth = defaultConfig.ResearchDepth
	}
	if userConfig.MaxResearcherIterations == 0 {
		userConfig.MaxResearcherIterations = defaultConfig.MaxResearcherIterations
	}
	if userConfig.ReportFormat == "" {
		userConfig.ReportFormat = defaultConfig.ReportFormat
	}
	if userConfig.ReportLang == "" {
		userConfig.ReportLang = defaultConfig.ReportLang
	}
	if userConfig.ExternalSearch.Engine == "" {
		userConfig.ExternalSearch = defaultConfig.ExternalSearch
	}

	return userConfig
}
