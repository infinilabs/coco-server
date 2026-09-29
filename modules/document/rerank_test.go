/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"fmt"
	"testing"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

func rerankTestPlan() rerankPlan {
	return rerankPlan{Model: &core.ModelId{ProviderID: "p1", ID: "qwen3-reranker"}}
}

func stubRerankProvider(t *testing.T) {
	restore := getRerankProviderFn
	getRerankProviderFn = func(id string) (*core.ModelProvider, error) {
		if id != "p1" {
			return nil, fmt.Errorf("provider %s not found", id)
		}
		return &core.ModelProvider{Name: "P1", APIType: "openai", BaseURL: "https://api.example.com/v1", APIKey: "sk-test"}, nil
	}
	t.Cleanup(func() { getRerankProviderFn = restore })
}

func stubRerankHTTP(t *testing.T, body string, err error) {
	restore := rerankHTTPPostFn
	rerankHTTPPostFn = func(ctx context.Context, url, apiKey string, payload []byte) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		return []byte(body), nil
	}
	t.Cleanup(func() { rerankHTTPPostFn = restore })
}

func TestRerankFusedHitsReorders(t *testing.T) {
	stubRerankProvider(t)
	stubRerankHTTP(t, `{"results":[
		{"index":2,"relevance_score":0.9},
		{"index":0,"relevance_score":0.8},
		{"index":1,"relevance_score":0.7}
	]}`, nil)

	fused := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 1), rrfHit("b", 2), rrfHit("c", 3)}
	out, changes, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", fused)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0].ID != "c" || out[1].ID != "a" || out[2].ID != "b" {
		t.Fatalf("rerank order wrong: %v %v %v", out[0].ID, out[1].ID, out[2].ID)
	}
	if len(changes) != 3 || changes[0].ID != "c" || changes[0].RerankRank != 1 || changes[0].RRFRank != 3 {
		t.Fatalf("changes wrong: %+v", changes)
	}
	if changes[0].RelevanceScore != 0.9 {
		t.Fatalf("relevance score wrong: %+v", changes[0])
	}
}

func TestRerankTiesKeepRRFOrder(t *testing.T) {
	stubRerankProvider(t)
	// all equal scores: RRF order must survive (stable sort over RRF input)
	stubRerankHTTP(t, `{"results":[
		{"index":0,"relevance_score":0.5},
		{"index":1,"relevance_score":0.5},
		{"index":2,"relevance_score":0.5}
	]}`, nil)

	fused := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 1), rrfHit("b", 2), rrfHit("c", 3)}
	out, _, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", fused)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].ID != "a" || out[1].ID != "b" || out[2].ID != "c" {
		t.Fatalf("ties must keep RRF order, got %v %v %v", out[0].ID, out[1].ID, out[2].ID)
	}
}

func TestRerankUnscoredHitsTrailBehind(t *testing.T) {
	stubRerankProvider(t)
	// only index 1 scored; the rest keep RRF order behind the scored one
	stubRerankHTTP(t, `{"results":[{"index":1,"relevance_score":0.99}]}`, nil)

	fused := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 1), rrfHit("b", 2), rrfHit("c", 3)}
	out, changes, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", fused)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0].ID != "b" || out[1].ID != "a" || out[2].ID != "c" {
		t.Fatalf("unscored hits must trail in RRF order: %v %v %v", out[0].ID, out[1].ID, out[2].ID)
	}
	if len(changes) != 1 {
		t.Fatalf("one change expected, got %+v", changes)
	}
}

func TestRerankIgnoresInvalidIndexes(t *testing.T) {
	stubRerankProvider(t)
	stubRerankHTTP(t, `{"results":[
		{"index":0,"relevance_score":0.4},
		{"index":7,"relevance_score":0.99},
		{"index":-1,"relevance_score":0.99},
		{"index":0,"relevance_score":0.3}
	]}`, nil)

	fused := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 1), rrfHit("b", 2)}
	out, changes, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", fused)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].ID != "a" {
		t.Fatalf("out-of-range and duplicate indexes must be ignored, got %+v", changes)
	}
	if out[0].ID != "a" {
		t.Fatalf("order wrong: %+v", out)
	}
}

func TestRerankDegradesOnError(t *testing.T) {
	stubRerankProvider(t)
	stubRerankHTTP(t, "", fmt.Errorf("connection refused"))

	fused := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 1), rrfHit("b", 2)}
	out, _, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", fused)
	if err == nil {
		t.Fatal("expected error from failed rerank call")
	}
	if len(out) != 2 || out[0].ID != "a" {
		t.Fatalf("degraded output must be the original fused list, got %+v", out)
	}

	// the production hook turns the error into a note, not a failure
	note := ""
	got, appliedNote := applyRerankWithPlan(context.Background(), rerankTestPlan(), "q", fused)
	if len(got) != 2 || got[0].ID != "a" {
		t.Fatalf("applyRerank must keep RRF order on failure, got %+v", got)
	}
	if appliedNote == note || appliedNote == "" {
		t.Fatalf("degrade note missing: %q", appliedNote)
	}

	// no model configured: unchanged, silent note
	plan := rerankPlan{Note: "no rerank model configured, fused order kept"}
	got, note = applyRerankWithPlan(context.Background(), plan, "q", fused)
	if len(got) != 2 || got[0].ID != "a" || note != plan.Note {
		t.Fatalf("unconfigured plan must pass through, got %+v / %q", got, note)
	}
}

func TestRerankWindowCap(t *testing.T) {
	stubRerankProvider(t)
	var documentsSeen = -1
	restore := rerankHTTPPostFn
	rerankHTTPPostFn = func(ctx context.Context, url, apiKey string, payload []byte) ([]byte, error) {
		var req rerankRequest
		if err := util.FromJSONBytes(payload, &req); err != nil {
			return nil, err
		}
		documentsSeen = len(req.Documents)
		return []byte(`{"results":[]}`), nil
	}
	t.Cleanup(func() { rerankHTTPPostFn = restore })

	fused := make([]elastic.DocumentWithMeta[core.Document], rerankCandidateWindow+10)
	for i := range fused {
		fused[i] = rrfHit(fmt.Sprintf("d%d", i), float32(i))
	}
	out, _, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", fused)
	if err != nil {
		t.Fatal(err)
	}
	if documentsSeen != rerankCandidateWindow {
		t.Fatalf("rerank must cap the scored window at %d, sent %d", rerankCandidateWindow, documentsSeen)
	}
	if len(out) != len(fused) {
		t.Fatalf("no hits may be dropped, got %d of %d", len(out), len(fused))
	}
	if out[0].ID != fused[0].ID {
		t.Fatalf("empty reranker response keeps RRF order, got %s", out[0].ID)
	}
}
