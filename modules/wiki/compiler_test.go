/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func TestParseCompileExtracts(t *testing.T) {
	raw := `[{"kind":"entity","name":"支付网关","aliases":["PGW"],"statement":"处理所有外部支付请求"},{"kind":"concept","name":"灰度发布","statement":"按比例放量"},{"kind":"junk","name":"x"}]`
	out, err := parseCompileExtracts(raw)
	require.NoError(t, err)
	require.Len(t, out, 2, "invalid kinds dropped")
	assert.Equal(t, "entity", out[0].Kind)

	fenced, err := parseCompileExtracts("```json\n" + raw + "\n```")
	require.NoError(t, err)
	require.Len(t, fenced, 2)

	_, err = parseCompileExtracts("没有可提取的内容。")
	assert.Error(t, err)
}

func TestReduceExtractsMergesByKey(t *testing.T) {
	extracts := []CompileExtract{
		{Kind: "entity", Name: "支付网关", Aliases: []string{"Payment Gateway"}, Statement: "处理支付", DocID: "d1"},
		{Kind: "entity", Name: "支付网关", Statement: "处理支付", DocID: "d2"},                                                   // dup statement text
		{Kind: "entity", Name: "支付网关", Aliases: []string{"payment gateway"}, Statement: "supports refunds", DocID: "d3"}, // same name → same key
		{Kind: "entity", Name: "订单服务", Statement: "管理订单", DocID: "d1"},
	}

	reduced := ReduceExtracts(extracts)
	require.Len(t, reduced, 2)

	// most evidence first: 支付网关 has 2 unique statements
	top := reduced[0]
	assert.Equal(t, "entity", top.Kind)
	require.Len(t, top.Statements, 2, "duplicate statement text deduped, distinct kept")
	assert.True(t, top.Aliases["Payment Gateway"], "aliases union across sources")

	// order independence
	shuffled := []CompileExtract{extracts[3], extracts[2], extracts[1], extracts[0]}
	reduced2 := ReduceExtracts(shuffled)
	require.Len(t, reduced2, 2)
	assert.Equal(t, reduced[0].NormKey, reduced2[0].NormKey)
}

func TestNormalizeCompileKeyStable(t *testing.T) {
	assert.Equal(t,
		NormalizeCompileKey("entity", "支付网关"),
		NormalizeCompileKey("entity", "支付 网关"),
		"whitespace and case fold to the same key")
	assert.NotEqual(t,
		NormalizeCompileKey("entity", "支付网关"),
		NormalizeCompileKey("concept", "支付网关"),
		"kind separates the space")
}

func TestPlanCompileCreateAndUpdate(t *testing.T) {
	existing := []core.WikiArticle{
		{PageType: core.WikiPageTypeEntity, Title: "支付网关"},
		{PageType: core.WikiPageTypeEntity, Title: "灰度发布"},
	}
	existing[0].ID = "art-existing"
	existing[0].LinkedPages = []core.WikiLinkedPage{{Name: "PGW"}}

	reduced := []ReducedKey{
		{Kind: "entity", NormKey: NormalizeCompileKey("entity", "支付网关"), Name: "PGW", Aliases: map[string]bool{"支付网关": true}}, // alias hit → update
		{Kind: "entity", NormKey: NormalizeCompileKey("entity", "订单服务"), Name: "订单服务"},                                        // no match → create
		{Kind: "concept", NormKey: NormalizeCompileKey("concept", "灰度发布"), Name: "灰度发布"},                                      // exact hit → update
	}

	plan := PlanCompile(reduced, existing)
	require.Len(t, plan, 3)
	assert.Equal(t, PlanUpdate, plan[0].Action, "alias match reconciles to the existing page")
	assert.Equal(t, "art-existing", plan[0].ExistingArticleID)
	assert.Equal(t, PlanCreate, plan[1].Action)
	assert.Equal(t, PlanUpdate, plan[2].Action)
}

func TestBatchChunkTextBudget(t *testing.T) {
	text := strings.Repeat("行内容\n", 100) // 400 runes
	batches := batchChunkText(text, 100)
	require.Greater(t, len(batches), 1)
	for _, b := range batches {
		assert.LessOrEqual(t, len([]rune(b)), 110, "batches respect the budget (line slack only)")
	}
	var rebuilt strings.Builder
	for _, b := range batches {
		rebuilt.WriteString(b)
	}
	assert.True(t, strings.HasPrefix(rebuilt.String(), strings.TrimRight(text, "\n")),
		"batches reassemble the content (at most a trailing newline differs)")

	assert.Equal(t, []string{"短"}, batchChunkText("短", 100))
}
