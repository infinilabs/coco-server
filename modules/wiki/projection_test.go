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
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

var projectionStoreOnce sync.Once

func projectionSetup(t *testing.T) {
	t.Helper()
	projectionStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-projection.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiKnowledgeBase{}, "wiki-kb-proj2"},
			{core.WikiArticle{}, "wiki-article-proj2"},
			{core.Document{}, "document-proj2"},
			{core.WikiOntologySchema{}, "wiki-schema-proj2"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-projection-test", handler)
	})
	restore := projectionConfigFn
	t.Cleanup(func() { projectionConfigFn = restore })
	projectionConfigFn = func() core.Config {
		return core.Config{ServerInfo: &core.ServerInfo{Endpoint: "http://coco.test"}}
	}
}

func projectionWrite(t *testing.T, obj interface{}) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	require.NoError(t, orm.Create(ctx, obj))
}

func loadProjectionRow(t *testing.T, articleID string) (core.Document, bool) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.Document{})
	doc := core.Document{}
	doc.ID = ProjectionDocID(articleID)
	exists, err := orm.GetV2(ctx, &doc)
	if err != nil {
		// the sqlite backend reports a missing row as an error — same
		// verdict as not-found for this helper's purpose
		return doc, false
	}
	return doc, exists
}

func TestProjectionSyncOnPublishAndUnpublish(t *testing.T) {
	projectionSetup(t)
	kb := &core.WikiKnowledgeBase{}
	kb.ID = "kb-proj2"
	kb.Name = "产品知识库"
	projectionWrite(t, kb)

	article := &core.WikiArticle{}
	article.ID = "art-proj2"
	article.KbID = kb.ID
	article.Title = "发布流程"
	article.Summary = "如何发布"
	article.Content = "正文内容"
	article.Status = core.WikiArticlePublished
	article.PageType = core.WikiPageTypeConcept
	article.Confidence = "high"
	article.Sources = []core.WikiSourceReference{{DocID: "d1"}, {DocID: "d2"}}

	SyncArticleProjection(article)

	row, exists := loadProjectionRow(t, article.ID)
	require.True(t, exists, "published article must project into the document index")
	assert.Equal(t, "wiki_article", row.Type)
	assert.Equal(t, "发布流程", row.Title)
	assert.Equal(t, "wiki_kb_kb-proj2", row.Source.ID, "the KB becomes a built-in datasource reference")
	assert.Equal(t, "产品知识库", row.Source.Name)
	assert.Contains(t, row.URL, "/wiki/article/art-proj2")
	assert.Equal(t, true, row.Metadata["wiki_article"])
	assert.Equal(t, core.DocumentStatusCompleted, row.Status)

	// unpublish: the projection row must go away
	article.Status = core.WikiArticleArchived
	SyncArticleProjection(article)
	_, exists = loadProjectionRow(t, article.ID)
	assert.False(t, exists, "archived article's projection is removed")

	// idempotent remove
	article.Status = core.WikiArticleDraft
	SyncArticleProjection(article)
	_, exists = loadProjectionRow(t, article.ID)
	assert.False(t, exists)
}

func TestProjectionGateSkipsLowConfidence(t *testing.T) {
	projectionSetup(t)
	low := &core.WikiArticle{}
	low.ID = "art-low"
	low.KbID = "kb-proj2"
	low.Title = "低置信页"
	low.Status = core.WikiArticlePublished
	low.Confidence = "low"

	SyncArticleProjection(low)
	_, exists := loadProjectionRow(t, low.ID)
	assert.False(t, exists, "low-confidence pages never project — the noise gate lives at projection time")
}

func TestProjectionBackfillIdempotent(t *testing.T) {
	projectionSetup(t)

	a1 := &core.WikiArticle{}
	a1.ID = "art-bf1"
	a1.KbID = "kb-proj2"
	a1.Title = "存量一"
	a1.Status = core.WikiArticlePublished
	a1.Confidence = "high"
	projectionWrite(t, a1)

	a2 := &core.WikiArticle{}
	a2.ID = "art-bf2"
	a2.KbID = "kb-proj2"
	a2.Title = "存量二"
	a2.Status = core.WikiArticlePublished
	a2.Confidence = "high"
	projectionWrite(t, a2)

	total, projected, skipped, err := backfillPublishedProjections()
	require.NoError(t, err)
	assert.Equal(t, 2, projected, "both legacy published articles project")
	assert.GreaterOrEqual(t, total, 2)

	// second run: every projection exists, nothing is rewritten
	_, projected2, _, err := backfillPublishedProjections()
	require.NoError(t, err)
	assert.Zero(t, projected2, "backfill is idempotent — existing projections are skipped")
	_ = skipped
}
