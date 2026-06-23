package deep_research

import (
	"fmt"

	"infini.sh/coco/core"
	llmmodule "infini.sh/coco/modules/llm"
)

// resolveStageModel resolves a deep research stage model config with fallback
// to the server-configured default language model when the stage model is not
// explicitly configured.
//
// Fallback chain:
//  1. The ModelConfig already has both ProviderID and Name set → use as-is.
//  2. The server Settings.DefaultModel.LanguageModel is fully populated → fill
//     ProviderID and Name from there.
//  3. Neither is available → return an error identifying the failing stage.
//
// The returned ModelConfig is a copy of `in` with only ProviderID and Name
// overwritten by the resolved identity; all other fields (settings, keepalive,
// prompt config) are preserved.
func resolveStageModel(in core.ModelConfig, stage string) (core.ModelConfig, error) {
	resolved := llmmodule.ResolveModel(core.LLMTypeLanguage, &core.ModelId{
		ProviderID: in.ProviderID,
		ID:         in.Name,
	})
	if resolved == nil {
		return core.ModelConfig{}, fmt.Errorf(
			"deep research %s stage: no model configured and no default language model in settings",
			stage,
		)
	}
	out := in
	out.ProviderID = resolved.ProviderID
	out.Name = resolved.ID
	return out, nil
}
