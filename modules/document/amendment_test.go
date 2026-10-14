/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAmendOutput(t *testing.T) {
	raw := `[{"action":"correct","claim":"报销周期为 30 天","current":"报销周期为 15 天","suggest":"报销周期为 30 天","excerpt":"报销应在 30 天内完成"},{"action":"supplement","claim":"新增电子发票支持","current":"","suggest":"支持电子发票","excerpt":"电子发票与纸质发票同等有效"}]`
	items, err := parseAmendOutput(raw)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "correct", items[0].Action)
	assert.Equal(t, "supplement", items[1].Action)

	// fenced + chatter
	items, err = parseAmendOutput("分析如下:\n```json\n" + raw + "\n```\n以上")
	require.NoError(t, err)
	require.Len(t, items, 2)

	// invalid actions dropped, cap respected
	big := `[`
	for i := 0; i < 12; i++ {
		if i > 0 {
			big += ","
		}
		act := "correct"
		if i%3 == 0 {
			act = "delete-everything" // invalid, dropped
		}
		big += `{"action":"` + act + `","claim":"c` + string(rune('a'+i)) + `"}`
	}
	big += `]`
	items, err = parseAmendOutput(big)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(items), amendMaxItems)
	for _, it := range items {
		assert.Contains(t, []string{"correct", "supplement", "obsolete"}, it.Action)
		assert.NotEmpty(t, it.Claim, "claim-less items are dropped")
	}

	// no array
	_, err = parseAmendOutput("两个文档一致,无需修改。")
	assert.Error(t, err)
}

func TestAmendEmptyArrayIsValid(t *testing.T) {
	items, err := parseAmendOutput("[]")
	require.NoError(t, err)
	assert.Empty(t, items, "an empty diff is a valid verdict — documents agree")
}
