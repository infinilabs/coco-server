package neo4j

import (
	"testing"
	"time"
)

func TestCursorFactoryFromValueWithTie(t *testing.T) {
	factory := cursorFactory{propertyType: "datetime"}
	prop := time.Date(2024, 10, 9, 12, 0, 0, 0, time.UTC)
	tie := "node-42"
	snapshot, err := factory.fromValue(prop, tie)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if got, ok := snapshot.property.(time.Time); !ok || !got.Equal(prop) {
		t.Fatalf("unexpected property value: %#v", snapshot.property)
	}
	if snapshot.tie != tie {
		t.Fatalf("unexpected tie value: %#v", snapshot.tie)
	}
	if snapshot.stored.Property.Type != "datetime" {
		t.Fatalf("expected stored property type datetime, got %s", snapshot.stored.Property.Type)
	}
	if snapshot.stored.Tie == nil || snapshot.stored.Tie.Type != "string" || snapshot.stored.Tie.Value != tie {
		t.Fatalf("unexpected stored tie: %#v", snapshot.stored.Tie)
	}
}

func TestCompareCursorUsesTie(t *testing.T) {
	factory := cursorFactory{propertyType: "int"}
	a, err := factory.fromValue(int64(10), "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := factory.fromValue(int64(10), "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmp := compareCursor(a, b, "int"); cmp >= 0 {
		t.Fatalf("expected a < b due to tie, got %d", cmp)
	}
	if cmp := compareCursor(b, a, "int"); cmp <= 0 {
		t.Fatalf("expected b > a due to tie, got %d", cmp)
	}

	// property difference should dominate tie
	c, err := factory.fromValue(int64(11), "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmp := compareCursor(c, a, "int"); cmp <= 0 {
		t.Fatalf("expected property comparison to win, got %d", cmp)
	}
}
