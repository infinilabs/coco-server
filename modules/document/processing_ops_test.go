/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
)

func opsCall(t *testing.T, h APIHandler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader = strings.NewReader(body)
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(security.AddUserToContext(req.Context(), &security.UserSessionInfo{UserID: "ops-user"}))
	w := httptest.NewRecorder()
	if method == http.MethodGet {
		h.processingOverviewHandler(w, req, httprouter.Params{})
	} else {
		h.retryFailedHandler(w, req, httprouter.Params{})
	}
	return w
}

func seedStatusDoc(t *testing.T, id, status, errMsg string, datasource string) {
	t.Helper()
	w0Capture(t) // swallow create-path enqueue
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	doc := core.Document{}
	doc.ID = id
	doc.Title = id
	doc.Content = "content for " + id
	doc.Status = status
	doc.ErrorMessage = errMsg
	doc.Source = core.DataSourceReference{ID: datasource, Name: datasource, Type: "connector"}
	require.NoError(t, orm.Create(ctx, &doc))
}

func TestProcessingOverviewCountsAndFailedList(t *testing.T) {
	w0Setup(t)
	seedStatusDoc(t, "p-ok", core.DocumentStatusCompleted, "", "ds-ops")
	seedStatusDoc(t, "p-run", core.DocumentStatusIndexing, "", "ds-ops")
	seedStatusDoc(t, "p-f1", core.DocumentStatusFailed, "tika timeout", "ds-ops")
	seedStatusDoc(t, "p-f2", core.DocumentStatusFailed, "vision model 500", "ds-ops")
	seedStatusDoc(t, "p-old", "", "", "ds-ops") // legacy: no status stamp

	w := opsCall(t, w0Setup(t), http.MethodGet, "/search/ops/processing?datasource=ds-ops", "")
	require.Equal(t, http.StatusOK, w.Code)
	out := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))

	statuses := out["statuses"].(map[string]interface{})
	assert.Equal(t, float64(1), statuses["completed"])
	assert.Equal(t, float64(1), statuses["indexing"])
	assert.Equal(t, float64(2), statuses["failed"])
	assert.Equal(t, float64(1), statuses["legacy"], "unstamped docs count as legacy, not silently dropped")

	failed := out["failed"].([]interface{})
	require.Len(t, failed, 2)
	first := failed[0].(map[string]interface{})
	assert.NotEmpty(t, first["error"])
	assert.Contains(t, first["reprocessurl"], "_reprocess")
}

func TestRetryFailedResetsAndEnqueues(t *testing.T) {
	h := w0Setup(t)
	seedStatusDoc(t, "r-f1", core.DocumentStatusFailed, "boom", "ds-retry")
	seedStatusDoc(t, "r-ok", core.DocumentStatusCompleted, "", "ds-retry")

	got := w0Capture(t)
	w := opsCall(t, h, http.MethodPost, "/document/_retry_failed", `{"datasource":"ds-retry"}`)
	require.Equal(t, http.StatusOK, w.Code)
	out := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, float64(1), out["retried"], "only the failed document retries")
	assert.Equal(t, float64(0), out["still_failed"], "nothing failed to reset or enqueue")

	require.Len(t, *got, 1)
	enq := (*got)[0]
	assert.Equal(t, "r-f1", enq.ID)
	assert.Equal(t, core.DocumentStatusIndexing, enq.Status)
	assert.Empty(t, enq.ErrorMessage, "the failure reason clears on retry")
}

func TestDocumentChunksListing(t *testing.T) {
	h := w0Setup(t)
	w0Capture(t)
	// seed a doc with structured chunks directly
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	doc := core.Document{}
	doc.ID = "chk-1"
	doc.Title = "切块样例"
	doc.Content = "正文"
	doc.Chunks = []core.DocumentChunk{
		{Text: strings.Repeat("块一内容", 200), Breadcrumb: "手册 > 安装", Range: core.ChunkRange{Start: 1, End: 2}, Embedding: core.Embedding{Embedding1024: []float32{0.1}}},
		{Text: "块二内容短", Breadcrumb: "手册 > 配置", Range: core.ChunkRange{Start: 3, End: 3}},
	}
	require.NoError(t, orm.Create(ctx, &doc))

	req := httptest.NewRequest(http.MethodGet, "/document/chk-1/_chunks", nil)
	w := httptest.NewRecorder()
	h.documentChunksHandler(w, req, httprouter.Params{{Key: "doc_id", Value: "chk-1"}})
	require.Equal(t, http.StatusOK, w.Code)
	out := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, float64(2), out["total"])
	chunks := out["chunks"].([]interface{})
	first := chunks[0].(map[string]interface{})
	assert.Equal(t, float64(0), first["index"])
	assert.Equal(t, "手册 > 安装", first["breadcrumb"])
	assert.Equal(t, true, first["vectorized"])
	assert.Equal(t, 500, len([]rune(first["excerpt"].(string))), "excerpts cap at 500 runes")
	second := chunks[1].(map[string]interface{})
	assert.Equal(t, false, second["vectorized"])

	// missing doc → 404
	req404 := httptest.NewRequest(http.MethodGet, "/document/ghost/_chunks", nil)
	w404 := httptest.NewRecorder()
	h.documentChunksHandler(w404, req404, httprouter.Params{{Key: "doc_id", Value: "ghost"}})
	assert.Equal(t, http.StatusNotFound, w404.Code)
}
