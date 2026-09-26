/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

type Config struct {
	ServerInfo         *ServerInfo         `config:"server" json:"server,omitempty"`
	AppSettings        *AppSettings        `config:"app_settings" json:"app_settings,omitempty"`
	SearchSettings     *SearchSettings     `config:"search_settings" json:"search_settings,omitempty"`
	DefaultModel       *DefaultModel       `config:"default_model" json:"default_model,omitempty"`
	DocumentProcessing *DocumentProcessing `config:"document_processing" json:"document_processing,omitempty"`
	DataSecurity       *DataSecurity       `config:"data_security" json:"data_security,omitempty"`
}

type AppSettings struct {
	Chat *ChatConfig `json:"chat,omitempty" config:"chat" `
}

type ChatConfig struct {
	ChatStartPageConfig *ChatStartPageConfig `config:"start_page" json:"start_page,omitempty"`
}

type SearchSettings struct {
	Enabled     bool   `json:"enabled"`
	Integration string `json:"integration"`
}

// Uniquely identifies a model.
//
// It is only a reference, it does not contain any model configurations.
type ModelId struct {
	// Model Provider ID
	ProviderID string `config:"provider_id" json:"provider_id,omitempty"`
	// Model ID
	ID string `config:"id" json:"id,omitempty"`
}

// Settings under the "Default Model" tab.
type DefaultModel struct {
	LanguageModel  *ModelId `config:"language_model" json:"language_model,omitempty"`
	VisionModel    *ModelId `config:"vision_model" json:"vision_model,omitempty"`
	EmbeddingModel *ModelId `config:"embedding_model" json:"embedding_model,omitempty"`

	/*
	 * Models used during chatting with various assistants.
	 *
	 * Fallback strategy:
	 *   1. Model specified in the assistant setting
	 *   2. The below default model
	 *   3. default language model
	 */
	IntentAnalysisModel *ModelId `config:"intent_analysis_model" json:"intent_analysis_model,omitempty"`
	PickingDocModel     *ModelId `config:"picking_doc_model" json:"picking_doc_model,omitempty"`
	PickingToolModel    *ModelId `config:"picking_tool_model" json:"picking_tool_model,omitempty"`
	AnsweringModel      *ModelId `config:"answering_model" json:"answering_model,omitempty"`
}

// Settings under the "Document Processing" tab.
type DocumentProcessing struct {
	// If the user didn't specify which pipeline to run to process the uploaded
	// attachments in an assistant's settings, use this one.
	DefaultPipelineForAttachment string `config:"default_pipeline_for_attachment" json:"default_pipeline_for_attachment,omitempty"`
	// If the user didn't specify which pipeline to run to process the fetched
	// documents in an data source's settings, use this one.
	DefaultPipelineForDocument string `config:"default_pipeline_for_document" json:"default_pipeline_for_document,omitempty"`
	// Default language used by pipeline stages that invoke an LLM to generate
	// content (summaries, tags, etc.) when no per-pipeline override is set.
	// Expected to be a BCP 47 tag, e.g. "en-US", "zh-CN".
	LLMGenerationLanguage string `config:"llm_generation_language" json:"llm_generation_language,omitempty"`
}

// Settings under the "Data Security" tab: dynamic content masking before
// recalled documents are handed to a model, and field-level access
// restrictions layered on top of the existing index/document sharing.
type DataSecurity struct {
	Masking     *MaskingSettings     `config:"masking" json:"masking,omitempty"`
	FieldAccess *FieldAccessSettings `config:"field_access" json:"field_access,omitempty"`
}

type MaskingSettings struct {
	Enabled bool          `json:"enabled"`
	Rules   []MaskingRule `json:"rules,omitempty"`
}

// MaskingRule is one regex substitution applied to document content on its
// way into a model prompt. Patterns use Go's RE2 syntax.
type MaskingRule struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement,omitempty"`
	Enabled     bool   `json:"enabled"`
}

type FieldAccessSettings struct {
	// One entry per role; a user matching several roles gets the union of
	// the excluded fields.
	Restrictions []FieldRestriction `json:"restrictions,omitempty"`
}

type FieldRestriction struct {
	Role          string   `json:"role"`
	ExcludeFields []string `json:"exclude_fields,omitempty"`
}
