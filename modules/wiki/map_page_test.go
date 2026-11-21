/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/modules/sqlite"

	"infini.sh/coco/core"
)

// mapStoreOnce gives the company-map tests their own article store: model
// routing across the shared test stores binds by latest registration, so
// files that run after graph_test would otherwise write into its store.
var mapStoreOnce sync.Once

func mapSetup(t *testing.T) APIHandler {
	t.Helper()
	mapStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-map.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.WikiArticle{}, "wiki-article-map"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-wiki-map-test", handler)
	})
	return APIHandler{}
}

func companyMapCall(t *testing.T, h APIHandler, roles []string, query ...string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	uri := "/wiki/company-map"
	if len(query) > 0 {
		uri += query[0]
	}
	req := httptest.NewRequest(http.MethodGet, uri, nil)
	user := &security.UserSessionInfo{UserID: "user-1", Roles: roles}
	req = req.WithContext(security.AddUserToContext(req.Context(), user))
	w := httptest.NewRecorder()
	h.getCompanyMap(w, req, httprouter.Params{})
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func seedMapArticle(t *testing.T, id, title, status string) {
	t.Helper()
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})
	article := core.WikiArticle{
		KbID:     "kb-map",
		Title:    title,
		Status:   status,
		PageType: core.WikiPageTypeMap,
		Content:  "# Company Map\n\n## What we do\n...",
	}
	article.SetID(id)
	require.NoError(t, orm.Create(ctx, &article))
}

func TestCompanyMapRequiresLogin(t *testing.T) {
	h := mapSetup(t)
	req := httptest.NewRequest(http.MethodGet, "/wiki/company-map", nil)
	w := httptest.NewRecorder()
	h.getCompanyMap(w, req, httprouter.Params{})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCompanyMapReturnsPublishedMap(t *testing.T) {
	h := mapSetup(t)

	restore := companyMapConfigFn
	companyMapConfigFn = func() core.Config {
		return core.Config{ServerInfo: &core.ServerInfo{Endpoint: "http://coco.test"}}
	}
	t.Cleanup(func() { companyMapConfigFn = restore })

	seedMapArticle(t, "map-draft", "Draft Map", core.WikiArticleDraft)
	seedMapArticle(t, "map-live", "The Company Map", core.WikiArticlePublished)

	w, out := companyMapCall(t, h, []string{security.RoleAdmin})
	require.Equal(t, http.StatusOK, w.Code)

	// the published map wins over the draft (WriteOKJSON writes the raw value)
	assert.Equal(t, true, out["found"])
	mp := out["map"].(map[string]interface{})
	assert.Equal(t, "The Company Map", mp["title"])
	assert.Contains(t, mp["content"], "Company Map")
	assert.Contains(t, mp["url"], "/#/wiki/article/map-live")

	// non-admin without the article-search permission is rejected
	w, _ = companyMapCall(t, h, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCompanyMapNotFound(t *testing.T) {
	h := mapSetup(t)
	// an empty kb keeps this test independent of maps seeded by other tests
	w, out := companyMapCall(t, h, []string{security.RoleAdmin}, "?kb=kb-without-map")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, false, out["found"])
	assert.Contains(t, out["note"], "page_type=map")
}

func TestCompanyMapTemplateSeeded(t *testing.T) {
	mapSetup(t)
	cfg := articleConfig()

	// empty map page starts from the navigation skeleton
	mapArticle := &core.WikiArticle{KbID: "kb", Title: "Map", PageType: core.WikiPageTypeMap}
	require.NoError(t, cfg.PrepareCreate(mapArticle))
	assert.Contains(t, mapArticle.Content, "Source priority")

	// an author's own content is never clobbered
	custom := &core.WikiArticle{KbID: "kb", Title: "M", PageType: core.WikiPageTypeMap, Content: "my own map"}
	require.NoError(t, cfg.PrepareCreate(custom))
	assert.Equal(t, "my own map", custom.Content)

	// non-map pages stay untouched
	concept := &core.WikiArticle{KbID: "kb", Title: "C"}
	require.NoError(t, cfg.PrepareCreate(concept))
	assert.Empty(t, concept.Content)
}
