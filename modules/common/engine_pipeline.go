/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

var (
	errEngineAIProviderNotFound = errors.New("embedding model provider not found")
	errEngineAIUnavailable      = errors.New("engine client is not available for pipeline operations")
)

// engineRawClient is the subset of the concrete ES client the pipeline
// provisioning needs; asserted off the API interface because the pipeline
// REST endpoints have no dedicated interface method.
type engineRawClient interface {
	Request(ctx context.Context, method, url string, body []byte) (*util.Result, error)
	GetEndpoint() string
}

// EngineAIPlan is the resolved intent for the engine's AI pipelines: which
// model provider backs them, the generated pipeline bodies, and any
// non-fatal notes about how the config was interpreted.
type EngineAIPlan struct {
	Enabled  bool
	Config   *core.EngineAI
	Model    *core.ModelId
	Provider *core.ModelProvider
	// EmbeddingURL is the OpenAI-compatible endpoint the engine will call.
	EmbeddingURL string
	// Vendor is the engine-side processor vendor for the provider's api type.
	Vendor         string
	IngestPipeline util.MapStr
	SearchPipeline util.MapStr
	DocumentIndex  string
	TextField      string
	VectorField    string
	Warnings       []string
}

// engineAIVectorFieldDefault derives the document vector field from the
// dimension the semantic search wiring expects.
func engineAIVectorFieldDefault() string {
	return "ai_insights.embedding.embedding" + strconv.Itoa(core.RequiredEmbeddingDimension)
}

// ResolveEngineAIPlan turns the EngineAI settings into everything the engine
// needs: the model (falling back to the default embedding model), the
// provider connection details, and the two generated pipeline bodies. It
// never touches the engine — ApplyEnginePipelines does.
func ResolveEngineAIPlan(cfg *core.EngineAI) (*EngineAIPlan, error) {
	plan := &EngineAIPlan{Config: cfg}
	if cfg == nil {
		return plan, nil
	}
	plan.Enabled = cfg.Enabled

	model := cfg.EmbeddingModel
	if model == nil || model.ProviderID == "" || model.ID == "" {
		if fallback := AppConfig().DefaultModel; fallback != nil && fallback.EmbeddingModel != nil &&
			fallback.EmbeddingModel.ProviderID != "" && fallback.EmbeddingModel.ID != "" {
			model = fallback.EmbeddingModel
			plan.Warnings = append(plan.Warnings, "embedding model not set, using the default embedding model")
		}
	}
	if model == nil || model.ProviderID == "" || model.ID == "" {
		return plan, nil
	}
	plan.Model = model

	provider, err := GetModelProvider(model.ProviderID)
	if err != nil || provider == nil {
		if err == nil {
			err = errEngineAIProviderNotFound
		}
		return plan, err
	}
	plan.Provider = provider

	// The engine's embedding callout speaks the OpenAI protocol; vendor is
	// fixed to openai regardless of which OpenAI-compatible provider Coco uses.
	plan.Vendor = "openai"
	if !strings.EqualFold(provider.APIType, "openai") {
		plan.Warnings = append(plan.Warnings, "model provider api type "+provider.APIType+" is not openai-compatible, the engine callout may fail")
	}
	plan.EmbeddingURL = strings.TrimSuffix(provider.BaseURL, "/") + "/embeddings"

	plan.TextField = cfg.TextField
	if plan.TextField == "" {
		plan.TextField = core.EngineAITextFieldDefault
	}
	plan.VectorField = cfg.VectorField
	if plan.VectorField == "" {
		plan.VectorField = engineAIVectorFieldDefault()
	}
	plan.IngestPipeline, plan.SearchPipeline = buildEnginePipelines(cfg, model, provider)

	plan.DocumentIndex = orm.GetIndexName(&core.Document{})
	return plan, nil
}

// buildEnginePipelines is the pure generator: the two pipeline bodies the
// engine gets, derived only from the settings, the model and its provider.
func buildEnginePipelines(cfg *core.EngineAI, model *core.ModelId, provider *core.ModelProvider) (util.MapStr, util.MapStr) {
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	rankConstant := cfg.RankConstant
	if rankConstant <= 0 {
		rankConstant = 60
	}
	textField := cfg.TextField
	if textField == "" {
		textField = core.EngineAITextFieldDefault
	}
	vectorField := cfg.VectorField
	if vectorField == "" {
		vectorField = engineAIVectorFieldDefault()
	}
	embeddingURL := strings.TrimSuffix(provider.BaseURL, "/") + "/embeddings"
	ingest := util.MapStr{
		"description": "Managed by Coco AI - do not edit manually",
		"processors": []interface{}{
			util.MapStr{
				"text_embedding": util.MapStr{
					"text_field":     textField,
					"vector_field":   vectorField,
					"vendor":         "openai",
					"api_key":        provider.APIKey,
					"url":            embeddingURL,
					"model_id":       model.ID,
					"dims":           core.RequiredEmbeddingDimension,
					"batch_size":     batchSize,
					"ignore_missing": true,
					"ignore_failure": true,
				},
			},
		},
	}
	search := util.MapStr{
		"description": "Managed by Coco AI - do not edit manually",
		"rewrite_processors": []interface{}{
			util.MapStr{
				"semantic_query_enricher": util.MapStr{
					"default_model_id": model.ID,
					"vendor":           "openai",
					"api_key":          provider.APIKey,
					"url":              embeddingURL,
				},
			},
		},
		"rerank_processors": []interface{}{
			util.MapStr{
				"hybrid_ranker_processor": util.MapStr{
					"combination": util.MapStr{
						"technique":  "rrf",
						"parameters": util.MapStr{"rank_constant": rankConstant},
					},
				},
			},
		},
	}
	return ingest, search
}

// engineAIRawClient returns the engine client behind the ORM, the same
// cluster the documents live in.
func engineAIRawClient() (engineRawClient, error) {
	client := elastic.GetClientNoPanic(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	if client == nil {
		return nil, errEngineAIUnavailable
	}
	raw, ok := client.(engineRawClient)
	if !ok {
		return nil, errEngineAIUnavailable
	}
	return raw, nil
}

// ApplyEnginePipelines pushes the plan to the engine: both pipeline
// definitions, and the document index's default pipeline so every future
// write gets vectorized by the engine.
func ApplyEnginePipelines(ctx context.Context, plan *EngineAIPlan) error {
	if plan == nil || !plan.Enabled || plan.Model == nil {
		return nil
	}
	raw, err := engineAIRawClient()
	if err != nil {
		return err
	}
	endpoint := raw.GetEndpoint()
	if _, err := raw.Request(ctx, util.Verb_PUT, endpoint+"/_ingest/pipeline/"+core.EngineIngestPipelineName, util.MustToJSONBytes(plan.IngestPipeline)); err != nil {
		return err
	}
	if _, err := raw.Request(ctx, util.Verb_PUT, endpoint+"/_search/pipeline/"+core.EngineSearchPipelineName, util.MustToJSONBytes(plan.SearchPipeline)); err != nil {
		return err
	}
	client := elastic.GetClientNoPanic(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	if client == nil {
		return errEngineAIUnavailable
	}
	return client.UpdateIndexSettings(plan.DocumentIndex, map[string]interface{}{
		"index.default_pipeline": core.EngineIngestPipelineName,
	})
}

// EnginePipelineActual is what the engine currently has.
type EnginePipelineActual struct {
	IngestPipeline  util.MapStr
	SearchPipeline  util.MapStr
	DefaultPipeline string
	Errors          []string
}

// ReadEnginePipelines reads the current pipeline state back from the engine
// for display (回显) and drift comparison.
func ReadEnginePipelines(ctx context.Context, plan *EngineAIPlan) *EnginePipelineActual {
	actual := &EnginePipelineActual{}
	raw, err := engineAIRawClient()
	if err != nil {
		actual.Errors = append(actual.Errors, err.Error())
		return actual
	}
	endpoint := raw.GetEndpoint()
	read := func(path string) util.MapStr {
		resp, err := raw.Request(ctx, util.Verb_GET, endpoint+path, nil)
		if err != nil || resp == nil || resp.StatusCode != 200 || len(resp.Body) == 0 {
			return nil
		}
		out := util.MapStr{}
		if err := util.FromJSONBytes(resp.Body, &out); err != nil {
			return nil
		}
		return out
	}
	actual.IngestPipeline = unwrapPipeline(read("/_ingest/pipeline/"+core.EngineIngestPipelineName), core.EngineIngestPipelineName)
	actual.SearchPipeline = unwrapPipeline(read("/_search/pipeline/"+core.EngineSearchPipelineName), core.EngineSearchPipelineName)

	client := elastic.GetClientNoPanic(global.MustLookupString(elastic.GlobalSystemElasticsearchID))
	if client != nil && plan != nil && plan.DocumentIndex != "" {
		if settings, err := client.GetIndexSettings(plan.DocumentIndex); err == nil && settings != nil {
			actual.DefaultPipeline = extractDefaultPipeline(*settings, plan.DocumentIndex)
		}
	}
	return actual
}

// unwrapPipeline lifts the definition out of the {"<name>": {...}} wrapper
// the pipeline GET responses use.
func unwrapPipeline(wrapped util.MapStr, name string) util.MapStr {
	if wrapped == nil {
		return nil
	}
	switch def := wrapped[name].(type) {
	case map[string]interface{}:
		return def
	case util.MapStr:
		return def
	}
	return nil
}

// extractDefaultPipeline digs index.default_pipeline out of a settings map,
// which nests as {"<index>":{"settings":{"index":{...}}}}.
func extractDefaultPipeline(settings util.MapStr, indexName string) string {
	perIndex, _ := settings[indexName].(map[string]interface{})
	if perIndex == nil {
		return ""
	}
	inner, _ := perIndex["settings"].(map[string]interface{})
	if inner == nil {
		return ""
	}
	idx, _ := inner["index"].(map[string]interface{})
	if idx == nil {
		return ""
	}
	v, _ := idx["default_pipeline"].(string)
	return v
}

// MaskEngineAISecret replaces api_key values with a mask so the generated
// pipelines can be echoed to the UI without leaking the provider key.
func MaskEngineAISecret(pipeline util.MapStr) util.MapStr {
	if pipeline == nil {
		return nil
	}
	var maskNode func(node interface{}) interface{}
	maskNode = func(node interface{}) interface{} {
		switch m := node.(type) {
		case map[string]interface{}:
			out := util.MapStr{}
			for k, v := range m {
				if k == "api_key" {
					out[k] = "******"
					continue
				}
				out[k] = maskNode(v)
			}
			return out
		case util.MapStr:
			return maskNode(map[string]interface{}(m))
		case []interface{}:
			out := make([]interface{}, len(m))
			for i, v := range m {
				out[i] = maskNode(v)
			}
			return out
		default:
			return node
		}
	}
	return maskNode(pipeline).(util.MapStr)
}

// EnginePipelineDrift summarizes whether the engine matches the plan.
type EnginePipelineDrift struct {
	IngestInSync  bool `json:"ingest_in_sync"`
	SearchInSync  bool `json:"search_in_sync"`
	DefaultInSync bool `json:"default_pipeline_in_sync"`
	Applied       bool `json:"applied"`
}

// CompareEnginePipelines reports drift between plan and actual with the
// api_key ignored on both sides (the engine may store it encrypted or
// masked, so it is not comparable).
func CompareEnginePipelines(plan *EngineAIPlan, actual *EnginePipelineActual) EnginePipelineDrift {
	drift := EnginePipelineDrift{}
	if plan == nil || !plan.Enabled || plan.Model == nil {
		return drift
	}
	if actual == nil {
		return drift
	}
	drift.IngestInSync = pipelineEqual(MaskEngineAISecret(plan.IngestPipeline), maskAny(actual.IngestPipeline))
	drift.SearchInSync = pipelineEqual(MaskEngineAISecret(plan.SearchPipeline), maskAny(actual.SearchPipeline))
	drift.DefaultInSync = actual.DefaultPipeline == core.EngineIngestPipelineName
	drift.Applied = drift.IngestInSync && drift.SearchInSync && drift.DefaultInSync
	return drift
}

func maskAny(pipeline util.MapStr) util.MapStr {
	if pipeline == nil {
		return nil
	}
	return MaskEngineAISecret(pipeline)
}

// pipelineEqual compares two pipeline definitions semantically: JSON bytes
// after sorting are still map-order dependent, so compare via DeepEqual on
// parsed maps.
func pipelineEqual(a, b util.MapStr) bool {
	if a == nil || b == nil {
		return false
	}
	return util.ToString(a) == util.ToString(b)
}
