/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "infini.sh/framework/core/orm"

// Golden-query evaluation set (D9): a fixed, human-annotated set of
// query → expected-document pairs, scored against the live search pipeline.
// Any change to the recall stack (RRF weights, rerank window, query rewrite)
// runs the set before and after — tuning without an evaluation set is
// guessing with extra steps.

// SearchEvalCase is one golden query: what a user would ask and which
// documents must come back for the answer to count as correct. ExpectedIDs
// may reference documents or wiki articles — matching happens on hit ids.
type SearchEvalCase struct {
	orm.ORMObjectBase
	Query          string   `json:"query" elastic_mapping:"query:{type:keyword}"`
	ExpectedIDs    []string `json:"expected_ids" elastic_mapping:"expected_ids:{type:keyword}"`
	ExpectedTitles []string `json:"expected_titles,omitempty" elastic_mapping:"expected_titles:{type:keyword}"`
	// Datasource optionally scopes the case to one datasource, mirroring how
	// the query was asked in the wild.
	Datasource string `json:"datasource,omitempty" elastic_mapping:"datasource:{type:keyword}"`
	Note       string `json:"note,omitempty" elastic_mapping:"note:{type:keyword}"`
}

// SearchEvalRun is one scoring pass over the whole set.
type SearchEvalRun struct {
	orm.ORMObjectBase
	TotalCases int64                  `json:"total_cases" elastic_mapping:"total_cases:{type:long}"`
	Top4Hits   int64                  `json:"top4_hits" elastic_mapping:"top4_hits:{type:long}"`
	Top4Rate   float64                `json:"top4_rate" elastic_mapping:"top4_rate:{type:float}"`
	MRR        float64                `json:"mrr" elastic_mapping:"mrr:{type:float}"`
	AvgTookMS  int64                  `json:"avg_took_ms" elastic_mapping:"avg_took_ms:{type:long}"`
	Cases      []SearchEvalCaseResult `json:"cases" elastic_mapping:"cases:{type:object,enabled:false}"`
}

// SearchEvalCaseResult is one case's outcome inside a run.
type SearchEvalCaseResult struct {
	Query string `json:"query"`
	// HitRank is the 1-based rank of the first expected document in the
	// fused (and reranked) result; 0 means no expected document returned.
	HitRank   int      `json:"hit_rank"`
	Total     int64    `json:"total"`
	TookMS    int64    `json:"took_ms"`
	TopIDs    []string `json:"top_ids,omitempty"`
	TopTitles []string `json:"top_titles,omitempty"`
	Error     string   `json:"error,omitempty"`
}
