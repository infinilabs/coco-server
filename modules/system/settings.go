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
			if t := incoming.SearchSettings; t != nil && t.SearchType != "" && !core.IsValidSearchType(t.SearchType) {
				return fmt.Errorf("invalid search_type %q: must be one of keyword, semantic, hybrid, hybrid_rrf", t.SearchType)
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
