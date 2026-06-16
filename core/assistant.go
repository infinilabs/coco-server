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
	Name         string       `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"`
	Description  string       `json:"description" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon         string       `json:"icon" elastic_mapping:"icon:{enabled:false}"`
	Type         string       `json:"type" elastic_mapping:"type:{type:keyword}"` // assistant type, default value: "simple", possible values: "simple", "deep_think", "external_workflow", "deep_research"
	Category     string       `json:"category,omitempty" elastic_mapping:"category:{type:keyword}"`
	Tags         []string     `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword}"`
	Keepalive    string       `json:"keepalive" elastic_mapping:"keepalive:{type:keyword}"`
	Enabled      bool         `json:"enabled" elastic_mapping:"enabled:{type:boolean}"`
	Builtin      bool         `json:"builtin" elastic_mapping:"builtin:{type:boolean}"` // Whether the model provider is builtin
	UploadConfig UploadConfig `json:"upload,omitempty" elastic_mapping:"upload:{type:object,enabled:false}"`

	// This field contains assistant-specific configuration settings
	//
	// After loading, Config is decoded into the corresponding typed field:
	// DeepThinkConfig or DeepResearchConfig (both tagged json:"-").
	Config interface{} `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"`
	// used by  simple/deep_think
	AnsweringModel ModelConfig `json:"answering_model" elastic_mapping:"answering_model:{type:object,enabled:false}"`
	// used by simple/deep_think; deep_research uses InternalSearch.DatasourceIDs instead
	Datasource DatasourceConfig `json:"datasource" elastic_mapping:"datasource:{type:object,enabled:false}"`
	// used by simple/deep_think
	ToolsConfig ToolsConfig `json:"tools,omitempty" elastic_mapping:"tools:{type:object,enabled:false}"`
	// used by simple/deep_think
	MCPConfig MCPConfig `json:"mcp_servers,omitempty" elastic_mapping:"mcp_servers:{type:object,enabled:false}"`
	// used by simple/deep_think
	ChatSettings ChatSettings `json:"chat_settings" elastic_mapping:"chat_settings:{type:object,enabled:false}"`
	// used by simple/deep_think (passed as system prompt to GenerateFinalResponse)
	RolePrompt string `json:"role_prompt" elastic_mapping:"role_prompt:{enabled:false}"`

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
	// Optional. Restricts internal search to these datasource IDs. Default: empty,
	// which allows all accessible datasources.
	DatasourceIDs []string `json:"datasource_ids,omitempty"`
}

// DeepResearchExternalSearchConfig controls external web search behaviour.
type DeepResearchExternalSearchConfig struct {
	// Optional. External search engine: "duckduckgo", "wikipedia", or "tavily".
	// Default: "duckduckgo".
	Engine string `json:"engine"`
	// Required when Engine is "tavily"; otherwise optional. Default: "".
	APIKey string `json:"api_key,omitempty"`
}

type DeepResearchConfig struct {
	// Models — one per pipeline stage; each may point to a different provider/model.
	// When a field is empty, the assistant's top-level AnsweringModel is used as
	// a fallback for that stage.

	// Optional. Model used to decompose the query into a step-by-step research
	// plan. Default: empty, which falls back to the assistant's AnsweringModel.
	PlanningModel ModelConfig `json:"planning_model"`
	// Optional. Model used to analyze search results for each individual research
	// step. Default: empty, which falls back to the assistant's AnsweringModel.
	ResearchModel ModelConfig `json:"research_model"`
	// Optional. Model used to synthesize findings across sources within a step.
	// Default: empty, which falls back to the assistant's AnsweringModel.
	SynthesisModel ModelConfig `json:"synthesis_model"`
	// Optional. Model used to write the final structured report. Default: empty,
	// which falls back to the assistant's AnsweringModel.
	ReportModel ModelConfig `json:"report_model"`

	// Execution limits

	// Optional. Maximum number of research steps the planner may generate.
	// Default: 5. If provided, it must be positive.
	MaxSteps int `json:"max_steps"`
	// Optional. Maximum number of researcher loop iterations over the plan.
	// Default: 5.
	MaxResearcherIterations int `json:"max_researcher_iterations"`
	// Optional. Maximum number of parallel research workers (v1 pipeline only).
	// Default: 5.
	MaxConcurrentResearchUnits int `json:"max_concurrent_research_units"`
	// Optional. Maximum number of search results fetched per query. Default: 5.
	// If provided, it must be positive.
	MaxResults int `json:"max_results"`
	// Optional. Total research deadline as a Go duration string, e.g. "30m" or
	// "1h". Default: "" (no timeout).
	Timeout string `json:"timeout"`
	// Optional. Effort level for the research pipeline: "basic", "comprehensive",
	// or "exhaustive". Default: "basic".
	ResearchDepth string `json:"research_depth"`

	// Output

	// Optional. When true, appends a citations section to the report. Default: false.
	IncludeSources bool `json:"include_sources"`
	// Optional. Citation style: "APA", "MLA", or "" for plain Markdown links.
	// Default: "".
	SourceFormat string `json:"source_format"`
	// Optional. Rendered output format: "markdown" or "html". Default: "markdown".
	ReportFormat string `json:"report_format"`
	// Optional. Report language as a BCP 47 tag, e.g. "en-US" or "zh-CN".
	// Default: "" (inherits system locale).
	ReportLang string `json:"report_lang"`

	// Search

	// Optional. Internal enterprise search settings. Default: zero value, which
	// allows all accessible datasources.
	InternalSearch DeepResearchInternalSearchConfig `json:"internal_search"`
	// Optional. External web search settings. Default: duckduckgo.
	ExternalSearch DeepResearchExternalSearchConfig `json:"external_search"`
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

// Validate validates the deep research configuration
func (cfg *DeepResearchConfig) Validate() error {
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
	// Validate external search engine if one is configured
	if cfg.ExternalSearch.Engine != "" {
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
	validFormats := []string{"markdown", "html", "pdf"}
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

// MergeDeepResearchConfig fills zero-valued fields in userConfig with hardcoded
// defaults. Fields whose zero value is already the intended default (e.g.
// Timeout, ReportLang, model configs) are left untouched.
func MergeDeepResearchConfig(userConfig *DeepResearchConfig) *DeepResearchConfig {
	d := DeepResearchConfig{
		MaxSteps:                   5,
		MaxResearcherIterations:    5,
		MaxConcurrentResearchUnits: 5,
		MaxResults:                 5,
		ResearchDepth:              "basic",
		ReportFormat:               "markdown",
		ExternalSearch:             DeepResearchExternalSearchConfig{Engine: "duckduckgo"},
	}
	if userConfig == nil {
		return &d
	}
	if userConfig.MaxSteps == 0 {
		userConfig.MaxSteps = d.MaxSteps
	}
	if userConfig.MaxResearcherIterations == 0 {
		userConfig.MaxResearcherIterations = d.MaxResearcherIterations
	}
	if userConfig.MaxConcurrentResearchUnits == 0 {
		userConfig.MaxConcurrentResearchUnits = d.MaxConcurrentResearchUnits
	}
	if userConfig.MaxResults == 0 {
		userConfig.MaxResults = d.MaxResults
	}
	if userConfig.ResearchDepth == "" {
		userConfig.ResearchDepth = d.ResearchDepth
	}
	if userConfig.ReportFormat == "" {
		userConfig.ReportFormat = d.ReportFormat
	}
	if userConfig.ExternalSearch.Engine == "" {
		userConfig.ExternalSearch.Engine = d.ExternalSearch.Engine
	}
	return userConfig
}
