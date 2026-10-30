/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"strconv"
	"strings"
)

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

// FieldWeight is one keyword-leg searchable field with its boost (BM25
// relevance weight). Weight 1 means no boost.
type FieldWeight struct {
	Field  string  `json:"field"`
	Weight float64 `json:"weight"`
}

type SearchSettings struct {
	Enabled     bool   `json:"enabled"`
	Integration string `json:"integration"`
	// SearchType is the default search strategy /query/_search falls back
	// to when the request carries no explicit search_type — one of the
	// values in searchTypes. Empty means keyword, the historic behavior,
	// so settings saved before this field existed keep their meaning.
	SearchType string `json:"search_type,omitempty"`
	// WikiProjection serves the wiki retrieval leg from the document index
	// (published-article projections, type=wiki_article) instead of the
	// wiki_article index — the "KB as a built-in datasource" unification
	// (W16a). Off by default: flip it after the one-off backfill so the
	// leg never serves a half-empty index.
	WikiProjection bool `json:"wiki_projection,omitempty"`
	// ChunkRoute adds the standalone KnowledgeChunk index as an extra RRF
	// leg (W3 方案 c): BM25 over retrievable child chunks, hit a chunk →
	// cite its mom section. Off by default — the text/semantic document
	// legs stay; flip after _reprocess has projected enough documents.
	ChunkRoute bool `json:"chunk_route,omitempty"`
	// RsaEnabled turns on the RSA federated leg (DESIGN-RSA.md): enabled
	// remote sources are queried at search time, mapped into standard
	// documents and fused via RRF — zero local copies. Off by default;
	// every source carries its own timeout/QPS/cache budget so the leg
	// degrades alone when a backend is slow.
	RsaEnabled bool `json:"rsa_enabled,omitempty"`
	// RsaMaxSourcesPerQuery caps the fan-out breadth of one search (the
	// anti-flatten-the-backend knob, default 8).
	RsaMaxSourcesPerQuery int `json:"rsa_max_sources_per_query,omitempty"`
	// RsaCacheSize bounds the in-memory per-source result cache entries
	// (default 256, shared across sources, FIFO eviction).
	RsaCacheSize int `json:"rsa_cache_size,omitempty"`

	// ---- retrieval strategy (搜索设置 · 检索策略) ----
	// NOTE: none of these carry omitempty on purpose — the settings merge is
	// a shallow per-key overwrite, so omitted keys could never be cleared
	// back to their zero value ("off"). The UI always saves the full section.

	// FieldWeights overrides the keyword/hybrid text leg's searchable fields
	// and their boosts ("title" weight 10 → title^10). Empty or nil keeps
	// the built-in defaults in QueryDocuments.
	FieldWeights []FieldWeight `json:"field_weights"`
	// DefaultSort orders results when the request carries no explicit sort:
	// "" / "relevance" (BM25 score, the historic behavior), "created_desc"
	// or "created_asc". Never applied to hybrid_rrf, whose ordering is the
	// fused rank.
	DefaultSort string `json:"default_sort"`
	// MinScore drops hits scoring below the threshold; 0 disables it. Not
	// applied to hybrid_rrf (its scores are rank-fusion values, not BM25).
	MinScore float64 `json:"min_score"`
	// MatchOperator combines terms inside every leaf match of the text leg:
	// "" / "OR" (any term, the historic behavior) or "AND" (all terms).
	MatchOperator string `json:"match_operator"`
	// MinimumShouldMatch is a raw ES minimum_should_match spec ("75%",
	// "2", "2<75%") applied to the text leg's should logic. Empty = off.
	MinimumShouldMatch string `json:"minimum_should_match"`

	// ---- presentation, suggestions, facets & protections (搜索设置 ·
	// 结果呈现 / 搜索建议 / 聚合与筛选 / 高级) ----
	// same no-omitempty rule as the strategy block above: the settings merge
	// is a shallow per-key overwrite and the UI always saves the full
	// section, so omitted keys could never be cleared back to "default".

	// PageSize is the default results-per-page /query/_search serves when
	// the request carries no explicit ?size=. 0 keeps the historic 10.
	PageSize int `json:"page_size"`
	// MaxPageSize clamps any single request's ?size= on /query/_search (a
	// per-page upper bound, 10..1000). 0 keeps the historic 100.
	MaxPageSize int `json:"max_page_size"`
	// MaxResultWindow caps from+size on /query/_search — deep-pagination
	// protection with the ES max_result_window semantics, enforced by
	// clamping from so the window fits instead of the engine's error.
	// 0 keeps 10000.
	MaxResultWindow int `json:"max_result_window"`
	// SuggestDisabled turns the search-box document suggestions
	// (/query/_suggest without a tag) off. Inverted on purpose so the zero
	// value keeps the historic enabled behavior: settings saved before this
	// field existed (and deployments that never touch the UI) decode as
	// enabled. Studio field-name/value suggestions are not affected.
	SuggestDisabled bool `json:"suggest_disabled"`
	// SuggestSize is the default suggestion count when the request carries
	// no explicit ?size=. 0 keeps 10.
	SuggestSize int `json:"suggest_size"`
	// AggregationBucketCap is the per-aggregation bucket cap on /query/_aggregations. 0 keeps
	// the historic 15.
	AggregationBucketCap int `json:"aggregation_bucket_cap"`
	// ResultSourceIncludes trims every search hit's _source to these fields
	// (ES _source.includes) — smaller payloads and hidden internals. Empty
	// keeps the full document, the historic behavior.
	ResultSourceIncludes []string `json:"result_source_includes"`
	// Fuzziness is the default typo-tolerance distance of the text leg when
	// the request carries no explicit ?fuzziness=. 0 keeps the historic 3
	// (exact matching stays reachable via the request param).
	Fuzziness int `json:"fuzziness"`
	// QueryTimeoutMS caps engine execution time per search (ES ?timeout=).
	// 0 = off, the historic behavior.
	QueryTimeoutMS int `json:"query_timeout_ms"`

	// AggregationsDisabled turns the /query/_aggregations endpoint off.
	// Inverted on purpose so the zero value keeps the historic enabled
	// behavior.
	AggregationsDisabled bool `json:"aggregations_disabled"`
	// AggregationBucketSort orders the aggregation buckets: "" / "count" (count desc, the
	// historic behavior) or "value" (alphabetical by bucket value).
	AggregationBucketSort string `json:"aggregation_bucket_sort"`
	// AggregationFieldsDisabled turns individual aggregation dimensions off
	// (semantic keys: category/source/type/tags/owner/time/color). Inverted
	// storage so the zero value keeps every dimension enabled — same
	// convention as AggregationsDisabled.
	AggregationFieldsDisabled []string `json:"aggregation_fields_disabled"`
	// HotSearchesDisabled turns the hot-queries endpoint (/query/_hot —
	// top search-log queries) off. Inverted so the zero value keeps the
	// historic... endpoint-is-new default: enabled.
	HotSearchesDisabled bool `json:"hot_searches_disabled"`
	// HotSearchesSize is the default top-N the hot-queries endpoint serves
	// when the request carries no explicit ?size=. 0 keeps 10.
	HotSearchesSize int `json:"hot_searches_size"`
	// RecommendTopics are the operator-managed recommended queries the
	// /query/_recommend endpoint serves (rendered under the search box).
	// Forward storage: empty falls back to the static `recommend:` config
	// block in coco.yml (historic behavior).
	RecommendTopics []string `json:"recommend_topics"`
	// RecommendTopicsDisabled turns the recommendation feature off without
	// wiping the stored topics (they reappear when re-enabled). Inverted so
	// the zero value keeps the default: enabled.
	RecommendTopicsDisabled bool `json:"recommend_topics_disabled"`
	// RelatedSearchesDisabled turns the related-searches endpoint
	// (/query/_related — the result-page sidebar suggestions) off. Inverted
	// so the zero value keeps the default: enabled.
	RelatedSearchesDisabled bool `json:"related_searches_disabled"`

	// ---- semantic / AI (搜索设置 · 语义与 AI) ----
	// SemanticMinScore drops semantic/hybrid recall hits scoring below the
	// threshold (cosine-scale scores are far smaller than BM25 — a BM25
	// min_score must not be reused there). 0 disables. Never applied to
	// hybrid_rrf: rank-fusion values are not comparable.
	SemanticMinScore float64 `json:"semantic_min_score"`
	// RRFRankConstant overrides the client-side fusion K of hybrid_rrf when
	// the request carries no explicit ?rrf_k=. 0 keeps the built-in default.
	RRFRankConstant int `json:"rrf_rank_constant"`
	// RouteWeights are the operator's default per-route RRF fusion weights
	// of hybrid_rrf, applied when the request carries no explicit weight
	// params (the studio test panel and the eval runs are the two callers
	// that differ here). Non-positive fields keep the built-in defaults
	// (wiki 2, the rest 1); a per-request ?rewrite=off stays the way to
	// mute a leg without touching the stored defaults.
	RouteWeights *SearchRouteWeights `json:"route_weights"`
	// RerankDisabled turns the post-fusion rerank leg off. Inverted so the
	// zero value keeps the historic enabled behavior.
	RerankDisabled bool `json:"rerank_disabled"`

	// ---- result intervention (搜索设置 · 结果干预) ----
	// BlockedDocIDs never surface in any recall leg that runs through
	// QueryDocuments (keyword/semantic/hybrid, hybrid_rrf routes, suggest).
	BlockedDocIDs []string `json:"blocked_doc_ids"`
	// BlockedDatasourceIDs hide whole datasources from search.
	BlockedDatasourceIDs []string `json:"blocked_datasource_ids"`
	// PinnedRules pin documents on top of the results for an
	// exactly-matched query (case-insensitive). First match wins.
	PinnedRules []PinnedRule `json:"pinned_rules"`

	// ---- security (搜索设置 · 安全) ----
	// ApplyMaskingToResults runs the Data Security masking rules over the
	// returned search hits (title/summary/content). Default off: masking
	// was designed for model prompts, search results keep the historic
	// unmasked behavior.
	ApplyMaskingToResults bool `json:"apply_masking_to_results"`
	// BlockedQueryPatterns are RE2 regexes; a query matching any of them
	// short-circuits to an empty result without touching the engine.
	BlockedQueryPatterns []string `json:"blocked_query_patterns"`
}

// PinnedRule is one 结果干预 · 置顶 rule: an exact (case-insensitive) query
// and the documents to place on top, in the given order.
type PinnedRule struct {
	Query       string   `json:"query"`
	DocumentIDs []string `json:"document_ids"`
}

// SearchRouteWeights mirrors the five studio-facing RRF routes (text /
// semantic / wiki / graph / rewrite). The entity and chunk legs keep their
// built-in neutral weight — the chunk leg is gray-switched by chunk_route.
type SearchRouteWeights struct {
	Text     float64 `json:"text"`
	Semantic float64 `json:"semantic"`
	Wiki     float64 `json:"wiki"`
	Graph    float64 `json:"graph"`
	Rewrite  float64 `json:"rewrite"`
}

// Historic defaults + hard ceilings for the pagination/facet knobs. The
// hardMaxPageSize ceiling keeps even a misconfigured max_page_size far below
// anything that could drag unbounded rows out of the engine.
const (
	defaultSearchPageSize       = 10
	defaultSearchMaxPage        = 100
	hardMaxSearchPageSize       = 1000
	defaultResultWindow         = 10000
	maxResultWindowCeiling      = 100000
	defaultSuggestSize          = 10
	defaultAggregationBucketCap = 15
	defaultSearchFuzziness      = 3
	defaultHotSearchesSize      = 10
)

// EffectivePageSize is the nil-safe default results-per-page: unset or out
// of range reads as the historic 10.
func (s *SearchSettings) EffectivePageSize() int {
	if s != nil && s.PageSize >= 1 && s.PageSize <= 100 {
		return s.PageSize
	}
	return defaultSearchPageSize
}

// EffectiveMaxPageSize is the nil-safe per-request ?size= clamp: unset or
// out of range reads as the historic 100, never above the hard ceiling.
func (s *SearchSettings) EffectiveMaxPageSize() int {
	if s != nil && s.MaxPageSize >= 10 && s.MaxPageSize <= hardMaxSearchPageSize {
		return s.MaxPageSize
	}
	return defaultSearchMaxPage
}

// EffectiveMaxResultWindow is the nil-safe from+size cap: unset or out of
// range reads as the ES default 10000.
func (s *SearchSettings) EffectiveMaxResultWindow() int {
	if s != nil && s.MaxResultWindow >= 100 && s.MaxResultWindow <= maxResultWindowCeiling {
		return s.MaxResultWindow
	}
	return defaultResultWindow
}

// SuggestEnabledVal is the nil-safe suggestion gate: unset reads as enabled.
func (s *SearchSettings) SuggestEnabledVal() bool {
	return s == nil || !s.SuggestDisabled
}

// EffectiveSuggestSize is the nil-safe default suggestion count: unset or
// out of range reads as the historic 10.
func (s *SearchSettings) EffectiveSuggestSize() int {
	if s != nil && s.SuggestSize >= 1 && s.SuggestSize <= 50 {
		return s.SuggestSize
	}
	return defaultSuggestSize
}

// EffectiveAggregationBucketCap is the nil-safe per-aggregation bucket cap: unset or out of
// range reads as the historic 15.
func (s *SearchSettings) EffectiveAggregationBucketCap() int {
	if s != nil && s.AggregationBucketCap >= 1 && s.AggregationBucketCap <= 50 {
		return s.AggregationBucketCap
	}
	return defaultAggregationBucketCap
}

// EffectiveRRFRankConstant is the nil-safe fusion-K override: unset or out
// of range reads as 0, meaning "keep the built-in defaultRRFK".
func (s *SearchSettings) EffectiveRRFRankConstant() int {
	if s != nil && s.RRFRankConstant >= 1 && s.RRFRankConstant <= 1000 {
		return s.RRFRankConstant
	}
	return 0
}

// EffectiveFuzziness is the nil-safe default typo-tolerance distance: unset
// or out of range reads as the historic 3.
func (s *SearchSettings) EffectiveFuzziness() int {
	if s != nil && s.Fuzziness >= 1 && s.Fuzziness <= 5 {
		return s.Fuzziness
	}
	return defaultSearchFuzziness
}

// AggregationsEnabledVal is the nil-safe aggregations gate: unset reads as enabled.
func (s *SearchSettings) AggregationsEnabledVal() bool {
	return s == nil || !s.AggregationsDisabled
}

// EffectiveAggregationBucketSort is the nil-safe bucket ordering: anything but "value"
// reads as the historic count-desc.
func (s *SearchSettings) EffectiveAggregationBucketSort() string {
	if s != nil && s.AggregationBucketSort == "value" {
		return "value"
	}
	return "count"
}

// AggregationFieldDisabled is the nil-safe per-dimension gate: unset (or an
// unknown key) reads as enabled, the historic behavior.
func (s *SearchSettings) AggregationFieldDisabled(key string) bool {
	if s == nil {
		return false
	}
	for _, k := range s.AggregationFieldsDisabled {
		if k == key {
			return true
		}
	}
	return false
}

// HotSearchesEnabledVal is the nil-safe hot-queries gate: unset reads as
// enabled (the endpoint is new; existing deployments keep working).
func (s *SearchSettings) HotSearchesEnabledVal() bool {
	return s == nil || !s.HotSearchesDisabled
}

// RelatedSearchesEnabledVal is the nil-safe related-searches gate: unset
// reads as enabled (same convention as HotSearchesEnabledVal).
func (s *SearchSettings) RelatedSearchesEnabledVal() bool {
	return s == nil || !s.RelatedSearchesDisabled
}

// EffectiveHotSearchesSize is the nil-safe default hot-queries count: unset
// or out of range reads as 10.
func (s *SearchSettings) EffectiveHotSearchesSize() int {
	if s != nil && s.HotSearchesSize >= 1 && s.HotSearchesSize <= 20 {
		return s.HotSearchesSize
	}
	return defaultHotSearchesSize
}

// Built-in keyword-leg fields — keep in sync with the historic defaults in
// QueryDocuments (exposed so the UI can pre-fill the weight editor).
var DefaultFieldWeights = []FieldWeight{
	{Field: "title.keyword", Weight: 100},
	{Field: "title", Weight: 10},
	{Field: "title.pinyin", Weight: 4},
	{Field: "combined_fulltext", Weight: 1},
}

// TextFieldWeights renders the configured field weights as the "field^boost"
// strings the query builder consumes. Nil when unconfigured (caller falls
// back to the built-in defaults).
func (s *SearchSettings) TextFieldWeights() []string {
	if s == nil || len(s.FieldWeights) == 0 {
		return nil
	}
	fields := make([]string, 0, len(s.FieldWeights))
	for _, fw := range s.FieldWeights {
		if fw.Field == "" {
			continue
		}
		if fw.Weight > 0 && fw.Weight != 1 {
			fields = append(fields, fw.Field+"^"+strconv.FormatFloat(fw.Weight, 'f', -1, 64))
		} else {
			fields = append(fields, fw.Field)
		}
	}
	return fields
}

// MatchOperatorVal is the nil-safe match operator: empty reads as OR.
func (s *SearchSettings) MatchOperatorVal() string {
	if s == nil || (s.MatchOperator != "AND" && s.MatchOperator != "OR") {
		return "OR"
	}
	return s.MatchOperator
}

// sortTypes are the values search_settings.default_sort accepts.
var sortTypes = map[string]bool{
	"":             true,
	"relevance":    true,
	"created_desc": true,
	"created_asc":  true,
}

// IsValidSortType reports whether the value is accepted for default_sort.
func IsValidSortType(t string) bool {
	return sortTypes[t]
}

// DefaultSortSpec resolves the configured default sort into an ES sort
// clause (field + "asc"/"desc"). ok=false means relevance (no explicit
// sort) — the historic behavior.
func (s *SearchSettings) DefaultSortSpec() (field string, order string, ok bool) {
	if s == nil {
		return "", "", false
	}
	switch s.DefaultSort {
	case "created_desc":
		return "created", "desc", true
	case "created_asc":
		return "created", "asc", true
	}
	return "", "", false
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
