/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package extract_tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterTagsByVocab(t *testing.T) {
	vocab := []string{"支付", "Risk", "合规"}
	tags := []string{"支付", "risk", "未知词", "合规", " another unknown "}

	kept, dropped := filterTagsByVocab(tags, vocab)
	assert.Equal(t, []string{"支付", "risk", "合规"}, kept, "known tags kept (case-insensitive), unknown dropped")
	assert.Equal(t, 2, dropped)

	// empty vocabulary = free-form mode, nothing filtered by this helper
	kept, dropped = filterTagsByVocab(tags, nil)
	assert.Equal(t, tags, kept)
	assert.Zero(t, dropped)
}

func TestBuildConstrainedTagPromptListsVocabulary(t *testing.T) {
	prompt := buildConstrainedTagPrompt("insights", "zh-CN", []string{"支付", "合规"})
	assert.Contains(t, prompt, `"支付"`)
	assert.Contains(t, prompt, `"合规"`)
	assert.Contains(t, prompt, "MUST choose ONLY from this vocabulary", "the model must be told the boundary explicitly")
	assert.Contains(t, prompt, "no new tags")
}
