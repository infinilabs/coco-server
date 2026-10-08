/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

// IsBuiltInIntegration reports whether the integration id is reserved for the
// app's own search page (see DefaultSearchIntegrationID).
func IsBuiltInIntegration(id string) bool {
	return id == DefaultSearchIntegrationID
}

// StripBuiltInIntegration removes the built-in search integration's hit from a
// list response and fixes the total accordingly. Total is a plain number on
// legacy ES and a {value, relation} object on 7.x+.
func StripBuiltInIntegration(res *elastic.SearchResponse) {
	kept := make([]elastic.IndexDocument, 0, len(res.Hits.Hits))
	removed := 0
	for _, hit := range res.Hits.Hits {
		if IsBuiltInIntegration(hit.ID) {
			removed++
			continue
		}
		kept = append(kept, hit)
	}
	if removed == 0 {
		return
	}
	res.Hits.Hits = kept
	switch total := res.Hits.Total.(type) {
	case map[string]interface{}:
		if v, ok := total["value"]; ok {
			total["value"] = util.GetInt64Value(v) - int64(removed)
		}
	case int64:
		res.Hits.Total = total - int64(removed)
	case int:
		res.Hits.Total = total - removed
	case float64:
		res.Hits.Total = total - float64(removed)
	}
}
