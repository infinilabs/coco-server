package langchain

import (
	"testing"
	"time"
)

func TestPromptWithTimeAppendsCurrentTime(t *testing.T) {
	now := time.Date(2026, 6, 11, 17, 12, 0, 0, time.Local)
	got := promptWithTime("You are a helpful assistant.", now)
	want := "You are a helpful assistant.\n\nThe current time is June 11, 2026 17:12."
	if got != want {
		t.Fatalf("unexpected system prompt:\nwant: %q\n got: %q", want, got)
	}
}

func TestPromptWithTimeHandlesEmptyPrompt(t *testing.T) {
	now := time.Date(2026, 6, 11, 17, 12, 0, 0, time.Local)
	got := promptWithTime("  ", now)
	want := "The current time is June 11, 2026 17:12."
	if got != want {
		t.Fatalf("unexpected empty system prompt:\nwant: %q\n got: %q", want, got)
	}
}
