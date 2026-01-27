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

type Assistant struct {
	CombinedFullText

	Name           string           `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext,fields:{text: {type: text}, pinyin: {type: text, analyzer: pinyin_analyzer}}}"`
	Description    string           `json:"description" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon           string           `json:"icon" elastic_mapping:"icon:{enabled:false}"`
	Type           string           `json:"type" elastic_mapping:"type:{type:keyword}"` // assistant type, default value: "simple", possible values: "simple", "deep_think", "external_workflow", "deep_research"
	Category       string           `json:"category,omitempty" elastic_mapping:"category:{type:keyword}"`
	Tags           []string         `json:"tags,omitempty" elastic_mapping:"tags:{type:keyword}"`
	Config         interface{}      `json:"config,omitempty" elastic_mapping:"config:{enabled:false}"` // Assistant-specific configuration settings with type
	AnsweringModel ModelConfig      `json:"answering_model" elastic_mapping:"answering_model:{type:object,enabled:false}"`
	Datasource     DatasourceConfig `json:"datasource" elastic_mapping:"datasource:{type:object,enabled:false}"`
	ToolsConfig    ToolsConfig      `json:"tools,omitempty" elastic_mapping:"tools:{type:object,enabled:false}"`
	MCPConfig      MCPConfig        `json:"mcp_servers,omitempty" elastic_mapping:"mcp_servers:{type:object,enabled:false}"`
	UploadConfig   UploadConfig     `json:"upload,omitempty" elastic_mapping:"upload:{type:object,enabled:false}"`
	Keepalive      string           `json:"keepalive" elastic_mapping:"keepalive:{type:keyword}"`
	Enabled        bool             `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`
	ChatSettings   ChatSettings     `json:"chat_settings" elastic_mapping:"chat_settings:{type:object,enabled:false}"`
	Builtin        bool             `json:"builtin" elastic_mapping:"builtin:{type:keyword}"`          // Whether the model provider is builtin
	RolePrompt     string           `json:"role_prompt" elastic_mapping:"role_prompt:{enabled:false}"` // Role prompt for the assistant

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

type DeepResearchConfig struct {
	// Research Model Configuration
	PlanningModel  ModelConfig `json:"planning_model"`  // For research planning and query decomposition
	ResearchModel  ModelConfig `json:"research_model"`  // For individual research step analysis
	SynthesisModel ModelConfig `json:"synthesis_model"` // For information synthesis across sources
	ReportModel    ModelConfig `json:"report_model"`    // For final report generation
	PodcastModel   ModelConfig `json:"podcast_model"`   // For podcast script generation

	// Research Execution Settings
	MaxSteps                   int    `json:"max_steps"` // Maximum steps in research workflow
	MaxResearcherIterations    int    `json:"max_researcher_iterations"`
	MaxConcurrentResearchUnits int    `json:"max_concurrent_research_units"`
	MaxResults                 int    `json:"max_results"`           // Maximum search results per query
	Timeout                    string `json:"timeout"`               // Research timeout (e.g., "1h", "30m")
	ResearchDepth              string `json:"research_depth"`        // "basic", "comprehensive", "exhaustive"
	IncludeSources             bool   `json:"include_sources"`       // Include sources in final report
	SourceFormat               string `json:"source_format"`         // "APA", "MLA", etc.
	HandleContradictions       bool   `json:"handle_contradictions"` // Detect and handle conflicting information

	// Search Configuration
	SearchEngines      []string `json:"search_engines"` // Enabled search engines ["duckduckgo", "wikipedia", "bing"]
	MaxSourcesPerQuery int      `json:"max_sources_per_query"`
	QualityThreshold   float64  `json:"quality_threshold"` // Minimum quality score (0.0-1.0)
	Language           string   `json:"language"`          // Target language ("zh-CN", "en" etc.)
	TimeHorizon        string   `json:"time_horizon"`      // Time range for research ("recent", "last_year", "custom")

	// Output Configuration
	ReportFormat    string `json:"report_format"`    // "markdown", "html", "pdf"
	GeneratePodcast bool   `json:"generate_podcast"` // Enable podcast generation
	IncludeImages   bool   `json:"include_images"`   // Include relevant images in report
	VisualElements  bool   `json:"visual_elements"`  // Include charts, timelines, etc.
	ReportLang      string `json:"report_lang"`      // Report language (BCP 47: "en-US", "zh-CN", etc.)

	// Tool Integration Settings
	ToolsConfig      ToolsConfig `json:"tools_config"`      // Tool availability settings
	EnableFactCheck  bool        `json:"enable_fact_check"` // Cross-reference facts across sources
	CitationTracking bool        `json:"citation_tracking"` // Track citations and references
	TavilyAPIKey     string      `json:"tavily_api_key"`    // Tavily API key for external web search

	// Advanced Settings
	RetryAttempts             int                `json:"retry_attempts"`     // Number of retry attempts on failure
	RateLimiting              RateLimitingConfig `json:"rate_limiting"`      // API rate limiting settings
	ProgressReporting         bool               `json:"progress_reporting"` // Enable detailed progress reporting
	Validation                ValidationConfig   `json:"validation"`         // Content validation settings
	MaxToolCallIterations     int                `json:"max_tool_call_iterations"`
	CompressionModelMaxTokens int                `json:"compression_model_max_tokens"`
}

// RateLimitingConfig defines rate limiting for external APIs
type RateLimitingConfig struct {
	WebSearchRequests int `json:"web_search_requests_per_minute"`
	WikipediaRequests int `json:"wikipedia_requests_per_minute"`
	LLMRequests       int `json:"llm_requests_per_minute"`
	RetryDelayMs      int `json:"retry_delay_ms"`
	MaxRetryAttempts  int `json:"max_retry_attempts"`
}

// ValidationConfig defines content validation settings
type ValidationConfig struct {
	MinSourceQuality     float64            `json:"min_source_quality"`     // Minimum source quality score
	MinRelevanceScore    float64            `json:"min_relevance_score"`    // Minimum relevance score
	ContentFreshnessDays int                `json:"content_freshness_days"` // Maximum age of research content in days
	DomainCredentials    map[string]float64 `json:"domain_credentials"`     // Domain reputation scores
}

type WorkflowConfig struct {
	// Workflow-specific configuration
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

type ModelConfig struct {
	ProviderID   string        `json:"provider_id,omitempty"`
	Name         string        `json:"name"`
	Settings     ModelSettings `json:"settings"`
	PromptConfig *PromptConfig `json:"prompt,omitempty"`
	Keepalive    string        `json:"keepalive"`
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
		PodcastModel: ModelConfig{
			ProviderID: "qianwen",
			Name:       "qwq-plus",
			Settings: ModelSettings{
				Temperature: 0.8, // Slightly more creative for podcast generation
				TopP:        0.95,
				MaxTokens:   2500,
			},
		},
		MaxSteps:                   50,
		MaxResearcherIterations:    10,
		MaxConcurrentResearchUnits: 3,
		MaxToolCallIterations:      20,
		CompressionModelMaxTokens:  8192,
		MaxResults:                 100,
		Timeout:                    "1h",
		ResearchDepth:              "comprehensive",
		IncludeSources:             true,
		SourceFormat:               "APA",
		HandleContradictions:       true,
		SearchEngines:              []string{"duckduckgo", "wikipedia", "bing"},
		MaxSourcesPerQuery:         20,
		QualityThreshold:           0.7,
		Language:                   "zh-CN",
		TimeHorizon:                "recent",
		ReportLang:                 "en-US",
		ReportFormat:               "html",
		GeneratePodcast:            false, // Default to false - can be explicitly enabled
		IncludeImages:              true,
		VisualElements:             true,
		ToolsConfig: ToolsConfig{
			Enabled: true,
			BuiltinTools: BuiltinToolsConfig{
				Calculator: false,
				Wikipedia:  true,
				Duckduckgo: true,
				Scraper:    true,
			},
		},
		EnableFactCheck:   true,
		CitationTracking:  true,
		TavilyAPIKey:      "", // Empty by default, user must configure
		RetryAttempts:     3,
		ProgressReporting: true,
		RateLimiting: RateLimitingConfig{
			WebSearchRequests: 30,
			WikipediaRequests: 60,
			LLMRequests:       120,
			RetryDelayMs:      1000,
			MaxRetryAttempts:  3,
		},
		Validation: ValidationConfig{
			MinSourceQuality:     0.5,
			MinRelevanceScore:    0.3,
			ContentFreshnessDays: 90,
			DomainCredentials: map[string]float64{
				"wikipedia.org":    0.9,
				"academic.com":     0.85,
				"researchgate.net": 0.8,
				"medium.com":       0.6,
				"blogspot.com":     0.5,
			},
		},
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
	if len(cfg.SearchEngines) == 0 {
		return fmt.Errorf("at least one search engine must be enabled")
	}
	if cfg.QualityThreshold < 0 || cfg.QualityThreshold > 1 {
		return fmt.Errorf("quality_threshold must be between 0 and 1")
	}

	// Validate research depth
	validDepths := []string{"basic", "comprehensive", "exhaustive"}
	if !slices.Contains(validDepths, cfg.ResearchDepth) {
		return fmt.Errorf("research_depth must be one of: %v", validDepths)
	}

	// Validate language
	if cfg.Language != "zh-CN" && cfg.Language != "en" {
		return fmt.Errorf("language must be either zh-CN or en")
	}

	// Validate time horizon
	validHorizons := []string{"recent", "last_year", "last_2_years", "custom"}
	if !slices.Contains(validHorizons, cfg.TimeHorizon) {
		return fmt.Errorf("time_horizon must be one of: %v", validHorizons)
	}

	// Validate report format
	validFormats := []string{"markdown", "html", "pdf"}
	if !slices.Contains(validFormats, cfg.ReportFormat) {
		return fmt.Errorf("report_format must be one of: %v", validFormats)
	}

	// Validate Tavily API key
	if cfg.TavilyAPIKey == "" {
		return fmt.Errorf("tavily_api_key is required")
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
	if userConfig.PodcastModel.Name == "" {
		userConfig.PodcastModel = defaultConfig.PodcastModel
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
	if len(userConfig.SearchEngines) == 0 {
		userConfig.SearchEngines = defaultConfig.SearchEngines
	}
	if userConfig.QualityThreshold == 0 {
		userConfig.QualityThreshold = defaultConfig.QualityThreshold
	}
	if userConfig.Language == "" {
		userConfig.Language = defaultConfig.Language
	}
	if userConfig.TimeHorizon == "" {
		userConfig.TimeHorizon = defaultConfig.TimeHorizon
	}
	if userConfig.ReportFormat == "" {
		userConfig.ReportFormat = defaultConfig.ReportFormat
	}
	if userConfig.RateLimiting.WebSearchRequests == 0 {
		userConfig.RateLimiting = defaultConfig.RateLimiting
	}
	if len(userConfig.Validation.DomainCredentials) == 0 {
		userConfig.Validation.DomainCredentials = defaultConfig.Validation.DomainCredentials
	}
	if userConfig.ReportLang == "" {
		userConfig.ReportLang = defaultConfig.ReportLang
	}

	return userConfig
}
