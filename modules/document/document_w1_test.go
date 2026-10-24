/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"

	"infini.sh/coco/core"
)

func w1Call(t *testing.T, h APIHandler, method, target string, params httprouter.Params) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req = req.WithContext(security.AddUserToContext(req.Context(), &security.UserSessionInfo{UserID: "w1-user"}))
	w := httptest.NewRecorder()
	if method == http.MethodGet {
		h.docTimeline(w, req, params)
	} else if len(params) > 0 && params[0].Key == "id" {
		h.reprocessDatasourceDocs(w, req, params)
	} else {
		h.reprocessDoc(w, req, params)
	}
	return w
}

func w1SeedDoc(t *testing.T, h APIHandler, mutate func(*core.Document)) string {
	t.Helper()
	w0Capture(t) // swallow the create-path enqueue
	w := w0Call(t, h, http.MethodPost, "/document/", `{"title":"w1 doc","content":"seed content for the w1 tests"}`, httprouter.Params{})
	require.Equal(t, http.StatusOK, w.Code)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	id, _ := out["_id"].(string)
	require.NotEmpty(t, id)

	if mutate != nil {
		ctx := orm.NewContext()
		ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
		ctx.Refresh = orm.WaitForRefresh
		doc := core.Document{}
		doc.ID = id
		_, err := orm.GetV2(ctx, &doc)
		require.NoError(t, err)
		mutate(&doc)
		require.NoError(t, orm.Save(ctx, &doc))
	}
	return id
}

func TestReprocessSkipsUnchangedCompleted(t *testing.T) {
	h := w0Setup(t)
	id := w1SeedDoc(t, h, func(d *core.Document) {
		d.Status = core.DocumentStatusCompleted
	})

	got := w0Capture(t)
	w := w1Call(t, h, http.MethodPost, "/document/"+id+"/_reprocess", httprouter.Params{{Key: "doc_id", Value: id}})
	require.Equal(t, http.StatusOK, w.Code)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	assert.Equal(t, "skipped", out["result"], "completed+unchanged fingerprint must be skipped")
	assert.Empty(t, *got, "skip must not enqueue")
}

func TestReprocessForceClearsAndEnqueues(t *testing.T) {
	h := w0Setup(t)
	id := w1SeedDoc(t, h, func(d *core.Document) {
		d.Status = core.DocumentStatusCompleted
		d.Chunks = []core.DocumentChunk{{Text: "stale chunk"}}
		d.Summary = "stale summary"
		d.ErrorMessage = "old failure"
	})

	got := w0Capture(t)
	w := w1Call(t, h, http.MethodPost, "/document/"+id+"/_reprocess?force=true", httprouter.Params{{Key: "doc_id", Value: id}})
	require.Equal(t, http.StatusOK, w.Code)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	assert.Equal(t, "reprocessing", out["result"])
	require.Len(t, *got, 1)

	enq := (*got)[0]
	assert.Equal(t, core.DocumentStatusIndexing, enq.Status)
	assert.Empty(t, enq.Chunks, "stale chunks must be cleared before re-entering the pipeline")
	assert.Empty(t, enq.Summary)
	assert.Empty(t, enq.ErrorMessage)
}

func TestReprocessFailedDocWithoutForce(t *testing.T) {
	h := w0Setup(t)
	id := w1SeedDoc(t, h, func(d *core.Document) {
		d.Status = core.DocumentStatusFailed
		d.ErrorMessage = "tika down"
	})

	got := w0Capture(t)
	w := w1Call(t, h, http.MethodPost, "/document/"+id+"/_reprocess", httprouter.Params{{Key: "doc_id", Value: id}})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, *got, 1, "failed documents re-run without force — the guard protects completed docs only")
}

func TestReprocessMissingDoc404(t *testing.T) {
	h := w0Setup(t)
	w := w1Call(t, h, http.MethodPost, "/document/ghost/_reprocess", httprouter.Params{{Key: "doc_id", Value: "ghost"}})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTimelineReturnsLifecycle(t *testing.T) {
	h := w0Setup(t)
	id := w1SeedDoc(t, h, func(d *core.Document) {
		d.Status = core.DocumentStatusFailed
		d.ErrorMessage = "boom"
		d.EmbeddingModel = "mock/embed-1024"
		d.Metadata = map[string]interface{}{"pipeline_runs": []interface{}{map[string]interface{}{"pipeline": "enrich_documents"}}}
	})

	w := w1Call(t, h, http.MethodGet, "/document/"+id+"/_timeline", httprouter.Params{{Key: "doc_id", Value: id}})
	require.Equal(t, http.StatusOK, w.Code)
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	assert.Equal(t, core.DocumentStatusFailed, out["status"])
	assert.Equal(t, "boom", out["error_message"])
	assert.Equal(t, "mock/embed-1024", out["embedding_model"])
	runs, ok := out["pipeline_runs"].([]interface{})
	require.True(t, ok, "pipeline_runs must always be an array, empty included")
	assert.Len(t, runs, 1)
}

func TestCreateMarksIndexingStatus(t *testing.T) {
	h := w0Setup(t)
	got := w0Capture(t)
	w := w0Call(t, h, http.MethodPost, "/document/", `{"title":"statusful","content":"content that will be enriched"}`, httprouter.Params{})
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, *got, 1)
	assert.Equal(t, core.DocumentStatusIndexing, (*got)[0].Status, "content-bearing creates enter the queue visibly in-flight")

	// metadata-only create keeps whatever the caller sent (no lifecycle promise)
	got2 := w0Capture(t)
	w = w0Call(t, h, http.MethodPost, "/document/", `{"title":"no content"}`, httprouter.Params{})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, *got2)
}
