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
