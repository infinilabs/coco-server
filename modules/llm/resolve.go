/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package llm

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
)

// ResolveModel picks the ModelId to use for the given LLM type:
//  1. `override` if it's fully populated (both ProviderID and ID set)
//  2. The matching default model from the current AppConfig's DefaultModel
//
// Returns nil if neither source yields a complete ModelId.
func ResolveModel(t core.LLMType, override *core.ModelId) *core.ModelId {
	if override != nil && override.ProviderID != "" && override.ID != "" {
		return override
	}
	cfg := common.AppConfig()
	m := defaultModelFor(cfg.DefaultModel, t)
	if m != nil && m.ProviderID != "" && m.ID != "" {
		return m
	}
	return nil
}

// defaultModelFor returns d's configured default ModelId for the given LLM
// type, or nil when nothing is configured.
func defaultModelFor(d *core.DefaultModel, t core.LLMType) *core.ModelId {
	if d == nil {
		return nil
	}
	switch t {
	case core.LLMTypeLanguage:
		return d.LanguageModel
	case core.LLMTypeVision:
		return d.VisionModel
	case core.LLMTypeEmbedding:
		return d.EmbeddingModel
	}
	return nil
}

// ResolveAssistantModel picks the ModelId to use for the given assistant model
// use case, applying the fallback chain:
//  1. `override` if it's fully populated (both ProviderID and ID set) — this is
//     the model configured at the assistant level
//  2. The matching default model for the use case from
//     Settings.DefaultModel (e.g. AnsweringModel, IntentAnalysisModel, ...)
//  3. Settings.DefaultModel.LanguageModel as a last resort, since these are all
//     language-model use cases
//
// Returns nil if none of the above yields a complete ModelId.
func ResolveAssistantModel(use core.AssistantModelUse, override *core.ModelId) *core.ModelId {
	if override != nil && override.ProviderID != "" && override.ID != "" {
		return override
	}
	cfg := common.AppConfig()
	if m := defaultModelForUse(cfg.DefaultModel, use); m != nil && m.ProviderID != "" && m.ID != "" {
		return m
	}
	if cfg.DefaultModel != nil {
		if m := cfg.DefaultModel.LanguageModel; m != nil && m.ProviderID != "" && m.ID != "" {
			return m
		}
	}
	return nil
}

// defaultModelForUse returns d's configured default ModelId for the given
// assistant model use, or nil when nothing is configured.
func defaultModelForUse(d *core.DefaultModel, use core.AssistantModelUse) *core.ModelId {
	if d == nil {
		return nil
	}
	switch use {
	case core.AssistantModelUseAnswering:
		return d.AnsweringModel
	case core.AssistantModelUseIntentAnalysis:
		return d.IntentAnalysisModel
	case core.AssistantModelUsePickingDoc:
		return d.PickingDocModel
	case core.AssistantModelUsePickingTool:
		return d.PickingToolModel
	}
	return nil
}
