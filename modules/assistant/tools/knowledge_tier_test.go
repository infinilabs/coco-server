/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package tools

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func init() { maskContentFn = func(s string) string { return s } }

func rawDoc(id string) core.Document {
	d := core.Document{}
	d.ID = id
	d.Title = id
	d.Source = core.DataSourceReference{ID: "ds-hr", Name: "hr"}
	return d
}

func curatedDoc(id, kind string) core.Document {
	d := core.Document{}
	d.ID = id
	d.Title = id
	if kind == "entity" {
		d.Type = "entity"
		d.Source = core.DataSourceReference{ID: "wiki", Name: "Wiki"}
	} else {
		d.Type = "wiki_article"
		d.Source = core.DataSourceReference{ID: "wiki_kb_kb1", Name: "产品库"}
	}
	return d
}

func TestPartitionCuratedFirstStableWithinTier(t *testing.T) {
	docs := []core.Document{
		rawDoc("r1"),
		curatedDoc("w1", "article"),
		rawDoc("r2"),
		curatedDoc("e1", "entity"),
		curatedDoc("w2", "article"),
	}
	curated, raw := partitionReferencesByKnowledgeTier(docs)
	require.Len(t, curated, 3)
	require.Len(t, raw, 2)
	assert.Equal(t, "w1", curated[0].ID, "tier order preserved, curated first")
	assert.Equal(t, "e1", curated[1].ID)
	assert.Equal(t, "w2", curated[2].ID)
	assert.Equal(t, "r1", raw[0].ID)
	assert.Equal(t, "r2", raw[1].ID)
}

func TestFormatReferencesLayerMarkers(t *testing.T) {
	docs := []core.Document{rawDoc("r1"), curatedDoc("w1", "article")}
	out := FormatDocumentForReplyReferences(docs)

	require.True(t, strings.Contains(out, "authoritative tier"), "the prompt states which tier wins")
	curatedPos := strings.Index(out, "Layer: curated")
	rawPos := strings.Index(out, "Layer: source")
	require.GreaterOrEqual(t, curatedPos, 0)
	require.GreaterOrEqual(t, rawPos, 0)
	assert.Less(t, curatedPos, rawPos, "curated references precede raw sources in the prompt")
}

func TestFormatReferencesNoCuratedNoNotice(t *testing.T) {
	out := FormatDocumentForReplyReferences([]core.Document{rawDoc("only-raw")})
	assert.NotContains(t, out, "authoritative tier", "no curated docs → no tier notice")
	assert.Contains(t, out, "Layer: source")
}
