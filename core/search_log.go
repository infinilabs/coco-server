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
	Query      string `json:"query" elastic_mapping:"query:{type:keyword}"`
	SearchType string `json:"search_type" elastic_mapping:"search_type:{type:keyword}"` // keyword | semantic | hybrid_rrf | ...
	Total      int64  `json:"total" elastic_mapping:"total:{type:long}"`
	TookMS     int64  `json:"took_ms" elastic_mapping:"took_ms:{type:long}"`
	ZeroHit    bool   `json:"zero_hit" elastic_mapping:"zero_hit:{type:boolean}"`
	// Rewritten marks searches where the query-rewrite leg fired and
	// contributed a rewritten keyword route to the fusion (D8) — the flag
	// is how the overview quantifies the rewrite's lift.
	Rewritten bool   `json:"rewritten" elastic_mapping:"rewritten:{type:boolean}"`
	UserID    string `json:"user_id,omitempty" elastic_mapping:"user_id:{type:keyword}"`
}
