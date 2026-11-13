/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
	"infini.sh/framework/core/api/crud"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/modules/sqlite"
)

var likeStoreOnce sync.Once

func likeHandlers(t *testing.T) crud.Handlers {
	t.Helper()
	likeStoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "wiki-like.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.WikiLike{}, "wiki-like-crud-test"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-wiki-like-test", handler)
	})
	return crud.NewHandlers(likeConfig())
}

func likeCall(t *testing.T, h crud.HandlerFunc, method, targetOrID, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	target := "/wiki/like/"
	if method == "GET" || method == "DELETE" {
		target += targetOrID
	}
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
	if rest := strings.TrimPrefix(target, "/wiki/like/"); rest != "" {
		ps = httprouter.Params{{Key: "id", Value: rest}}
	}
	w := httptest.NewRecorder()
	h(w, req, ps)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestLikeCreateValidation(t *testing.T) {
	h := likeHandlers(t)

	// missing article_id is rejected
	w, _ := likeCall(t, h.Create, "POST", "", `{"user_id":"u1"}`)
	assert.Equal(t, 400, w.Code)

	// a like round-trips
	w, out := likeCall(t, h.Create, "POST", "", `{"article_id":"a1","user_id":"u1","user_name":"medcl"}`)
	require.Equal(t, 200, w.Code)
	id, _ := out["_id"].(string)
	require.NotEmpty(t, id)

	// payload cannot rewrite the author after the fact (protected fields);
	// update is skipped entirely, so this is a route-level 405/404 absence —
	// verify by checking the record is intact through Get
	w, out = likeCall(t, h.Get, "GET", id, "")
	require.Equal(t, 200, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	assert.Equal(t, "a1", src["article_id"])
	assert.Equal(t, "u1", src["user_id"])

	// delete removes the like
	w, _ = likeCall(t, h.Delete, "DELETE", id, "")
	require.Equal(t, 200, w.Code)
}
