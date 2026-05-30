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
