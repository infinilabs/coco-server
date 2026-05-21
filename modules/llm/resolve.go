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
