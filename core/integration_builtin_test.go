/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import (
	"testing"

	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

func TestIsBuiltInIntegration(t *testing.T) {
	if !IsBuiltInIntegration(DefaultSearchIntegrationID) {
		t.Errorf("expected %q to be the built-in integration", DefaultSearchIntegrationID)
	}
	for _, id := range []string{"", "my-widget", "FULL-SCREEN-WIDGET-DEFAULT", DefaultSearchIntegrationID + " "} {
		if IsBuiltInIntegration(id) {
			t.Errorf("expected %q to NOT be the built-in integration", id)
		}
	}
}

func builtInHit() elastic.IndexDocument {
	return elastic.IndexDocument{ID: DefaultSearchIntegrationID, Index: "integration"}
}

func TestStripBuiltInIntegration(t *testing.T) {
	t.Run("removes the built-in hit and fixes object total", func(t *testing.T) {
		res := elastic.SearchResponse{}
		res.Hits.Hits = []elastic.IndexDocument{builtInHit(), {ID: "abc"}, {ID: "def"}}
		res.Hits.Total = map[string]interface{}{"value": 3, "relation": "eq"}

		StripBuiltInIntegration(&res)

		if len(res.Hits.Hits) != 2 {
			t.Fatalf("expected 2 hits after strip, got %d", len(res.Hits.Hits))
		}
		for _, hit := range res.Hits.Hits {
			if IsBuiltInIntegration(hit.ID) {
				t.Fatalf("built-in integration still present after strip")
			}
		}
		total := res.Hits.Total.(map[string]interface{})
		if total["value"] != int64(2) {
			t.Errorf("expected total fixed to 2, got %v", total["value"])
		}
	})

	t.Run("fixes legacy numeric total", func(t *testing.T) {
		res := elastic.SearchResponse{}
		res.Hits.Hits = []elastic.IndexDocument{builtInHit()}
		res.Hits.Total = float64(1)

		StripBuiltInIntegration(&res)

		if len(res.Hits.Hits) != 0 {
			t.Fatalf("expected 0 hits after strip, got %d", len(res.Hits.Hits))
		}
		if res.Hits.Total != float64(0) {
			t.Errorf("expected total fixed to 0, got %v", res.Hits.Total)
		}
	})

	t.Run("no-op when the built-in integration is absent", func(t *testing.T) {
		res := elastic.SearchResponse{}
		res.Hits.Hits = []elastic.IndexDocument{{ID: "abc"}}
		res.Hits.Total = map[string]interface{}{"value": 1, "relation": "eq"}
		before := util.MustToJSONBytes(res)

		StripBuiltInIntegration(&res)

		if string(util.MustToJSONBytes(res)) != string(before) {
			t.Errorf("expected response untouched when built-in absent")
		}
	})
}
