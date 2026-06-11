package tools

import (
	"strings"
	"testing"
)

func TestFormatToolCallChunkIncludesOnlyToolFields(t *testing.T) {
	chunk := formatToolCallChunk("search", `{"query":"coco"}`, "found result")

	for _, expected := range []string{
		"* search",
		"Arguments:",
		`"query": "coco"`,
		"Output:",
		"found result",
	} {
		if !strings.Contains(chunk, expected) {
			t.Fatalf("expected chunk to contain %q, got %q", expected, chunk)
		}
	}

	for _, unexpected := range []string{"Thought:", "Final Answer:", "AI:"} {
		if strings.Contains(chunk, unexpected) {
			t.Fatalf("expected chunk to omit model output marker %q, got %q", unexpected, chunk)
		}
	}
}
