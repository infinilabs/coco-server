/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package system

import (
	"fmt"
	"net/http"

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

func (h *APIHandler) updateServerSettings(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	appConfig := core.Config{}
	if err := h.DecodeJSON(req, &appConfig); err != nil {
		_ = log.Error(err)
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	oldAppConfig := common.AppConfig()
	if appConfig.ServerInfo != nil {
		//merge settings
		serverCfg := core.ServerInfo{}
		err := mergeSettings(oldAppConfig.ServerInfo, appConfig.ServerInfo, &serverCfg)
		if err != nil {
			_ = log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.ServerInfo = &serverCfg
	}
	if appConfig.AppSettings != nil {
		//merge settings
		appSettings := core.AppSettings{}
		err := mergeSettings(oldAppConfig.AppSettings, appConfig.AppSettings, &appSettings)
		if err != nil {
			_ = log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.AppSettings = &appSettings
	}
	if appConfig.SearchSettings != nil {
		//merge settings
		searchSettings := core.SearchSettings{}
		err := mergeSettings(oldAppConfig.SearchSettings, appConfig.SearchSettings, &searchSettings)
		if err != nil {
			_ = log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.SearchSettings = &searchSettings
	}
	if appConfig.DefaultModel != nil {
		// validate that role-specific models are language models
		for _, check := range []struct {
			model *core.ModelId
			name  string
		}{
			{appConfig.DefaultModel.AnsweringModel, "answering_model"},
			{appConfig.DefaultModel.PickingToolModel, "picking_tool_model"},
			{appConfig.DefaultModel.PickingDocModel, "picking_doc_model"},
			{appConfig.DefaultModel.IntentAnalysisModel, "intent_analysis_model"},
		} {
			if err := validateLanguageModelType(check.model, check.name); err != nil {
				h.WriteError(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		//merge settings
		defaultModel := core.DefaultModel{}
		err := mergeSettings(oldAppConfig.DefaultModel, appConfig.DefaultModel, &defaultModel)
		if err != nil {
			_ = log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.DefaultModel = &defaultModel
	}
	if appConfig.DocumentProcessing != nil {
		// Validate language settings.
		if lang := appConfig.DocumentProcessing.LLMGenerationLanguage; lang != "" {
			if _, err := language.Parse(lang); err != nil {
				h.WriteError(w, fmt.Sprintf("invalid llm_generation_language %q: %v", lang, err), http.StatusBadRequest)
				return
			}
		}
		//merge settings
		docProcessing := core.DocumentProcessing{}
		err := mergeSettings(oldAppConfig.DocumentProcessing, appConfig.DocumentProcessing, &docProcessing)
		if err != nil {
			_ = log.Error(err)
			h.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		oldAppConfig.DocumentProcessing = &docProcessing
	}
	common.SetAppConfig(&oldAppConfig)
	h.WriteAckOKJSON(w)
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
