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
