/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"

	"infini.sh/coco/core"
)

func correctionCall(t *testing.T, h APIHandler, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/wiki/governance/_correction", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(security.AddUserToContext(req.Context(), &security.UserSessionInfo{UserID: "user-1"}))
	w := httptest.NewRecorder()
	h.createCorrection(w, req, httprouter.Params{})
	out := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestCorrectionValidation(t *testing.T) {
	_, _, h := setupFlow(t)

	for _, body := range []string{
		`{}`, // no comment
		`{"comment":"  ","route_hint":"fact_missing"}`, // blank comment
		`{"comment":"fix","route_hint":"nope"}`,        // bad hint
	} {
		w, _ := correctionCall(t, h, body)
		assert.Equal(t, http.StatusBadRequest, w.Code, "body should be rejected: %.40s", body)
	}

	// an oversized *answer* is truncated server-side, not rejected: the
	// answer excerpt is attached context, the comment is the payload
	w, _ := correctionCall(t, h, `{"comment":"fix","route_hint":"fact_missing","answer":"`+strings.Repeat("a", correctionMaxAnswer+1)+`"}`)
	assert.Equal(t, http.StatusOK, w.Code)

	// no user in context -> 401
	req := httptest.NewRequest(http.MethodPost, "/wiki/governance/_correction", bytes.NewReader([]byte(`{"comment":"fix","route_hint":"fact_missing"}`)))
	w401 := httptest.NewRecorder()
	h.createCorrection(w401, req, httprouter.Params{})
	assert.Equal(t, http.StatusUnauthorized, w401.Code)
}

func TestCorrectionCreatesProposal(t *testing.T) {
	_, _, h := setupFlow(t)

	w, out := correctionCall(t, h, `{
		"query": "how do we rotate the gateway certs",
		"answer": "run /opt/gw/rotate.sh",
		"message_id": "msg-42",
		"cited_doc_ids": ["doc-1", "doc-2"],
		"route_hint": "fact_outdated",
		"comment": "rotate.sh was replaced by gwctl in March"
	}`)
	require.Equal(t, http.StatusOK, w.Code)
	id, ok := out["id"].(string)
	require.True(t, ok, "expected proposal id, got %v", out)

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	anchor := correctionAnchor("msg-42", "how do we rotate the gateway certs", "rotate.sh was replaced by gwctl in March")
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(10).
		Filter(orm.TermQuery("article_id", anchor)))
	require.NoError(t, err)
	proposals, _, err := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	require.NoError(t, err)
	require.Len(t, proposals, 1)

	p := proposals[0]
	assert.Equal(t, id, p.ID)
	assert.Equal(t, core.WikiGovernanceOpen, p.Status)
	assert.Equal(t, anchor, p.ArticleID)
	assert.Equal(t, "how do we rotate the gateway certs", p.ArticleTitle)
	assert.NotEmpty(t, p.Reason, "reason carries the routing-hint explanation")
	assert.Equal(t, "fact_outdated", p.Evidence["route_hint"])
	assert.Equal(t, "user-1", p.Evidence["reported_by"])
	assert.Equal(t, float64(1), p.Evidence["report_count"])
}

func TestCorrectionIdempotentPerMessage(t *testing.T) {
	_, _, h := setupFlow(t)

	body := `{"query":"q1","message_id":"msg-7","route_hint":"preference","comment":"prefer runbooks in Chinese"}`
	_, out1 := correctionCall(t, h, body)
	_, out2 := correctionCall(t, h, body)

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiGovernanceProposal{})
	anchor := correctionAnchor("msg-7", "q1", "prefer runbooks in Chinese")
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(10).
		Filter(orm.TermQuery("article_id", anchor), orm.TermQuery("type", core.WikiGovernanceCorrection)))
	require.NoError(t, err)
	proposals, _, err := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	require.NoError(t, err)

	// one proposal per anchor, counter bumped, same id both times
	require.Len(t, proposals, 1)
	assert.Equal(t, out1["id"], out2["id"])
	assert.Equal(t, float64(2), proposals[0].Evidence["report_count"])

	// a different correction on the same message opens its own proposal
	w, out3 := correctionCall(t, h, `{"query":"q1","message_id":"msg-7","route_hint":"fact_missing","comment":"different problem"}`)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotEqual(t, out1["id"], out3["id"])
}
