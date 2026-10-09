/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func TestChunkRowIDDeterministic(t *testing.T) {
	assert.Equal(t, "doc1_mom_000000", chunkRowID("doc1", "mom", 0))
	assert.Equal(t, "doc1_c_000042", chunkRowID("doc1", "c", 42))
}

func TestChunkTypeOfTableDetection(t *testing.T) {
	assert.Equal(t, core.ChunkTypeTable, chunkTypeOf(core.DocumentChunk{Text: "| 指标 | 值 |\n| a | b |"}))
	assert.Equal(t, core.ChunkTypeText, chunkTypeOf(core.DocumentChunk{Text: "普通文本"}))
}

func TestLocatorOf(t *testing.T) {
	assert.Nil(t, locatorOf(core.DocumentChunk{}))
	loc := locatorOf(core.DocumentChunk{Range: core.ChunkRange{Start: 2, End: 3}})
	require.NotNil(t, loc)
	pages := loc["pages"].(map[string]interface{})
	assert.Equal(t, 2, pages["start"])
	assert.Equal(t, 3, pages["end"])
}

func TestChunkToDocumentHit(t *testing.T) {
	c := &core.KnowledgeChunk{
		DocID:      "d1",
		ChunkType:  core.ChunkTypeText,
		Breadcrumb: "手册 > 安装",
		Text:       strings.Repeat("内容", 300), // 600 runes → quote caps at 300
		MomID:      "d1_mom_000000",
		Seq:        3,
		ModelID:    "mock/embed",
	}
	c.Source = core.DataSourceReference{ID: "ds", Name: "DS"}

	hit := chunkToDocumentHit(c, 7)
	assert.Equal(t, "d1", hit.ID)
	assert.Equal(t, "knowledge_chunk", hit.Index)
	assert.Equal(t, "chunk", hit.Source.Type)
	assert.Equal(t, 300, len([]rune(hit.Source.Metadata["quote"].(string))), "quote caps at 300")
	assert.Equal(t, "d1_mom_000000", hit.Source.Metadata["mom_id"])
	assert.Equal(t, "mock/embed", hit.Source.Metadata["chunk_model"])
}
