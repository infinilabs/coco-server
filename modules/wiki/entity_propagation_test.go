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

var propagationStoreOnce sync.Once

func propagationSetup(t *testing.T) {
	t.Helper()
	propagationStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-propagation.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiEntity{}, "wiki-entity-prop"},
			{core.WikiArticle{}, "wiki-article-prop"},
			{core.WikiKnowledgeBase{}, "wiki-kb-prop"},
			{core.WikiGovernanceProposal{}, "wiki-proposal-prop"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-propagation-test", handler)
	})
}

func propagationWrite(t *testing.T, obj interface{}) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	require.NoError(t, orm.Create(ctx, obj))
}

func TestEntityEditPropagationFilesRefreshProposals(t *testing.T) {
	propagationSetup(t)
	kb := &core.WikiKnowledgeBase{}
	kb.ID = "kb-prop"
	propagationWrite(t, kb)

	entity := &core.WikiEntity{}
	entity.ID = "ent-prop"
	entity.Name = "支付网关"
	entity.Type = "product"
	entity.Status = core.WikiEntityReviewed
	entity.ArticleID = "art-own"
	propagationWrite(t, entity)

	own := &core.WikiArticle{}
	own.ID = "art-own"
	own.KbID = kb.ID
	own.Title = "支付网关(实体页)"
	own.Status = core.WikiArticlePublished
	propagationWrite(t, own)

	linker := &core.WikiArticle{}
	linker.ID = "art-link"
	linker.KbID = kb.ID
	linker.Title = "引用它的文章"
	linker.Status = core.WikiArticlePublished
	linker.LinkedPages = []core.WikiLinkedPage{{EntityID: "ent-prop", Name: "支付网关"}}
	propagationWrite(t, linker)

	// stage 1: prepare records the rename while the store holds the old row
	renamed := *entity
	renamed.Name = "Payment Gateway"
	recordEntityEditForPropagation(&renamed)

	// stage 2: post-save files one refresh proposal per affected article
	propagateEntityEditAfterSave(&renamed)

	proposals := listProposalsForTest(t)
	var refresh []core.WikiGovernanceProposal
	for _, p := range proposals {
		if p.Type == core.WikiGovernanceArticleRefresh {
			refresh = append(refresh, p)
		}
	}
	require.Len(t, refresh, 2, "own article + backlink article each get a refresh proposal")
	byArticle := map[string]core.WikiGovernanceProposal{}
	for _, p := range refresh {
		byArticle[p.ArticleID] = p
	}
	require.Contains(t, byArticle, "art-own")
	require.Contains(t, byArticle, "art-link")
	assert.Equal(t, "entity-edit", byArticle["art-own"].Evidence["cascade"])

	// a second edit of the same entity folds into the open twin — one open
	// proposal per article, not a growing pile
	recordEntityEditForPropagation(&renamed)
	aliased := renamed
	aliased.Aliases = []string{"PGW"}
	recordEntityEditForPropagation(&aliased)
	propagateEntityEditAfterSave(&aliased)
	refresh2 := 0
	for _, p := range listProposalsForTest(t) {
		if p.Type == core.WikiGovernanceArticleRefresh && p.Status == core.WikiGovernanceOpen {
			refresh2++
		}
	}
	assert.Equal(t, 2, refresh2, "open twins fold, no duplicate open proposals")
}

func TestEntityEditPropagationSkipsTrivialEdits(t *testing.T) {
	propagationSetup(t)
	entity := &core.WikiEntity{}
	entity.ID = "ent-trivial"
	entity.Name = "未变实体"
	entity.Type = "product"
	propagationWrite(t, entity)

	same := *entity
	recordEntityEditForPropagation(&same)
	_, pending := pendingEntityEdits.Load(entity.ID)
	assert.False(t, pending, "a facet-identical edit records nothing")

	// PostUpdate without a pending entry is a no-op — scoped to THIS
	// entity: the other propagation test's proposals share the store
	propagateEntityEditAfterSave(&same)
	for _, p := range listProposalsForTest(t) {
		if ev, _ := p.Evidence["entity_id"].(string); ev == entity.ID {
			t.Fatalf("trivial edit must not file a proposal, got %+v", p)
		}
	}
}

func listProposalsForTest(t *testing.T) []core.WikiGovernanceProposal {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100).
		Filter(orm.TermQuery("type", core.WikiGovernanceArticleRefresh)))
	require.NoError(t, err)
	hits, _, err := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	require.NoError(t, err)
	return hits
}
