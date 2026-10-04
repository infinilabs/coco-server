/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/tmc/langchaingo/llms"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	llmmodule "infini.sh/coco/modules/llm"
)

// Query rewrite leg (D8): the vocabulary-mismatch problem. Users ask in
// colloquial language ("十三薪"), documents answer in formal vocabulary
// ("年终双薪") — BM25 cannot bridge that gap and the semantic leg only
// partially does. A cheap language model rewrites the query into document
// language and the rewritten text runs as one more keyword route in the RRF
// fusion; the original routes keep running, so a bad rewrite can only add
// candidates, never remove them. No model, a timeout or any failure
// degrades to the leg simply not firing — the same fallback discipline as
// the semantic and rerank legs.

const (
	// rewriteTimeout caps the model call; past it the leg degrades instead
	// of stalling the search. Cache hits return in microseconds.
	rewriteTimeout = 2500 * time.Millisecond
	// rewriteCacheTTL keeps repeated colloquial phrasings from re-billing
	// the model — the article's "cache the rewrite" discipline.
	rewriteCacheTTL = 24 * time.Hour
	// rewriteCacheMax bounds the cache; overflow flushes it whole (the
	// cache is a warm-up optimization, not a registry).
	rewriteCacheMax = 4096
	// rewriteBreakerCooldown is how long the leg stays off after repeated
	// failures, so a down model does not add a timeout to every search.
	rewriteBreakerCooldown  = time.Minute
	rewriteBreakerThreshold = 3
	rewriteMinQueryLen      = 2
	rewriteMaxQueryLen      = 256
	rewriteMaxOutputLen     = 512
	rewriteMaxOutputTokens  = 128
)

// rewriteQueryEnabled is the global kill switch (SEARCH_QUERY_REWRITE=off).
var rewriteQueryEnabled = envFlagEnabled("SEARCH_QUERY_REWRITE", true)

// envFlagEnabled reads a boolean flag with the shared off vocabulary
// ("off"/"disabled"/"0"/"false"), defaulting to def when unset.
func envFlagEnabled(key string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "off", "disabled", "0", "false":
		return false
	case "on", "enabled", "1", "true":
		return true
	}
	return def
}

var (
	// resolveRewriteModelFn and rewriteCallModelFn are swapped out in tests
	// (the real paths touch the provider store and the network).
	resolveRewriteModelFn = func() *core.ModelId {
		return llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	}
	rewriteCallModelFn = rewriteCallModel
)

// rewriteResult reports what the rewrite leg did. Query carries the text to
// search with (the rewrite when applied, the original otherwise).
type rewriteResult struct {
	Applied  bool
	Query    string
	Original string
	Cached   bool
	TookMS   int64
	Note     string // why the leg did not fire
}

// rewriteSystemPrompt asks for a bare rewritten query — same intent,
// document vocabulary, no commentary.
const rewriteSystemPrompt = `你是企业知识库的检索查询改写器。用户提问常用口语,而文档使用正式词汇。请把提问改写成文档中更可能出现的正式表述:口语词换成同义术语、缩写展开、口语转书面语。
要求:
- 只输出改写后的查询文本本身,一行,不要解释,不要加引号或前缀
- 不改变提问意图,不添加原文没有的限定条件
- 保留原文中已经是正式术语的部分
- 如果提问已经是正式表述,原样输出`

// rewriteSearchQuery resolves the rewrite for one query: cheap guards, the
// cache, then the model call behind a timeout. Failures never propagate —
// the caller keeps the original query and the fusion runs without the leg.
func rewriteSearchQuery(ctx context.Context, query string) rewriteResult {
	started := time.Now()
	res := rewriteResult{Query: query, Original: query}
	if !rewriteQueryEnabled {
		res.Note = "query rewrite disabled (SEARCH_QUERY_REWRITE=off)"
		return res
	}
	runes := utf8.RuneCountInString(query)
	if runes < rewriteMinQueryLen || runes > rewriteMaxQueryLen {
		res.Note = "query length outside the rewrite range"
		return res
	}
	if rewriteBreakerOpen() {
		res.Note = "query rewrite cooling down after repeated failures"
		return res
	}
	m := resolveRewriteModelFn()
	if m == nil || m.ProviderID == "" || m.ID == "" {
		res.Note = "no default language model configured, rewrite leg off"
		return res
	}

	key := normalizeSearchQuery(query)
	if entry, ok := rewriteCacheGet(key); ok {
		res.Query, res.Applied, res.Cached = entry.query, entry.applied, true
		res.TookMS = time.Since(started).Milliseconds()
		return res
	}

	callCtx, cancel := context.WithTimeout(ctx, rewriteTimeout)
	defer cancel()
	rewritten, err := rewriteCallModelFn(callCtx, m, query)
	res.TookMS = time.Since(started).Milliseconds()
	if err != nil {
		rewriteBreakerFail()
		res.Note = fmt.Sprintf("query rewrite failed: %v", err)
		return res
	}
	rewriteBreakerOK()

	rewritten = sanitizeRewrite(rewritten)
	if rewritten == "" || normalizeSearchQuery(rewritten) == key {
		res.Note = "rewrite unchanged, original kept"
		rewriteCachePut(key, rewriteCacheEntry{query: query, applied: false, expires: time.Now().Add(rewriteCacheTTL)})
		return res
	}
	res.Query, res.Applied = rewritten, true
	rewriteCachePut(key, rewriteCacheEntry{query: rewritten, applied: true, expires: time.Now().Add(rewriteCacheTTL)})
	return res
}

// rewriteCallModel runs the rewrite through the default language model
// (openai-compatible or ollama, whichever the provider speaks).
func rewriteCallModel(ctx context.Context, m *core.ModelId, query string) (string, error) {
	llm, err := langchain.SimplyGetLLM(m.ProviderID, m.ID, "")
	if err != nil {
		return "", err
	}
	messages := []llms.MessageContent{
		langchain.SystemTextParts(rewriteSystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, query),
	}
	resp, err := llm.GenerateContent(ctx, messages, llms.WithMaxTokens(rewriteMaxOutputTokens))
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty rewrite response")
	}
	return resp.Choices[0].Content, nil
}

var (
	rewriteThinkPattern  = regexp.MustCompile(`(?s)<think>.*?</think>`)
	rewriteFencePattern  = regexp.MustCompile("(?s)```[a-zA-Z]*\\n?(.*?)```")
	rewritePrefixPattern = regexp.MustCompile(`(?i)^(改写后|改写|rewritten? query|rewrite)\s*[:：]\s*`)
)

// sanitizeRewrite strips the debris small models leave around the answer:
// reasoning blocks, markdown fences, "改写:" prefixes and wrapping quotes.
// An output longer than the cap is rejected (a rewrite that long is
// commentary, not a query).
func sanitizeRewrite(text string) string {
	text = rewriteThinkPattern.ReplaceAllLiteralString(text, "")
	if m := rewriteFencePattern.FindStringSubmatch(text); m != nil {
		text = m[1]
	}
	text = rewritePrefixPattern.ReplaceAllLiteralString(text, "")
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "\"'`“”‘’「」《《》》")
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > rewriteMaxOutputLen {
		return ""
	}
	return text
}

// --- rewrite cache (bounded, TTL, process-local) ---

type rewriteCacheEntry struct {
	query   string
	applied bool
	expires time.Time
}

var (
	rewriteCacheMu sync.Mutex
	rewriteCache   = map[string]rewriteCacheEntry{}
)

func rewriteCacheGet(key string) (rewriteCacheEntry, bool) {
	rewriteCacheMu.Lock()
	defer rewriteCacheMu.Unlock()
	entry, ok := rewriteCache[key]
	if !ok {
		return entry, false
	}
	if time.Now().After(entry.expires) {
		delete(rewriteCache, key)
		return rewriteCacheEntry{}, false
	}
	return entry, true
}

func rewriteCachePut(key string, entry rewriteCacheEntry) {
	rewriteCacheMu.Lock()
	defer rewriteCacheMu.Unlock()
	if len(rewriteCache) >= rewriteCacheMax {
		now := time.Now()
		for k, v := range rewriteCache {
			if now.After(v.expires) {
				delete(rewriteCache, k)
			}
		}
		// still full after purging expired entries: flush whole — the cache
		// refills from traffic, bounded memory beats perfect retention
		if len(rewriteCache) >= rewriteCacheMax {
			rewriteCache = map[string]rewriteCacheEntry{}
		}
	}
	rewriteCache[key] = entry
}

// rewriteCacheReset drops the cache (tests).
func rewriteCacheReset() {
	rewriteCacheMu.Lock()
	defer rewriteCacheMu.Unlock()
	rewriteCache = map[string]rewriteCacheEntry{}
}

// --- failure breaker (a down model must not tax every search) ---

var rewriteBreaker struct {
	mu        sync.Mutex
	fails     int
	openUntil time.Time
}

func rewriteBreakerOpen() bool {
	rewriteBreaker.mu.Lock()
	defer rewriteBreaker.mu.Unlock()
	return time.Now().Before(rewriteBreaker.openUntil)
}

func rewriteBreakerFail() {
	rewriteBreaker.mu.Lock()
	defer rewriteBreaker.mu.Unlock()
	rewriteBreaker.fails++
	if rewriteBreaker.fails >= rewriteBreakerThreshold {
		rewriteBreaker.openUntil = time.Now().Add(rewriteBreakerCooldown)
		rewriteBreaker.fails = 0
	}
}

func rewriteBreakerOK() {
	rewriteBreaker.mu.Lock()
	defer rewriteBreaker.mu.Unlock()
	rewriteBreaker.fails = 0
}

// rewriteBreakerReset clears the breaker (tests).
func rewriteBreakerReset() {
	rewriteBreaker.mu.Lock()
	defer rewriteBreaker.mu.Unlock()
	rewriteBreaker.fails = 0
	rewriteBreaker.openUntil = time.Time{}
}
