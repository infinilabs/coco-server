/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"

	"infini.sh/coco/core"
)

func fromChatCall(t *testing.T, h APIHandler, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/wiki/article/_from_chat", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.createArticleFromChat(w, req, httprouter.Params{})
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestFromChatValidation(t *testing.T) {
	_, _, h := setupFlow(t)

	for _, body := range []string{
		`{}`,
		`{"kb_id":"","title":"t","content":"c"}`,
		`{"kb_id":"x","title":"  ","content":"c"}`,
		`{"kb_id":"x","title":"t","content":"   "}`,
	} {
		w, _ := fromChatCall(t, h, body)
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
	}

	// unknown kb -> 404
	w, _ := fromChatCall(t, h, `{"kb_id":"nope","title":"t","content":"c"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// content size cap
	big := `{"kb_id":"x","title":"t","content":"` + string(make([]byte, fromChatMaxContentBytes+1)) + `"}`
	w, _ = fromChatCall(t, h, big)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// too many sources
	w, _ = fromChatCall(t, h, `{"kb_id":"x","title":"t","content":"c","sources":[`+repeatJSONSource(fromChatMaxSources+1)+`]}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func repeatJSONSource(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ","
		}
		s += `{"doc_id":"d` + string(rune('0'+i%10)) + `"}`
	}
	return s
}

func TestFromChatCreatesDraftArticle(t *testing.T) {
	kbH, artH, h := setupFlow(t)

	// a KB to land into
	w, out := call(t, kbH.Create, "POST", "/wiki/kb/", `{"name":"Ops Runbook","visibility":"team"}`)
	require.Equal(t, http.StatusOK, w.Code)
	kbID, _ := out["_id"].(string)
	require.NotEmpty(t, kbID)

	// save an answer with citations
	w, out = fromChatCall(t, h, `{
		"kb_id":"`+kbID+`",
		"title":"Restart policy",
		"summary":"How to restart safely",
		"content":"Restart takes effect after flush [1]. See also [[concept:Rolling Update]].",
		"sources":[{"doc_id":"doc-1","title":"Runbook","excerpt":"flush then restart"},{"doc_id":""},{"doc_id":"doc-2","title":"Wiki","excerpt":"rolling"}],
		"message_id":"msg-9"
	}`)
	require.Equal(t, http.StatusOK, w.Code, out)
	articleID, _ := out["_id"].(string)
	require.NotEmpty(t, articleID)

	// D1: lands as a draft, flagged AI-generated, with citations preserved
	w, out = call(t, artH.Get, "GET", "/wiki/article/"+articleID, "")
	require.Equal(t, http.StatusOK, w.Code)
	src, _ := out["_source"].(map[string]interface{})
	assert.Equal(t, core.WikiArticleDraft, src["status"])
	assert.Equal(t, true, src["ai_generated"])
	sources, _ := src["sources"].([]interface{})
	require.Len(t, sources, 2) // the doc_id-less entry is dropped (D2 backlink contract)
	first, _ := sources[0].(map[string]interface{})
	assert.Equal(t, "doc-1", first["doc_id"])

	// entered the TOC (createDraftArticle contract)
	tocReq := httptest.NewRequest("GET", "/wiki/kb/"+kbID+"/toc", nil)
	tocW := httptest.NewRecorder()
	h.getToc(tocW, tocReq, httprouter.Params{{Key: "id", Value: kbID}})
	require.Equal(t, http.StatusOK, tocW.Code)
	var tocOut struct {
		Source struct {
			Nodes []map[string]interface{} `json:"nodes"`
		} `json:"_source"`
	}
	require.NoError(t, json.Unmarshal(tocW.Body.Bytes(), &tocOut))
	require.Len(t, tocOut.Source.Nodes, 1)

	// version 1 snapshot records the chat provenance
	versions := listVersions(t, h, articleID)
	require.Len(t, versions, 1)
	assert.Equal(t, core.WikiChangeAIGenerated, versions[0]["change_type"])
	assert.Contains(t, versions[0]["change_summary"], "saved from chat answer")
}
