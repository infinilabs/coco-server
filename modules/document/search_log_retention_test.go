/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"testing"
	"time"
)

func TestSearchLogRetentionCutoff(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

	cutoff := searchLogRetentionCutoff(now, 30)
	if diff := now.Sub(cutoff).Hours(); diff < 29*24 || diff > 31*24 {
		t.Fatalf("30-day cutoff wrong: %v", cutoff)
	}

	// zero/negative days fall back to the default, never "delete everything"
	fallback := searchLogRetentionCutoff(now, 0)
	if !fallback.Equal(searchLogRetentionCutoff(now, searchLogDefaultRetentionDays)) {
		t.Fatal("invalid retention must fall back to the default window")
	}
	if !cutoff.Before(now) {
		t.Fatal("cutoff must be in the past")
	}
}

func TestParsePositiveInt(t *testing.T) {
	if v, err := parsePositiveInt("14"); err != nil || v != 14 {
		t.Fatalf("valid input must parse: %d %v", v, err)
	}
	for _, bad := range []string{"", "x", "0", "-3"} {
		if _, err := parsePositiveInt(bad); err == nil {
			t.Fatalf("%q must be rejected", bad)
		}
	}
}
