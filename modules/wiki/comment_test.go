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

	"infini.sh/coco/core"
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"
)

func commentCall(t *testing.T, h crud.HandlerFunc, method, target, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
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
	ps := httprouter.Params{}
	if rest := strings.TrimPrefix(target, "/wiki/comment/"); rest != "" && !strings.Contains(rest, "/") {
		ps = httprouter.Params{{Key: "id", Value: rest}}
	}
	w := httptest.NewRecorder()
	h(w, req, ps)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

var commentStoreOnce sync.Once

func commentHandlers(t *testing.T) crud.Handlers {
	t.Helper()
	commentStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-comment.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.WikiComment{}, "wiki-comment-test"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-wiki-comment-test", handler)
	})
	return crud.NewHandlers(commentConfig())
}

func TestCommentCreateValidation(t *testing.T) {
	h := commentHandlers(t)

	// missing article_id / blank content are rejected
	w, _ := commentCall(t, h.Create, "POST", "/wiki/comment/", `{"content":"no article"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w, _ = commentCall(t, h.Create, "POST", "/wiki/comment/", `{"article_id":"a1","content":"   "}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// a valid comment round-trips with its author
	w, out := commentCall(t, h.Create, "POST", "/wiki/comment/", `{"article_id":"a1","user_id":"u1","user_name":"medcl","content":"nice article"}`)
	require.Equal(t, http.StatusOK, w.Code)
	id, _ := out["_id"].(string)
	require.NotEmpty(t, id)

	w, out = commentCall(t, h.Get, "GET", "/wiki/comment/"+id, "")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	assert.Equal(t, "nice article", src["content"])
	assert.Equal(t, "medcl", src["user_name"])

	// a later patch cannot rewrite the author (protected fields)
	w, _ = commentCall(t, h.Update, "PUT", "/wiki/comment/"+id, `{"user_id":"u2","content":"edited"}`)
	require.Equal(t, http.StatusOK, w.Code)
	w, out = commentCall(t, h.Get, "GET", "/wiki/comment/"+id, "")
	src, _ = out["_source"].(map[string]interface{})
	assert.Equal(t, "edited", src["content"])
	assert.Equal(t, "u1", src["user_id"])
	assert.Equal(t, "medcl", src["user_name"])
}
