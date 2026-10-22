/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	log "github.com/cihub/seelog"
	"golang.org/x/text/language"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/util"
)

type ServerSettings struct {
}

func (h *APIHandler) getServerSettings(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	appConfig := common.AppConfig()
	h.WriteJSON(w, appConfig, http.StatusOK)
}

// mergeSection merges one settings section: incoming over old, writing the
// result back via writeBack. A nil old section is seeded with an empty value —
// util.MergeFields is a no-op against a nil destination map, which would
// otherwise silently drop the first save of a section.
func mergeSection[T any](old, incoming *T, writeBack func(*T)) error {
	if incoming == nil {
		return nil
	}
	base := old
	if base == nil {
		base = new(T)
	}
	merged := new(T)
	if err := mergeSettings(base, incoming, merged); err != nil {
		return err
	}
	writeBack(merged)
	return nil
}

// settingsSections lists every mergeable section of the server settings.
// Adding a section is one entry here: optional validation, then the merge.
var settingsSections = []struct {
	name  string
	apply func(incoming, old *core.Config) error
}{
	{
		name: "server",
		apply: func(incoming, old *core.Config) error {
			return mergeSection(old.ServerInfo, incoming.ServerInfo, func(v *core.ServerInfo) { old.ServerInfo = v })
		},
	},
	{
		name: "app_settings",
		apply: func(incoming, old *core.Config) error {
			return mergeSection(old.AppSettings, incoming.AppSettings, func(v *core.AppSettings) { old.AppSettings = v })
		},
	},
	{
		name: "search_settings",
		apply: func(incoming, old *core.Config) error {
			if t := incoming.SearchSettings; t != nil {
				if t.SearchType != "" && !core.IsValidSearchType(t.SearchType) {
					return fmt.Errorf("invalid search_type %q: must be one of keyword, semantic, hybrid, hybrid_rrf", t.SearchType)
				}
				if err := validateSearchStrategy(t); err != nil {
					return err
				}
			}
			return mergeSection(old.SearchSettings, incoming.SearchSettings, func(v *core.SearchSettings) { old.SearchSettings = v })
		},
	},
	{
		name: "default_model",
		apply: func(incoming, old *core.Config) error {
			if incoming.DefaultModel != nil {
				if err := validateDefaultModel(incoming.DefaultModel); err != nil {
					return err
				}
			}
			return mergeSection(old.DefaultModel, incoming.DefaultModel, func(v *core.DefaultModel) { old.DefaultModel = v })
		},
	},
	{
		name: "document_processing",
		apply: func(incoming, old *core.Config) error {
			if incoming.DocumentProcessing != nil {
				// Validate language settings.
				if lang := incoming.DocumentProcessing.LLMGenerationLanguage; lang != "" {
					if _, err := language.Parse(lang); err != nil {
						return fmt.Errorf("invalid llm_generation_language %q: %v", lang, err)
					}
				}
				if err := validateDocumentProcessingTika(incoming.DocumentProcessing); err != nil {
					return err
				}
			}
			return mergeSection(old.DocumentProcessing, incoming.DocumentProcessing, func(v *core.DocumentProcessing) { old.DocumentProcessing = v })
		},
	},
	{
		name: "data_security",
		apply: func(incoming, old *core.Config) error {
			if incoming.DataSecurity != nil {
				if err := validateDataSecurity(incoming.DataSecurity); err != nil {
					return err
				}
			}
			return mergeSection(old.DataSecurity, incoming.DataSecurity, func(v *core.DataSecurity) { old.DataSecurity = v })
		},
	},
	{
		name: "engine_ai",
		apply: func(incoming, old *core.Config) error {
			if incoming.EngineAI != nil {
				if err := validateEngineAI(incoming.EngineAI); err != nil {
					return err
				}
			}
			return mergeSection(old.EngineAI, incoming.EngineAI, func(v *core.EngineAI) { old.EngineAI = v })
		},
	},
	{
		name: "appearance",
		apply: func(incoming, old *core.Config) error {
			if incoming.Appearance != nil {
				if err := validateAppearance(incoming.Appearance); err != nil {
					return err
				}
			}
			return mergeSection(old.Appearance, incoming.Appearance, func(v *core.AppearanceSettings) { old.Appearance = v })
		},
	},
}

// maxSearchFieldWeights caps the operator-configured field weight list — the
// text leg fires one match clause per field, a runaway list multiplies query cost.
const maxSearchFieldWeights = 20

// minSearchWeight / maxSearchWeight bound one field's boost. Extremely large
// boosts drown out every other relevance signal; zero/negative is invalid.
const (
	minSearchWeight = 0.0001
	maxSearchWeight = 10000
)

// searchMSMPattern loosely validates an ES minimum_should_match spec: a
// percentage, an integer, or the "N<pct|n>" combination form.
var searchMSMPattern = regexp.MustCompile(`^-?\d{1,3}%?$|^-?\d{1,3}<(-?\d{1,3}%?)$`)

// validateSearchStrategy rejects retrieval-strategy values that would corrupt
// the query DSL at recall time — better to fail the save than to 400 every search.
func validateSearchStrategy(cfg *core.SearchSettings) error {
	if cfg.DefaultSort != "" && !core.IsValidSortType(cfg.DefaultSort) {
		return fmt.Errorf("invalid default_sort %q: must be one of relevance, created_desc, created_asc", cfg.DefaultSort)
	}
	if cfg.MatchOperator != "" && cfg.MatchOperator != "OR" && cfg.MatchOperator != "AND" {
		return fmt.Errorf("invalid match_operator %q: must be OR or AND", cfg.MatchOperator)
	}
	if cfg.MinScore < 0 || cfg.MinScore > 10000 {
		return fmt.Errorf("min_score must be between 0 and 10000")
	}
	if msm := cfg.MinimumShouldMatch; msm != "" && !searchMSMPattern.MatchString(strings.TrimSpace(msm)) {
		return fmt.Errorf("invalid minimum_should_match %q: expected an ES spec like 75%%, 2 or 2<75%%", msm)
	}
	seen := map[string]bool{}
	for i, fw := range cfg.FieldWeights {
		if fw.Field == "" {
			return fmt.Errorf("field_weights[%d]: field is required", i)
		}
		if fw.Weight < minSearchWeight || fw.Weight > maxSearchWeight {
			return fmt.Errorf("field_weights[%d] (%s): weight must be between %v and %v", i, fw.Field, minSearchWeight, maxSearchWeight)
		}
		if seen[fw.Field] {
			return fmt.Errorf("field_weights[%d]: duplicate field %q", i, fw.Field)
		}
		seen[fw.Field] = true
	}
	if len(cfg.FieldWeights) > maxSearchFieldWeights {
		return fmt.Errorf("field_weights: at most %d fields", maxSearchFieldWeights)
	}
	// pagination / suggestion / facet knobs: 0 means "keep the built-in
	// default", anything set must sit in the same range the UI enforces
	if cfg.PageSize != 0 && (cfg.PageSize < 1 || cfg.PageSize > 100) {
		return fmt.Errorf("page_size must be between 1 and 100 (0 = default)")
	}
	if cfg.MaxPageSize != 0 && (cfg.MaxPageSize < 10 || cfg.MaxPageSize > 1000) {
		return fmt.Errorf("max_page_size must be between 10 and 1000 (0 = default)")
	}
	if cfg.MaxResultWindow != 0 && (cfg.MaxResultWindow < 100 || cfg.MaxResultWindow > 100000) {
		return fmt.Errorf("max_result_window must be between 100 and 100000 (0 = default)")
	}
	if cfg.SuggestSize != 0 && (cfg.SuggestSize < 1 || cfg.SuggestSize > 50) {
		return fmt.Errorf("suggest_size must be between 1 and 50 (0 = default)")
	}
	if cfg.AggregationBucketCap != 0 && (cfg.AggregationBucketCap < 1 || cfg.AggregationBucketCap > 50) {
		return fmt.Errorf("aggregation_bucket_cap must be between 1 and 50 (0 = default)")
	}
	// low-difficulty sweep group
	if cfg.Fuzziness != 0 && (cfg.Fuzziness < 1 || cfg.Fuzziness > 5) {
		return fmt.Errorf("fuzziness must be between 1 and 5 (0 = default 3; use the request param for exact matching)")
	}
	if cfg.AggregationBucketSort != "" && cfg.AggregationBucketSort != "count" && cfg.AggregationBucketSort != "value" {
		return fmt.Errorf("aggregation_bucket_sort must be \"count\" or \"value\"")
	}
	for _, dim := range cfg.AggregationFieldsDisabled {
		switch dim {
		case "category", "source", "type", "tags", "owner", "time", "color":
		default:
			return fmt.Errorf("aggregation_fields_disabled contains unknown dimension %q (allowed: category/source/type/tags/owner/time/color)", dim)
		}
	}
	if cfg.QueryTimeoutMS != 0 && (cfg.QueryTimeoutMS < 100 || cfg.QueryTimeoutMS > 60000) {
		return fmt.Errorf("query_timeout_ms must be between 100 and 60000 (0 = off)")
	}
	if cfg.HotSearchesSize != 0 && (cfg.HotSearchesSize < 1 || cfg.HotSearchesSize > 20) {
		return fmt.Errorf("hot_searches_size must be between 1 and 20 (0 = default)")
	}
	if len(cfg.RecommendTopics) > 20 {
		return fmt.Errorf("recommend_topics: at most 20 topics")
	}
	for _, topic := range cfg.RecommendTopics {
		if strings.TrimSpace(topic) == "" {
			return fmt.Errorf("recommend_topics: blank topics are not allowed")
		}
	}
	if len(cfg.ResultSourceIncludes) > 30 {
		return fmt.Errorf("result_source_includes: at most 30 fields")
	}
	// semantic / AI group
	if cfg.SemanticMinScore < 0 || cfg.SemanticMinScore > 10000 {
		return fmt.Errorf("semantic_min_score must be between 0 and 10000")
	}
	if cfg.RRFRankConstant != 0 && (cfg.RRFRankConstant < 1 || cfg.RRFRankConstant > 1000) {
		return fmt.Errorf("rrf_rank_constant must be between 1 and 1000 (0 = default)")
	}
	// result intervention group
	if len(cfg.BlockedDocIDs) > 100 {
		return fmt.Errorf("blocked_doc_ids: at most 100 documents")
	}
	for i, id := range cfg.BlockedDocIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("blocked_doc_ids[%d]: empty id", i)
		}
	}
	if len(cfg.BlockedDatasourceIDs) > 50 {
		return fmt.Errorf("blocked_datasource_ids: at most 50 datasources")
	}
	for i, id := range cfg.BlockedDatasourceIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("blocked_datasource_ids[%d]: empty id", i)
		}
	}
	if len(cfg.PinnedRules) > 20 {
		return fmt.Errorf("pinned_rules: at most 20 rules")
	}
	for i, rule := range cfg.PinnedRules {
		if strings.TrimSpace(rule.Query) == "" {
			return fmt.Errorf("pinned_rules[%d]: query is required", i)
		}
		if len(rule.DocumentIDs) == 0 || len(rule.DocumentIDs) > 100 {
			return fmt.Errorf("pinned_rules[%d]: needs 1..100 document ids", i)
		}
	}
	// security group
	if len(cfg.BlockedQueryPatterns) > 50 {
		return fmt.Errorf("blocked_query_patterns: at most 50 patterns")
	}
	for i, p := range cfg.BlockedQueryPatterns {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("blocked_query_patterns[%d]: empty pattern", i)
		}
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("blocked_query_patterns[%d]: invalid regex %q: %v", i, p, err)
		}
	}
	return nil
}

// maxAppearanceImageLength caps one branding image field (a data URL or a
// remote URL) at roughly 2MB of text, so a careless upload can't blow up the
// settings document that every client fetches.
const maxAppearanceImageLength = 2 * 1024 * 1024

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

var appearanceImagePattern = regexp.MustCompile(`^(data:image/(png|jpe?g|gif|webp|svg\+xml);base64,[A-Za-z0-9+/=\s]+|https?://\S+)$`)

// validateAppearance rejects malformed branding values before they are
// persisted: colors must be hex, images must be data URLs or http(s) URLs.
func validateAppearance(cfg *core.AppearanceSettings) error {
	checkColors := func(prefix string, colors ...string) error {
		for _, c := range colors {
			if c != "" && !hexColorPattern.MatchString(c) {
				return fmt.Errorf("%s: invalid color %q (expected hex, e.g. #0087FF)", prefix, c)
			}
		}
		return nil
	}
	checkImage := func(field, v string) error {
		if v == "" {
			return nil
		}
		if len(v) > maxAppearanceImageLength {
			return fmt.Errorf("%s: image too large (max %d bytes)", field, maxAppearanceImageLength)
		}
		if !appearanceImagePattern.MatchString(v) {
			return fmt.Errorf("%s: must be a data:image URL or an http(s) URL", field)
		}
		return nil
	}

	if c := cfg.ThemeColors; c != nil {
		if err := checkColors("appearance.theme_colors", c.Primary, c.PrimaryDark, c.Success, c.Warning, c.Error); err != nil {
			return err
		}
		for _, n := range []struct {
			name  string
			value *core.NeutralColors
		}{
			{"light", c.Light},
			{"dark", c.Dark},
		} {
			if n.value == nil {
				continue
			}
			if err := checkColors("appearance.theme_colors."+n.name, n.value.Layout, n.value.Container, n.value.BaseText); err != nil {
				return err
			}
		}
	}
	if l := cfg.Logo; l != nil {
		for _, img := range []struct{ name, value string }{
			{"logo.light", l.Light}, {"logo.dark", l.Dark}, {"logo.icon", l.Icon},
		} {
			if err := checkImage("appearance."+img.name, img.value); err != nil {
				return err
			}
		}
	}
	if s := cfg.Search; s != nil {
		for _, pair := range []struct {
			name  string
			value *core.AppearanceImagePair
		}{
			{"search.logo", s.Logo},
			{"search.background", s.Background},
		} {
			if pair.value == nil {
				continue
			}
			if err := checkImage("appearance."+pair.name+".light", pair.value.Light); err != nil {
				return err
			}
			if err := checkImage("appearance."+pair.name+".dark", pair.value.Dark); err != nil {
				return err
			}
		}
	}
	if l := cfg.Login; l != nil {
		if err := checkColors("appearance.login.background_color", l.BackgroundColor); err != nil {
			return err
		}
		if err := checkImage("appearance.login.background_image", l.BackgroundImage); err != nil {
			return err
		}
	}
	return nil
}

func (h *APIHandler) updateServerSettings(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	appConfig := core.Config{}
	if err := h.DecodeJSON(req, &appConfig); err != nil {
		_ = log.Error(err)
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	oldAppConfig := common.AppConfig()
	for _, section := range settingsSections {
		if err := section.apply(&appConfig, &oldAppConfig); err != nil {
			_ = log.Error(err)
			h.WriteError(w, fmt.Sprintf("%s: %v", section.name, err), http.StatusBadRequest)
			return
		}
	}
	common.SetAppConfig(&oldAppConfig)

	// Engine AI is save-then-sync: the settings persist regardless of engine
	// state, and the pipelines are pushed best-effort in the background —
	// /search/engine-ai reports drift if this push does not land.
	if cfg := oldAppConfig.EngineAI; cfg != nil && cfg.Enabled {
		go func(cfg *core.EngineAI) {
			plan, err := common.ResolveEngineAIPlan(cfg)
			if err != nil {
				log.Warnf("engine_ai: resolve failed: %v", err)
				return
			}
			if err := common.ApplyEnginePipelines(context.Background(), plan); err != nil {
				log.Warnf("engine_ai: pipeline sync failed: %v", err)
			} else {
				log.Info("engine_ai: engine pipelines synced")
			}
		}(cfg)
	}

	h.WriteAckOKJSON(w)
}

// validateEngineAI checks that an enabled EngineAI section can actually be
// turned into engine pipelines: the model must resolve to an existing
// provider, and the knobs must stay in sane ranges.
func validateEngineAI(cfg *core.EngineAI) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.BatchSize < 0 || cfg.BatchSize > 100 {
		return fmt.Errorf("batch_size must be between 0 and 100")
	}
	if cfg.RankConstant < 0 || cfg.RankConstant > 1000 {
		return fmt.Errorf("rank_constant must be between 0 and 1000")
	}
	plan, err := common.ResolveEngineAIPlan(cfg)
	if err != nil {
		return err
	}
	if plan.Model == nil {
		return fmt.Errorf("no embedding model: set engine_ai.embedding_model or default_model.embedding_model")
	}
	return nil
}

// validateDefaultModel checks that role-specific models resolve to language
// models. Vision and embedding models are rejected.
func validateDefaultModel(cfg *core.DefaultModel) error {
	for _, check := range []struct {
		model *core.ModelId
		name  string
	}{
		{cfg.AnsweringModel, "answering_model"},
		{cfg.PickingToolModel, "picking_tool_model"},
		{cfg.PickingDocModel, "picking_doc_model"},
		{cfg.IntentAnalysisModel, "intent_analysis_model"},
	} {
		if err := validateLanguageModelType(check.model, check.name); err != nil {
			return err
		}
	}
	return nil
}

// validateDocumentProcessingTika rejects a Tika endpoint that isn't a valid
// http(s) URL, and timeout values outside the sane range, before they are
// persisted — the processors silently fall back to defaults otherwise, and a
// typo in the address would surface only as mysteriously failed extraction.
func validateDocumentProcessingTika(cfg *core.DocumentProcessing) error {
	if endpoint := strings.TrimSpace(cfg.TikaEndpoint); endpoint != "" {
		u, err := url.Parse(endpoint)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("invalid tika_endpoint %q: must be an http(s) URL", endpoint)
		}
	}
	if t := cfg.TikaTimeoutInSeconds; t < 0 || t > 3600 {
		return fmt.Errorf("tika_timeout_in_seconds must be between 1 and 3600")
	}
	return nil
}

// validateDataSecurity rejects masking rules whose pattern cannot compile
// before they are persisted — a broken regex must never take the masking
// pipeline down at recall time.
func validateDataSecurity(cfg *core.DataSecurity) error {
	if cfg.Masking != nil {
		for i, rule := range cfg.Masking.Rules {
			if rule.Pattern == "" {
				continue
			}
			if _, err := regexp.Compile(rule.Pattern); err != nil {
				return fmt.Errorf("masking rule #%d (%s): invalid pattern %q: %v", i+1, rule.Name, rule.Pattern, err)
			}
		}
	}
	if cfg.FieldAccess != nil {
		for i, restriction := range cfg.FieldAccess.Restrictions {
			if strings.TrimSpace(restriction.Role) == "" {
				return fmt.Errorf("field restriction #%d: role is required", i+1)
			}
		}
	}
	return nil
}

// validateLanguageModelType checks that the given model (if specified) resolves
// to a language model. Vision and embedding models are rejected.
func validateLanguageModelType(modelId *core.ModelId, fieldName string) error {
	if modelId == nil || modelId.ProviderID == "" || modelId.ID == "" {
		return nil
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		return fmt.Errorf("%s: provider %q not found", fieldName, modelId.ProviderID)
	}
	m := provider.GetModel(modelId.ID)
	if m == nil {
		// model not in provider's builtin list; skip type check
		return nil
	}
	if m.Type != "" && m.Type != core.LLMTypeLanguage {
		return fmt.Errorf("%s: model %q must be a language model, got %q", fieldName, modelId.ID, m.Type)
	}
	return nil
}

func mergeSettings(old, new, merged interface{}) error {
	newSettings := util.MapStr{}
	buf := util.MustToJSONBytes(new)
	util.MustFromJSONBytes(buf, &newSettings)
	buf = util.MustToJSONBytes(old)
	oldSettings := util.MapStr{}
	util.MustFromJSONBytes(buf, &oldSettings)
	err := util.MergeFields(oldSettings, newSettings, true)
	if err != nil {
		return err
	}
	buf = util.MustToJSONBytes(oldSettings)
	util.MustFromJSONBytes(buf, merged)
	return nil
}
