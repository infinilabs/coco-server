/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "strings"

type Config struct {
	ServerInfo         *ServerInfo         `config:"server" json:"server,omitempty"`
	AppSettings        *AppSettings        `config:"app_settings" json:"app_settings,omitempty"`
	SearchSettings     *SearchSettings     `config:"search_settings" json:"search_settings,omitempty"`
	DefaultModel       *DefaultModel       `config:"default_model" json:"default_model,omitempty"`
	DocumentProcessing *DocumentProcessing `config:"document_processing" json:"document_processing,omitempty"`
	DataSecurity       *DataSecurity       `config:"data_security" json:"data_security,omitempty"`
	EngineAI           *EngineAI           `config:"engine_ai" json:"engine_ai,omitempty"`
	Appearance         *AppearanceSettings `config:"appearance" json:"appearance,omitempty"`
}

// Settings under the "Appearance" tab: site branding. Images are stored as
// data URLs (or http(s) URLs) directly in the section; the whole section is
// exposed unauthenticated via GET /setting/application so the login page, the
// boot splash and the search page can pick branding up before any login.
type AppearanceSettings struct {
	// Title is the site name: appended as the browser-tab title suffix and
	// shown next to the logo in the app shell. Empty keeps the built-ins.
	Title string `config:"title" json:"title,omitempty"`
	// Slogan is shown on the login page's left panel and falls back as the
	// search home welcome text when the integration doesn't define one.
	Slogan string `config:"slogan" json:"slogan,omitempty"`
	// ThemeColors are enforced on every client: each boot (and each light/dark
	// switch) re-applies them over any locally cached theme.
	ThemeColors *AppearanceThemeColors `config:"theme_colors" json:"theme_colors,omitempty"`
	Logo        *AppearanceLogo        `config:"logo" json:"logo,omitempty"`
	Search      *AppearanceSearch      `config:"search" json:"search,omitempty"`
	Login       *AppearanceLogin       `config:"login" json:"login,omitempty"`
}

// AppearanceThemeColors: one brand color per mode plus shared functional
// colors and per-mode neutral surface colors. Empty values keep the built-in
// theme defaults from the web client.
type AppearanceThemeColors struct {
	// Primary for light mode; PrimaryDark for dark mode (falls back to Primary).
	Primary     string         `config:"primary" json:"primary,omitempty"`
	PrimaryDark string         `config:"primary_dark" json:"primary_dark,omitempty"`
	Success     string         `config:"success" json:"success,omitempty"`
	Warning     string         `config:"warning" json:"warning,omitempty"`
	Error       string         `config:"error" json:"error,omitempty"`
	Light       *NeutralColors `config:"light" json:"light,omitempty"`
	Dark        *NeutralColors `config:"dark" json:"dark,omitempty"`
}

// NeutralColors are the per-mode surface colors of the web theme
// (theme settings tokens). Values are hex colors, translated to rgb() by the
// web client before they reach the CSS vars.
type NeutralColors struct {
	Layout    string `config:"layout" json:"layout,omitempty"`
	Container string `config:"container" json:"container,omitempty"`
	BaseText  string `config:"base_text" json:"base_text,omitempty"`
}

// AppearanceLogo is the app-shell logo: Light for light mode, Dark for dark
// mode, Icon for the collapsed sider (square).
type AppearanceLogo struct {
	Light string `config:"light" json:"light,omitempty"`
	Dark  string `config:"dark" json:"dark,omitempty"`
	Icon  string `config:"icon" json:"icon,omitempty"`
}

// AppearanceSearch brands the search home: the banner logo above the search
// box and the page background image, each with a light and dark variant.
type AppearanceSearch struct {
	Logo       *AppearanceImagePair `config:"logo" json:"logo,omitempty"`
	Background *AppearanceImagePair `config:"background" json:"background,omitempty"`
}

type AppearanceImagePair struct {
	Light string `config:"light" json:"light,omitempty"`
	Dark  string `config:"dark" json:"dark,omitempty"`
}

// AppearanceLogin personalizes the login page's left panel: a background
// color (defaults to the historic #0087FF) and a background image replacing
// the bundled illustration.
type AppearanceLogin struct {
	BackgroundColor string `config:"background_color" json:"background_color,omitempty"`
	BackgroundImage string `config:"background_image" json:"background_image,omitempty"`
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
	// SearchType is the default search strategy /query/_search falls back
	// to when the request carries no explicit search_type — one of the
	// values in searchTypes. Empty means keyword, the historic behavior,
	// so settings saved before this field existed keep their meaning.
	SearchType string `json:"search_type,omitempty"`
}

// DefaultType is the nil-safe resolution of the configured default: no
// saved search_settings section (or a stale value) reads as keyword.
func (s *SearchSettings) DefaultType() string {
	if s == nil || !IsValidSearchType(s.SearchType) {
		return "keyword"
	}
	return s.SearchType
}

// searchTypes are the strategies /query/_search accepts, both as an
// explicit ?search_type= and as the operator-configured default
// (search_settings.search_type). semantic and hybrid degrade gracefully
// when the engine has no AI pipelines — the plan notes travel in the
// Warning header.
var searchTypes = map[string]bool{
	"keyword":    true,
	"semantic":   true,
	"hybrid":     true,
	"hybrid_rrf": true,
}

func IsValidSearchType(t string) bool {
	return searchTypes[t]
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
	RerankModel    *ModelId `config:"rerank_model" json:"rerank_model,omitempty"`

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
	// Tika server used by document-processing pipeline stages. A pipeline may
	// still pin its own tika_endpoint; this global setting only fills the gap
	// when it doesn't.
	TikaEndpoint string `config:"tika_endpoint" json:"tika_endpoint,omitempty"`
	// TikaTimeoutInSeconds is the per-request timeout for Tika calls that have
	// no pipeline-level override.
	TikaTimeoutInSeconds int `config:"tika_timeout_in_seconds" json:"tika_timeout_in_seconds,omitempty"`
}

// DefaultTikaEndpoint matches the value the setup templates shipped with
// before the setting moved into this section, so pre-existing deployments
// that never touch the UI keep talking to the same server.
const DefaultTikaEndpoint = "http://127.0.0.1:9998"

// DefaultTikaTimeoutInSeconds covers OCR-heavy PDFs through Tika's full
// extraction; the setup templates used the same value.
const DefaultTikaTimeoutInSeconds = 360

// EffectiveTikaEndpoint is the nil-safe resolution of the configured Tika
// address: empty reads as the historic default.
func (s *DocumentProcessing) EffectiveTikaEndpoint() string {
	if s == nil || strings.TrimSpace(s.TikaEndpoint) == "" {
		return DefaultTikaEndpoint
	}
	return strings.TrimSpace(s.TikaEndpoint)
}

// EffectiveTikaTimeoutInSeconds is the nil-safe resolution of the configured
// Tika timeout: zero (unset) reads as the historic default.
func (s *DocumentProcessing) EffectiveTikaTimeoutInSeconds() int {
	if s == nil || s.TikaTimeoutInSeconds <= 0 {
		return DefaultTikaTimeoutInSeconds
	}
	return s.TikaTimeoutInSeconds
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

// Names of the engine pipelines Coco manages. Stable so operators can find
// them on the engine; the description marks them Coco-managed.
const (
	EngineIngestPipelineName = "coco-embedding"
	EngineSearchPipelineName = "coco-semantic-rrf"
	// EngineAITextFieldDefault is the field the text_embedding processor reads.
	EngineAITextFieldDefault = "ai_insights.text"
)

// Settings under the "Engine AI" tab: the AI capabilities Coco curates onto
// the backing engine. Coco generates the engine's ingest pipeline
// (text_embedding) and search pipeline (semantic_query_enricher +
// hybrid_ranker_processor) from these values plus the referenced model
// provider, keeps them in sync, and passes the search pipeline on
// semantic/hybrid searches — one model config in Coco, applied at both
// ingest time and query time on the engine.
type EngineAI struct {
	Enabled bool `json:"enabled"`
	// EmbeddingModel selects the Coco model provider used for both engine
	// pipelines. Falls back to DefaultModel.EmbeddingModel when empty.
	EmbeddingModel *ModelId `config:"embedding_model" json:"embedding_model,omitempty"`
	// TextField is what the ingest-side text_embedding processor embeds.
	TextField string `config:"text_field" json:"text_field,omitempty"`
	// VectorField is where the ingest processor writes vectors; defaults to
	// the semantic search field derived from RequiredEmbeddingDimension.
	VectorField string `config:"vector_field" json:"vector_field,omitempty"`
	// BatchSize batches inputs per embedding call at ingest time.
	BatchSize int `config:"batch_size" json:"batch_size,omitempty"`
	// RankConstant is the k of the engine-side RRF fusion.
	RankConstant int `config:"rank_constant" json:"rank_constant,omitempty"`
}
