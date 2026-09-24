/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	"infini.sh/coco/modules/common"
	llmmodule "infini.sh/coco/modules/llm"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// KM agent generation (design doc §5.1, stage C). The pipeline is a plain
// sequential scope→cluster→outline→draft→assemble→deliver flow over the
// KB's scoped documents; pages are drafted through a bounded worker pool
// and delivered as draft articles — the review gate stays on
// PUT /wiki/article/:id/status (D1: AI never publishes).

const (
	defaultGeneratePages       = 5
	defaultGenerateScopeDocs   = 30
	defaultGenerateDocChars    = 4000
	defaultGenerateConcurrency = 2
	maxGeneratePages           = 20
)

var citationPattern = regexp.MustCompile(`\[(\d+)\]`)

var removeThinkPattern = regexp.MustCompile(`(?s)<think>.*?</think>`)

// resolveLanguageLLM resolves the model used by KM generation: explicit
// provider/model, else the server default language model. Package var so
// tests can stub the LLM.
var resolveLanguageLLM = func(providerID, model string) (llms.Model, error) {
	modelId := llmmodule.ResolveModel(core.LLMTypeLanguage, &core.ModelId{ProviderID: providerID, ID: model})
	if modelId == nil {
		return nil, fmt.Errorf("no language model configured: pass model_provider/model in the request or configure a default language model in settings")
	}
	provider, err := common.GetModelProvider(modelId.ProviderID)
	if err != nil {
		return nil, err
	}
	return langchain.GetLLM(provider.BaseURL, provider.APIType, modelId.ID, provider.APIKey, ""), nil
}

func defaultGenerationLang() string {
	// AppConfig reads settings from kv, whose handler is only registered
	// in a booted server; outside one (unit tests) fall back to en-US
	lang := ""
	func() {
		defer func() { _ = recover() }()
		if appCfg := common.AppConfig(); appCfg.DocumentProcessing != nil {
			lang = appCfg.DocumentProcessing.LLMGenerationLanguage
		}
	}()
	if lang != "" {
		return lang
	}
	return "en-US"
}

/* ---------------- SSE stream ---------------- */

// sseStream writes the KM agent event contract (design doc §5.1):
//
//	event: progress data: {phase, phaseCurrent, phaseTotal, progress, eta}
//	event: article  data: {…WikiArticle}
//	event: done     data: {generated, updated, failed}
type sseStream struct {
	w   http.ResponseWriter
	f   http.Flusher
	mu  sync.Mutex
	ctx context.Context
}

func newSSEStream(w http.ResponseWriter, ctx context.Context) (*sseStream, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("http.Flusher not supported")
	}
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &sseStream{w: w, f: flusher, ctx: ctx}, nil
}

func (s *sseStream) emit(event string, data interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, util.MustToJSON(data)); err != nil {
		return err
	}
	s.f.Flush()
	return nil
}

type progressEvent struct {
	Phase        string  `json:"phase"`
	PhaseCurrent int     `json:"phaseCurrent"`
	PhaseTotal   int     `json:"phaseTotal"`
	Progress     float64 `json:"progress"`
	Eta          int64   `json:"eta"` // seconds, rough
}

type doneEvent struct {
	Generated int      `json:"generated"`
	Updated   int      `json:"updated"`
	Failed    []string `json:"failed,omitempty"`
	Reason    string   `json:"reason,omitempty"`
	// token budget placeholder until WS6 usage tracking lands; char counts
	// approximate the per-page budget accounting
	Usage *llmUsage `json:"usage,omitempty"`
}

/* ---------------- LLM helpers ---------------- */

// llmUsage accumulates prompt/completion sizes across a generation run.
type llmUsage struct {
	PromptChars     int64 `json:"prompt_chars"`
	CompletionChars int64 `json:"completion_chars"`
}

func (u *llmUsage) add(promptLen, completionLen int) {
	if u == nil {
		return
	}
	u.PromptChars += int64(promptLen)
	u.CompletionChars += int64(completionLen)
}

// callLLM runs one round-trip and returns the think-stripped text. The
// accumulated text prefers streamed chunks and falls back to the response
// choices when the model doesn't drive the streaming callback.
func callLLM(ctx context.Context, llm llms.Model, system, user string, usage *llmUsage) (string, error) {
	message := []llms.MessageContent{
		langchain.SystemTextParts(system),
		llms.TextParts(llms.ChatMessageTypeHuman, user),
	}
	var builder strings.Builder
	resp, err := llm.GenerateContent(ctx, message, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
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
	usage.add(len(system)+len(user), builder.Len())
	return removeThinkPattern.ReplaceAllLiteralString(builder.String(), ""), nil
}

// decodeLLMJSON unmarshals the first JSON object embedded in an LLM reply
// (markdown fences and surrounding prose tolerated).
func decodeLLMJSON[T any](raw string, out *T) error {
	trimmed := strings.TrimSpace(raw)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || start > end {
		return fmt.Errorf("no JSON object found in LLM response")
	}
	if err := json.Unmarshal([]byte(trimmed[start:end+1]), out); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

/* ---------------- pipeline types ---------------- */

type generationOptions struct {
	MaxPages     int
	MaxScopeDocs int
	MaxDocChars  int
	Concurrency  int
	Lang         string
	Hint         string
}

func (o *generationOptions) applyDefaults() {
	if o.MaxPages <= 0 {
		o.MaxPages = defaultGeneratePages
	}
	if o.MaxPages > maxGeneratePages {
		o.MaxPages = maxGeneratePages
	}
	if o.MaxScopeDocs <= 0 {
		o.MaxScopeDocs = defaultGenerateScopeDocs
	}
	if o.MaxDocChars <= 0 {
		o.MaxDocChars = defaultGenerateDocChars
	}
	if o.Concurrency <= 0 {
		o.Concurrency = defaultGenerateConcurrency
	}
	if o.Lang == "" {
		o.Lang = defaultGenerationLang()
	}
}

// pageCandidate is one planned page from the cluster phase.
type pageCandidate struct {
	Title    string `json:"title"`
	PageType string `json:"page_type"`
	Subtype  string `json:"subtype,omitempty"`
	Reason   string `json:"reason,omitempty"`
	DocRefs  []int  `json:"doc_refs"` // 1-based indexes into the scoped docs
}

type outlineResponse struct {
	Summary  string        `json:"summary"`
	Chapters []chapterPlan `json:"chapters"`
}

type chapterPlan struct {
	Title     string   `json:"title"`
	KeyPoints []string `json:"key_points,omitempty"`
}

// draftedPage is a fully assembled page waiting for delivery.
type draftedPage struct {
	candidate  pageCandidate
	summary    string
	content    string
	sources    []core.WikiSourceReference
	confidence string
	citedDocs  int
}

/* ---------------- phase 1: scope ---------------- */

// scopeKbDocuments returns the KB's freshest documents (§5.1 phase 1:
// kb-bound datasources in coco_document, freshness-ordered sample when
// over the cap).
func scopeKbDocuments(ctx *orm.Context, kb *core.WikiKnowledgeBase, limit int) ([]core.Document, error) {
	if len(kb.DatasourceIDs) == 0 {
		return nil, nil
	}
	orm.WithModel(ctx, &core.Document{})
	builder := orm.NewQuery().Size(limit).
		Filter(orm.TermsQuery("source.id", kb.DatasourceIDs)).
		SortBy(orm.Sort{Field: "updated", SortType: orm.DESC})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, err
	}
	docs, _, err := elastic.DecodeHits[core.Document](res)
	return docs, err
}

/* ---------------- phase 2: cluster ---------------- */

// clusterPages asks the LLM for a complementary page plan over the scoped
// documents (PlannerNode rewritten for a doc set instead of a query).
func clusterPages(ctx context.Context, llm llms.Model, docs []core.Document, opts generationOptions, usage *llmUsage) ([]pageCandidate, error) {
	var listing strings.Builder
	for i := range docs {
		doc := &docs[i]
		summary := doc.Summary
		if summary == "" {
			summary = doc.AiInsights.Text
		}
		if len(summary) > 200 {
			summary = summary[:200]
		}
		updated := ""
		if doc.Updated != nil {
			updated = doc.Updated.Format(time.RFC3339)
		}
		fmt.Fprintf(&listing, "[%d] %s — %s (source: %s, updated: %s)\n",
			i+1, doc.Title, summary, doc.Source.Name, updated)
	}

	userPrompt := fmt.Sprintf(
		"Plan wiki pages for a knowledge base from the documents below.\n\n"+
			"Requirements:\n"+
			"- Return ONLY a valid JSON object, no markdown fences\n"+
			"- Plan at most %d pages covering complementary topics; do not split one topic across pages\n"+
			"- \"page_type\" is one of: concept (default), entity (a specific thing worth its own page), source (a document-level summary page)\n"+
			"- \"doc_refs\": the [n] indexes of documents this page should be built from, at least 1\n"+
			"- Titles must be in %s\n"+
			"%s"+
			"Format:\n"+
			`{"pages":[{"title":"","page_type":"concept","subtype":"","reason":"","doc_refs":[1]}]}`+"\n\n"+
			"Documents:\n%s\n\n"+
			"Generate the JSON object now.",
		opts.MaxPages, opts.Lang, clusterHintLine(opts.Hint), listing.String())

	system := "You are a knowledge-management agent planning wiki knowledge-base pages. Your response MUST be in " + opts.Lang + "."
	raw, err := callLLM(ctx, llm, system, userPrompt, usage)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Pages []pageCandidate `json:"pages"`
	}
	if err := decodeLLMJSON(raw, &parsed); err != nil {
		return nil, err
	}
	return validateCandidates(parsed.Pages, docs, opts.MaxPages), nil
}

func clusterHintLine(hint string) string {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return ""
	}
	return fmt.Sprintf("- Focus the plan on: %s\n", hint)
}

// validateCandidates normalizes LLM output: clamps page types, drops
// candidates without usable doc refs, dedupes titles and caps the count.
func validateCandidates(pages []pageCandidate, docs []core.Document, maxPages int) []pageCandidate {
	validTypes := map[string]bool{
		core.WikiPageTypeEntity:  true,
		core.WikiPageTypeConcept: true,
		core.WikiPageTypeSource:  true,
	}
	seen := map[string]bool{}
	out := make([]pageCandidate, 0, len(pages))
	for _, page := range pages {
		page.Title = strings.TrimSpace(page.Title)
		if page.Title == "" {
			continue
		}
		key := strings.ToLower(page.Title)
		if seen[key] {
			continue
		}
		if !validTypes[page.PageType] {
			page.PageType = core.WikiPageTypeConcept
		}
		refs := make([]int, 0, len(page.DocRefs))
		for _, ref := range page.DocRefs {
			if ref >= 1 && ref <= len(docs) {
				refs = append(refs, ref)
			}
		}
		if len(refs) == 0 {
			continue
		}
		page.DocRefs = refs
		seen[key] = true
		out = append(out, page)
		if len(out) >= maxPages {
			break
		}
	}
	return out
}

/* ---------------- phase 3+4: outline & draft ---------------- */

// docMaterial renders one document as citation-indexed LLM material.
func docMaterial(doc *core.Document, index, maxChars int) string {
	body := doc.Content
	if body == "" {
		body = doc.Summary
	}
	if len(body) > maxChars {
		body = body[:maxChars]
	}
	return fmt.Sprintf("[%d] %s\n%s", index, doc.Title, body)
}

// outlinePage generates the page summary and chapter plan (§5.1 phase 3,
// ChapterOutline shape).
func outlinePage(ctx context.Context, llm llms.Model, candidate pageCandidate, docs []core.Document, opts generationOptions, usage *llmUsage) (*outlineResponse, error) {
	material := joinDocMaterials(candidate, docs, opts.MaxDocChars)
	userPrompt := fmt.Sprintf(
		"Design the chapter outline of a wiki page.\n\n"+
			"Page title: %s (page_type: %s)\n\n"+
			"Requirements:\n"+
			"- Return ONLY a valid JSON object, no markdown fences\n"+
			"- 2 to 5 chapters; for entity pages prefer: overview/definition, key characteristics, relationships & context, sources\n"+
			"- \"summary\": 2 sentences summarizing the page, in %s\n"+
			"- Chapter titles in %s\n"+
			"Format:\n"+
			`{"summary":"","chapters":[{"title":"","key_points":[""]}]}`+"\n\n"+
			"Source documents:\n%s\n\n"+
			"Generate the JSON object now.",
		candidate.Title, candidate.PageType, opts.Lang, opts.Lang, material)
	raw, err := callLLM(ctx, llm, "You are a knowledge-management agent drafting wiki page outlines. Your response MUST be in "+opts.Lang+".", userPrompt, usage)
	if err != nil {
		return nil, err
	}
	outlined := &outlineResponse{}
	if err := decodeLLMJSON(raw, outlined); err != nil {
		return nil, err
	}
	if len(outlined.Chapters) == 0 {
		return nil, fmt.Errorf("empty outline for page %q", candidate.Title)
	}
	return outlined, nil
}

func joinDocMaterials(candidate pageCandidate, docs []core.Document, maxChars int) string {
	parts := make([]string, 0, len(candidate.DocRefs))
	for _, ref := range candidate.DocRefs {
		parts = append(parts, docMaterial(&docs[ref-1], ref, maxChars))
	}
	return strings.Join(parts, "\n\n")
}

// draftPage runs outline + per-chapter drafting + assembly and returns the
// page with only cited sources (§5.1 phases 3–5). Chapters that produced
// no citation are dropped: their assertions have no verifiable origin
// (design doc C2, 无引用的断言剔除).
func draftPage(ctx context.Context, llm llms.Model, candidate pageCandidate, docs []core.Document, opts generationOptions, usage *llmUsage) (*draftedPage, error) {
	outlined, err := outlinePage(ctx, llm, candidate, docs, opts, usage)
	if err != nil {
		return nil, err
	}

	page := &draftedPage{candidate: candidate, summary: outlined.Summary}

	sections := make([]string, 0, len(outlined.Chapters))
	cited := map[int]bool{}
	citations := 0
	for _, chapter := range outlined.Chapters {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		body, err := draftChapter(ctx, llm, candidate, chapter, docs, opts, usage)
		if err != nil {
			log.Warnf("wiki: chapter %q of page %q failed: %v", chapter.Title, candidate.Title, err)
			continue
		}
		if len(citationPattern.FindAllString(body, -1)) == 0 {
			log.Warnf("wiki: chapter %q of page %q produced no citations, dropping", chapter.Title, candidate.Title)
			continue
		}
		for _, match := range citationPattern.FindAllStringSubmatch(body, -1) {
			if n, err := strconv.Atoi(match[1]); err == nil && n >= 1 && n <= len(docs) {
				cited[n] = true
			}
			citations++
		}
		sections = append(sections, fmt.Sprintf("## %s\n\n%s", chapter.Title, strings.TrimSpace(body)))
	}
	if len(sections) == 0 {
		return nil, fmt.Errorf("page %q produced no cited content", candidate.Title)
	}

	page.content = strings.Join(sections, "\n\n")
	for n := range cited {
		page.sources = append(page.sources, sourceReferenceFor(&docs[n-1]))
	}
	page.citedDocs = len(cited)
	page.confidence = confidenceFor(citations, len(cited), len(candidate.DocRefs))
	return page, nil
}

// draftChapter generates one chapter body with mandatory [n] citations.
func draftChapter(ctx context.Context, llm llms.Model, candidate pageCandidate, chapter chapterPlan, docs []core.Document, opts generationOptions, usage *llmUsage) (string, error) {
	keyPoints := strings.Join(chapter.KeyPoints, "; ")
	material := joinDocMaterials(candidate, docs, opts.MaxDocChars)
	userPrompt := fmt.Sprintf(
		"Write one chapter of a wiki page.\n\n"+
			"Page: %s\nChapter: %s\nPoints to cover: %s\n\n"+
			"Requirements:\n"+
			"- Write in %s, markdown body only (no chapter heading, no page title)\n"+
			"- Every factual statement MUST cite its source as [n], where n is the document index in the material below\n"+
			"- Do not invent facts; if the material is insufficient, write less\n"+
			"- When first mentioning another notable entity or concept from the material, link it as [[type:Name]] (type such as person/organization/product/concept/event/location)\n\n"+
			"Source documents:\n%s\n\n"+
			"Write the chapter body now.",
		candidate.Title, chapter.Title, keyPoints, opts.Lang, material)
	return callLLM(ctx, llm, "You are a knowledge-management agent writing cited wiki content. Your response MUST be in "+opts.Lang+".", userPrompt, usage)
}

// sourceReferenceFor adapts a scoped document to the citation backlink
// contract (doc_id + excerpt, design doc D2).
func sourceReferenceFor(doc *core.Document) core.WikiSourceReference {
	excerpt := doc.Summary
	if excerpt == "" {
		excerpt = doc.Content
	}
	if len(excerpt) > 200 {
		excerpt = excerpt[:200]
	}
	return core.WikiSourceReference{
		DocID:      doc.ID,
		SourceType: doc.Source.Type,
		SourceName: doc.Source.Name,
		Title:      doc.Title,
		URL:        doc.URL,
		Excerpt:    excerpt,
	}
}

// confidenceFor scores a page by citation density × source diversity
// (design doc §5.1 phase 5) and maps to the article confidence label.
func confidenceFor(citations, distinctDocs, scopedDocs int) string {
	density := float64(citations) / 8 // ~8 citations saturate a page
	if density > 1 {
		density = 1
	}
	diversity := 1.0
	if scopedDocs > 0 {
		diversity = float64(distinctDocs) / float64(scopedDocs)
	}
	score := 0.6*density + 0.4*diversity
	switch {
	case score >= 0.7:
		return "high"
	case score >= 0.4:
		return "medium"
	default:
		return "low"
	}
}

/* ---------------- phase 6: deliver ---------------- */

// deliverPage persists a drafted page as a draft article with its first
// version snapshot, linked pages and KB counters (§5.1 phase 6).
func deliverPage(kb *core.WikiKnowledgeBase, page *draftedPage) (*core.WikiArticle, error) {
	return createDraftArticle(kb, page.candidate.Title, page.summary, page.content,
		page.candidate.PageType, page.candidate.Subtype, page.sources, page.confidence,
		fmt.Sprintf("KM agent generated from %d source documents", len(page.sources)),
		fmt.Sprintf("AI generated draft page %q for knowledge base %s", page.candidate.Title, kb.Name))
}

// createDraftArticle is the single delivery path for AI-produced content:
// draft status, first version snapshot, TOC entry, linked pages, KB counter
// and owner notification (D1: every AI origin stops at draft — publish stays
// on PUT /wiki/article/:id/status).
func createDraftArticle(kb *core.WikiKnowledgeBase, title, summary, content, pageType, subtype string,
	sources []core.WikiSourceReference, confidence, versionNote, notificationMsg string) (*core.WikiArticle, error) {
	article := &core.WikiArticle{
		KbID:        kb.ID,
		Title:       title,
		Summary:     summary,
		Content:     content,
		PageType:    pageType,
		Subtype:     subtype,
		Status:      core.WikiArticleDraft, // D1: generation stops at draft
		AIGenerated: true,
		Confidence:  confidence,
		Sources:     sources,
	}

	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true) // pipeline context has no session user
	orm.WithModel(ctx, &core.WikiArticle{})
	if err := orm.Create(ctx, article); err != nil {
		return nil, err
	}
	if err := writeVersionSnapshot(article, 1, core.WikiChangeAIGenerated, versionNote); err != nil {
		return nil, err
	}
	if err := addArticleToToc(kb.ID, article.ID, article.Title); err != nil {
		return nil, err
	}
	persistLinkedPages(article)
	if err := bumpKbArticleCount(kb.ID, 1); err != nil {
		return nil, err
	}
	notifyOwner(kb.GetOwnerID(), "article", article.ID, "ai-draft", notificationMsg)
	return article, nil
}

// notifyOwner records a wiki event for a user (no-op without a target).
func notifyOwner(userID, targetType, targetID, action, message string) {
	if userID == "" {
		return
	}
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiNotification{})
	notification := &core.WikiNotification{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
		Action:     action,
		Message:    message,
	}
	if err := orm.Create(ctx, notification); err != nil {
		log.Warnf("wiki: failed to record notification for user %s: %v", userID, err)
	}
}

/* ---------------- C3: POST /wiki/kb/:id/ai/generate ---------------- */

type aiGenerateRequest struct {
	Hint          string `json:"hint"`
	MaxPages      int    `json:"max_pages"`
	ModelProvider string `json:"model_provider"`
	Model         string `json:"model"`
	Lang          string `json:"lang"`
}

func (h *APIHandler) aiGenerate(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")

	body := aiGenerateRequest{}
	if req.ContentLength != 0 {
		if err := h.DecodeJSON(req, &body); err != nil {
			h.Error400(w, err.Error())
			return
		}
	}
	opts := generationOptions{MaxPages: body.MaxPages, Lang: body.Lang, Hint: body.Hint}
	opts.applyDefaults()

	readCtx := orm.NewContextWithParent(req.Context())
	readCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(readCtx, &core.WikiKnowledgeBase{})
	var kb core.WikiKnowledgeBase
	kb.SetID(kbID)
	exists, err := orm.GetV2(readCtx, &kb)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, kbID)
		return
	}
	if len(kb.DatasourceIDs) == 0 {
		h.Error400(w, "knowledge base has no bound datasources")
		return
	}

	llm, err := resolveLanguageLLM(body.ModelProvider, body.Model)
	if err != nil {
		h.Error400(w, err.Error())
		return
	}

	stream, err := newSSEStream(w, req.Context())
	if err != nil {
		h.Error500(w, err.Error())
		return
	}

	start := time.Now()
	usage := &llmUsage{}
	fail := func(reason string) {
		_ = stream.emit("done", doneEvent{Reason: reason, Usage: usage})
	}
	// Once streaming has started the API router's recover middleware can no
	// longer report a panic to the client — without this guard a panic
	// strands the browser on an open SSE connection forever.
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("wiki: generation panicked: %v\n%s", r, debug.Stack())
			_ = stream.emit("done", doneEvent{Reason: fmt.Sprintf("internal error: %v", r), Usage: usage})
		}
	}()

	if err := stream.emit("progress", progressEvent{Phase: "scope", PhaseCurrent: 1, PhaseTotal: 1, Progress: 0.05}); err != nil {
		return
	}
	docs, err := scopeKbDocuments(readCtx, &kb, opts.MaxScopeDocs)
	if err != nil {
		fail(fmt.Sprintf("scope failed: %v", err))
		return
	}
	if len(docs) == 0 {
		fail("no documents found in the bound datasources")
		return
	}

	if err := stream.emit("progress", progressEvent{Phase: "cluster", PhaseCurrent: 1, PhaseTotal: 1, Progress: 0.2}); err != nil {
		return
	}
	candidates, err := clusterPages(req.Context(), llm, docs, opts, usage)
	if err != nil {
		fail(fmt.Sprintf("cluster failed: %v", err))
		return
	}
	candidates = dropExistingTitles(readCtx, kb.ID, candidates)
	if len(candidates) == 0 {
		fail("no new pages to generate")
		return
	}

	// phases 3–5 with a bounded worker pool (并发页数 ≤N, §5.1)
	type draftResult struct {
		page  *draftedPage
		title string
		err   error
	}
	jobs := make(chan pageCandidate)
	results := make(chan draftResult)
	var workers sync.WaitGroup
	for i := 0; i < opts.Concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for candidate := range jobs {
				if req.Context().Err() != nil {
					results <- draftResult{title: candidate.Title, err: req.Context().Err()}
					continue
				}
				page, err := draftPage(req.Context(), llm, candidate, docs, opts, usage)
				results <- draftResult{page: page, title: candidate.Title, err: err}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, candidate := range candidates {
			select {
			case jobs <- candidate:
			case <-req.Context().Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()

	pages := make([]*draftedPage, 0, len(candidates))
	failed := make([]string, 0)
	completed := 0
	for result := range results {
		completed++
		switch {
		case result.err != nil:
			failed = append(failed, result.title)
			log.Warnf("wiki: generation of page %q failed: %v", result.title, result.err)
		default:
			pages = append(pages, result.page)
		}
		if err := stream.emit("progress", progressEvent{
			Phase:        "draft",
			PhaseCurrent: completed,
			PhaseTotal:   len(candidates),
			Progress:     0.2 + 0.6*float64(completed)/float64(len(candidates)),
			Eta:          etaSeconds(start, 0.2+0.6*float64(completed)/float64(len(candidates))),
		}); err != nil {
			return
		}
	}

	if err := stream.emit("progress", progressEvent{Phase: "deliver", PhaseCurrent: 1, PhaseTotal: 1, Progress: 0.95}); err != nil {
		return
	}
	generated := 0
	for _, page := range pages {
		article, err := deliverPage(&kb, page)
		if err != nil {
			failed = append(failed, page.candidate.Title)
			log.Warnf("wiki: delivery of page %q failed: %v", page.candidate.Title, err)
			continue
		}
		generated++
		if err := stream.emit("article", article); err != nil {
			return
		}
	}

	_ = stream.emit("done", doneEvent{Generated: generated, Failed: failed, Usage: usage})
}

func etaSeconds(start time.Time, progress float64) int64 {
	if progress <= 0 {
		return 0
	}
	elapsed := time.Since(start).Seconds()
	return int64(elapsed/progress - elapsed)
}

// dropExistingTitles removes candidates whose title already exists in the
// KB so re-running generation complements instead of duplicating.
func dropExistingTitles(ctx *orm.Context, kbID string, candidates []pageCandidate) []pageCandidate {
	if len(candidates) == 0 {
		return candidates
	}
	orm.WithModel(ctx, &core.WikiArticle{})
	builder := orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("kb_id", kbID)).
		Include("title")
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		log.Warnf("wiki: failed to list existing titles for dedup: %v", err)
		return candidates
	}
	hits, _, err := elastic.DecodeHits[util.MapStr](res)
	if err != nil {
		return candidates
	}
	existing := make(map[string]bool, len(hits))
	for _, hit := range hits {
		if title, ok := hit["title"].(string); ok {
			existing[strings.ToLower(title)] = true
		}
	}
	out := candidates[:0]
	for _, candidate := range candidates {
		if !existing[strings.ToLower(candidate.Title)] {
			out = append(out, candidate)
		}
	}
	return out
}

/* ---------------- C4: POST /wiki/article/:id/ai/edit ---------------- */

type aiEditRequest struct {
	Instruction   string `json:"instruction"`
	Selection     string `json:"selection"`
	ModelProvider string `json:"model_provider"`
	Model         string `json:"model"`
	Lang          string `json:"lang"`
}

// aiEdit rewrites an article (or a selected fragment) per instruction and
// streams the rewritten text; the result lands as a new ai-generated
// version while the status machine stays untouched (review gate, D1).
func (h *APIHandler) aiEdit(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")

	body := aiEditRequest{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if strings.TrimSpace(body.Instruction) == "" {
		h.Error400(w, "instruction is required")
		return
	}

	readCtx := orm.NewContextWithParent(req.Context())
	readCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(readCtx, &core.WikiArticle{})
	var article core.WikiArticle
	article.SetID(articleID)
	exists, err := orm.GetV2(readCtx, &article)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, articleID)
		return
	}
	if body.Selection != "" && !strings.Contains(article.Content, body.Selection) {
		h.Error400(w, "selection not found in the article content")
		return
	}

	llm, err := resolveLanguageLLM(body.ModelProvider, body.Model)
	if err != nil {
		h.Error400(w, err.Error())
		return
	}

	stream, err := newSSEStream(w, req.Context())
	if err != nil {
		h.Error500(w, err.Error())
		return
	}

	if err := stream.emit("progress", progressEvent{Phase: "editing", PhaseCurrent: 1, PhaseTotal: 1, Progress: 0.1}); err != nil {
		return
	}

	lang := body.Lang
	if lang == "" {
		lang = defaultGenerationLang()
	}
	system := "You are a knowledge-management agent editing wiki content. Your response MUST be in " + lang + "."
	var userPrompt string
	if body.Selection != "" {
		userPrompt = fmt.Sprintf(
			"Rewrite ONLY the selected fragment of the wiki article below according to the instruction.\n\n"+
				"Instruction: %s\n\n"+
				"Rules:\n"+
				"- Keep the original language (%s) unless the instruction says otherwise\n"+
				"- Preserve existing [n] citations and [[type:Name]] wikilinks when still applicable\n"+
				"- Return ONLY the rewritten fragment, no commentary, no markdown fences\n\n"+
				"Selected fragment:\n%s\n\nFull article (context only):\n%s",
			body.Instruction, lang, body.Selection, truncate(article.Content, 8000))
	} else {
		userPrompt = fmt.Sprintf(
			"Rewrite the wiki article below according to the instruction.\n\n"+
				"Instruction: %s\n\n"+
				"Rules:\n"+
				"- Keep the original language (%s) unless the instruction says otherwise\n"+
				"- Keep the structured markdown: ## section headings, [n] citations and [[type:Name]] wikilinks\n"+
				"- Return ONLY the full rewritten article, no commentary, no markdown fences\n\n"+
				"Article:\n%s",
			body.Instruction, lang, article.Content)
	}

	usage := &llmUsage{}
	// Guard mirrors aiGenerate: report a panic as a terminal done event
	// instead of stranding the client on an open SSE connection.
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("wiki: AI edit panicked: %v\n%s", r, debug.Stack())
			_ = stream.emit("done", doneEvent{Reason: fmt.Sprintf("internal error: %v", r), Usage: usage})
		}
	}()
	message := []llms.MessageContent{
		langchain.SystemTextParts(system),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}
	var builder strings.Builder
	resp, err := llm.GenerateContent(req.Context(), message, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
		builder.Write(chunk)
		return stream.emit("chunk", util.MapStr{"text": string(chunk)})
	}))
	if err != nil {
		_ = stream.emit("done", doneEvent{Reason: fmt.Sprintf("edit failed: %v", err), Usage: usage})
		return
	}
	usage.add(len(system)+len(userPrompt), builder.Len())
	if builder.Len() == 0 && resp != nil {
		for _, choice := range resp.Choices {
			builder.WriteString(choice.Content)
		}
	}

	rewritten := strings.TrimSpace(removeThinkPattern.ReplaceAllLiteralString(builder.String(), ""))
	if rewritten == "" {
		_ = stream.emit("done", doneEvent{Reason: "model returned empty content", Usage: usage})
		return
	}
	newContent := rewritten
	if body.Selection != "" {
		newContent = strings.Replace(article.Content, body.Selection, rewritten, 1)
	}

	writeCtx := orm.NewContext()
	writeCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	writeCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(writeCtx, &core.WikiArticle{})
	article.Content = newContent
	if err := orm.Update(writeCtx, &article); err != nil {
		_ = stream.emit("done", doneEvent{Reason: fmt.Sprintf("failed to persist edit: %v", err), Usage: usage})
		return
	}
	version := nextVersionNumber(article.ID)
	if err := writeVersionSnapshot(&article, version, core.WikiChangeAIGenerated,
		fmt.Sprintf("AI edit: %s", truncate(strings.TrimSpace(body.Instruction), 80))); err != nil {
		_ = stream.emit("done", doneEvent{Reason: fmt.Sprintf("failed to record version: %v", err), Usage: usage})
		return
	}
	persistLinkedPages(&article)

	_ = stream.emit("done", util.MapStr{"_id": article.ID, "version": version, "usage": usage})
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
