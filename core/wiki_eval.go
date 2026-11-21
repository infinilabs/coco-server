/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"infini.sh/framework/core/orm"
)

// Per-KB validation set (LLM-wiki P3): a human-curated set of query →
// expected-page pairs for one knowledge base, scored by replaying each
// query through the same KB-scoped published-page retrieval the chat
// knowledge tier uses. The design (§4.2 G4) makes replay a progressive
// capability — KBs without cases keep the LLM-review + human-approval
// gate. A skill proposal that lands in a KB WITH cases additionally
// triggers a replay so the score trend lands in the evolution log next
// to the skill it gated; answer-level (LLM) scoring and automatic
// rollback on regression are explicitly future work, not silently
// implied by these numbers.

// WikiEvalCase is one golden query for a KB. Upsert-keyed on the
// normalized query within the KB: re-annotating a question replaces its
// expectation instead of stacking a twin.
type WikiEvalCase struct {
	orm.ORMObjectBase
	KbID string `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
	// Query is the normalized golden query (trimmed, whitespace-folded).
	Query string `json:"query" elastic_mapping:"query:{type:keyword}"`
	// ExpectedIDs are wiki article ids the retrieval must surface; at
	// least one. Titles ride along for display only.
	ExpectedIDs    []string `json:"expected_ids" elastic_mapping:"expected_ids:{type:keyword}"`
	ExpectedTitles []string `json:"expected_titles,omitempty" elastic_mapping:"expected_titles:{type:keyword}"`
	Note           string   `json:"note,omitempty" elastic_mapping:"note:{type:keyword}"`
}

// WikiEvalCaseResult is one case's outcome inside a run.
type WikiEvalCaseResult struct {
	Query string `json:"query"`
	// HitRank is the 1-based rank of the first expected page inside the
	// scoring window; 0 = not found.
	HitRank int `json:"hit_rank"`
	Total   int `json:"total,omitempty"`
	// TopIDs/TopTitles carry the first ranks for display (what DID show).
	TopIDs    []string `json:"top_ids,omitempty"`
	TopTitles []string `json:"top_titles,omitempty"`
	Error     string   `json:"error,omitempty"`
	TookMS    int64    `json:"took_ms,omitempty"`
}

// WikiEvalRun is one scoring pass over a KB's whole set. Stored every
// time so tuning shows its trend (same contract as the search eval set).
type WikiEvalRun struct {
	orm.ORMObjectBase
	KbID       string               `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
	TotalCases int                  `json:"total_cases" elastic_mapping:"total_cases:{type:integer}"`
	Top4Hits   int                  `json:"top4_hits" elastic_mapping:"top4_hits:{type:integer}"`
	Top4Rate   float64              `json:"top4_rate" elastic_mapping:"top4_rate:{type:float}"`
	MRR        float64              `json:"mrr" elastic_mapping:"mrr:{type:float}"`
	AvgTookMS  int64                `json:"avg_took_ms,omitempty" elastic_mapping:"avg_took_ms:{type:integer}"`
	Cases      []WikiEvalCaseResult `json:"cases,omitempty" elastic_mapping:"cases:{type:object,enabled:false}"`
	// Trigger names what started the run: "manual" (operator hit the
	// button) or "skill_applied" (the post-apply gate tie-in).
	Trigger string `json:"trigger,omitempty" elastic_mapping:"trigger:{type:keyword}"`
}
