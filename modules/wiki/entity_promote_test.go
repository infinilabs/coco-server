/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

var promoteStoreOnce sync.Once

func promoteSetup(t *testing.T) {
	t.Helper()
	promoteStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-entity-promote.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiEntity{}, "wiki-entity-promote"},
			{core.WikiArticle{}, "wiki-article-promote"},
			{core.WikiKnowledgeBase{}, "wiki-kb-promote"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-entity-promote-test", handler)
	})
}

func TestPromoteTopicEntities(t *testing.T) {
	promoteSetup(t)
	kb := &core.WikiKnowledgeBase{}
	kb.ID = "kb-promote"
	propagationWrite(t, kb)

	// pre-existing entity: "美心集团" already in the pool — promotion must
	// absorb rather than stack a twin
	existing := &core.WikiEntity{}
	existing.ID = "ent-promote-existing"
	existing.KbID = kb.ID
	existing.Name = "美心集团"
	existing.Type = "organization"
	existing.Status = core.WikiEntityReviewed
	existing.Aliases = []string{"Maxim's"}
	propagationWrite(t, existing)

	ctx := orm.NewContext()
	ctx.DirectAccess()

	types := map[string]string{"奇华月饼": "product", "美心集团": "organization"}
	descs := entityDescriptionsFromStatements(
		[]string{"美心集团", "奇华月饼", "陈锡培"},
		[]compileStatement{
			{DocID: "d1", Claim: "奇华月饼是美心集团旗下的港式月饼品牌，主打低端市场"},
			{DocID: "d2", Claim: "与美心集团无关的其他主张"},
		}, "奇华月饼专题摘要")
	n := promoteTopicEntities(ctx, kb.ID, []string{"美心集团", "奇华月饼", "  ", "陈锡培"}, types, descs)
	// 美心集团 exists (skipped), 奇华月饼 typed product, 陈锡培 untyped → concept
	assert.Equal(t, 2, n)

	// typed entity landed with the LLM type
	typed := findKBEntityByNameOrAlias(ctx, "奇华月饼", kb.ID)
	require.NotNil(t, typed, "typed entity must be promoted")
	assert.Equal(t, "product", typed.Type)
	assert.Equal(t, core.WikiEntityProposed, typed.Status)
	// grounded description: the first claim naming the entity, not the
	// later unrelated one
	assert.Equal(t, "奇华月饼是美心集团旗下的港式月饼品牌，主打低端市场", typed.Description)

	// untyped entity fell back to concept (person typing absent in this
	// test because the map lacks the name — fallback covers LLM misses)
	fallback := findKBEntityByNameOrAlias(ctx, "陈锡培", kb.ID)
	require.NotNil(t, fallback, "untyped entity must still be promoted")
	assert.Equal(t, "concept", fallback.Type)

	// duplicate guard: re-running the same promotion is a no-op
	n2 := promoteTopicEntities(ctx, kb.ID, []string{"美心集团", "奇华月饼", "陈锡培"}, types, descs)
	assert.Equal(t, 0, n2, "second run must absorb every name")

	// idempotent pool: exactly one row per name
	eres, err := orm.SearchV2(ctx, orm.NewQuery().Filter(orm.TermQuery("kb_id", kb.ID)).Size(10))
	require.NoError(t, err)
	pool, _, derr := elastic.DecodeHits[core.WikiEntity](eres)
	require.NoError(t, derr)
	assert.Len(t, pool, 3, "pool must stay one-row-per-concept")
}
