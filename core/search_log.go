/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package core

import "infini.sh/framework/core/orm"

// SearchLog is one completed search: what was asked, which strategy ran,
// how many hits came back and how long it took. It is the raw material for
// the operations overview (P2) and the knowledge-gap loop (D5) — a query
// that keeps returning zero hits is the cheapest signal there is that the
// knowledge base lacks a page for it.
type SearchLog struct {
	orm.ORMObjectBase
	// Query is stored lowercased/trimmed as a keyword so repeated queries
	// aggregate exactly.
	Query string `json:"query" elastic_mapping:"query:{type:keyword}"`
	// Raw is the user's input as typed (whitespace-trimmed only, case and
	// inner spacing preserved) — diagnostics want the original text; the
	// normalized Query above stays the aggregation/filter key. Empty on
	// rows written before this field existed.
	Raw        string `json:"raw,omitempty" elastic_mapping:"raw:{type:keyword}"`
	SearchType string `json:"search_type" elastic_mapping:"search_type:{type:keyword}"` // keyword | semantic | hybrid_rrf | ...
	Total      int64  `json:"total" elastic_mapping:"total:{type:long}"`
	TookMS     int64  `json:"took_ms" elastic_mapping:"took_ms:{type:long}"`
	ZeroHit    bool   `json:"zero_hit" elastic_mapping:"zero_hit:{type:boolean}"`
	// Rewritten marks searches where the query-rewrite leg fired and
	// contributed a rewritten keyword route to the fusion (D8) — the flag
	// is how the overview quantifies the rewrite's lift.
	Rewritten bool   `json:"rewritten" elastic_mapping:"rewritten:{type:boolean}"`
	UserID    string `json:"user_id,omitempty" elastic_mapping:"user_id:{type:keyword}"`
	// Diagnostics: per-leg timings and outcome evidence so "why slow / why
	// these results" triage closes inside the log, without a reproduction.
	// All optional — rows written before these existed (and trivial keyword
	// searches) leave them out.
	LegKeywordMS  int64   `json:"leg_keyword_ms,omitempty" elastic_mapping:"leg_keyword_ms:{type:long}"`
	LegSemanticMS int64   `json:"leg_semantic_ms,omitempty" elastic_mapping:"leg_semantic_ms:{type:long}"`
	LegRewriteMS  int64   `json:"leg_rewrite_ms,omitempty" elastic_mapping:"leg_rewrite_ms:{type:long}"`
	EngineTookMS  int64   `json:"engine_took_ms,omitempty" elastic_mapping:"engine_took_ms:{type:long}"`
	TopHit        string  `json:"top_hit,omitempty" elastic_mapping:"top_hit:{type:keyword}"`
	TopScore      float64 `json:"top_score,omitempty" elastic_mapping:"top_score:{type:float}"`
	// Error is set when the search itself failed (engine down, leg error);
	// failed searches previously never reached the log at all
	Error string `json:"error,omitempty" elastic_mapping:"error:{type:keyword}"`
}
