/* Copyright © INFINI LTD.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/llms"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	fpipeline "infini.sh/framework/core/pipeline"
	"infini.sh/framework/core/util"
	pipemod "infini.sh/framework/modules/pipeline"
)

const (
	maxAIDocuments      = 5
	maxAIDocumentChars  = 4000
	maxAIRequirementLen = 2000
)

var (
	mdFenceRe   = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")
	thinkBlock  = regexp.MustCompile("(?s)<think>.*?</think>")
	removeThink = regexp.MustCompile(`(?s)<think>.*?</think>`)
)

// resolveStudioLLM resolves the default language model for chain generation.
// Package var so tests can stub the LLM (same pattern as wiki).
var resolveStudioLLM = func() (llms.Model, error) {
	modelId := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if modelId == nil {
		return nil, fmt.Errorf("no language model configured: configure a default language model in settings")
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		return nil, err
	}
	return langchain.GetLLM(provider.BaseURL, provider.APIType, modelId.ID, provider.APIKey, ""), nil
}

// callStudioLLM runs one round-trip and returns the think-stripped text —
// streamed chunks accumulate, with a fallback to response choices for
// providers that never invoke the streaming callback.
func callStudioLLM(ctx context.Context, llm llms.Model, system, user string) (string, error) {
	messages := []llms.MessageContent{
		langchain.SystemTextParts(system),
		llms.TextParts(llms.ChatMessageTypeHuman, user),
	}
	var builder strings.Builder
	resp, err := llm.GenerateContent(ctx, messages, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
		builder.Write(chunk)
		return nil
	}))
	if err != nil {
		return "", err
	}
	if builder.Len() == 0 && resp != nil {
		for _, choice := range resp.Choices {
			builder.WriteString(choice.Content)
		}
	}
	return removeThink.ReplaceAllLiteralString(builder.String(), ""), nil
}

// studioNamingConvention is the field naming contract sent with every
// generation request. Documents (not log lines) flow through coco pipelines,
// so the original document fields are the contract's protected core.
const studioNamingConvention = `Field addressing rules (must follow):
- Records follow the document codec: the payload tree rides under document.* — address fields as document.title, document.content, document.source.id, ...
- New fields use snake_case, lowercase; no camelCase or hyphens
- Always keep the original document.* fields — never delete or rename them
- Language detection: document.lang; tags: document.tags (array of strings); summary: document.summary
- Extracted values use semantic names (document.author, document.category), never field1/value2
- Only add fields a later consumer would query or filter by; do not duplicate content`

// studioSystemPrompt renders the instruction block including the live
// processor catalog, so the model can only propose processors that actually
// exist in this build.
func studioSystemPrompt() string {
	var catalog strings.Builder
	meta := fpipeline.GetProcessorMetadata()
	for name, v := range meta {
		entry, _ := v.(util.MapStr)
		category := ""
		var keys []string
		if entry != nil {
			if c, ok := entry["category"].(string); ok {
				category = c
			}
			if props, ok := entry["properties"].(util.MapStr); ok {
				for k := range props {
					keys = append(keys, k)
				}
			} else if props, ok := entry["properties"].(map[string]interface{}); ok {
				for k := range props {
					keys = append(keys, k)
				}
			}
		}
		sortStrings(keys)
		catalog.WriteString("- ")
		catalog.WriteString(name)
		if category != "" {
			catalog.WriteString(" (category: " + category + ")")
		}
		if len(keys) > 0 {
			catalog.WriteString(" config keys: " + strings.Join(keys, ", "))
		}
		catalog.WriteString("\n")
	}

	return "You design document processing pipelines for Coco. " +
		"A pipeline is a JSON array of processor entries. Each entry is either " +
		`{"processor_name": {config}} (exactly one key) or {"if": <condition>, "then": [...], "else": [...]} for conditional branching.` + "\n\n" +
		"Available processors in this system:\n" + catalog.String() + "\n" +
		studioNamingConvention + "\n\n" +
		"Output rules (must follow):\n" +
		"- Reply with ONLY a JSON array of processor entries — no explanations, no markdown fences\n" +
		"- Use only processors from the list above, with valid config keys\n" +
		"- Prefer the fewest processors that satisfy the requirement"
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// extractJSONArray pulls the first complete JSON array text out of a model
// reply: strips inline <think> blocks, unwraps markdown fences, accepts
// object-wrapped arrays like {"chain": [...]}, and finally falls back to a
// bracket-balanced scan that ignores brackets inside string literals.
func extractJSONArray(resp string) string {
	s := strings.TrimSpace(resp)
	if strings.Contains(s, "<think>") {
		s = thinkBlock.ReplaceAllString(s, "")
		s = strings.TrimSpace(s)
	}
	if m := mdFenceRe.FindStringSubmatch(s); m != nil {
		s = strings.TrimSpace(m[1])
	}
	if strings.HasPrefix(s, "[") {
		if raw, ok := balancedJSONSpan(s); ok {
			return raw
		}
	}
	if strings.HasPrefix(s, "{") {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(s), &obj); err == nil {
			for _, v := range obj {
				if arr, ok := v.([]interface{}); ok {
					if b, err := json.Marshal(arr); err == nil {
						return string(b)
					}
				}
			}
		}
	}
	if idx := strings.IndexByte(s, '['); idx >= 0 {
		if raw, ok := balancedJSONSpan(s[idx:]); ok {
			return raw
		}
	}
	return ""
}

// balancedJSONSpan returns the first balanced JSON span of s (which must
// start with [ or {), skipping brackets inside string literals and escapes.
func balancedJSONSpan(s string) (string, bool) {
	depth, inStr, esc := 0, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth == 0 {
				return s[:i+1], true
			}
		}
	}
	return "", false
}

func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// cocoChainGenerator adapts coco's LLM chain generation (prompts, catalog,
// extraction) to the framework ChainGenerator contract: the unified
// POST /pipeline/ai-generate endpoint owns validation, replay is the
// caller's /pipeline/dry-run follow-up.
type cocoChainGenerator struct{}

func (cocoChainGenerator) Name() string { return "coco" }

func (cocoChainGenerator) Generate(genReq pipemod.ChainGenerateRequest) ([]map[string]interface{}, error) {
	sampleDocs := genReq.Documents

	llm, err := resolveStudioLLM()
	if err != nil {
		return nil, err
	}

	var user strings.Builder
	user.WriteString("Sample documents (one JSON object per block):\n")
	for i, doc := range sampleDocs {
		user.WriteString(fmt.Sprintf("--- document %d ---\n%s\n", i+1, truncateRunes(util.MustToJSON(doc), maxAIDocumentChars)))
	}
	if requirement := truncateRunes(strings.TrimSpace(genReq.Requirements), maxAIRequirementLen); requirement != "" {
		user.WriteString("\nRequirement:\n" + requirement + "\n")
	}
	if genReq.Mode == "refine" {
		user.WriteString("\nCurrent chain (edit this chain to satisfy the requirement; keep the parts not related to it):\n")
		user.WriteString(util.MustToJSON(genReq.CurrentChain))
		user.WriteString("\n")
	}

	system := studioSystemPrompt()
	reply, err := callStudioLLM(genReq.Context, llm, system, user.String())
	if err != nil {
		return nil, fmt.Errorf("AI chat failed: %s", truncateRunes(err.Error(), 300))
	}

	// Extract, and give the model one corrective retry when the reply has no
	// parseable array.
	raw := extractJSONArray(reply)
	if raw == "" {
		correction := "Your previous reply could not be parsed as a JSON array of processor entries. " +
			"Reply again with ONLY the JSON array, no prose, no markdown fences."
		reply2, err2 := callStudioLLM(genReq.Context, llm, system, user.String()+"\n\n"+correction)
		if err2 != nil {
			return nil, fmt.Errorf("AI chat failed: %s", truncateRunes(err2.Error(), 300))
		}
		raw = extractJSONArray(reply2)
		if raw == "" {
			return nil, fmt.Errorf("AI response contains no valid JSON array after retry: %s", truncateRunes(reply, 300))
		}
	}

	chain := []map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &chain); err != nil {
		return nil, fmt.Errorf("AI response is not a processor array: %s", truncateRunes(err.Error(), 300))
	}
	return chain, nil
}

func init() {
	if err := pipemod.RegisterChainGeneratorOrReplace(cocoChainGenerator{}); err != nil {
		panic(err)
	}
}
