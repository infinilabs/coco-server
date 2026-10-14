/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"strings"
	"testing"

	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// the highlight section survives only when there is a query term to mark;
// pure-filter browsing gets it stripped before the DSL reaches the engine
func TestStripHighlightWithoutQuery(t *testing.T) {
	highlightBody := `{"highlight":{"pre_tags":["<em>"],"fields":{"title":{}}},"aggs":{"counts":{"terms":{"field":"updated"}}}}`

	newBuilder := func(body string) *orm.QueryBuilder {
		qb := orm.NewQuery()
		if body != "" {
			qb.SetRequestBodyBytes([]byte(body))
		}
		qb.EnableBodyBytes()
		return qb
	}

	cases := []struct {
		name       string
		query      string
		body       string
		wantDrop   bool
		wantAggsIn bool
	}{
		{"empty query drops the posted highlight", "", highlightBody, true, true},
		{"whitespace-only query drops it too", "   ", highlightBody, true, true},
		{"a real query keeps the highlight", "coco", highlightBody, false, true},
		{"no body is a no-op", "", "", false, false},
		{"non-JSON body is left untouched", "", "{not-json", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			qb := newBuilder(c.body)
			stripHighlightWithoutQuery(qb, c.query)

			body := qb.RequestBodyBytesVal()
			if c.body == "" {
				if len(body) != 0 {
					t.Fatalf("body should stay empty, got %q", string(body))
				}
				return
			}
			if c.body == "{not-json" {
				if string(body) != c.body {
					t.Fatalf("non-JSON body should pass through unchanged, got %q", string(body))
				}
				return
			}

			var dsl map[string]interface{}
			if err := util.FromJSONBytes(body, &dsl); err != nil {
				t.Fatalf("stripped body must stay valid JSON: %v", err)
			}
			_, hasHighlight := dsl["highlight"]
			if hasHighlight == c.wantDrop {
				t.Fatalf("highlight present = %v, want dropped = %v (body: %s)", hasHighlight, c.wantDrop, string(body))
			}
			_, hasAggs := dsl["aggs"]
			if hasAggs != c.wantAggsIn {
				t.Fatalf("aggs present = %v, want %v (body: %s)", hasAggs, c.wantAggsIn, string(body))
			}
			if !strings.Contains(string(body), "counts") {
				t.Fatalf("sibling sections must survive the strip, got %s", string(body))
			}
		})
	}
}
