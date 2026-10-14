/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"bytes"
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

var w0StoreOnce sync.Once

// w0Setup points the global ORM at a sqlite store carrying the document
// model, mirroring the wiki module's per-file isolated store (a shared
// store binds routes by last registration — file ordering pollution).
func w0Setup(t *testing.T) APIHandler {
	t.Helper()
	w0StoreOnce.Do(func() {
		handler := &sqlite.SQLiteORM{Config: sqlite.SQLiteConfig{
			Enabled: true,
			DBPath:  filepath.Join(t.TempDir(), "document-w0.db"),
		}}
		if err := handler.Open(); err != nil {
			panic(err)
		}
		if err := handler.RegisterSchemaWithName(core.Document{}, "document-w0-test"); err != nil {
			panic(err)
		}
		orm.Register("sqlite-document-w0-test", handler)
	})
	return APIHandler{}
}

// w0Capture swaps the enqueue hook for a recorder and restores it after.
func w0Capture(t *testing.T) *[]*core.Document {
	t.Helper()
	var got []*core.Document
	orig := enqueueForIndexing
	enqueueForIndexing = func(doc *core.Document) error {
		d := *doc
		got = append(got, &d)
		return nil
	}
	t.Cleanup(func() { enqueueForIndexing = orig })
	return &got
}

func w0Call(t *testing.T, h APIHandler, method, target, body string, params httprouter.Params) *httptest.ResponseRecorder {
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
	req = req.WithContext(security.AddUserToContext(req.Context(), &security.UserSessionInfo{UserID: "w0-user"}))
	w := httptest.NewRecorder()
	switch method {
	case http.MethodPost:
		h.createDoc(w, req, params)
	case http.MethodPut:
		h.updateDoc(w, req, params)
	}
	return w
}

func TestDocumentContentChangedPure(t *testing.T) {
	same := &core.Document{Content: "same content"}
	h1, _, ok := contentFingerprint(same.Content)
	require.True(t, ok)

	// changed: different fingerprint
	assert.True(t, documentContentChanged(&core.Document{ContentHash: "zzz"}, same))
	// unchanged: fingerprint matches stored hash
	assert.False(t, documentContentChanged(&core.Document{ContentHash: h1}, same))
	// blank / too-short payloads never trigger
	assert.False(t, documentContentChanged(&core.Document{ContentHash: "abc"}, &core.Document{Content: ""}))
	assert.False(t, documentContentChanged(&core.Document{ContentHash: "abc"}, &core.Document{Content: "tiny"}))
	assert.False(t, documentContentChanged(&core.Document{ContentHash: "abc"}, nil))
	// legacy doc without stored hash: any fingerprint-bearing edit counts
	assert.True(t, documentContentChanged(&core.Document{}, same))
}

func TestCreateDocEnqueuesContentBearingDocument(t *testing.T) {
	h := w0Setup(t)
	got := w0Capture(t)

	w := w0Call(t, h, http.MethodPost, "/document/", `{"title":"api doc","content":"some real content for chunking and embedding"}`, httprouter.Params{})
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, *got, 1, "content-bearing create must enter the enrichment queue")
	assert.Equal(t, "api doc", (*got)[0].Title)
	assert.NotEmpty(t, (*got)[0].ContentHash, "enqueued doc must already carry the orm-hook fingerprint")

	// metadata-only create: nothing to enrich, no queue churn
	got2 := w0Capture(t)
	w = w0Call(t, h, http.MethodPost, "/document/", `{"title":"empty doc"}`, httprouter.Params{})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, *got2, "content-less create must not enqueue")
}

func w0CreateForUpdate(t *testing.T, h APIHandler) string {
	t.Helper()
	// the create itself enqueues (entry closure) — swallow it here, this
	// helper only exists to seed a stored document for the edit test
	w0Capture(t)
	out := map[string]interface{}{}
	w := w0Call(t, h, http.MethodPost, "/document/", `{"title":"editable","content":"initial content body"}`, httprouter.Params{})
	require.Equal(t, http.StatusOK, w.Code)
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	id, _ := out["_id"].(string)
	require.NotEmpty(t, id)
	return id
}

func TestUpdateDocReprocessesOnContentChangeOnly(t *testing.T) {
	h := w0Setup(t)
	id := w0CreateForUpdate(t, h)

	// unchanged content (same payload) → no reprocess
	w0Capture(t)
	same := w0Call(t, h, http.MethodPut, "/document/"+id, `{"title":"editable","content":"initial content body"}`,
		httprouter.Params{{Key: "doc_id", Value: id}})
	require.Equal(t, http.StatusOK, same.Code)

	// real content change → re-enqueued for re-chunking/re-embedding
	got := w0Capture(t)
	changed := w0Call(t, h, http.MethodPut, "/document/"+id, `{"title":"editable","content":"rewritten content that invalidates old chunks"}`,
		httprouter.Params{{Key: "doc_id", Value: id}})
	require.Equal(t, http.StatusOK, changed.Code)
	require.Len(t, *got, 1, "content edit must re-enter the pipeline")
	assert.Equal(t, id, (*got)[0].ID)

	// shrunken-below-floor payload (UI partial update) → no reprocess
	got2 := w0Capture(t)
	shrunk := w0Call(t, h, http.MethodPut, "/document/"+id, `{"title":"editable","content":"x"}`,
		httprouter.Params{{Key: "doc_id", Value: id}})
	require.Equal(t, http.StatusOK, shrunk.Code)
	assert.Empty(t, *got2, "too-short payload must not trigger reprocessing")
}

func TestUpdateDocMissingOldDocStillSaves(t *testing.T) {
	h := w0Setup(t)
	got := w0Capture(t)
	// pre-read misses (never-created id): save must still proceed
	w := w0Call(t, h, http.MethodPut, "/document/no-such-doc", `{"title":"ghost","content":"never existed before"}`,
		httprouter.Params{{Key: "doc_id", Value: "no-such-doc"}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, *got, "no stored fingerprint to compare against → no reprocess decision")
}
