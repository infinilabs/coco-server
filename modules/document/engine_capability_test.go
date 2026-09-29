/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"strings"
	"testing"
	"time"

	"infini.sh/coco/core"
)

// resetEngineCapabilityCache clears the probe cache so test scenarios are
// independent of each other's cached verdicts.
func resetEngineCapabilityCache() {
	engineCapabilityMu.Lock()
	engineCapabilityCached = nil
	engineCapabilityUntil = time.Time{}
	engineCapabilityMu.Unlock()
}

func TestClassifyEngineSemanticError(t *testing.T) {
	cases := []struct {
		name        string
		errText     string
		available   bool
		definitive  bool
		reasonParts string
	}{
		{
			name:        "engine without embedding service (real 500 body)",
			errText:     `Cannot invoke "org.easysearch.ai.action.embedding.EmbeddingRequest.dimensions(java.lang.Integer)" because "this.embeddingRequest" is null`,
			available:   false,
			definitive:  true,
			reasonParts: "engine embedding service is not configured",
		},
		{
			name:        "semantic query rejected",
			errText:     "[semantic] query does not support [model_id]",
			available:   false,
			definitive:  true,
			reasonParts: "engine rejected the semantic query",
		},
		{
			name:        "knn query rejected",
			errText:     "[knn] query does not support [vec1]",
			available:   false,
			definitive:  true,
			reasonParts: "engine rejected the semantic query",
		},
		{
			name:        "parser noise",
			errText:     "[semantic] unknown token [START_ARRAY] after [query_vector]",
			available:   false,
			definitive:  true,
			reasonParts: "engine rejected the semantic query",
		},
		{
			name:       "success",
			errText:    "",
			available:  true,
			definitive: true,
		},
		{
			name:       "transient network error is not cached",
			errText:    "connection refused 127.0.0.1:9200",
			available:  false,
			definitive: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cap, definitive := classifyEngineSemanticError(c.errText)
			if cap.Available != c.available {
				t.Fatalf("available = %v, want %v", cap.Available, c.available)
			}
			if definitive != c.definitive {
				t.Fatalf("definitive = %v, want %v", definitive, c.definitive)
			}
			if c.reasonParts != "" && !strings.Contains(cap.Reason, c.reasonParts) {
				t.Fatalf("reason %q does not contain %q", cap.Reason, c.reasonParts)
			}
		})
	}
}

func TestEngineVectorCapabilityCachesDefinitiveVerdicts(t *testing.T) {
	restore := probeEngineSemanticFn
	defer func() { probeEngineSemanticFn = restore }()
	resetEngineCapabilityCache()

	calls := 0
	probeEngineSemanticFn = func(ctx context.Context) EngineVectorCapability {
		calls++
		return EngineVectorCapability{Available: false, Reason: "engine embedding service is not configured"}
	}

	ctx := context.Background()
	if engineVectorCapability(ctx).Available {
		t.Fatal("expected unavailable")
	}
	if engineVectorCapability(ctx).Available {
		t.Fatal("expected unavailable")
	}
	if calls != 1 {
		t.Fatalf("definitive verdict should be cached, probe ran %d times", calls)
	}

	// transient verdicts must not be cached
	resetEngineCapabilityCache()
	probeEngineSemanticFn = func(ctx context.Context) EngineVectorCapability {
		calls++
		return EngineVectorCapability{Available: false, Reason: "connection refused"}
	}
	engineVectorCapability(ctx)
	engineVectorCapability(ctx)
	if calls != 3 {
		t.Fatalf("transient verdict should not be cached, probe ran %d times total", calls)
	}
}

func TestPlanSemanticRoutes(t *testing.T) {
	restoreProbe, restoreResolve := probeEngineSemanticFn, resolveEmbeddingModelFn
	defer func() {
		probeEngineSemanticFn, resolveEmbeddingModelFn = restoreProbe, restoreResolve
	}()

	ctx := context.Background()

	probeEngineSemanticFn = func(ctx context.Context) EngineVectorCapability {
		return EngineVectorCapability{Available: true}
	}
	if got := planSemantic(ctx).Route; got != semanticRouteEngine {
		t.Fatalf("route = %v, want engine", got)
	}

	resetEngineCapabilityCache()
	probeEngineSemanticFn = func(ctx context.Context) EngineVectorCapability {
		return EngineVectorCapability{Available: false, Reason: "engine embedding service is not configured"}
	}
	resolveEmbeddingModelFn = func() *core.ModelId { return nil }
	if got := planSemantic(ctx).Route; got != semanticRouteNone {
		t.Fatalf("route = %v, want none", got)
	}

	resolveEmbeddingModelFn = func() *core.ModelId { return &core.ModelId{ProviderID: "p", ID: "m"} }
	plan := planSemantic(ctx)
	if plan.Route != semanticRouteClient {
		t.Fatalf("route = %v, want client", plan.Route)
	}
	if plan.Reason == "" || plan.EmbeddingModel == nil {
		t.Fatalf("client plan should carry a reason and the model, got %+v", plan)
	}
}
