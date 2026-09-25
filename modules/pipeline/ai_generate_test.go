/* Copyright © INFINI LTD.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package pipeline

import (
	"strings"
	"testing"

	"infini.sh/framework/core/config"
	fpipeline "infini.sh/framework/core/pipeline"
)

func TestExtractJSONArray(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // "" means no extraction
	}{
		{"bare array", `[{"a":{}}]`, `[{"a":{}}]`},
		{"fenced", "```json\n[{\"echo\":{}}]\n```", `[{"echo":{}}]`},
		{"think block", "<think>maybe [not this]</think>\n[{\"a\":{}}]", `[{"a":{}}]`},
		{"object wrapped", `{"chain": [{"a":{}}], "note": "x"}`, `[{"a":{}}]`},
		{"prose around", `Here is the chain:\n[{"a":{}}]\nhope it helps`, `[{"a":{}}]`},
		{"brackets in strings", `[{"pattern":"[INFO] %{DATA}"}]`, `[{"pattern":"[INFO] %{DATA}"}]`},
		{"unbalanced", `[{"a":{}`, ``},
		{"no array at all", "I cannot do that", ``},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractJSONArray(c.in)
			if got != c.want {
				t.Fatalf("extractJSONArray(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestBalancedJSONSpan(t *testing.T) {
	if s, ok := balancedJSONSpan(`[{"a":"b]c"},2] trailing`); !ok || s != `[{"a":"b]c"},2]` {
		t.Fatalf("string-internal bracket mishandled: %q %v", s, ok)
	}
}

func TestStudioSystemPromptContainsCatalog(t *testing.T) {
	fpipeline.RegisterProcessorPlugin("studio_prompt_probe", func(cfg *config.Config) (fpipeline.Processor, error) {
		return &studioStubProcessor{name: "studio_prompt_probe", mutate: func(pctx *fpipeline.Context) error { return nil }}, nil
	})
	prompt := studioSystemPrompt()
	if !strings.Contains(prompt, "studio_prompt_probe") {
		t.Fatal("prompt should list the registered processor")
	}
	if !strings.Contains(prompt, "JSON array") {
		t.Fatal("prompt should state the output contract")
	}
}

func TestTruncateRunes(t *testing.T) {
	in := strings.Repeat("界", 10)
	out := truncateRunes(in, 5)
	if len([]rune(strings.TrimSuffix(out, "…"))) != 5 {
		t.Fatalf("rune truncation broke: %q", out)
	}
	if truncateRunes("short", 10) != "short" {
		t.Fatal("short strings must pass through")
	}
}
