/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
	out, changes, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", nil, fused)
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

	fused := []elastic.DocumentWithMeta[core.Document]{rrfHit("a", 3), rrfHit("b", 2), rrfHit("c", 1)}
	out, _, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", nil, fused)
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
	out, changes, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", nil, fused)
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
	out, changes, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", nil, fused)
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
	out, _, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", nil, fused)
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
	out, _, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", nil, fused)
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

/* ---------------- D4.5 composite + MMR ---------------- */

func TestSourceWeightFor(t *testing.T) {
	assert.Equal(t, 1.0, sourceWeightFor("wiki", nil), "built-in knowledge source tops the scale")
	assert.InDelta(t, 1.0, sourceWeightFor("ds-first", []string{"ds-first", "ds-second"}), 1e-9)
	assert.InDelta(t, 0.75, sourceWeightFor("ds-second", []string{"ds-first", "ds-second"}), 1e-9)
	assert.InDelta(t, 0.5, sourceWeightFor("ds-unranked", []string{"ds-first"}), 1e-9, "unlisted sources sit at the baseline")
	assert.InDelta(t, 0.5, sourceWeightFor("anything", nil), 1e-9)
}

func TestRerankCompositeSourcePriorityWins(t *testing.T) {
	stubRerankProvider(t)
	// identical model scores and identical RRF scores: the source weight
	// decides — the priority-listed source outranks the unranked twin
	stubRerankHTTP(t, `{"results":[
		{"index":0,"relevance_score":0.8},
		{"index":1,"relevance_score":0.8}
	]}`, nil)

	plain := rrfHit("plain", 5)
	plain.Source.Source.ID = "ds-plain"
	listed := rrfHit("listed", 5)
	listed.Source.Source.ID = "ds-trusted"
	fused := []elastic.DocumentWithMeta[core.Document]{plain, listed}

	out, changes, err := rerankFusedHits(context.Background(), rerankTestPlan(), "q", []string{"ds-trusted"}, fused)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].ID != "listed" {
		t.Fatalf("priority-listed source must win the tie, got %s first", out[0].ID)
	}
	if len(changes) != 2 || changes[0].CompositeScore <= changes[1].CompositeScore {
		t.Fatalf("composite scores must be reported and ordered, got %+v", changes)
	}
}

func TestMMROrderDemotesNearDuplicates(t *testing.T) {
	// b is a near-duplicate of a (identical text); c is diverse with a
	// slightly lower composite. MMR must push b below c.
	scored := []rerankComposite{
		{idx: 0, score: 0.90},
		{idx: 1, score: 0.85},
		{idx: 2, score: 0.80},
	}
	texts := []string{
		"年终奖 发放 政策 修订",
		"年终奖 发放 政策 修订",
		"季度 考核 制度 变更",
	}
	mmrOrder(scored, texts)
	if scored[0].idx != 0 {
		t.Fatalf("highest composite stays first, got %d", scored[0].idx)
	}
	if scored[1].idx != 2 {
		t.Fatalf("diverse hit must outrank the near-duplicate, got order %d,%d,%d", scored[0].idx, scored[1].idx, scored[2].idx)
	}
	if scored[2].idx != 1 {
		t.Fatalf("near-duplicate sinks last, got %d", scored[2].idx)
	}
}
