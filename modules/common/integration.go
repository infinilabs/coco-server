/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

const (
	IntegrationTypeEmbedded = "embedded"
	IntegrationTypeFloating = "floating"
)

type Integration struct {
	CombinedFullText

	Type       string   `json:"type,omitempty" elastic_mapping:"type:{type:keyword,copy_to:combined_fulltext}"` // Type of the Integeration, eg: embedded, floating
	Name       string   `json:"name,omitempty" elastic_mapping:"name:{type:keyword,copy_to:combined_fulltext}"` // Display name of this embedding
	Datasource []string `json:"datasource,omitempty" elastic_mapping:"datasource:{type:keyword}"`               // Datasource ID

	EnabledModule ModuleConfig        `json:"enabled_module,omitempty" elastic_mapping:"enabled_module:{type:object}"`                      // Enabled module configuration
	AccessControl AccessControlConfig `json:"access_control,omitempty" elastic_mapping:"access_control:{type:object}"`                      // Access control configuration
	Appearance    AppearanceConfig    `json:"appearance,omitempty" elastic_mapping:"appearance:{type:object}"`                              // Appearance configuration
	Cors          CorsConfig          `json:"cors,omitempty" elastic_mapping:"cors:{type:object}"`                                          // CORS configuration
	Token         string              `json:"token,omitempty" elastic_mapping:"token:{type:keyword}"`                                       // Token for authentication
	Description   string              `json:"description,omitempty" elastic_mapping:"description:{type:keyword,copy_to:combined_fulltext}"` // Description of the embedding
	Enabled       bool                `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`                                             // Whether the embedding is enabled
}

type CorsConfig struct {
	Enabled        bool     `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`                           // Whether CORS is enabled
	AllowedOrigins []string `json:"allowed_origins,omitempty" elastic_mapping:"allowed_origins:{type:keyword}"` // Allowed origins
}

type AppearanceConfig struct {
	Theme string `json:"theme,omitempty" elastic_mapping:"theme:{type:keyword}"` // Theme of the embedding
}
type AccessControlConfig struct {
	Authentication bool `json:"authentication" elastic_mapping:"authentication:{type:keyword}"` // Whether authentication is required
	ChatHistory    bool `json:"chat_history" elastic_mapping:"chat_history:{type:keyword}"`     // Whether chat history is enabled
}

type ModuleConfig struct {
	Search   SearchModuleConfig `json:"search,omitempty" elastic_mapping:"search:{type:object}"`      // Search configuration
	AIChat   AIChatModuleConfig `json:"ai_chat,omitempty" elastic_mapping:"ai_chat:{type:object}"`    // AI Chat configuration
	Features []string           `json:"features,omitempty" elastic_mapping:"features:{type:keyword}"` // Features enabled
}

type SearchModuleConfig struct {
	Enabled     bool   `json:"enabled" elastic_mapping:"enabled:{type:keyword}"`
	Placeholder string `json:"placeholder,omitempty" elastic_mapping:"placeholder:{type:keyword}"` // Placeholder text for search input
}

type AIChatModuleConfig SearchModuleConfig
