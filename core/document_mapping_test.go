/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"infini.sh/framework/core/util"
)

// aiInsightsTextTag walks the elastic_mapping annotation tree the way the
// schema registration does (nested structs expand into their parent's tag)
// and returns the annotation of AiInsights.Text.
func aiInsightsTextTag(t *testing.T) string {
	t.Helper()
	for _, a := range util.GetTagsByTagName(Document{}, "elastic_mapping") {
		if a.Field != "AiInsights" {
			continue
		}
		for _, nested := range a.Annotation {
			if nested.Field == "Text" {
				return nested.Tag
			}
		}
	}
	t.Fatal("AiInsights.Text not found in elastic_mapping annotations")
	return ""
}

// The AI interpretation must reach combined_fulltext: keyword search only
// queries title + combined_fulltext (QueryDocuments defaultFields), so a
// missing copy_to here makes ai_insights.text unsearchable by keyword.
func TestAiInsightsTextCopiesToCombinedFulltext(t *testing.T) {
	assert.Contains(t, aiInsightsTextTag(t), "copy_to:combined_fulltext")
	assert.Contains(t, aiInsightsTextTag(t), "type:text")
}
