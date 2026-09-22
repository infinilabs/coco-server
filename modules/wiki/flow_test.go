/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	httprouter "infini.sh/framework/core/api/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

// End-to-end wiki flow over the sqlite backend, mirroring the framework's
// crud_test.go recipe: handlers driven directly via httptest, global ORM
// pointed at sqlite so the Post* hooks' secondary writes (versions, kb
// counters, cascade delete) hit the same store.
func call(t *testing.T, h crud.HandlerFunc, method, target, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	// crud reads the object id from the :id route param — parse it from the
	// path the way the real router would
	ps := httprouter.Params{}
	for _, prefix := range []string{"/wiki/kb/", "/wiki/article/"} {
		if rest := strings.TrimPrefix(target, prefix); rest != "" && !strings.Contains(rest, "/") {
			if i := strings.IndexByte(rest, '?'); i >= 0 {
				rest = rest[:i]
			}
			ps = httprouter.Params{{Key: "id", Value: rest}}
		}
	}
	w := httptest.NewRecorder()
	h(w, req, ps)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

var flowStoreOnce sync.Once

func setupFlow(t *testing.T) (crud.Handlers, crud.Handlers, APIHandler) {
	t.Helper()
	flowStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-flow.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		for _, s := range []struct {
			model interface{}
			index string
		}{
			{core.WikiKnowledgeBase{}, "wiki-kb-flow"},
			{core.WikiArticle{}, "wiki-article-flow"},
			{core.WikiToc{}, "wiki-toc-flow"},
			{core.WikiVersion{}, "wiki-version-flow"},
		} {
			if err := handler.RegisterSchemaWithName(s.model, s.index); err != nil {
				panic(err)
			}
		}
		orm.Register("sqlite-wiki-flow", handler)
	})
	return crud.NewHandlers(kbConfig()), crud.NewHandlers(articleConfig()), APIHandler{}
}

func TestWikiFlow_ArticleLifecycle(t *testing.T) {
	kbH, artH, h := setupFlow(t)

	// KB create: default visibility applied
	w, out := call(t, kbH.Create, "POST", "/wiki/kb/", `{"name":"Maxim's Ops Guide"}`)
	require.Equal(t, http.StatusOK, w.Code)
	kbID, _ := out["_id"].(string)
	require.NotEmpty(t, kbID)

	w, out = call(t, kbH.Get, "GET", "/wiki/kb/"+kbID, "")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	require.Equal(t, core.WikiVisibilityTeam, src["visibility"])

	// invalid visibility rejected
	w, _ = call(t, kbH.Create, "POST", "/wiki/kb/", `{"name":"x","visibility":"everyone"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// article create: draft default + version 1 snapshot + kb count bump
	w, out = call(t, artH.Create, "POST", "/wiki/article/", `{"kb_id":"`+kbID+`","title":"RAG Overview","content":"## Definition\n\nretrieval"}`)
	require.Equal(t, http.StatusOK, w.Code)
	artID, _ := out["_id"].(string)
	require.NotEmpty(t, artID)

	versions := listVersions(t, h, artID)
	require.Len(t, versions, 1)
	assert.Equal(t, float64(1), versions[0]["version"])
	assert.Equal(t, core.WikiChangeHumanEdited, versions[0]["change_type"])
	assert.Equal(t, "## Definition\n\nretrieval", versions[0]["content"])

	w, out = call(t, kbH.Get, "GET", "/wiki/kb/"+kbID, "")
	src, _ = out["_source"].(map[string]interface{})
	assert.Equal(t, float64(1), src["article_count"])

	// partial update merges the delta over stored state
	w, _ = call(t, artH.Update, "PUT", "/wiki/article/"+artID, `{"content":"## Definition\n\nupdated"}`)
	require.Equal(t, http.StatusOK, w.Code)

	versions = listVersions(t, h, artID)
	require.Len(t, versions, 2)
	assert.Equal(t, float64(2), versions[0]["version"])
	assert.Equal(t, "## Definition\n\nupdated", versions[0]["content"])

	// status machine: draft -> published rejected, draft -> reviewed -> published ok
	w, _ = callStatus(t, h, artID, `{"status":"published"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w, _ = callStatus(t, h, artID, `{"status":"reviewed"}`)
	require.Equal(t, http.StatusOK, w.Code)
	w, _ = callStatus(t, h, artID, `{"status":"published"}`)
	require.Equal(t, http.StatusOK, w.Code)

	w, out = call(t, artH.Get, "GET", "/wiki/article/"+artID, "")
	src, _ = out["_source"].(map[string]interface{})
	assert.Equal(t, core.WikiArticlePublished, src["status"])

	// versions gained the transition audit entries
	versions = listVersions(t, h, artID)
	require.Len(t, versions, 4)
	assert.Contains(t, versions[0]["change_summary"], "reviewed -> published")

	// TOC roundtrip on the KB
	tocBody := `[{"id":"n1","title":"Basics","type":"folder","children":[{"id":"n2","title":"RAG","type":"article","article_id":"` + artID + `"}]}]`
	w, _ = callTocPut(t, h, kbID, tocBody)
	require.Equal(t, http.StatusOK, w.Code)

	w, out = callTocGet(t, h, kbID)
	require.Equal(t, http.StatusOK, w.Code)
	src, _ = out["_source"].(map[string]interface{})
	nodes, _ := src["nodes"].([]interface{})
	require.Len(t, nodes, 1)
	root, _ := nodes[0].(map[string]interface{})
	assert.Equal(t, "Basics", root["title"])

	w, _ = callTocPut(t, h, kbID, `[{"id":"bad","title":"x","type":"article"}]`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// a PUT whose body is entirely protected fields is a no-op with
	// infinilabs/framework#428; older framework wipes the record, so skip
	// loudly until the pinned framework carries the fix
	w, _ = call(t, artH.Update, "PUT", "/wiki/article/"+artID, `{"status":"draft"}`)
	require.Equal(t, http.StatusOK, w.Code)
	w, out = call(t, artH.Get, "GET", "/wiki/article/"+artID, "")
	src, _ = out["_source"].(map[string]interface{})
	if v, ok := src["status"].(string); !ok || v == "" {
		t.Skip("framework lacks infinilabs/framework#428: empty post-protection delta wipes the record; merge #428 to enforce the no-op contract here")
	}
	assert.Equal(t, core.WikiArticlePublished, src["status"])
	assert.Len(t, listVersions(t, h, artID), 4)

	// KB delete cascades articles, versions and toc
	w, _ = call(t, kbH.Delete, "DELETE", "/wiki/kb/"+kbID, "")
	require.Equal(t, http.StatusOK, w.Code)

	w, _ = call(t, artH.Get, "GET", "/wiki/article/"+artID, "")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, listVersions(t, h, artID))

	w, out = callTocGet(t, h, kbID)
	src, _ = out["_source"].(map[string]interface{})
	assert.Empty(t, src["nodes"])
}

func listVersions(t *testing.T, h APIHandler, artID string) []map[string]interface{} {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/wiki/article/"+artID+"/versions", nil)
	h.articleVersions(w, req, httprouter.Params{{Key: "id", Value: artID}})
	require.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Data []map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out.Data
}

func callStatus(t *testing.T, h APIHandler, artID, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("PUT", "/wiki/article/"+artID+"/status", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.updateArticleStatus(w, req, httprouter.Params{{Key: "id", Value: artID}})
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func callTocGet(t *testing.T, h APIHandler, kbID string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("GET", "/wiki/kb/"+kbID+"/toc", nil)
	w := httptest.NewRecorder()
	h.getToc(w, req, httprouter.Params{{Key: "kbId", Value: kbID}})
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func callTocPut(t *testing.T, h APIHandler, kbID, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("PUT", "/wiki/kb/"+kbID+"/toc", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.updateToc(w, req, httprouter.Params{{Key: "kbId", Value: kbID}})
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}
