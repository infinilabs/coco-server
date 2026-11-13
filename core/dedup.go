/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "infini.sh/framework/core/orm"

// DocumentDedupDismissal records a reviewer's "these two are NOT duplicates"
// verdict. The dedup scan consults these pairs and never reports them again
// — the human call outranks the math permanently.
type DocumentDedupDismissal struct {
	orm.ORMObjectBase
	// PairKey is the stable key of an unordered document id pair.
	PairKey string `json:"pair_key" elastic_mapping:"pair_key:{type:keyword}"`
	Reason  string `json:"reason,omitempty" elastic_mapping:"reason:{type:text}"`
}

// DocumentDedupGroup persists one operator-confirmed near-duplicate group
// (W12): near-duplicate candidates (simhash/phash tiers) only fold in
// search results after this confirmation; exact hash groups fold without
// it. GroupKey is the hash of the sorted member ids — stable across scans.
type DocumentDedupGroup struct {
	orm.ORMObjectBase
	GroupKey  string   `json:"group_key" elastic_mapping:"group_key:{type:keyword}"`
	MemberIDs []string `json:"member_ids" elastic_mapping:"member_ids:{type:keyword}"`
	Tier      string   `json:"tier,omitempty" elastic_mapping:"tier:{type:keyword}"`
}
