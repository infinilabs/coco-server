/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rsa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/util"
)

func TestLookupPath(t *testing.T) {
	m := map[string]interface{}{
		"title": "hello",
		"nested": map[string]interface{}{
			"deep": map[string]interface{}{"leaf": "found"},
		},
	}
	if got := lookupPath(m, "title"); got != "hello" {
		t.Fatalf("title: got %v", got)
	}
	if got := lookupPath(m, "nested.deep.leaf"); got != "found" {
		t.Fatalf("deep leaf: got %v", got)
	}
	if got := lookupPath(m, "nested.missing"); got != nil {
		t.Fatalf("missing should be nil, got %v", got)
	}
	if got := lookupPath(m, "title.sub"); got != nil {
		t.Fatalf("non-map descent should be nil, got %v", got)
	}
}

func TestMapHitDefaultsAndCustomPaths(t *testing.T) {
	src := &core.RemoteSource{}
	src.ID = "rsas_test"
	src.Name = "测试源"
	src.Adapter = AdapterTypeEsCompatible
	src.Mapping.Title = "doc.title"

	raw := RawHit{ID: "abc", Index: "logs-1", Fields: map[string]interface{}{
		"doc": map[string]interface{}{"title": "自定义标题"},
		"content": "正文",
		"tags":   []interface{}{"a", "b"},
	}}
	hit := MapHit(src, raw, 3)

	if hit.Document.Title != "自定义标题" {
		t.Fatalf("custom title path: %q", hit.Document.Title)
	}
	if hit.Document.Content != "正文" {
		t.Fatalf("default content path: %q", hit.Document.Content)
	}
	if !strings.HasPrefix(hit.Document.ID, "rsa_rsas_test_") {
		t.Fatalf("pseudo id prefix: %q", hit.Document.ID)
	}
	if len(hit.Document.Tags) != 2 {
		t.Fatalf("tags: %v", hit.Document.Tags)
	}
	md, ok := hit.Document.Metadata["rsa"].(util.MapStr)
	if !ok || md["raw_id"] != "abc" || md["rank"] != 3 {
		t.Fatalf("rsa metadata: %#v", hit.Document.Metadata["rsa"])
	}
	if hit.Document.Source.Type != "rsa" || hit.Document.Source.Name != "测试源" {
		t.Fatalf("source ref: %#v", hit.Document.Source)
	}

	// mapping miss: row survives with empty field
	miss := MapHit(src, RawHit{ID: "x", Fields: map[string]interface{}{}}, 1)
	if miss.Document.Title != "" || miss.Document.ID == "" {
		t.Fatalf("mapping miss must keep the row: %#v", miss.Document)
	}
}

func TestMapHitDeterministicPseudoID(t *testing.T) {
	src := &core.RemoteSource{}
	src.ID = "s1"
	a := MapHit(src, RawHit{ID: "row-1"}, 1).Document.ID
	b := MapHit(src, RawHit{ID: "row-1"}, 2).Document.ID
	if a != b {
		t.Fatalf("same raw id must map to same pseudo id: %s vs %s", a, b)
	}
	c := MapHit(src, RawHit{ID: "row-2"}, 1).Document.ID
	if a == c {
		t.Fatal("different raw ids must differ")
	}
}

func TestClampWindow(t *testing.T) {
	src := &core.RemoteSource{}
	from, size := ClampWindow(src, 0, 50)
	if from != 0 || size != 50 {
		t.Fatalf("plain window: %d/%d", from, size)
	}
	from, size = ClampWindow(src, 90, 50) // default cap 100
	if from != 90 || size != 10 {
		t.Fatalf("over-cap must shrink size: %d/%d", from, size)
	}
	from, size = ClampWindow(src, 150, 50)
	if size != 0 {
		t.Fatalf("beyond cap must be empty: %d/%d", from, size)
	}
	big := &core.RemoteSource{}
	big.Controls.MaxWindow = 5000 // hard cap trims to 1000
	from, size = ClampWindow(big, 900, 200)
	if from != 900 || size != 100 {
		t.Fatalf("hard cap: %d/%d", from, size)
	}
}

func TestRetryableClass(t *testing.T) {
	cases := []struct {
		err      error
		class    string
		retriable bool
	}{
		{errors.New("dial tcp: connection refused"), "conn", true},
		{errors.New("Get \"http://x\": EOF"), "conn", true},
		{&httpStatusError{Code: 503, Body: "busy"}, "5xx", true},
		{&httpStatusError{Code: 400, Body: "bad"}, "error", false},
		{errSourceTimeout, "timeout", false},
		{context.DeadlineExceeded, "timeout", false},
	}
	for _, c := range cases {
		class, retriable := retryableClass(c.err)
		if class != c.class || retriable != c.retriable {
			t.Errorf("%v: got (%s,%v) want (%s,%v)", c.err, class, retriable, c.class, c.retriable)
		}
	}
}

// fakeAdapter drives executor behavior tests without a live backend.
type fakeAdapter struct {
	calls   int
	failFor int // fail the first N calls with conn error
	lastErr error
	result  Result
}

func (f *fakeAdapter) Type() string { return "fake" }
func (f *fakeAdapter) Validate(src *core.RemoteSource) error { return nil }
func (f *fakeAdapter) Search(ctx context.Context, src *core.RemoteSource, endpoint string, q Query) (Result, error) {
	f.calls++
	if f.calls <= f.failFor {
		f.lastErr = fmt.Errorf("dial tcp %s: connection refused", endpoint)
		return Result{}, f.lastErr
	}
	return f.result, nil
}

func TestExecutorFailoverRetryRecovers(t *testing.T) {
	Register(&fakeAdapter{failFor: 1, result: Result{Total: 5, Hits: []MappedHit{{Rank: 1}}}})
	// NOTE: registry is global; this test relies on execution order —
	// restore by overwriting with a fresh default at the end
	defer Register(&fakeAdapter{})

	src := &core.RemoteSource{}
	src.ID = "rsas_retry"
	src.Adapter = "fake"
	src.Connection.Endpoints = []string{"http://a", "http://b"}
	src.Controls.RetryMaxAttempts = 2

	res, diag := ExecuteQuery(context.Background(), src, Query{Text: "q", Size: 10})
	if diag.Status != "ok" {
		t.Fatalf("second attempt must recover: status=%s err=%s attempts=%v", diag.Status, diag.Err, diag.Attempts)
	}
	if len(diag.Attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %v", diag.Attempts)
	}
	if diag.Attempts[0].Endpoint == diag.Attempts[1].Endpoint {
		t.Fatalf("attempt 2 must failover to the other endpoint: %v", diag.Attempts)
	}
	if res == nil || res.Total != 5 {
		t.Fatalf("result: %#v", res)
	}
}

func TestExecutorTimeoutNotRetriedByDefault(t *testing.T) {
	f := &timeoutAdapter{}
	Register(f)
	defer Register(&fakeAdapter{})

	src := &core.RemoteSource{}
	src.ID = "rsas_to"
	src.Adapter = f.Type()
	src.Connection.Endpoints = []string{"http://slow"}
	src.Controls.TimeoutMS = 300
	src.Controls.RetryMaxAttempts = 3

	_, diag := ExecuteQuery(context.Background(), src, Query{Text: "q", Size: 5})
	if diag.Status != "timeout" {
		t.Fatalf("expected timeout, got %s (%s)", diag.Status, diag.Err)
	}
	if f.calls > 1 {
		t.Fatalf("timeout must not consume further attempts by default, calls=%d", f.calls)
	}
}

type timeoutAdapter struct{ calls int }

func (t *timeoutAdapter) Type() string        { return "timeoutfake" }
func (t *timeoutAdapter) Validate(*core.RemoteSource) error { return nil }
func (t *timeoutAdapter) Search(ctx context.Context, src *core.RemoteSource, endpoint string, q Query) (Result, error) {
	t.calls++
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-time.After(2 * time.Second):
		return Result{}, nil
	}
}

func TestCacheTTLZeroDisables(t *testing.T) {
	src := &core.RemoteSource{}
	src.ID = "rsas_cache"
	src.Controls.CacheTTLSeconds = 0 // off
	if cacheGet(src, Query{Text: "x"}) != nil {
		t.Fatal("ttl 0 must never read")
	}
	cachePut(src, Query{Text: "x"}, Result{Total: 1})
	if cacheGet(src, Query{Text: "x"}) != nil {
		t.Fatal("ttl 0 must never write")
	}
	src.Controls.CacheTTLSeconds = 60
	cachePut(src, Query{Text: "x"}, Result{Total: 1})
	if r := cacheGet(src, Query{Text: "x"}); r == nil || r.Total != 1 {
		t.Fatalf("60s ttl must round-trip: %#v", r)
	}
	InvalidateSource("rsas_cache")
	if cacheGet(src, Query{Text: "x"}) != nil {
		t.Fatal("invalidate must drop the entry")
	}
}

func TestEsAdapterValidate(t *testing.T) {
	a := &AdapterEsCompatible{}
	if err := a.Validate(&core.RemoteSource{}); err == nil {
		t.Fatal("empty source must fail validation")
	}
	ok := &core.RemoteSource{}
	ok.Connection.Endpoints = []string{"http://127.0.0.1:9200"}
	ok.Query.Indices = "logs-*"
	if err := a.Validate(ok); err != nil {
		t.Fatalf("valid source: %v", err)
	}
	badTpl := *ok
	badTpl.Query.QueryTemplate = "{not json"
	if err := a.Validate(&badTpl); err == nil {
		t.Fatal("invalid template JSON must fail")
	}
}

func TestEsAdapterSearchAgainstTestServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/logs-2026/_search") {
			http.Error(w, "unexpected path "+r.URL.Path, 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"took":2,"hits":{"total":{"value":1},"hits":[{"_id":"hit1","_index":"logs-2026","_score":1.5,"_source":{"title":"外部命中","content":"外部内容"}}]}}`)
	}))
	defer srv.Close()

	src := &core.RemoteSource{}
	src.ID = "rsas_live"
	src.Name = "本地测试"
	src.Adapter = AdapterTypeEsCompatible
	src.Connection.Endpoints = []string{srv.URL}
	src.Query.Indices = "logs-2026"

	a := &AdapterEsCompatible{}
	res, err := a.Search(context.Background(), src, srv.URL, Query{Text: "外部", Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 1 || res.Hits[0].Document.Title != "外部命中" {
		t.Fatalf("hits: %#v", res.Hits)
	}
	if res.Hits[0].Document.Source.Type != "rsa" {
		t.Fatalf("mapped doc source type: %#v", res.Hits[0].Document.Source)
	}
}

func TestEsAdapterTemplateRender(t *testing.T) {
	a := &AdapterEsCompatible{}
	src := &core.RemoteSource{}
	src.ID = "rsas_tpl"
	src.Query.Indices = "idx"
	src.Query.QueryTemplate = `{"size":{{.Size}},"from":{{.From}},"query":{"term":{"msg":"{{.Text}}"}}}`
	body, err := a.buildBody(src, Query{Text: `含"引号"的查询`, From: 3, Size: 7})
	if err != nil {
		t.Fatal(err)
	}
	if body["size"].(float64) != 7 || body["from"].(float64) != 3 {
		t.Fatalf("window substitution: %#v", body)
	}
	q := body["query"].(map[string]interface{})["term"].(map[string]interface{})["msg"].(string)
	if q != `含"引号"的查询` {
		t.Fatalf("text must be JSON-escaped safely: %q", q)
	}
}

func TestEndpointRotation(t *testing.T) {
	src := &core.RemoteSource{}
	src.ID = "rsas_rotate_test"
	src.Connection.Endpoints = []string{"http://a", "http://b", "http://c"}
	// consecutive queries start at rotating bases — traffic spreads
	bases := map[int]bool{}
	for i := 0; i < 3; i++ {
		bases[beginRotation(src)] = true
	}
	if len(bases) != 3 {
		t.Fatalf("sticky cursor must spread across the ring: %v", bases)
	}
	// attempts from one base walk the ring without landing on the
	// just-failed endpoint (2-endpoint ring regression)
	small := &core.RemoteSource{}
	small.ID = "rsas_rotate_small"
	small.Connection.Endpoints = []string{"http://a", "http://b"}
	base := beginRotation(small)
	if endpointAt(small, base, 0) == endpointAt(small, base, 1) {
		t.Fatalf("attempt 1 must failover to the other endpoint: base=%d", base)
	}
}
