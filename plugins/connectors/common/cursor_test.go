package common

import (
	"testing"
	"time"
)

func TestCursorFactoryFromValueWithTie(t *testing.T) {
	factory := CursorSerializer{PropertyType: "datetime"}
	prop := time.Date(2024, 10, 9, 12, 0, 0, 0, time.UTC)
	tie := "node-42"
	snapshot, err := factory.FromValue(prop, tie)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if got, ok := snapshot.Property.(time.Time); !ok || !got.Equal(prop) {
		t.Fatalf("unexpected property value: %#v", snapshot.Property)
	}
	if snapshot.Tie != tie {
		t.Fatalf("unexpected tie value: %#v", snapshot.Tie)
	}
	if snapshot.Stored.Property.Type != "datetime" {
		t.Fatalf("expected stored property type datetime, got %s", snapshot.Stored.Property.Type)
	}
	if snapshot.Stored.Tie == nil || snapshot.Stored.Tie.Type != "string" || snapshot.Stored.Tie.Value != tie {
		t.Fatalf("unexpected stored tie: %#v", snapshot.Stored.Tie)
	}
}

func TestCompareCursorUsesTie(t *testing.T) {
	factory := CursorSerializer{PropertyType: "int"}
	a, err := factory.FromValue(int64(10), "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := factory.FromValue(int64(10), "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmp := CompareCursors(a, b, "int"); cmp >= 0 {
		t.Fatalf("expected a < b due to tie, got %d", cmp)
	}
	if cmp := CompareCursors(b, a, "int"); cmp <= 0 {
		t.Fatalf("expected b > a due to tie, got %d", cmp)
	}

	// property difference should dominate tie
	c, err := factory.FromValue(int64(11), "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmp := CompareCursors(c, a, "int"); cmp <= 0 {
		t.Fatalf("expected property comparison to win, got %d", cmp)
	}
}
