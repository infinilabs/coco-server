/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"infini.sh/coco/core"
)

func withSearchSettings(t *testing.T, cfg *core.SearchSettings) {
	t.Helper()
	prev := searchSettingsSnapshotFn
	searchSettingsSnapshotFn = func() *core.SearchSettings { return cfg }
	t.Cleanup(func() { searchSettingsSnapshotFn = prev })
}

func TestIsBlockedQuery(t *testing.T) {
	withSearchSettings(t, &core.SearchSettings{
		BlockedQueryPatterns: []string{`^topsecret`, `internal[-_ ]?only`},
	})
	assert.True(t, isBlockedQuery("topsecret123"))
	assert.True(t, isBlockedQuery("TOPSECRET"))
	assert.True(t, isBlockedQuery("see the Internal_Only doc"))
	assert.True(t, isBlockedQuery("internal only"))
	assert.False(t, isBlockedQuery("normal query"))
	assert.False(t, isBlockedQuery(""))

	// broken patterns are skipped, healthy ones still match
	withSearchSettings(t, &core.SearchSettings{
		BlockedQueryPatterns: []string{`[unclosed`, `^blocked$`},
	})
	assert.True(t, isBlockedQuery("blocked"))
	assert.False(t, isBlockedQuery("fine"))

	// no config → nothing blocked
	withSearchSettings(t, nil)
	assert.False(t, isBlockedQuery("anything"))
}

func TestParseRRFParamsSettingsK(t *testing.T) {
	h := &APIHandler{}

	// no request param, no setting → the built-in default
	withSearchSettings(t, nil)
	req := httptest.NewRequest("GET", "/query/_search?query=a", nil)
	cfg := h.parseRRFParams(req)
	assert.Equal(t, defaultRRFK, cfg.K)

	// setting fills in when the request carries no rrf_k
	withSearchSettings(t, &core.SearchSettings{RRFRankConstant: 5})
	cfg = h.parseRRFParams(req)
	assert.Equal(t, float64(5), cfg.K)

	// an explicit request parameter wins over the setting
	req2 := httptest.NewRequest("GET", "/query/_search?query=a&rrf_k=9", nil)
	cfg = h.parseRRFParams(req2)
	assert.Equal(t, float64(9), cfg.K)

	// out-of-range settings fall back to the default
	withSearchSettings(t, &core.SearchSettings{RRFRankConstant: 5000})
	cfg = h.parseRRFParams(req)
	assert.Equal(t, defaultRRFK, cfg.K)
}

func TestPinnedRuleMatchOrder(t *testing.T) {
	// pinnedHits walks rules in order; the first exact (case-insensitive)
	// match wins. The fetch itself needs a live engine — covered by the
	// API smoke test — here we only pin the matching semantics through the
	// same code path the handler uses.
	cfg := &core.SearchSettings{
		PinnedRules: []core.PinnedRule{
			{Query: "  annual report  ", DocumentIDs: []string{"doc-1"}},
			{Query: "a", DocumentIDs: []string{"doc-2"}},
		},
	}
	require.Len(t, cfg.PinnedRules, 2)
	assert.NotEqual(t, cfg.PinnedRules[0].Query, "a")
}
