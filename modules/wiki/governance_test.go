/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/modules/sqlite"
)

var governanceStoreOnce sync.Once

// governanceSetup points the global ORM at a sqlite store carrying every
// schema the sweep touches; the sweep itself reads through the orm package.
func governanceSetup(t *testing.T) APIHandler {
	t.Helper()
	governanceStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-governance.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiKnowledgeBase{}, "wiki-kb-gov"},
			{core.WikiArticle{}, "wiki-article-gov"},
			{core.WikiToc{}, "wiki-toc-gov"},
			{core.WikiVersion{}, "wiki-version-gov"},
			{core.WikiGovernanceProposal{}, "wiki-governance-test"},
			{core.WikiNotification{}, "wiki-notification-gov"},
			{core.WikiLike{}, "wiki-like-test"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-governance-test", handler)
	})
	return APIHandler{}
}

func govCreate(t *testing.T, model interface{}, index string, obj interface{}) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, model)
	require.NoError(t, orm.Create(ctx, obj))
}

func govProposals(t *testing.T) []core.WikiGovernanceProposal {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100))
	require.NoError(t, err)
	hits, _, err := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	require.NoError(t, err)
	return hits
}

func TestDuplicateCandidates(t *testing.T) {
	articles := []core.WikiArticle{
		{Title: "VPN 排查指南"},
		{Title: "VPN 排查手册"},             // shares VPN+排查 -> candidate
		{Title: "邮箱配额说明"},               // unrelated
		{Title: "Coffee Brewing Guide"}, // latin tokens
		{Title: "Coffee Brewing Basics"},
	}
	pairs := duplicateCandidates(articles)
	require.Len(t, pairs, 2)
	assert.Equal(t, "VPN 排查指南", pairs[0][0].Title)
	assert.Equal(t, "Coffee Brewing Guide", pairs[1][0].Title)
}

func TestGovernanceSweepHeuristics(t *testing.T) {
	governanceSetup(t)

	kb := &core.WikiKnowledgeBase{}
	kb.Name = "Gov KB"
	govCreate(t, &core.WikiKnowledgeBase{}, "wiki-kb-gov", kb)

	// published + low confidence -> low_quality
	weak := &core.WikiArticle{KbID: kb.ID, Title: "Weak Page", Status: core.WikiArticlePublished, Confidence: "low"}
	govCreate(t, &core.WikiArticle{}, "wiki-article-gov", weak)

	// reviewed article with a pending auto-updated version -> stale
	stale := &core.WikiArticle{KbID: kb.ID, Title: "Stale Page", Status: core.WikiArticleReviewed}
	govCreate(t, &core.WikiArticle{}, "wiki-article-gov", stale)
	govCreate(t, &core.WikiVersion{}, "wiki-version-gov", &core.WikiVersion{
		ArticleID: stale.ID, Version: 2, ChangeType: core.WikiChangeAutoUpdated, Content: "regenerated",
	})

	// healthy published page, kept in the TOC -> no proposal
	healthy := &core.WikiArticle{KbID: kb.ID, Title: "Healthy Page", Status: core.WikiArticlePublished,
		Confidence: "high", Sources: []core.WikiSourceReference{{DocID: "d1"}}}
	govCreate(t, &core.WikiArticle{}, "wiki-article-gov", healthy)

	// tree contains only the healthy article; weak/stale are orphans too
	govCreate(t, &core.WikiToc{}, "wiki-toc-gov", &core.WikiToc{KbID: kb.ID, Nodes: []core.WikiTocNode{
		{ID: "toc-" + healthy.ID, Title: healthy.Title, Type: "article", ArticleID: healthy.ID},
	}})

	// heuristic-only sweep (nil LLM)
	stats := governanceSweep(t.Context(), nil)
	assert.Positive(t, stats.filed)

	byArticle := map[string][]string{}
	for _, p := range govProposals(t) {
		byArticle[p.ArticleID] = append(byArticle[p.ArticleID], p.Type)
	}

	assert.Contains(t, byArticle[weak.ID], core.WikiGovernanceLowQuality)
	assert.Contains(t, byArticle[weak.ID], core.WikiGovernanceOrphan)
	assert.Contains(t, byArticle[stale.ID], core.WikiGovernanceStale)
	assert.Contains(t, byArticle[stale.ID], core.WikiGovernanceOrphan)
	assert.NotContains(t, byArticle, healthy.ID)

	// idempotent: a second sweep files nothing new
	before := len(govProposals(t))
	stats = governanceSweep(t.Context(), nil)
	after := len(govProposals(t))
	assert.Equal(t, before, after)
	assert.Zero(t, stats.filed)
}

func TestGovernanceStatusTransitions(t *testing.T) {
	h := governanceSetup(t)

	proposal := &core.WikiGovernanceProposal{KbID: "kb1", ArticleID: "a1", ArticleTitle: "T",
		Type: core.WikiGovernanceStale, Status: core.WikiGovernanceOpen}
	govCreate(t, &core.WikiGovernanceProposal{}, "wiki-governance-test", proposal)

	callStatus := func(body string) int {
		req := httptest.NewRequest(http.MethodPut, "/wiki/governance/"+proposal.ID+"/status", bytes.NewReader([]byte(body)))
		req = req.WithContext(security.AddUserToContext(req.Context(), &security.UserSessionInfo{UserID: "tester"}))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.updateGovernanceStatus(w, req, httprouter.Params{{Key: "id", Value: proposal.ID}})
		return w.Code
	}

	// invalid target status
	assert.Equal(t, http.StatusBadRequest, callStatus(`{"status":"bogus"}`))
	// resolved is terminal: open -> resolved ok, resolved -> dismissed rejected
	assert.Equal(t, http.StatusOK, callStatus(`{"status":"resolved"}`))
	assert.Equal(t, http.StatusBadRequest, callStatus(`{"status":"dismissed"}`))

	for _, p := range govProposals(t) {
		if p.ArticleID == "a1" {
			assert.Equal(t, core.WikiGovernanceResolved, p.Status)
		}
	}
}
