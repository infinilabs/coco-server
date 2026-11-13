/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package common

import (
	"strings"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func testEngineAIProvider() *core.ModelProvider {
	return &core.ModelProvider{
		BaseURL: "https://api.example.com/v1",
		APIType: "openai",
		APIKey:  "sk-secret",
	}
}

func TestBuildEnginePipelines(t *testing.T) {
	cfg := &core.EngineAI{Enabled: true}
	model := &core.ModelId{ProviderID: "prov1", ID: "embed-1024"}
	ingest, search := buildEnginePipelines(cfg, model, testEngineAIProvider())

	te, _ := ingest["processors"].([]interface{})
	if len(te) != 1 {
		t.Fatalf("expected one processor, got %v", ingest["processors"])
	}
	body := te[0].(util.MapStr)["text_embedding"].(util.MapStr)
	if body["text_field"] != core.EngineAITextFieldDefault {
		t.Fatalf("text_field default = %v", body["text_field"])
	}
	if body["vector_field"] != "ai_insights.embedding.embedding1024" {
		t.Fatalf("vector_field default = %v", body["vector_field"])
	}
	if body["url"] != "https://api.example.com/v1/embeddings" {
		t.Fatalf("url = %v", body["url"])
	}
	if body["dims"] != core.RequiredEmbeddingDimension {
		t.Fatalf("dims = %v", body["dims"])
	}
	if body["batch_size"] != 10 {
		t.Fatalf("batch_size default = %v", body["batch_size"])
	}

	rewrite := search["rewrite_processors"].([]interface{})
	enricher := rewrite[0].(util.MapStr)["semantic_query_enricher"].(util.MapStr)
	if enricher["default_model_id"] != "embed-1024" {
		t.Fatalf("default_model_id = %v", enricher["default_model_id"])
	}
	rerank := search["rerank_processors"].([]interface{})
	ranker := rerank[0].(util.MapStr)["hybrid_ranker_processor"].(util.MapStr)
	combination := ranker["combination"].(util.MapStr)
	if combination["technique"] != "rrf" {
		t.Fatalf("technique = %v", combination["technique"])
	}
	if combination["parameters"].(util.MapStr)["rank_constant"] != 60 {
		t.Fatalf("rank_constant default = %v", combination["parameters"])
	}

	// explicit knobs flow through
	cfg2 := &core.EngineAI{Enabled: true, TextField: "content", VectorField: "vec1", BatchSize: 3, RankConstant: 100}
	ingest2, search2 := buildEnginePipelines(cfg2, model, testEngineAIProvider())
	body2 := ingest2["processors"].([]interface{})[0].(util.MapStr)["text_embedding"].(util.MapStr)
	if body2["text_field"] != "content" || body2["vector_field"] != "vec1" || body2["batch_size"] != 3 {
		t.Fatalf("explicit knobs not applied: %v", body2)
	}
	params := search2["rerank_processors"].([]interface{})[0].(util.MapStr)["hybrid_ranker_processor"].(util.MapStr)["combination"].(util.MapStr)["parameters"].(util.MapStr)
	if params["rank_constant"] != 100 {
		t.Fatalf("rank_constant = %v", params)
	}
}

func TestMaskEngineAISecret(t *testing.T) {
	ingest, _ := buildEnginePipelines(&core.EngineAI{}, &core.ModelId{ProviderID: "p", ID: "m"}, testEngineAIProvider())
	masked := MaskEngineAISecret(ingest)
	body := util.MapStr{}
	for k, v := range masked {
		body[k] = v
	}
	s := util.ToString(masked)
	if strings.Contains(s, "sk-secret") {
		t.Fatal("api_key leaked in masked pipeline")
	}
	if !strings.Contains(s, "******") {
		t.Fatal("api_key not masked")
	}
	// the unmasked original still carries the key
	if !strings.Contains(util.ToString(ingest), "sk-secret") {
		t.Fatal("masking must not mutate the original")
	}
}

func TestCompareEnginePipelines(t *testing.T) {
	model := &core.ModelId{ProviderID: "p", ID: "m"}
	cfg := &core.EngineAI{Enabled: true}
	ingest, search := buildEnginePipelines(cfg, model, testEngineAIProvider())
	plan := &EngineAIPlan{
		Enabled: true, Model: model,
		IngestPipeline: ingest, SearchPipeline: search,
	}

	// what ReadEnginePipelines stores: unwrapped definitions, with the
	// api_key stored differently than Coco sent it (encrypted/masked)
	actual := &EnginePipelineActual{
		IngestPipeline:  maskDeep(ingest).(util.MapStr),
		SearchPipeline:  maskDeep(search).(util.MapStr),
		DefaultPipeline: core.EngineIngestPipelineName,
	}
	drift := CompareEnginePipelines(plan, actual)
	if !drift.Applied {
		t.Fatalf("identical pipelines should be in sync: %+v", drift)
	}

	// api_key-insensitive: engine stores a different key spelling but same shape
	actual.DefaultPipeline = "other-pipeline"
	drift = CompareEnginePipelines(plan, actual)
	if drift.DefaultInSync || drift.Applied {
		t.Fatalf("drift should be reported: %+v", drift)
	}

	// missing pipelines on the engine
	drift = CompareEnginePipelines(plan, &EnginePipelineActual{})
	if drift.IngestInSync || drift.SearchInSync || drift.DefaultInSync || drift.Applied {
		t.Fatalf("empty engine should report drift everywhere: %+v", drift)
	}

	// disabled config never reports applied
	drift = CompareEnginePipelines(&EngineAIPlan{Enabled: false}, actual)
	if drift.Applied {
		t.Fatal("disabled plan must not report applied")
	}
}

// maskDeep mimics the engine storing the key differently than Coco sent it.
func maskDeep(node interface{}) interface{} {
	switch m := node.(type) {
	case map[string]interface{}:
		out := util.MapStr{}
		for k, v := range m {
			if k == "api_key" {
				out[k] = "::encrypted::"
				continue
			}
			out[k] = maskDeep(v)
		}
		return out
	case util.MapStr:
		return maskDeep(map[string]interface{}(m))
	case []interface{}:
		out := make([]interface{}, len(m))
		for i, v := range m {
			out[i] = maskDeep(v)
		}
		return out
	default:
		return node
	}
}

func TestUnwrapPipeline(t *testing.T) {
	if unwrapPipeline(nil, "x") != nil {
		t.Fatal("nil wrapper should stay nil")
	}
	wrapped := util.MapStr{"coco-embedding": util.MapStr{"processors": []interface{}{}}}
	if unwrapPipeline(wrapped, "coco-embedding") == nil {
		t.Fatal("definition not lifted out of the wrapper")
	}
}
