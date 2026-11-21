/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"testing"

	"infini.sh/coco/core"
)

func TestScoreEvalRun(t *testing.T) {
	// no cases: zeros, no division
	if total, hits, mrr, avg := scoreEvalRun(nil); total != 0 || hits != 0 || mrr != 0 || avg != 0 {
		t.Fatalf("empty run should score zero, got %d/%d/%v/%d", total, hits, mrr, avg)
	}

	results := []core.SearchEvalCaseResult{
		{Query: "a", HitRank: 1, TookMS: 100}, // top-4 hit, rank 1
		{Query: "b", HitRank: 4, TookMS: 200}, // top-4 hit at the boundary
		{Query: "c", HitRank: 5, TookMS: 300}, // recalled but outside top-4
		{Query: "d", HitRank: 0, TookMS: 400}, // miss
	}
	total, hits, mrr, avg := scoreEvalRun(results)
	if total != 4 || hits != 2 {
		t.Fatalf("total=%d top4=%d, want 4/2", total, hits)
	}
	wantMRR := (1.0 + 1.0/4.0 + 1.0/5.0) / 4.0
	if mrr != wantMRR {
		t.Fatalf("mrr = %v, want %v", mrr, wantMRR)
	}
	if avg != 250 {
		t.Fatalf("avg took = %d, want 250", avg)
	}
}

func TestNormalizeIDList(t *testing.T) {
	out := normalizeIDList([]string{" a ", "", "b", "a", " "})
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Fatalf("normalizeIDList = %v", out)
	}
	if got := normalizeIDList(nil); len(got) != 0 {
		t.Fatalf("nil input should stay empty, got %v", got)
	}
}

func TestNormalizeTitleList(t *testing.T) {
	out := normalizeTitleList([]string{" 一 ", "二"}, 3)
	if len(out) != 3 || out[0] != "一" || out[1] != "二" || out[2] != "" {
		t.Fatalf("normalizeTitleList = %v", out)
	}
}

func TestEvalGapEvidence(t *testing.T) {
	ev := evalGapEvidence("十三薪", &core.SearchEvalCase{
		ExpectedIDs:    []string{"doc1"},
		ExpectedTitles: []string{"年终双薪政策"},
	})
	if ev["query"] != "十三薪" || ev["source"] != "eval_set" {
		t.Fatalf("evidence = %v", ev)
	}
}
