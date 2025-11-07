/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

type Assistant struct {
	CombinedFullText
	Name           string           `json:"name" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"`
	Description    string           `json:"description" elastic_mapping:"description:{type:text,copy_to:combined_fulltext}"`
	Icon           string           `json:"icon" elastic_mapping:"icon:{enabled:false}"`
	Type           string           `json:"type" elastic_mapping:"type:{type:keyword}"` // assistant type, default value: "simple", possible values: "simple", "deep_think", "external_workflow"
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
