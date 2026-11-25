/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/sqlite"

	"infini.sh/coco/core"
)

// rebuildStoreOnce gives these tests their own sqlite store: model→index
// mappings are global and last-registration-wins, so borrowing another
// file's setup breaks once the full suite shuffles registration order.
var rebuildStoreOnce sync.Once

func rebuildSetup(t *testing.T) {
	t.Helper()
	rebuildStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-rebuild.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiArticle{}, "wiki-article-rebuild"},
			{core.WikiVersion{}, "wiki-version-rebuild"},
			{core.WikiContribution{}, "wiki-contribution-rebuild"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-rebuild-test", handler)
	})
}

// applyRebuildProposal used to overwrite the live article body in place
// with no snapshot of what it replaced — an auto-approve sweep swapped 28
// published bodies for thin claims lists and the content was gone. These
// tests pin the contract: pending version content wins, the live body is
// snapshotted before every overwrite, a published page folding a reviewed
// pending version stays published (it must not vanish from the TOC), and
// the synthetic fallback rebuilds land as draft.
func TestApplyRebuildProposalFoldsPendingVersion(t *testing.T) {
	rebuildSetup(t)
	ctx := orm.NewContext()
	ctx.DirectAccess()
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiArticle{})
	orm.WithModel(ctx, &core.WikiVersion{})
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})

	live := &core.WikiArticle{KbID: "kb-fold", Title: "Fold Page", Status: core.WikiArticlePublished, Content: "PUBLISHED BODY"}
	govCreate(t, &core.WikiArticle{}, "wiki-article-gov", live)

	// a pending auto-updated version produced by freshness
	govCreate(t, &core.WikiVersion{}, "wiki-version-gov", &core.WikiVersion{
		ArticleID: live.ID, Version: 2, ChangeType: core.WikiChangeAutoUpdated, Content: "REGENERATED BODY",
	})

	p := &core.WikiGovernanceProposal{KbID: "kb-fold", ArticleID: live.ID,
		Type: core.WikiGovernanceStale, Status: core.WikiGovernanceOpen,
		Evidence: util.MapStr{"pending_version": 2}}
	outcome, err := applyRebuildProposal(ctx, p)
	require.NoError(t, err)
	assert.Contains(t, outcome, "v2")

	// applying the proposal IS the review: the live row carries the
	// regenerated body and the published status is preserved — dropping it
	// to draft would hide the page from the TOC and read as content loss
	folded := core.WikiArticle{}
	folded.ID = live.ID
	ok, err := orm.GetV2(ctx, &folded)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "REGENERATED BODY", folded.Content)
	assert.Equal(t, core.WikiArticlePublished, folded.Status)

	// the pre-apply published body must be recoverable from version history
	snap, err := latestVersions(t, live.ID, 2)
	require.NoError(t, err)
	bodies := map[string]bool{}
	for _, v := range snap {
		bodies[v.Content] = true
	}
	assert.True(t, bodies["PUBLISHED BODY"], "pre-apply live body must be in version history")
	assert.True(t, bodies["REGENERATED BODY"], "applied body recorded by pipeline snapshot")
}

func TestApplyRebuildProposalProvenanceSnapshotsLiveBody(t *testing.T) {
	rebuildSetup(t)
	ctx := orm.NewContext()
	ctx.DirectAccess()
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiArticle{})
	orm.WithModel(ctx, &core.WikiVersion{})

	// no pending version in evidence: falls back to the deterministic
	// provenance rebuild; the published body must still be snapshotted
	live := &core.WikiArticle{KbID: "kb-prov", Title: "Provenance Page", Status: core.WikiArticlePublished,
		Content: "LONG PUBLISHED BODY",
		Sources: []core.WikiSourceReference{{DocID: "d1", Title: "Doc One", Excerpt: "claim one"}}}
	govCreate(t, &core.WikiArticle{}, "wiki-article-gov", live)

	p := &core.WikiGovernanceProposal{KbID: "kb-prov", ArticleID: live.ID,
		Type: core.WikiGovernanceStale, Status: core.WikiGovernanceOpen, Evidence: util.MapStr{}}
	_, err := applyRebuildProposal(ctx, p)
	require.NoError(t, err)

	folded := core.WikiArticle{}
	folded.ID = live.ID
	ok, err := orm.GetV2(ctx, &folded)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Contains(t, folded.Content, "claim one")
	assert.Equal(t, core.WikiArticleDraft, folded.Status)

	snap, err := latestVersions(t, live.ID, 3)
	require.NoError(t, err)
	bodies := map[string]bool{}
	for _, v := range snap {
		bodies[v.Content] = true
	}
	assert.True(t, bodies["LONG PUBLISHED BODY"], "pre-apply live body must be in version history")
}

// latestVersions returns the newest n versions of an article, newest first.
func latestVersions(t *testing.T, articleID string, n int) ([]core.WikiVersion, error) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.DirectReadAccess()
	orm.WithModel(ctx, &core.WikiVersion{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(n).
		Filter(orm.TermQuery("article_id", articleID)).
		SortBy(orm.Sort{Field: "version", SortType: orm.DESC}))
	if err != nil {
		return nil, err
	}
	versions, _, err := elastic.DecodeHits[core.WikiVersion](res)
	return versions, err
}
