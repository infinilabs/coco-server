/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"fmt"
	"net/http"
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
	h.WriteAckOKJSON(w)
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
