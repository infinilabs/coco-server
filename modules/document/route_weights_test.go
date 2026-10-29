/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http/httptest"
	"testing"

	"infini.sh/coco/core"
)

// the settings-side route weights must fill in before the per-request
// overrides: an explicit request weight wins, a non-positive stored value
// keeps the built-in default, and no stored section at all leaves the
// built-ins untouched.
func TestParseRRFParamsRouteWeightsFromSettings(t *testing.T) {
	h := &APIHandler{}

	// no stored section: built-in defaults (wiki 2, rest 1)
	searchSettingsSnapshotFn = func() *core.SearchSettings { return nil }
	cfg := h.parseRRFParams(httptest.NewRequest("GET", "/search/?q=x", nil))
	if w := cfg.weight(rrfRouteWiki); w != defaultWikiRRFWeight {
		t.Fatalf("wiki weight = %v, want built-in %v", w, defaultWikiRRFWeight)
	}
	if w := cfg.weight(rrfRouteText); w != defaultRRFWeight {
		t.Fatalf("text weight = %v, want built-in %v", w, defaultRRFWeight)
	}

	// stored wiki=3, text=0 (0 = keep built-in): wiki applies, text does not
	searchSettingsSnapshotFn = func() *core.SearchSettings {
		return &core.SearchSettings{RouteWeights: &core.SearchRouteWeights{Wiki: 3}}
	}
	cfg = h.parseRRFParams(httptest.NewRequest("GET", "/search/?q=x", nil))
	if w := cfg.weight(rrfRouteWiki); w != 3 {
		t.Fatalf("wiki weight = %v, want 3 from settings", w)
	}
	if w := cfg.weight(rrfRouteText); w != defaultRRFWeight {
		t.Fatalf("text weight = %v, want built-in %v (stored 0 means default)", w, defaultRRFWeight)
	}

	// an explicit request weight still wins over the stored default
	cfg = h.parseRRFParams(httptest.NewRequest("GET", "/search/?q=x&wiki_weight=0.5", nil))
	if w := cfg.weight(rrfRouteWiki); w != 0.5 {
		t.Fatalf("wiki weight = %v, want 0.5 from the request", w)
	}
}
