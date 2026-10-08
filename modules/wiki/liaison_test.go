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

var liaisonStoreOnce sync.Once

func liaisonSetup(t *testing.T) {
	t.Helper()
	liaisonStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-liaison.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiKnowledgeBase{}, "wiki-kb-liaison"},
			{core.WikiArticle{}, "wiki-article-liaison"},
			{core.Document{}, "document-liaison"},
			{core.WikiGovernanceProposal{}, "wiki-proposal-liaison"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-liaison-test", handler)
	})
}

func liaisonWrite(t *testing.T, obj interface{}) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	require.NoError(t, orm.Create(ctx, obj))
}

func liaisonProposals(t *testing.T, pType string) []core.WikiGovernanceProposal {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100).
		Filter(orm.TermQuery("type", pType)))
	require.NoError(t, err)
	hits, _, err := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	require.NoError(t, err)
	return hits
}

func TestLiaisonChangedDocumentFilesRefresh(t *testing.T) {
	liaisonSetup(t)
	kb := &core.WikiKnowledgeBase{}
	kb.ID = "kb-l"
	kb.Name = "联动库"
	liaisonWrite(t, kb)

	doc := &core.Document{}
	doc.ID = "doc-l"
	doc.ContentHash = "old-hash"
	liaisonWrite(t, doc)

	article := &core.WikiArticle{}
	article.ID = "art-cites"
	article.KbID = kb.ID
	article.Title = "引用文档的文章"
	article.Status = core.WikiArticlePublished
	article.Sources = []core.WikiSourceReference{{DocID: "doc-l", Title: "源文档"}}
	liaisonWrite(t, article)

	// content genuinely changed → refresh proposal
	onDocumentProcessed("doc-l", "new-hash")
	refresh := liaisonProposals(t, core.WikiGovernanceArticleRefresh)
	require.Len(t, refresh, 1)
	assert.Equal(t, "art-cites", refresh[0].ArticleID)
	assert.Equal(t, "document-change", refresh[0].Evidence["cascade"])
	assert.Equal(t, "new-hash", refresh[0].Evidence["content_hash"])

	// dispatcher resync with the SAME fingerprint → no second proposal
	onDocumentProcessed("doc-l", "new-hash")
	refresh2 := liaisonProposals(t, core.WikiGovernanceArticleRefresh)
	assert.Len(t, refresh2, 1, "open-twin folding keeps one open proposal per article")
}

func TestLiaisonUnchangedResyncStaysSilent(t *testing.T) {
	liaisonSetup(t)
	// a dedicated doc+article pair so the other tests' proposals can't
	// collide with this guard assertion (shared-store isolation)
	quiet := &core.Document{}
	quiet.ID = "doc-quiet"
	quiet.ContentHash = "same-hash"
	liaisonWrite(t, quiet)

	watcher := &core.WikiArticle{}
	watcher.ID = "art-quiet"
	watcher.KbID = "kb-l"
	watcher.Title = "安静的文章"
	watcher.Status = core.WikiArticlePublished
	watcher.Sources = []core.WikiSourceReference{{DocID: "doc-quiet"}}
	liaisonWrite(t, watcher)

	// stored hash == fresh hash: pure dispatcher resync, zero proposals
	onDocumentProcessed("doc-quiet", "same-hash")
	for _, p := range liaisonProposals(t, core.WikiGovernanceArticleRefresh) {
		if p.Evidence["doc_id"] == "doc-quiet" {
			t.Fatalf("unchanged resync must not file: %+v", p)
		}
	}
}

func TestLiaisonDeletionMarksStale(t *testing.T) {
	liaisonSetup(t)
	onDocumentDeleted("doc-l")
	stale := liaisonProposals(t, core.WikiGovernanceStale)
	require.Len(t, stale, 1)
	assert.Equal(t, "art-cites", stale[0].ArticleID)
	assert.Equal(t, "document-deleted", stale[0].Evidence["cascade"])

	// the citing article itself is untouched — marking only, never removal
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})
	a := core.WikiArticle{}
	a.ID = "art-cites"
	exists, err := orm.GetV2(ctx, &a)
	require.NoError(t, err)
	assert.True(t, exists, "evidence loss marks, never deletes")
}
