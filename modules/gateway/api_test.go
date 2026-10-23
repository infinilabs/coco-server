/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package gateway

import (
	"testing"

	"infini.sh/framework/core/model"
)

// A bare-bones sync client may omit host info entirely — the stored record
// then carries a nil Host and every read must be guarded or the list panics.
func TestHostGuardsNilSafe(t *testing.T) {
	bare := model.Instance{}
	if hostNameOf(&bare) != "" {
		t.Fatalf("expected empty hostname for nil Host, got %q", hostNameOf(&bare))
	}
	if hostOSOf(&bare) != "" {
		t.Fatalf("expected empty os for nil Host, got %q", hostOSOf(&bare))
	}

	full := model.Instance{}
	full.Host = &model.HostInfo{Name: "host-a"}
	full.Host.OS.Name = "darwin"
	if hostNameOf(&full) != "host-a" || hostOSOf(&full) != "darwin" {
		t.Fatalf("guard helpers must pass through populated Host values")
	}
}

// The reverse proxy is the one code path that reaches INTO a managed
// instance: exact matches first, then prefixes, never anything else.
func TestMatchProxyRead(t *testing.T) {
	allowed := map[string]string{
		"/_info":                    "version",
		"/queue/stats":              "queues",
		"/queue/abc/_scroll":        "queues",
		"/queue/abc/consumer/c/off": "queues",
		"/logging/files":            "logs",
		"/logging/tail":             "logs",
		"/pipeline/tasks/":          "pipelines",
		"/pipeline/task/x/stages":   "pipelines",
		"/pipelines/p1/versions":    "pipeline_config",
		"/config/runtime":           "runtime_config",
	}
	for path, wantLabel := range allowed {
		label, ok := matchProxyRead(path)
		if !ok || label != wantLabel {
			t.Errorf("matchProxyRead(%q) = (%q,%v), want (%q,true)", path, label, ok, wantLabel)
		}
	}
	// query strings are stripped before matching
	if _, ok := matchProxyRead("/logging/tail?file=a.log"); !ok {
		t.Errorf("query suffix must not break the whitelist match")
	}
	// escapes stay closed
	for _, path := range []string{"/", "", "/keystore", "/admin", "/queue", "/instance/_register", "/../_info"} {
		if _, ok := matchProxyRead(path); ok {
			t.Errorf("matchProxyRead(%q) must be rejected", path)
		}
	}
}
