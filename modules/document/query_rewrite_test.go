/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"errors"
	"strings"
	"testing"

	"infini.sh/coco/core"
)

// resetRewriteLeg clears every piece of rewrite leg state so tests stay
// independent.
func resetRewriteLeg() {
	rewriteCacheReset()
	rewriteBreakerReset()
	rewriteQueryEnabled = true
}

func rewriteTestModel() *core.ModelId {
	return &core.ModelId{ProviderID: "test-provider", ID: "test-model"}
}

func TestSanitizeRewrite(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"年终双薪政策", "年终双薪政策"},
		{"  \n年终双薪政策 \t", "年终双薪政策"},
		{`"年终双薪政策"`, "年终双薪政策"},
		{"“年终双薪政策”", "年终双薪政策"},
		{"改写后：年终双薪政策", "年终双薪政策"},
		{"Rewrite: year-end double pay", "year-end double pay"},
		{"```\n年终双薪政策\n```", "年终双薪政策"},
		{"<think>用户想要正式说法</think>年终双薪政策", "年终双薪政策"},
	}
	for _, c := range cases {
		if got := sanitizeRewrite(c.in); got != c.want {
			t.Errorf("sanitizeRewrite(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// commentary-length output is rejected, not truncated
	if got := sanitizeRewrite(strings.Repeat("字", rewriteMaxOutputLen+1)); got != "" {
		t.Errorf("over-length rewrite should be rejected, got %d runes", len(got))
	}
}

func TestRewriteSearchQueryAppliedAndCached(t *testing.T) {
	resetRewriteLeg()
	calls := 0
	resolveRewriteModelFn = rewriteTestModel
	rewriteCallModelFn = func(_ context.Context, _ *core.ModelId, query string) (string, error) {
		calls++
		if query != "十三薪怎么发" {
			t.Errorf("model got %q, want the original query", query)
		}
		return "年终双薪发放政策", nil
	}

	first := rewriteSearchQuery(context.Background(), "十三薪怎么发")
	if !first.Applied || first.Query != "年终双薪发放政策" || first.Cached {
		t.Fatalf("first call: %+v", first)
	}
	if calls != 1 {
		t.Fatalf("model calls after first = %d", calls)
	}

	second := rewriteSearchQuery(context.Background(), "  十三薪怎么发 \t") // same normalized form, different spacing
	if !second.Applied || second.Query != "年终双薪发放政策" || !second.Cached {
		t.Fatalf("second call should hit the cache: %+v", second)
	}
	if calls != 1 {
		t.Fatalf("cache miss went to the model, calls = %d", calls)
	}
}

func TestRewriteSearchQueryUnchangedNotApplied(t *testing.T) {
	resetRewriteLeg()
	calls := 0
	resolveRewriteModelFn = rewriteTestModel
	rewriteCallModelFn = func(_ context.Context, _ *core.ModelId, _ string) (string, error) {
		calls++
		return "报销流程", nil // same as the query
	}

	first := rewriteSearchQuery(context.Background(), "报销流程")
	if first.Applied || first.Query != "报销流程" || first.Note == "" {
		t.Fatalf("unchanged rewrite should not apply: %+v", first)
	}
	second := rewriteSearchQuery(context.Background(), "报销流程")
	if second.Applied || !second.Cached || calls != 1 {
		t.Fatalf("unchanged verdict should cache too: %+v, calls=%d", second, calls)
	}
}

func TestRewriteSearchQueryFailureThenBreaker(t *testing.T) {
	resetRewriteLeg()
	calls := 0
	resolveRewriteModelFn = rewriteTestModel
	rewriteCallModelFn = func(_ context.Context, _ *core.ModelId, _ string) (string, error) {
		calls++
		return "", errors.New("model down")
	}

	query := "怎么请年假"
	for i := 0; i < rewriteBreakerThreshold; i++ {
		res := rewriteSearchQuery(context.Background(), query)
		if res.Applied || res.Note == "" {
			t.Fatalf("failure #%d should degrade with a note: %+v", i+1, res)
		}
	}
	if calls != rewriteBreakerThreshold {
		t.Fatalf("calls = %d, want %d", calls, rewriteBreakerThreshold)
	}

	// breaker open: the model is not called again, the leg reports cooldown
	res := rewriteSearchQuery(context.Background(), query)
	if calls != rewriteBreakerThreshold {
		t.Fatalf("breaker open but model called again, calls = %d", calls)
	}
	if !strings.Contains(res.Note, "cooling down") {
		t.Fatalf("expected cooldown note, got %+v", res)
	}
}

func TestRewriteSearchQueryGuards(t *testing.T) {
	resetRewriteLeg()
	calls := 0
	resolveRewriteModelFn = rewriteTestModel
	rewriteCallModelFn = func(_ context.Context, _ *core.ModelId, _ string) (string, error) {
		calls++
		return "never", nil
	}

	// too short
	res := rewriteSearchQuery(context.Background(), "a")
	if res.Applied || calls != 0 {
		t.Fatalf("short query should be skipped: %+v", res)
	}
	// no model configured
	resolveRewriteModelFn = func() *core.ModelId { return nil }
	res = rewriteSearchQuery(context.Background(), "十三薪怎么发")
	if res.Applied || !strings.Contains(res.Note, "no default language model") {
		t.Fatalf("missing model should degrade: %+v", res)
	}
	// global kill switch
	resolveRewriteModelFn = rewriteTestModel
	rewriteQueryEnabled = false
	res = rewriteSearchQuery(context.Background(), "十三薪怎么发")
	if res.Applied || calls != 0 {
		t.Fatalf("disabled leg must not call the model: %+v", res)
	}
}
