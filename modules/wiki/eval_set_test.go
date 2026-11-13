/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

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

// Own store, same discipline as skill_share_test: the global orm handler
// is last-registration-wins and this file must not shift it.
var evalStoreOnce sync.Once

func setupEvalStore(t *testing.T) {
	t.Helper()
	evalStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-eval.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiArticle{}, "wiki-article-eval"},
			{core.WikiEvalCase{}, "wiki-evalcase-eval"},
			{core.WikiEvalRun{}, "wiki-evalrun-eval"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-eval", handler)
	})
}

func TestScoreWikiEvalRun(t *testing.T) {
	total, hits, mrr, avg := scoreWikiEvalRun(nil)
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, hits)
	assert.InDelta(t, 0.0, mrr, 1e-9)
	assert.EqualValues(t, 0, avg)

	results := []core.WikiEvalCaseResult{
		{HitRank: 3, TookMS: 100}, // inside top-4
		{HitRank: 7, TookMS: 300}, // found but outside top-4
		{HitRank: 0, TookMS: 50},  // miss
		{Error: "boom"},           // errored case scores as miss
	}
	total, hits, mrr, avg = scoreWikiEvalRun(results)
	assert.Equal(t, 4, total)
	assert.Equal(t, 1, hits)
	assert.InDelta(t, (1.0/3+1.0/7)/4, mrr, 1e-9)
	assert.EqualValues(t, 112, avg) // (100+300+50+0)/4
}

func TestNormalizeEvalHelpers(t *testing.T) {
	assert.Equal(t, "foo bar", normalizeEvalQuery("  Foo   BAR "))
	assert.Equal(t, "", normalizeEvalQuery("   "))
	assert.Equal(t, []string{"a", "b"}, normalizeEvalIDList([]string{" a ", "", "b", "a"}))
	assert.Equal(t, []string{"t1", "t2"}, normalizeEvalTitleList([]string{" t1", "t2", "t2", ""}, 3))
}

func TestRunWikiEvalReplayScopedAndGated(t *testing.T) {
	setupEvalStore(t)

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	kbA := "kb-eval-a"
	kbB := "kb-eval-b"

	// KB A: one published page the golden query must find, one draft the
	// gate must hide, one published distractor. sqlite compiles match to
	// exact equality (no analyzer), so the title equals the query — this
	// test pins the replay plumbing (scope/status gate/miss handling);
	// text relevance is ES behavior
	pay := core.WikiArticle{KbID: kbA, Title: "payment gateway", Status: core.WikiArticlePublished}
	require.NoError(t, orm.Create(ctx, &pay))
	require.NoError(t, orm.Create(ctx, &core.WikiArticle{
		KbID: kbA, Title: "payment gateway draft", Status: core.WikiArticleDraft,
	}))
	require.NoError(t, orm.Create(ctx, &core.WikiArticle{
		KbID: kbA, Title: "unrelated policy", Status: core.WikiArticlePublished,
	}))
	// KB B page must never surface for KB A's replay
	other := core.WikiArticle{KbID: kbB, Title: "payment gateway", Status: core.WikiArticlePublished}
	require.NoError(t, orm.Create(ctx, &other))

	hit := core.WikiEvalCase{KbID: kbA, Query: "payment gateway", ExpectedIDs: []string{pay.ID}}
	require.NoError(t, orm.Create(ctx, &hit))
	miss := core.WikiEvalCase{KbID: kbA, Query: "zzqq nothing matches", ExpectedIDs: []string{pay.ID}}
	require.NoError(t, orm.Create(ctx, &miss))

	run, err := RunWikiEvalReplay(ctx, kbA, "manual")
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, 2, run.TotalCases)
	assert.Equal(t, "manual", run.Trigger)
	// the hit case ranks the published page; the miss case records 0
	// honestly instead of erroring
	byQuery := map[string]core.WikiEvalCaseResult{}
	for _, c := range run.Cases {
		byQuery[c.Query] = c
	}
	assert.GreaterOrEqual(t, byQuery["payment gateway"].HitRank, 1)
	assert.NotContains(t, byQuery["payment gateway"].TopIDs, other.ID)
	assert.Equal(t, 0, byQuery["zzqq nothing matches"].HitRank)
	assert.Equal(t, 1, run.Top4Hits)
	assert.InDelta(t, 0.5, run.Top4Rate, 1e-9)

	// the run persisted
	runctx := orm.NewContext()
	runctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(runctx, &core.WikiEvalRun{})
	stored := &core.WikiEvalRun{}
	stored.SetID(run.ID)
	exists, err := orm.GetV2(runctx, stored)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, kbA, stored.KbID)

	// KB B has no cases — the replay says so instead of scoring nothing
	_, err = RunWikiEvalReplay(ctx, kbB, "manual")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestWikiEvalCaseByQueryKBScoped(t *testing.T) {
	setupEvalStore(t)

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	kbA := "kb-eval-q-a"
	kbB := "kb-eval-q-b"
	// same normalized query in two KBs are two rows — the upsert key is
	// (kb, query), not query alone
	require.NoError(t, orm.Create(ctx, &core.WikiEvalCase{KbID: kbA, Query: "same query", ExpectedIDs: []string{"x"}}))
	require.NoError(t, orm.Create(ctx, &core.WikiEvalCase{KbID: kbB, Query: "same query", ExpectedIDs: []string{"y"}}))

	a := wikiEvalCaseByQuery(ctx, kbA, "same query")
	require.NotNil(t, a)
	assert.Equal(t, kbA, a.KbID)
	b := wikiEvalCaseByQuery(ctx, kbB, "same query")
	require.NotNil(t, b)
	assert.NotEqual(t, a.ID, b.ID)
	assert.Nil(t, wikiEvalCaseByQuery(ctx, "kb-eval-q-c", "same query"))
}
