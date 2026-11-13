/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"net/http"
	"net/url"
	"testing"
)

func withFakeConnectorDatasources(t *testing.T, m map[string][]string) {
	t.Helper()
	prev := datasourceIDsByConnector
	datasourceIDsByConnector = func(connectorID string) []string {
		return m[connectorID]
	}
	t.Cleanup(func() {
		datasourceIDsByConnector = prev
	})
}

func TestTranslateConnectorFilterClause(t *testing.T) {
	withFakeConnectorDatasources(t, map[string][]string{
		"github": {"ds-1", "ds-2"},
		"yuque":  {"ds-3"},
	})

	cases := []struct {
		name       string
		clause     string
		want       string
		translated bool
	}{
		{
			name:       "single connector expands to its datasources",
			clause:     "source.connector_id:any(github)",
			want:       "source.id:any(ds-1,ds-2)",
			translated: true,
		},
		{
			name:       "multiple connectors union",
			clause:     "source.connector_id:any(github,yuque)",
			want:       "source.id:any(ds-1,ds-2,ds-3)",
			translated: true,
		},
		{
			name:       "negation is preserved",
			clause:     "!source.connector_id:any(github)",
			want:       "!source.id:any(ds-1,ds-2)",
			translated: true,
		},
		{
			name:       "unknown connector matches nothing via sentinel",
			clause:     "source.connector_id:any(nosuch)",
			want:       "source.id:any(__no_datasource__)",
			translated: true,
		},
		{
			name:   "other fields pass through",
			clause: "source.id:any(ds-1)",
			want:   "source.id:any(ds-1)",
		},
		{
			name:   "non-any connector clause passes through",
			clause: "source.connector_id:github",
			want:   "source.connector_id:github",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := translateConnectorFilterClause(c.clause)
			if ok != c.translated {
				t.Fatalf("translated = %v, want %v", ok, c.translated)
			}
			if got != c.want {
				t.Fatalf("clause = %q, want %q", got, c.want)
			}
		})
	}
}

func TestTranslateConnectorFiltersRewritesRequest(t *testing.T) {
	withFakeConnectorDatasources(t, map[string][]string{
		"github": {"ds-1"},
	})

	req := &http.Request{URL: &url.URL{RawQuery: "query=k8s&filter=source.connector_id:any(github)&filter=tags:any(deploy)"}}
	translateConnectorFilters(req)

	filters := req.URL.Query()["filter"]
	if len(filters) != 2 {
		t.Fatalf("filter params = %v, want 2", filters)
	}
	if filters[0] != "source.id:any(ds-1)" {
		t.Fatalf("filter[0] = %q, want translated datasource terms", filters[0])
	}
	if filters[1] != "tags:any(deploy)" {
		t.Fatalf("filter[1] = %q, want untouched", filters[1])
	}
	if q := req.URL.Query().Get("query"); q != "k8s" {
		t.Fatalf("query param = %q, want k8s", q)
	}
}

func TestTranslateConnectorFiltersNoopWithoutConnectorFilters(t *testing.T) {
	req := &http.Request{URL: &url.URL{RawQuery: "query=k8s"}}
	before := req.URL.RawQuery
	translateConnectorFilters(req)
	if req.URL.RawQuery != before {
		t.Fatalf("raw query rewritten to %q, want untouched", req.URL.RawQuery)
	}
}
