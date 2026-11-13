package common

import (
	"testing"
	"time"
)

func TestParseColumnTime(t *testing.T) {
	cases := []struct {
		name    string
		in      interface{}
		wantNil bool
		check   func(*testing.T, *time.Time)
	}{
		{
			name: "time.Time passthrough",
			in:   time.Date(2026, 11, 10, 8, 30, 0, 0, time.UTC),
			check: func(t *testing.T, got *time.Time) {
				want := time.Date(2026, 11, 10, 8, 30, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			},
		},
		{
			name: "[]byte RFC3339",
			in:   []byte("2026-11-10T08:30:00Z"),
			check: func(t *testing.T, got *time.Time) {
				want := time.Date(2026, 11, 10, 8, 30, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			},
		},
		{
			name: "string mysql datetime",
			in:   "2026-11-10 08:30:00",
			check: func(t *testing.T, got *time.Time) {
				want := time.Date(2026, 11, 10, 8, 30, 0, 0, time.UTC)
				if got.Unix() != want.Unix() {
					t.Fatalf("got %v (%d), want %v", got, got.Unix(), want)
				}
			},
		},
		{
			name: "string with fractional seconds",
			in:   "2026-11-10 08:30:00.123456",
			check: func(t *testing.T, got *time.Time) {
				if got.Nanosecond() != 123456000 {
					t.Fatalf("fractional seconds lost: %v", got)
				}
			},
		},
		{
			name: "string date only",
			in:   "2026-11-10",
			check: func(t *testing.T, got *time.Time) {
				if got.Year() != 2026 || got.Month() != time.November || got.Day() != 10 {
					t.Fatalf("got %v", got)
				}
			},
		},
		{
			name:  "unix seconds",
			in:    int64(1762000000),
			check: func(t *testing.T, got *time.Time) { assertUnix(t, got, 1762000000) },
		},
		{
			name:  "unix millis (>1e12 heuristic)",
			in:    int64(1762000000000),
			check: func(t *testing.T, got *time.Time) { assertUnix(t, got, 1762000000) },
		},
		{
			name:  "float64 epoch (some drivers)",
			in:    float64(1762000000),
			check: func(t *testing.T, got *time.Time) { assertUnix(t, got, 1762000000) },
		},
		{
			name:    "invalid string",
			in:      "not-a-timestamp",
			wantNil: true,
		},
		{
			name:    "empty bytes",
			in:      []byte(""),
			wantNil: true,
		},
		{
			name:    "unsupported type",
			in:      map[string]string{"x": "y"},
			wantNil: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseColumnTime(c.in)
			if c.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected non-nil, got nil for input %v", c.in)
			}
			c.check(t, got)
		})
	}
}

func assertUnix(t *testing.T, got *time.Time, wantSec int64) {
	t.Helper()
	if got.Unix() != wantSec {
		t.Fatalf("got unix %d, want %d", got.Unix(), wantSec)
	}
}

// The watermark should track the max of a batch even when rows arrive out of order.
func TestWatermarkMaxTracking(t *testing.T) {
	base := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	values := []interface{}{
		base.Add(3 * time.Minute), // row 3
		base.Add(1 * time.Minute), // row 1 (earlier, arrives later)
		base.Add(5 * time.Minute), // row 5 — max
		base.Add(2 * time.Minute), // row 2
	}

	var watermark *time.Time
	for _, v := range values {
		if ts := parseColumnTime(v); ts != nil && (watermark == nil || ts.After(*watermark)) {
			watermark = ts
		}
	}

	if watermark == nil {
		t.Fatal("watermark should be set")
	}
	want := base.Add(5 * time.Minute)
	if !watermark.Equal(want) {
		t.Fatalf("watermark = %v, want max %v", watermark, want)
	}
}

// GetLastSyncValue rewinds the persisted cursor by watermarkLookback so
// mutations during the scan window are re-read. Verify the arithmetic
// mirrors what the production code does (1 minute rewind).
func TestWatermarkLookbackSemantics(t *testing.T) {
	persisted := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	effective := persisted.Add(-watermarkLookback)
	if effective.Minute() != 59 {
		t.Fatalf("effective cursor should rewind 1m: %v", effective)
	}
	// a row mutated 30s after the persisted watermark must fall inside the window
	mutatedDuringScan := persisted.Add(30 * time.Second)
	if !mutatedDuringScan.After(effective) {
		t.Fatal("row mutated during scan window must be re-read (after effective cursor)")
	}
}

func TestParseTimeStringLayouts(t *testing.T) {
	// MySQL driver often returns Go-format time strings via []byte
	goStyle := "2026-11-10 08:30:05 +0000 UTC"
	if got := parseTimeString(goStyle); got == nil {
		t.Fatalf("go-style time string not parsed: %q", goStyle)
	}
	// postgres style with timezone offset
	pg := "2026-11-10T08:30:05+08:00"
	got := parseTimeString(pg)
	if got == nil {
		t.Fatalf("pg-style time string not parsed: %q", pg)
	}
	if offset := got.Format("-07:00"); offset != "+08:00" {
		t.Fatalf("offset lost: got %v", got)
	}
}
