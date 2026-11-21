/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func TestParseDistillOutput(t *testing.T) {
	// plain JSON array
	items, err := parseDistillOutput(`[{"kind":"preference","content":"Reply in English"},{"kind":"fact","content":"Works on payments"}]`)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "preference", items[0].Kind)

	// fenced with chatter
	items, err = parseDistillOutput("Here you go:\n```json\n[{\"kind\":\"task\",\"content\":\"Ship the report\"}]\n```\nDone!")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "task", items[0].Kind)

	// capped at five
	big := `[`
	for i := 0; i < 8; i++ {
		if i > 0 {
			big += ","
		}
		big += `{"kind":"fact","content":"f` + string(rune('0'+i)) + `"}`
	}
	big += `]`
	items, err = parseDistillOutput(big)
	require.NoError(t, err)
	assert.Len(t, items, 5, "distillation caps at 5 memories per run")

	// no array
	_, err = parseDistillOutput("I could not find any memories.")
	assert.Error(t, err)
}

func TestKindValidation(t *testing.T) {
	for _, k := range core.ResidentMemoryKinds {
		assert.True(t, core.ValidMemoryKind(k))
		assert.True(t, isResidentKind(k))
	}
	for _, k := range core.RecalledMemoryKinds {
		assert.True(t, core.ValidMemoryKind(k))
		assert.False(t, isResidentKind(k))
	}
	assert.False(t, core.ValidMemoryKind("snack"))
}

func TestKeywordOverlap(t *testing.T) {
	assert.True(t, keywordOverlap("user works on payment gateway", "payment gateway"))
	assert.True(t, keywordOverlap("季度考核制度", "考核"))
	assert.False(t, keywordOverlap("payment gateway", "vacation policy"))
	assert.False(t, keywordOverlap("anything", ""), "empty query recalls nothing")
}
