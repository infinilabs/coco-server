/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package engineguard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleFacets() TierFacets {
	return TierFacets{
		DatasourceIDs: []string{"ds-b", "ds-a"},
		OwnerID:       "u-123",
		FieldExcludes: []string{"payload", "document_chunk"},
		MaskRules:     []MaskRule{{Field: "metadata.phone", Action: "regex", Pattern: `(\\d{3})\\d{4}(\\d{4})`, Replace: "$1****$2"}},
	}
}

func TestTierSignatureDeterministic(t *testing.T) {
	a := sampleFacets()
	b := sampleFacets()
	// order of datasource ids must not matter
	b.DatasourceIDs = []string{"ds-a", "ds-b"}
	assert.Equal(t, TierSignature(a), TierSignature(b), "same facets (any order) share one tier")

	c := sampleFacets()
	c.DatasourceIDs = append(c.DatasourceIDs, "ds-c")
	assert.NotEqual(t, TierSignature(a), TierSignature(c), "a visible-set difference is a different tier")
}

func TestTierUserNamePrefix(t *testing.T) {
	sig := TierSignature(sampleFacets())
	name := TierUserName(sig)
	assert.Contains(t, name, rolePrefix)
	assert.Len(t, name, len(rolePrefix)+8)
}

func TestCompileRoleBodyShape(t *testing.T) {
	body := CompileRoleBody(sampleFacets(), []string{"coco_document-v2"})

	cluster, _ := body["cluster"].([]string)
	assert.Empty(t, cluster)

	blocks := body["indices"].([]interface{})
	require.Len(t, blocks, 1)
	block := blocks[0].(map[string]interface{})
	assert.Equal(t, []string{"coco_document-v2"}, block["names"])

	dls := block["dls"].(string)
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(dls), &parsed))
	should := parsed["bool"].(map[string]interface{})["should"].([]interface{})
	require.Len(t, should, 2, "datasource terms + owner term")

	fls := block["fls"].([]string)
	assert.Contains(t, fls, "~payload")
	assert.Contains(t, fls, "~document_chunk")

	mask := block["field_mask"].([]string)
	require.Len(t, mask, 1)
	assert.Contains(t, mask[0], "metadata.phone::")
}

func TestCompileRoleBodyAttrTier(t *testing.T) {
	f := TierFacets{UserAttrs: []string{"dept:finance"}}
	body := CompileRoleBody(f, []string{"idx"})
	block := body["indices"].([]interface{})[0].(map[string]interface{})
	dls := block["dls"].(string)
	assert.Contains(t, dls, "_system.attrs")
}

/* ---------------- adapter against a fake engine ---------------- */

func fakeEngine(t *testing.T) *httptest.Server {
	t.Helper()
	roles := map[string]map[string]interface{}{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/_security/role/admin":
			if r.Header.Get(RunAsHeader) != "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`{"admin": {}}`))
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/_security/role/"):
			name := strings.TrimPrefix(r.URL.Path, "/_security/role/")
			roles[name] = map[string]interface{}{}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"OK"}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/_security/role/"):
			name := strings.TrimPrefix(r.URL.Path, "/_security/role/")
			if _, ok := roles[name]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{name: roles[name]})
		case r.Method == http.MethodGet && r.URL.Path == "/coco_document-v2/_search":
			if r.Header.Get(RunAsHeader) != "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"hits":{"total":{"value":0}}}`))
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/_security/user/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestProbeAgainstFakeEngine(t *testing.T) {
	srv := fakeEngine(t)
	restore := secClientFn
	t.Cleanup(func() { secClientFn = restore })
	secClientFn = func() *client {
		return &client{baseURL: srv.URL, http: srv.Client(), username: "x", password: "y"}
	}

	caps := Probe()
	assert.True(t, caps.SecurityAPI)
	assert.True(t, caps.RunAs, "a 401 on the impersonated probe means run-as is honored")
}

func TestApplyFetchRoundTrip(t *testing.T) {
	srv := fakeEngine(t)
	restore := secClientFn
	t.Cleanup(func() { secClientFn = restore })
	secClientFn = func() *client {
		return &client{baseURL: srv.URL, http: srv.Client(), username: "x", password: "y"}
	}

	sig := TierSignature(sampleFacets())
	name := TierUserName(sig)
	require.NoError(t, ApplyTierRole(name, CompileRoleBody(sampleFacets(), []string{"idx"})))

	got, err := FetchTierRole(name)
	require.NoError(t, err)
	require.NotNil(t, got, "applied role reads back")

	missing, err := FetchTierRole(TierUserName("ffffffff"))
	require.NoError(t, err)
	assert.Nil(t, missing, "unknown role reads as nil, not an error")
}
