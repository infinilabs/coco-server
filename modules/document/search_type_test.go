/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"testing"

	"infini.sh/coco/core"
)

// the default-strategy resolution: an explicit ?search_type= always wins,
// the operator's configured default comes next, keyword is the floor.
func TestConfiguredSearchTypeFallbacks(t *testing.T) {
	cases := []struct {
		name     string
		settings *core.SearchSettings
		expected string
	}{
		{"no saved section reads as keyword", nil, "keyword"},
		{"empty value keeps historic keyword", &core.SearchSettings{}, "keyword"},
		{"stale value falls back to keyword", &core.SearchSettings{SearchType: "banana"}, "keyword"},
		{"keyword round-trips", &core.SearchSettings{SearchType: "keyword"}, "keyword"},
		{"semantic is configurable", &core.SearchSettings{SearchType: "semantic"}, "semantic"},
		{"hybrid is configurable", &core.SearchSettings{SearchType: "hybrid"}, "hybrid"},
		{"hybrid_rrf is configurable", &core.SearchSettings{SearchType: "hybrid_rrf"}, "hybrid_rrf"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.settings.DefaultType(); got != c.expected {
				t.Fatalf("DefaultType() = %q, want %q", got, c.expected)
			}
		})
	}
}
