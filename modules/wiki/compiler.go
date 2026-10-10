/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common/fingerprint"
	llmmodule "infini.sh/coco/modules/llm"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"github.com/tmc/langchaingo/llms"
)

// Knowledge compiler (W9, design doc 第七篇): turn a KB's source
// documents into curated wiki pages through a four-stage pipeline whose
// every output is a governance PROPOSAL — the human publishes, always.
//
//	MAP     batches of chunks → LLM extracts entities/concepts/claims
//	        (each carrying SourceChunkIDs), results cached by content
//	        fingerprint + model + schema fingerprint — unchanged chunks
//	        cost zero LLM on recompile
//	REDUCE  deterministic merge by normalized key (pure code, no LLM):
//	        same-key extracts across documents merge source lists
//	PLAN    reconcile against existing pages: names that already exist
//	        become UPDATE proposals, new ones CREATE
//	REFINE  one LLM writing pass per planned page — the draft IS the
//	        proposal payload, never a live article
//
// v1 scope: manual trigger, name-match reconciliation (embedding
// candidate matching when the page count justifies it), deterministic
// proposal anchors for idempotent re-runs.

const (
	compileChunkBatchRunes = 2000
	compileMaxDocs         = 200
	compileProposalType    = "knowledge_compile"
	compileFenceRe         = "(?s)```[a-zA-Z]*\\n?(.*?)```"
)

/* ---------------- extract types (MAP output) ---------------- */

type CompileExtract struct {
	Kind      string   `json:"kind"` // entity | concept | claim
	Name      string   `json:"name"`
	Aliases   []string `json:"aliases,omitempty"`
	Statement string   `json:"statement,omitempty"`
	DocID     string   `json:"-"` // stamped by the driver, not the model
	ChunkHint string   `json:"-"` // excerpt for evidence
}

type compileBatchResult struct {
	Fingerprint string           `json:"fingerprint"`
	Extracts    []CompileExtract `json:"extracts"`
}

// compileMapCache: batch fingerprint → cached result (zero LLM on
// recompile). Written through to badger so a restart (or a crashed
// compile hours into its MAP) resumes warm instead of re-paying the LLM;
// the in-memory map stays as the hot read path.
var (
	compileCacheMu    sync.RWMutex
	compileCache      = map[string][]CompileExtract{}
	compileCacheKVKey = "wiki-compile-map-cache"
)

// compileCacheGet looks the batch fingerprint up in memory, then in the
// persistent cache. Store trouble degrades to a miss — the compile pays
// the LLM again rather than failing.
func compileCacheGet(fp string) ([]CompileExtract, bool) {
	compileCacheMu.RLock()
	extracts, hit := compileCache[fp]
	compileCacheMu.RUnlock()
	if hit {
		return extracts, true
	}
	buf, err := kv.GetValue(compileCacheKVKey, []byte(fp))
	if err != nil || buf == nil {
		return nil, false
	}
	if err := util.FromJSONBytes(buf, &extracts); err != nil {
		log.Debugf("wiki compiler: decode cached MAP batch [%s] failed: %v", fp, err)
		return nil, false
	}
	compileCacheMu.Lock()
	compileCache[fp] = extracts
	compileCacheMu.Unlock()
	return extracts, true
}

// compileCachePut writes the batch result through to the store. A failed
// persist only costs a recompute after restart, so it logs and moves on.
func compileCachePut(fp string, extracts []CompileExtract) {
	compileCacheMu.Lock()
	compileCache[fp] = extracts
	compileCacheMu.Unlock()
	if err := kv.AddValueCompress(compileCacheKVKey, []byte(fp), util.MustToJSONBytes(extracts)); err != nil {
		log.Debugf("wiki compiler: persist MAP batch [%s] failed: %v", fp, err)
	}
}

const mapExtractPrompt = `You extract knowledge from document chunks for a wiki compiler.
Return ONLY a JSON array of extracts, each:
{"kind": "entity" | "concept", "name": "<canonical name>", "aliases": ["<alt names>"], "statement": "<one-sentence factual claim this source makes>"}
Rules: entities are named things (products, services, people, orgs); concepts are abstract topics.
Statements must be factual and attributable to THIS chunk. Max 8 per batch. No explanations, just the JSON array.`

// compileMAPFn is swappable in tests.
var compileMAPFn = compileMAP

// compileMAP runs one chunk batch through the LLM.
func compileMAP(ctx context.Context, model *core.ModelId, batchText string) ([]CompileExtract, error) {
	llm, err := langchain.SimplyGetLLM(model.ProviderID, model.ID, "")
	if err != nil {
		return nil, err
	}
	messages := []llms.MessageContent{
		langchain.SystemTextParts(mapExtractPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, batchText),
	}
	// the upstream occasionally answers 200 with a zero-byte completion
	// (observed under concurrent enrichment pressure) — retry: the batches
	// are idempotent reads, a flaky empty must not drop a document's
	// knowledge out of the compile
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
		extracts, err := compileMAPOnce(ctx, llm, messages)
		if err == nil {
			return extracts, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func compileMAPOnce(ctx context.Context, llm llms.Model, messages []llms.MessageContent) ([]CompileExtract, error) {
	// no token cap and stream like the enrich processors do: reasoning
	// models burn hundreds of tokens thinking before the JSON, a tight cap
	// leaves nothing but the thinking and the parse sees no array at all
	var out strings.Builder
	resp, err := llm.GenerateContent(ctx, messages,
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			out.Write(chunk)
			return nil
		}))
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty MAP response")
	}
	content := resp.Choices[0].Content
	if strings.TrimSpace(content) == "" {
		content = out.String()
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("empty MAP response (finish_reason=%v, choices=%d, stream_bytes=%d)",
			resp.Choices[0].StopReason, len(resp.Choices), out.Len())
	}
	return parseCompileExtracts(content)
}

var compileFencePattern = regexp.MustCompile(compileFenceRe)

func parseCompileExtracts(s string) ([]CompileExtract, error) {
	// reasoning models may inline their thinking — it must not leak into
	// the bracket slice (a stray "[" inside the think block breaks it)
	if i := strings.Index(s, "</think>"); i >= 0 {
		s = s[i+len("</think>"):]
	}
	if m := compileFencePattern.FindStringSubmatch(s); len(m) > 1 {
		s = m[1]
	}
	start, end := strings.Index(s, "["), strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		preview := strings.TrimSpace(s)
		if len(preview) > 300 {
			preview = preview[:300] + "…"
		}
		return nil, fmt.Errorf("no JSON array in MAP output, got: %q", preview)
	}
	var out []CompileExtract
	if err := json.Unmarshal([]byte(s[start:end+1]), &out); err != nil {
		return nil, err
	}
	valid := out[:0]
	for _, e := range out {
		if (e.Kind == "entity" || e.Kind == "concept") && strings.TrimSpace(e.Name) != "" {
			valid = append(valid, e)
		}
	}
	return valid, nil
}

/* ---------------- REDUCE (pure) ---------------- */

// ReducedKey is one merged knowledge unit after REDUCE.
type ReducedKey struct {
	Kind       string
	NormKey    string
	Name       string
	Aliases    map[string]bool
	Statements []ReducedStatement
}

// ReducedStatement carries its source for the evidence chain.
type ReducedStatement struct {
	DocID   string `json:"doc_id"`
	Excerpt string `json:"excerpt"`
	Text    string `json:"text"`
}

// NormalizeCompileKey: the deterministic identity of an extract.
// fingerprint.Normalize keeps single spaces; compile keys strip ALL
// whitespace — "支付 网关" and "支付网关" are the same label to a
// compiler even if they differ to a search index.
func NormalizeCompileKey(kind, name string) string {
	return kind + "|" + strings.ReplaceAll(fingerprint.Normalize(name), " ", "")
}

// ReduceExtracts merges MAP output by normalized key — pure code, no
// LLM, order-independent. Statements dedup by text hash per key.
func ReduceExtracts(extracts []CompileExtract) []ReducedKey {
	byKey := map[string]*ReducedKey{}
	order := []string{}
	for _, e := range extracts {
		key := NormalizeCompileKey(e.Kind, e.Name)
		rk, ok := byKey[key]
		if !ok {
			rk = &ReducedKey{Kind: e.Kind, NormKey: key, Name: e.Name, Aliases: map[string]bool{}}
			byKey[key] = rk
			order = append(order, key)
		}
		for _, a := range e.Aliases {
			if a = strings.TrimSpace(a); a != "" {
				rk.Aliases[a] = true
			}
		}
		if e.Statement != "" && e.DocID != "" {
			dup := false
			stmtHash := fingerprint.Normalize(e.Statement)
			for _, s := range rk.Statements {
				if fingerprint.Normalize(s.Text) == stmtHash {
					dup = true
					break
				}
			}
			if !dup {
				rk.Statements = append(rk.Statements, ReducedStatement{
					DocID: e.DocID, Excerpt: e.ChunkHint, Text: e.Statement,
				})
			}
		}
	}
	out := make([]ReducedKey, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	// most-evidence first — the compile capacity goes to what the corpus
	// actually talks about
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].Statements) > len(out[j].Statements) })
	return out
}

/* ---------------- PLAN (pure) ---------------- */

type CompilePlanAction int

const (
	PlanCreate CompilePlanAction = iota
	PlanUpdate
)

type CompilePlanItem struct {
	Action            CompilePlanAction
	Key               ReducedKey
	ExistingArticleID string
}

// PlanCompile reconciles reduced keys against the KB's existing entity
// pages: a page whose title (or alias set) matches the key's normalized
// name becomes an UPDATE; everything else is a CREATE. Name-match v1 —
// embedding candidates when page counts justify it.
func PlanCompile(reduced []ReducedKey, existing []core.WikiArticle) []CompilePlanItem {
	type existingEntry struct {
		id    string
		names map[string]bool // normalized title + aliases
	}
	entries := make([]existingEntry, 0, len(existing))
	for _, a := range existing {
		if a.PageType != core.WikiPageTypeEntity {
			continue
		}
		e := existingEntry{id: a.ID, names: map[string]bool{}}
		e.names[fingerprint.Normalize(a.Title)] = true
		// resolved wikilinks carry the aliases the page is known by
		for _, lp := range a.LinkedPages {
			if lp.Name != "" {
				e.names[fingerprint.Normalize(lp.Name)] = true
			}
		}
		entries = append(entries, e)
	}

	out := make([]CompilePlanItem, 0, len(reduced))
	for _, rk := range reduced {
		item := CompilePlanItem{Action: PlanCreate, Key: rk}
		for _, e := range entries {
			if e.names[fingerprint.Normalize(rk.Name)] {
				item.Action = PlanUpdate
				item.ExistingArticleID = e.id
				break
			}
			for alias := range rk.Aliases {
				if e.names[fingerprint.Normalize(alias)] {
					item.Action = PlanUpdate
					item.ExistingArticleID = e.id
					break
				}
			}
			if item.Action == PlanUpdate {
				break
			}
		}
		out = append(out, item)
	}
	return out
}

/* ---------------- driver ---------------- */

// compileKBHandler: POST /wiki/kb/:id/_compile — runs the pipeline
// against the KB's bound documents and files proposals.
func (h *APIHandler) compileKBHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")

	kb := core.WikiKnowledgeBase{}
	kb.ID = kbID
	kctx := orm.NewContextWithParent(req.Context())
	kctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	exists, err := orm.GetV2(kctx, &kb)
	if err != nil || !exists {
		h.WriteOpRecordNotFoundJSON(w, kbID)
		return
	}

	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		h.WriteError(w, "no default language model configured — the compiler needs one", http.StatusServiceUnavailable)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("wiki compiler: run panicked for kb [%s]: %v", kbID, r)
			}
		}()
		summary, err := runCompilePipeline(req.Context(), &kb, model)
		if err != nil {
			log.Warnf("wiki compiler: kb [%s] run failed: %v", kbID, err)
			return
		}
		log.Infof("wiki compiler: kb [%s] done — %d docs, %d batches, %d extracts, %d reduced, %d proposals (%d cached, %d failed batches)",
			kb.Name, summary.Docs, summary.Batches, summary.Extracts, summary.Reduced, summary.Proposals, summary.CachedBatches, summary.FailedBatches)
	}()

	h.WriteOKJSON(w, util.MapStr{
		"result": "compile started",
		"kb_id":  kbID,
		"note":   "output lands in the governance queue as proposals — publishing stays human",
	})
}

type compileSummary struct {
	Docs          int
	Batches       int
	Extracts      int
	Reduced       int
	Proposals     int
	CachedBatches int
	FailedBatches int
}

// runCompilePipeline: MAP (cached) → REDUCE → PLAN → REFINE → propose.
func runCompilePipeline(ctx context.Context, kb *core.WikiKnowledgeBase, model *core.ModelId) (*compileSummary, error) {
	summary := &compileSummary{}

	// source documents: the KB's bound datasources
	docs, err := compileSourceDocuments(ctx, kb)
	if err != nil {
		return nil, err
	}
	summary.Docs = len(docs)

	// MAP over chunk batches with fingerprint cache
	var allExtracts []CompileExtract
	for _, doc := range docs {
		batches := batchChunkText(doc.Content, compileChunkBatchRunes)
		summary.Batches += len(batches)
		for bi, batch := range batches {
			fp, _, okFp := fingerprint.Compute(fmt.Sprintf("%s|%s/%s|%s", model.ProviderID, model.ID, doc.ID, fingerprint.Normalize(batch)))
			if !okFp {
				continue
			}
			extracts, hit := compileCacheGet(fp)
			if hit {
				summary.CachedBatches++
				allExtracts = append(allExtracts, extracts...)
				continue
			}
			extracts, err := compileMAPFn(context.WithoutCancel(ctx), model, batch)
			if err != nil {
				summary.FailedBatches++
				if summary.FailedBatches <= 3 {
					log.Warnf("wiki compiler: MAP batch %s/%d failed (model %s/%s): %v", doc.ID, bi, model.ProviderID, model.ID, err)
				}
				continue
			}
			for i := range extracts {
				extracts[i].DocID = doc.ID
				extracts[i].ChunkHint = truncateCompile(batch, 200)
			}
			compileCachePut(fp, extracts)
			allExtracts = append(allExtracts, extracts...)
		}
	}
	summary.Extracts = len(allExtracts)
	if len(allExtracts) == 0 {
		return summary, nil
	}

	// REDUCE
	reduced := ReduceExtracts(allExtracts)
	summary.Reduced = len(reduced)

	// PLAN against existing entity pages
	existing := compileExistingArticles(ctx, kb.ID)
	plan := PlanCompile(reduced, existing)

	// REFINE → proposals (single-source concepts stay proposals too —
	// the noise gate applies at PUBLISH time, not compile time)
	pctx := orm.NewContext()
	pctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	for _, item := range plan {
		if summary.Proposals >= 50 {
			log.Warnf("wiki compiler: proposal cap 50 reached, truncating")
			break
		}
		if fileCompileProposal(pctx, kb, item, model) {
			summary.Proposals++
		}
	}
	return summary, nil
}

func compileSourceDocuments(ctx context.Context, kb *core.WikiKnowledgeBase) ([]core.Document, error) {
	if len(kb.DatasourceIDs) == 0 {
		return nil, nil
	}
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.Document{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(compileMaxDocs).
		Filter(orm.TermsQuery("source.id", kb.DatasourceIDs)).
		SortBy(orm.Sort{Field: "updated", SortType: orm.DESC}).
		Include("id", "title", "content"))
	if err != nil {
		return nil, err
	}
	docs, _, err := elastic.DecodeHits[core.Document](res)
	return docs, err
}

func compileExistingArticles(ctx context.Context, kbID string) []core.WikiArticle {
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiArticle{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("kb_id", kbID)))
	if err != nil {
		return nil
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		return nil
	}
	return articles
}

// fileCompileProposal files one compile proposal — the REFINE output in
// v1 is the deterministic evidence draft (statements + sources); the
// LLM writing pass lands when compiles are frequent enough to justify
// its cost. Anchored per key+generation for idempotent re-runs.
func fileCompileProposal(ctx *orm.Context, kb *core.WikiKnowledgeBase, item CompilePlanItem, model *core.ModelId) bool {
	key := item.Key
	actionLabel := "create"
	if item.Action == PlanUpdate {
		actionLabel = "update"
	}

	// evidence chain: every statement with its document id
	evidence := make([]interface{}, 0, len(key.Statements))
	for _, s := range key.Statements {
		evidence = append(evidence, util.MapStr{
			"doc_id":  s.DocID,
			"excerpt": s.Excerpt,
			"claim":   s.Text,
		})
	}

	proposal := &core.WikiGovernanceProposal{
		KbID:         kb.ID,
		Type:         compileProposalType,
		Status:       core.WikiGovernanceOpen,
		ArticleTitle: key.Name,
		Reason:       fmt.Sprintf("%s %s page from %d source(s)", actionLabel, key.Kind, len(key.Statements)),
	}
	proposal.ID = core.AnchoredProposalID("compile:"+kb.ID+":"+key.NormKey, compileProposalType,
		compileGeneration(ctx, kb.ID, key.NormKey))
	proposal.Evidence = util.MapStr{
		"cascade":     "knowledge-compile",
		"kind":        key.Kind,
		"norm_key":    key.NormKey,
		"aliases":     aliasList(key.Aliases),
		"statements":  evidence,
		"existing_id": item.ExistingArticleID,
		"model":       model.ProviderID + "/" + model.ID,
	}
	if err := orm.Create(ctx, proposal); err != nil {
		log.Debugf("wiki compiler: proposal create failed for [%s]: %v", key.Name, err)
		return false
	}
	// the human gate needs to know the gate has a new item — the KB owner
	// gets the same notification the scanner's proposals carry
	if kb.GetOwnerID() != "" {
		notifyOwner(kb.GetOwnerID(), "article", kb.ID, "governance",
			fmt.Sprintf("知识编译:%s %s 页提议(来自 %d 条来源主张)", actionLabel, key.Name, len(key.Statements)))
	}
	return true
}

func compileGeneration(ctx *orm.Context, kbID, normKey string) int {
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(100).
		Filter(orm.TermQuery("type", compileProposalType)))
	if err != nil || res == nil {
		return 0
	}
	proposals, _, derr := elastic.DecodeHits[core.WikiGovernanceProposal](res)
	if derr != nil {
		return 0
	}
	count := 0
	for _, p := range proposals {
		if p.KbID == kbID && p.Evidence["norm_key"] == normKey {
			count++
		}
	}
	return count
}

func aliasList(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// batchChunkText splits long content into rune-budget batches on line
// boundaries (falls back to hard cut).
func batchChunkText(text string, budget int) []string {
	if budget <= 0 || len([]rune(text)) <= budget {
		return []string{text}
	}
	var out []string
	current := strings.Builder{}
	currentRunes := 0
	for _, line := range strings.Split(text, "\n") {
		lineRunes := len([]rune(line)) + 1
		if currentRunes+lineRunes > budget && currentRunes > 0 {
			out = append(out, current.String())
			current.Reset()
			currentRunes = 0
		}
		current.WriteString(line)
		current.WriteString("\n")
		currentRunes += lineRunes
	}
	if current.Len() > 0 {
		out = append(out, current.String())
	}
	return out
}

func truncateCompile(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

/* ---------------- incremental trigger (W9) ---------------- */

// dirtyKBs accumulates KB ids whose sources changed since the last sweep.
// The debounce is coarse (5 min trailing edge) — compilation is expensive
// (LLM MAP per chunk batch) and document bursts should coalesce into one run.
var (
	compileMu       sync.Mutex
	compileDirty    = map[string]bool{}
	compileTimerSet bool
	compileTimer    *time.Timer
)

const compileDebounce = 5 * time.Minute

// MarkKBForCompile notes that a KB's source corpus changed and schedules a
// debounced compile run. Safe for concurrent callers; the first mark starts
// the timer, later marks before it fires just stay dirty.
func MarkKBForCompile(kbID string) {
	if kbID == "" {
		return
	}
	compileMu.Lock()
	compileDirty[kbID] = true
	needTimer := !compileTimerSet
	compileTimerSet = true
	compileMu.Unlock()

	if needTimer {
		time.AfterFunc(compileDebounce, func() {
			compileMu.Lock()
			dirty := compileDirty
			compileDirty = map[string]bool{}
			compileTimerSet = false
			compileMu.Unlock()

			for id := range dirty {
				runDebouncedCompile(id)
			}
		})
	}
}

// runDebouncedCompile runs the pipeline for one KB (best-effort, logged).
func runDebouncedCompile(kbID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("wiki compiler: debounced run panicked for kb [%s]: %v", kbID, r)
		}
	}()
	kb := core.WikiKnowledgeBase{}
	kb.ID = kbID
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil || !exists {
		return
	}
	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		log.Debugf("wiki compiler: debounced run skipped for [%s], no language model", kbID)
		return
	}
	summary, err := runCompilePipeline(ctx, &kb, model)
	if err != nil {
		log.Warnf("wiki compiler: debounced run failed for [%s]: %v", kbID, err)
		return
	}
	log.Infof("wiki compiler: debounced run for [%s] — %d extracts, %d reduced, %d proposals",
		kb.Name, summary.Extracts, summary.Reduced, summary.Proposals)
}
