/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func withRecall(t *testing.T, fn func(ctx context.Context, user, query string, limit int) []core.MemoryRecord) {
	t.Helper()
	restore := memoryRecallFn
	t.Cleanup(func() { memoryRecallFn = restore })
	memoryRecallFn = fn
}

func TestBuildMemorySectionRenders(t *testing.T) {
	withRecall(t, func(_ context.Context, user, query string, limit int) []core.MemoryRecord {
		assert.Equal(t, "u-1", user)
		return []core.MemoryRecord{
			{Kind: core.MemoryKindPreference, Content: "偏好中文回复"},
			{Kind: core.MemoryKindFact, Content: "负责支付网关"},
		}
	})

	out := buildMemorySection("u-1", "支付网关的负责人")
	require.True(t, strings.Contains(out, "Long-term Memory"))
	assert.Contains(t, out, "[preference] 偏好中文回复")
	assert.Contains(t, out, "[fact] 负责支付网关")
}

func TestBuildMemorySectionEmptyWhenNoMemories(t *testing.T) {
	withRecall(t, func(_ context.Context, _, _ string, _ int) []core.MemoryRecord { return nil })
	assert.Empty(t, buildMemorySection("u-1", "query"), "no memories → no section, no noise")
}

func TestBuildMemorySectionBudget(t *testing.T) {
	withRecall(t, func(_ context.Context, _, _ string, _ int) []core.MemoryRecord {
		var out []core.MemoryRecord
		for i := 0; i < 50; i++ {
			out = append(out, core.MemoryRecord{Kind: core.MemoryKindFact, Content: strings.Repeat("长内容", 20)})
		}
		return out
	})
	out := buildMemorySection("u-1", "query")
	require.NotEmpty(t, out)
	assert.LessOrEqual(t, len(out), memorySectionMaxChars+40, "section respects the budget cap")
}
