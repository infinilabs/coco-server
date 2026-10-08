/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package fileproc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func splitStructured(t *testing.T, pages []string, size int) []core.DocumentChunk {
	t.Helper()
	return SplitStructuredPages(pages, StructuredChunkConfig{ChunkSize: size})
}

func TestStructuredChunksBreadcrumbPerSection(t *testing.T) {
	pages := []string{
		"# 年度报告\n第一段正文。",
		"# 年度报告\n## 财务\n营收内容 A。\n营收内容 B。",
		"# 年度报告\n## 财务\n### 营收\n细节内容。",
	}
	chunks := splitStructured(t, pages, 200)

	var crumbs []string
	for _, c := range chunks {
		crumbs = append(crumbs, c.Breadcrumb)
	}
	require.GreaterOrEqual(t, len(chunks), 3)
	assert.Equal(t, "年度报告", chunks[0].Breadcrumb)
	assert.Contains(t, crumbs, "年度报告 > 财务")
	assert.Equal(t, "年度报告 > 财务 > 营收", chunks[len(chunks)-1].Breadcrumb,
		"a deeper heading replaces the shallower tail of the breadcrumb")
}

func TestStructuredChunksNeverSpanHeadings(t *testing.T) {
	pages := []string{"# A\n" + strings.Repeat("甲", 50) + "\n# B\n" + strings.Repeat("乙", 50)}
	chunks := splitStructured(t, pages, 30) // small cap forces multiple chunks per section

	for _, c := range chunks {
		if strings.Contains(c.Text, "甲") {
			assert.NotContains(t, c.Text, "乙", "section A chunks must not carry section B text")
			assert.Equal(t, "A", c.Breadcrumb)
		}
		if strings.Contains(c.Text, "乙") {
			assert.Equal(t, "B", c.Breadcrumb)
		}
	}
}

func TestStructuredChunksHardCapAndOverlap(t *testing.T) {
	text := "# S\n" + strings.Repeat("字", 500)
	chunks := splitStructured(t, []string{text}, 100)
	require.NotEmpty(t, chunks)

	// hard cap: no chunk (including its overlap prefix) exceeds the budget
	for _, c := range chunks {
		assert.LessOrEqual(t, len([]rune(c.Text)), 100, "hard cap must hold")
	}
	// bounded overlap: consecutive chunks in one section overlap — chunk 2
	// begins with the tail of chunk 1 (cut to a line start; this fixture
	// has no newlines, so the whole tail carries)
	require.Greater(t, len(chunks), 1)
	tail := chunks[0].Text[len(chunks[0].Text)-15:]
	assert.True(t, strings.HasPrefix(chunks[1].Text, tail),
		"next chunk should open with the overlap prefix, got %.30s…", chunks[1].Text)
}

func TestStructuredChunksLosslessWithoutOverlap(t *testing.T) {
	body := "# 节\n" + strings.Repeat("行内容\n", 40)
	chunks := SplitStructuredPages([]string{body}, StructuredChunkConfig{ChunkSize: 60, OverlapRunes: 1})

	// reassembly rule: a line-aligned cut consumes the separator newline
	// and carries NO overlap prefix; a mid-line hard cut duplicates exactly
	// one rune (prev's tail becomes next's prefix). Strip it only when the
	// first rune repeats the previous chunk's last rune.
	var rebuilt strings.Builder
	prevTail := rune(0)
	for i, c := range chunks {
		runes := []rune(c.Text)
		if i > 0 && len(runes) > 0 && len(runes) > 0 && runes[0] == prevTail {
			runes = runes[1:]
		}
		if len(runes) > 0 {
			prevTail = runes[len(runes)-1]
		}
		rebuilt.WriteString(string(runes))
		rebuilt.WriteString("\n")
	}
	assert.Equal(t, strings.TrimRight(body, "\n"), strings.TrimRight(rebuilt.String(), "\n"),
		"chunks must reassemble to the section text losslessly")
}

func TestStructuredChunksPageRange(t *testing.T) {
	pages := []string{
		"# 首页\n短内容。",
		strings.Repeat("跨页正文。", 100),
	}
	chunks := splitStructured(t, pages, 80)
	require.NotEmpty(t, chunks)
	assert.Equal(t, 1, chunks[0].Range.Start, "first chunk starts on its section's first page")
	// the continuation chunks of the root section span into page 2
	last := chunks[len(chunks)-1]
	assert.Equal(t, 2, last.Range.End)
}

func TestStructuredChunksUnstructuredFallback(t *testing.T) {
	// no heading markers: one root section, empty breadcrumb, still capped
	pages := []string{strings.Repeat("无结构文本", 200)}
	chunks := splitStructured(t, pages, 100)
	require.Greater(t, len(chunks), 1)
	for _, c := range chunks {
		assert.Empty(t, c.Breadcrumb)
		assert.LessOrEqual(t, len([]rune(c.Text)), 100)
	}
}

func TestStructuredChunksHeadingOnlySection(t *testing.T) {
	// a heading immediately followed by another heading yields no chunk —
	// an "orphan heading" carries no content to retrieve
	chunks := splitStructured(t, []string{"# 孤儿\n# 下一节\n正文。"}, 100)
	require.Len(t, chunks, 1)
	assert.Equal(t, "下一节", chunks[0].Breadcrumb)
	assert.Contains(t, chunks[0].Text, "正文")
}

func TestStructuredChunksAlwaysProgress(t *testing.T) {
	// adversarial config (overlap ≥ cap) must not hang — the loop drops
	// the prefix and forces progress
	chunks := SplitStructuredPages([]string{strings.Repeat("快", 300)}, StructuredChunkConfig{ChunkSize: 10, OverlapRunes: 50})
	assert.NotEmpty(t, chunks)
	for _, c := range chunks {
		assert.LessOrEqual(t, len([]rune(c.Text)), 10)
	}
}
