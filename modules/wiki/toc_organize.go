/* Copyright © INFINI LTD. All rights reserved. */

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
	"unicode"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	llmmodule "infini.sh/coco/modules/llm"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"github.com/tmc/langchaingo/llms"
)

// Semantic TOC grouping: articles publish into the tree root in creation
// order and stay there until someone drags them by hand, so a KB that grew
// fast shows a flat pile of loose pages. This runs the default language
// model over the loose root titles in two phases — first propose 4-10
// category names from a sample, then assign every loose article to one of
// those categories — and re-files the loose nodes into AI category folders.
// Manually arranged folders and their subtrees are never touched: the human
// layout wins, the AI only tidies what was left unfiled.

const (
	organizeSampleSize = 80  // titles shown to the category-proposal phase
	organizeBatchSize  = 120 // titles per assignment call — fewer calls on slow models
	organizeMinCats    = 4   // proposal prompt bounds; fewer live cats are fine
	organizeMaxCats    = 10  // — empty categories are dropped after assignment
	// fallback bucket for articles the model left out or filed into a
	// category it invented; keeping the name stable keeps folder ids stable
	organizeFallbackCategory = "其他"
)

// organizeRunGuard keeps one organize pass per KB — a second click while the
// first is still classifying would race on the tree save — and remembers the
// last outcome so the status endpoint can answer across page reloads.
type organizeRunGuard struct {
	mu sync.Mutex
	m  map[string]*organizeRunState
}

type organizeRunState struct {
	Running    bool
	StartLoose int
	Last       *organizeStats
	LastErr    string
	LastAt     time.Time
}

func (g *organizeRunGuard) TryReserve(kbID string, loose int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.m == nil {
		g.m = map[string]*organizeRunState{}
	}
	st := g.m[kbID]
	if st != nil && st.Running {
		return false
	}
	g.m[kbID] = &organizeRunState{Running: true, StartLoose: loose}
	return true
}

// Finish closes a run and records its outcome for the status endpoint.
func (g *organizeRunGuard) Finish(kbID string, stats *organizeStats, runErr error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	st := g.m[kbID]
	if st == nil {
		return
	}
	st.Running = false
	st.Last, st.LastAt = stats, time.Now()
	if runErr != nil {
		st.LastErr = runErr.Error()
	} else {
		st.LastErr = ""
	}
}

func (g *organizeRunGuard) State(kbID string) organizeRunState {
	g.mu.Lock()
	defer g.mu.Unlock()
	if st := g.m[kbID]; st != nil {
		return *st
	}
	return organizeRunState{}
}

var organizeRuns organizeRunGuard

// organizeFolderAIPrefix marks folders this feature created — a merge into
// one of them re-sorts the children (they were alphabetical anyway), while
// hand-built folders keep their manual child order.
const organizeFolderAIPrefix = "folder-ai-"

// buildProposalPrompt adapts the category proposal to the run: small page
// sets would be shredded by the full 4-10 spread, and existing folder names
// come first so repeated runs consolidate instead of spawning synonyms.
func buildProposalPrompt(pageCount int, existingTitles []string) string {
	minCats := organizeMinCats
	if pageCount < 24 {
		minCats = 2
	}
	var b strings.Builder
	b.WriteString("You are organizing the table of contents of a knowledge base.\n")
	fmt.Fprintf(&b, "You will see a sample of pages (title, sometimes with a one-line excerpt). Propose between %d and %d category names that group them meaningfully by topic.\n", minCats, organizeMaxCats)
	if len(existingTitles) > 0 {
		fmt.Fprintf(&b, "Existing folders: %s. Reuse these names where they fit; propose new ones only for topics with no home among them.\n", strings.Join(existingTitles, " | "))
	}
	b.WriteString("Rules: short names (2-8 characters for Chinese), the same language as the titles, mutually exclusive, no numbering, no explanations.\n")
	b.WriteString(`Return ONLY a JSON array of strings, e.g. ["产品与服务","数据与检索","部署与运维"].`)
	return b.String()
}

const organizeAssignPrompt = `You are filing knowledge-base pages into fixed categories.
For every page id given, choose exactly ONE category from the allowed list. Judge by the title.
Rules: every id in the input must appear exactly once in the output; use only allowed categories; no explanations.
Return ONLY a JSON array of objects {"id":"...","category":"..."} .`

// organizeStats reports what one organize pass did — surfaced in the HTTP
// response, the evolution log and the UI toast. Moved counts every re-filed
// page (new folders + merges alike); Merged is the subset that landed in
// folders the tree already had.
type organizeStats struct {
	Loose       int      `json:"loose"`
	Moved       int      `json:"moved"`
	Merged      int      `json:"merged"`
	Categories  []string `json:"categories"`
	KeptFolders int      `json:"kept_folders"`
	Other       int      `json:"other"`
}

// organizeFallbackName picks the fallback folder's name in the loose
// pages' dominant script — an English KB must not grow a Chinese 其他
// folder. Han-dominant titles → 其他, anything else → Other.
func organizeFallbackName(order []core.WikiTocNode) string {
	cjk, letters := 0, 0
	for _, n := range order {
		for _, r := range n.Title {
			switch {
			case unicode.Is(unicode.Han, r):
				cjk++
				letters++
			case unicode.IsLetter(r):
				letters++
			}
		}
	}
	if letters > 0 && cjk*2 >= letters {
		return "其他"
	}
	return "Other"
}

// sampleForProposal spreads the proposal sample evenly across the sorted
// loose list — a plain prefix would show the model only the A–D slice of
// an English KB and skew the categories accordingly.
func sampleForProposal(order []core.WikiTocNode, size int) []core.WikiTocNode {
	if len(order) <= size {
		return order
	}
	step := float64(len(order)) / float64(size)
	out := make([]core.WikiTocNode, 0, size)
	for i := 0; i < size; i++ {
		out = append(out, order[int(float64(i)*step)])
	}
	return out
}

// parseCategoryList extracts the JSON array of category names from a model
// answer (bare array or fenced code block). The cap flexes with the folder
// count the tree already has — a KB with ten folders must still leave the
// model room to propose a genuinely new topic.
func parseCategoryList(s string, maxCats int) ([]string, error) {
	raw := extractJSONArray(s, "")
	if raw == "" {
		return nil, fmt.Errorf("no JSON array found in category answer")
	}
	var cats []string
	if err := json.Unmarshal([]byte(raw), &cats); err != nil {
		return nil, fmt.Errorf("category answer is not a string array: %w", err)
	}
	out := make([]string, 0, len(cats))
	seen := map[string]bool{}
	for _, c := range cats {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
		if len(out) >= maxCats {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("category answer carried no usable names")
	}
	return out, nil
}

type organizeAssignment struct {
	ID       string `json:"id"`
	Category string `json:"category"`
}

// parseAssignments extracts {"id","category"} pairs; malformed entries are
// skipped rather than failing the whole batch.
func parseAssignments(s string) map[string]string {
	raw := extractJSONArray(s, "")
	out := map[string]string{}
	if raw == "" {
		return out
	}
	var pairs []organizeAssignment
	if err := json.Unmarshal([]byte(raw), &pairs); err != nil {
		return out
	}
	for _, p := range pairs {
		if p.ID != "" && p.Category != "" {
			out[p.ID] = strings.TrimSpace(p.Category)
		}
	}
	return out
}

// extractJSONArray returns the JSON array inside a fenced block, or between
// the first '[' and the last ']' of the raw text as a fallback.
func extractJSONArray(s, _ string) string {
	m := regexp.MustCompile("(?s)```[a-zA-Z]*\\n?(.*?)```").FindStringSubmatch(s)
	if len(m) >= 2 && strings.Contains(m[1], "[") {
		s = m[1]
	}
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}

// buildOrganizedToc re-files the loose root articles into category folders.
// Existing folders keep their order at the top (whole subtrees intact); a
// category whose name matches an existing folder files INTO it instead of
// spawning a sibling synonym; brand-new category folders follow in proposal
// order; anything unassigned or filed into an invented category lands in the
// fallback folder (only created when non-empty). A loose article that was
// never shown to the model (created after the pass snapshot) stays at the
// root — the next pass picks it up instead of burying it in the fallback.
// An AI folder left empty by all this is dropped (it was auto-created, an
// empty one has no meaning); a hand-built empty folder stays.
func buildOrganizedToc(existing []core.WikiTocNode, assigns map[string]string, categories []string, classified map[string]bool, fallback, folderIDSuffix string) ([]core.WikiTocNode, organizeStats) {
	stats := organizeStats{Categories: []string{}}

	folders := []core.WikiTocNode{}
	folderIdx := map[string]int{} // title → position in folders, for merging
	var loose []core.WikiTocNode
	var fresh []core.WikiTocNode // created mid-pass, left at the root
	for _, n := range existing {
		if n.Type == "folder" {
			if _, dup := folderIdx[n.Title]; !dup {
				folderIdx[n.Title] = len(folders)
			}
			folders = append(folders, n)
			continue
		}
		if n.ArticleID != "" && !classified[n.ArticleID] {
			fresh = append(fresh, n)
			continue
		}
		loose = append(loose, n)
	}
	stats.Loose = len(loose)
	preExisting := len(folders)

	buckets := map[string][]core.WikiTocNode{}
	catSet := map[string]bool{}
	for _, c := range categories {
		catSet[c] = true
	}
	for _, n := range loose {
		cat, ok := assigns[n.ArticleID]
		if !ok || !catSet[cat] {
			cat = fallback
		}
		buckets[cat] = append(buckets[cat], n)
	}

	// deterministic article order inside every folder
	byTitle := func(ns []core.WikiTocNode) {
		sort.SliceStable(ns, func(i, j int) bool { return ns[i].Title < ns[j].Title })
	}

	// fileBucket lands one category's pages, merging into an existing folder
	// of the same title when there is one (AI folders re-sort alphabetically,
	// hand-built folders keep their manual child order and just append).
	fileBucket := func(cat string, ns []core.WikiTocNode) {
		if idx, ok := folderIdx[cat]; ok {
			f := folders[idx]
			f.Children = append(f.Children, ns...)
			if strings.HasPrefix(f.ID, organizeFolderAIPrefix) {
				byTitle(f.Children)
			}
			folders[idx] = f
			stats.Merged += len(ns)
			return
		}
		byTitle(ns)
		folders = append(folders, core.WikiTocNode{
			ID:       organizeFolderID(cat, folderIDSuffix),
			Title:    cat,
			Type:     "folder",
			Children: ns,
		})
	}

	for _, c := range categories {
		ns := buckets[c]
		if len(ns) == 0 {
			continue // a category nobody landed in must not leave an empty folder
		}
		fileBucket(c, ns)
		stats.Categories = append(stats.Categories, c)
		stats.Moved += len(ns)
	}
	if ns := buckets[fallback]; len(ns) > 0 {
		fileBucket(fallback, ns)
		stats.Categories = append(stats.Categories, fallback)
		stats.Moved += len(ns)
		stats.Other = len(ns)
	}

	// drop AI folders nobody landed in (the user emptying one by drag is a
	// delete, not a refile) — manual folders survive even when empty
	out := make([]core.WikiTocNode, 0, len(folders)+len(fresh))
	dropped := 0
	for _, f := range folders {
		if strings.HasPrefix(f.ID, organizeFolderAIPrefix) && len(f.Children) == 0 {
			dropped++
			continue
		}
		out = append(out, f)
	}
	stats.KeptFolders = preExisting - dropped
	return append(out, fresh...), stats
}

// organizeFolderID derives a stable id from the category so a re-run moves
// articles into the same folders instead of stacking duplicates.
func organizeFolderID(category, kbID string) string {
	slug := strings.Builder{}
	for _, r := range strings.ToLower(category) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			slug.WriteRune(r)
		case r == ' ', r == '-', r == '_', r == '/', r == '.':
			slug.WriteRune('-')
		}
		// CJK and other non-ASCII contribute nothing — the hash keeps ids unique
	}
	s := slug.String()
	if len(s) > 24 {
		s = s[:24]
	}
	if s == "" {
		s = "cat"
	}
	return fmt.Sprintf("folder-ai-%s-%s", s, util.MD5digest(kbID + "|" + category)[:8])
}

// OrganizeTocSemantically is the whole pass: LLM classification plus the
// tree rewrite, ready to be called from the HTTP handler.
func OrganizeTocSemantically(parent context.Context, kbID string) (*organizeStats, error) {
	// read the current tree
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiToc{})
	toc, _, err := findToc(ctx, kbID)
	if err != nil {
		return nil, err
	}

	var loose []core.WikiTocNode
	for _, n := range toc.Nodes {
		if n.Type != "folder" && n.ArticleID != "" {
			loose = append(loose, n)
		}
	}
	if len(loose) == 0 {
		return &organizeStats{KeptFolders: len(toc.Nodes)}, nil
	}

	// stable input order → stable prompts → reproducible grouping
	order := make([]core.WikiTocNode, len(loose))
	copy(order, loose)
	sort.SliceStable(order, func(i, j int) bool { return order[i].Title < order[j].Title })

	// existing folder names go into the proposal so repeat runs consolidate
	// into the folders the tree already has instead of spawning synonyms
	existingTitles := make([]string, 0, len(toc.Nodes))
	for _, n := range toc.Nodes {
		if n.Type == "folder" {
			existingTitles = append(existingTitles, n.Title)
		}
	}

	// summaries sharpen filing for vague titles ("Division", "Case") — one
	// terms query, one capped line per page in the prompt
	summaries := fetchArticleSummaries(parent, loose)

	categories, assigns, err := classifyForOrganize(parent, order, existingTitles, summaries)
	if err != nil {
		return nil, err
	}

	classified := make(map[string]bool, len(order))
	for _, n := range order {
		classified[n.ArticleID] = true
	}
	// the fallback folder's name follows the loose pages' dominant script
	fallback := organizeFallbackName(order)
	var stats *organizeStats
	saveErr := tocMutate(kbID, func(nodes []core.WikiTocNode) ([]core.WikiTocNode, bool) {
		next, s := buildOrganizedToc(nodes, assigns, categories, classified, fallback, kbID)
		stats = &s
		return next, true
	})
	if saveErr != nil {
		return nil, saveErr
	}
	return stats, nil
}

// classifyForOrganize runs the two LLM phases: propose categories from a
// sample, then assign every loose article to one of them.
func classifyForOrganize(ctx context.Context, order []core.WikiTocNode, existingTitles []string, summaries map[string]string) ([]string, map[string]string, error) {
	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		return nil, nil, fmt.Errorf("no default language model configured")
	}
	llm, err := langchain.SimplyGetLLM(model.ProviderID, model.ID, "")
	if err != nil {
		return nil, nil, err
	}

	sample := sampleForProposal(order, organizeSampleSize)
	catResp, err := generateWithRetry(ctx, llm, []llms.MessageContent{
		langchain.SystemTextParts(buildProposalPrompt(len(order), existingTitles)),
		llms.TextParts(llms.ChatMessageTypeHuman, entriesBlock(sample, summaries)),
	}, 400)
	if err != nil {
		return nil, nil, fmt.Errorf("category proposal failed: %w", err)
	}
	if catResp == nil || len(catResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("empty category proposal response")
	}
	// existing folder names count against the cap too when the model echoes
	// them — a KB with ten folders must still leave room for a new topic
	categories, err := parseCategoryList(catResp.Choices[0].Content, organizeMaxCats+len(existingTitles))
	if err != nil {
		return nil, nil, err
	}

	allowed := strings.Join(categories, " | ")
	assigns := map[string]string{}
	for start := 0; start < len(order); start += organizeBatchSize {
		end := start + organizeBatchSize
		if end > len(order) {
			end = len(order)
		}
		batch := order[start:end]
		userText := "Allowed categories: " + allowed + "\n\nPages:\n" + entriesBlock(batch, summaries)
		resp, err := generateWithRetry(ctx, llm, []llms.MessageContent{
			langchain.SystemTextParts(organizeAssignPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, userText),
		}, 4000)
		if err != nil {
			return nil, nil, fmt.Errorf("assignment batch failed: %w", err)
		}
		if resp == nil || len(resp.Choices) == 0 {
			return nil, nil, fmt.Errorf("empty assignment response")
		}
		for id, cat := range parseAssignments(resp.Choices[0].Content) {
			assigns[id] = cat
		}
	}
	return categories, assigns, nil
}

// generateWithRetry rides out transient provider errors with one retry —
// without it a single hiccup fails the whole pass and the next trigger
// (button or auto-archive) starts the model calls over from scratch.
func generateWithRetry(ctx context.Context, llm llms.Model, msgs []llms.MessageContent, maxTokens int) (*llms.ContentResponse, error) {
	resp, err := llm.GenerateContent(ctx, msgs, llms.WithMaxTokens(maxTokens))
	if err == nil {
		return resp, nil
	}
	log.Warnf("wiki: toc organize llm call failed, retrying once: %v", err)
	return llm.GenerateContent(ctx, msgs, llms.WithMaxTokens(maxTokens))
}

// fetchArticleSummaries pulls the loose articles' summaries in one terms
// query; on any error classification falls back to titles alone.
func fetchArticleSummaries(parent context.Context, loose []core.WikiTocNode) map[string]string {
	ids := make([]string, 0, len(loose))
	for _, n := range loose {
		ids = append(ids, n.ArticleID)
	}
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(len(ids)).Filter(orm.TermsQuery("id", ids)))
	if err != nil {
		return nil
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		return nil
	}
	out := make(map[string]string, len(articles))
	for i := range articles {
		out[articles[i].ID] = articles[i].Summary
	}
	return out
}

// entriesBlock renders one "id. title — excerpt" line per article — the id
// is what the assignment phase echoes back, the title plus the one-line
// summary excerpt is what the model judges by.
func entriesBlock(nodes []core.WikiTocNode, summaries map[string]string) string {
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(n.ArticleID)
		b.WriteString(". ")
		b.WriteString(n.Title)
		if s := excerptLine(summaries[n.ArticleID], 120); s != "" {
			b.WriteString(" — ")
			b.WriteString(s)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// excerptLine flattens a summary to one space-separated line capped at
// maxRunes, so a multi-line summary cannot smuggle fake "id. title" rows
// into the prompt.
func excerptLine(s string, maxRunes int) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return s
}

// countLooseRootArticles reads a KB's toc and counts the un-filed root
// articles — the precheck for "nothing to do" and the number the UI shows.
func countLooseRootArticles(kbID string) (int, error) {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiToc{})
	toc, _, err := findToc(ctx, kbID)
	if err != nil {
		return 0, err
	}
	loose := 0
	for _, n := range toc.Nodes {
		if n.Type != "folder" && n.ArticleID != "" {
			loose++
		}
	}
	return loose, nil
}

func (h *APIHandler) organizeToc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	userID := requestUserID(req)

	started, loose, err := startOrganizeRun(kbID, userID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !started {
		if loose > 0 {
			h.WriteJSON(w, util.MapStr{"started": false, "already_running": true, "loose": loose}, http.StatusTooManyRequests)
			return
		}
		h.WriteJSON(w, util.MapStr{"started": false, "loose": 0}, http.StatusOK)
		return
	}
	h.WriteJSON(w, util.MapStr{"started": true, "loose": loose}, http.StatusOK)
}

// startOrganizeRun launches one background pass (guard-protected): the HTTP
// button and the per-KB auto-archive hook both come through here. Returns
// started plus the loose count it picked up; !started with loose > 0 means
// a pass is already running.
func startOrganizeRun(kbID, userID string) (bool, int, error) {
	loose, err := countLooseRootArticles(kbID)
	if err != nil {
		return false, 0, err
	}
	if loose == 0 {
		return false, 0, nil
	}
	if !organizeRuns.TryReserve(kbID, loose) {
		return false, loose, nil
	}
	go func() {
		stats, runErr := func() (stats *organizeStats, runErr error) {
			defer func() {
				if r := recover(); r != nil {
					stats, runErr = nil, fmt.Errorf("panic: %v", r)
					log.Warnf("wiki: toc organize recovered: %v", r)
				}
			}()
			return OrganizeTocSemantically(context.Background(), kbID)
		}()
		organizeRuns.Finish(kbID, stats, runErr)
		if runErr != nil {
			log.Warnf("wiki: toc organize failed for kb %s: %v", kbID, runErr)
			return
		}
		AppendEvolutionEntry(orm.NewContext(), kbID, core.WikiEvolutionTocOrganize, userID,
			"kb", kbID,
			fmt.Sprintf("semantic toc organize: %d loose pages filed into %d categories (%d into existing folders, %d folders kept)", stats.Moved, len(stats.Categories), stats.Merged, stats.KeptFolders),
			map[string]interface{}{"categories": stats.Categories, "moved": stats.Moved, "merged": stats.Merged, "other": stats.Other, "kept_folders": stats.KeptFolders})
		log.Infof("wiki: toc organize kb %s: %d pages into %v (%d merged)", kbID, stats.Moved, stats.Categories, stats.Merged)
	}()
	return true, loose, nil
}

// maybeAutoOrganizeToc fires the background pass when the KB opted into
// 自动归档 (KB.AutoOrganizeToc) — new pages publish to the tree root, this
// keeps the tree filed without anyone clicking the toolbar button.
// Best-effort: the guard collapses create bursts (every page in a batch
// tries, one pass wins) and a page created mid-pass just stays at the root
// for the next trigger.
func maybeAutoOrganizeToc(kbID string) {
	if kbID == "" {
		return
	}
	kctx := orm.NewContext()
	kctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(kctx, &core.WikiKnowledgeBase{})
	var kb core.WikiKnowledgeBase
	kb.SetID(kbID)
	exists, err := orm.GetV2(kctx, &kb)
	if err != nil || !exists || !kb.AutoOrganizeToc {
		return
	}
	if _, _, err := startOrganizeRun(kbID, "auto-organize"); err != nil {
		log.Warnf("wiki: auto toc organize failed to start for kb %s: %v", kbID, err)
	}
}

// organizeTocStatus answers "is a pass running and what did the last one
// do" — the polling target that survives page reloads (a UI that reloads
// mid-run picks its spinner back up from here).
func (h *APIHandler) organizeTocStatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	st := organizeRuns.State(kbID)
	resp := util.MapStr{"running": st.Running}
	if st.Running {
		resp["loose"] = st.StartLoose
	} else if loose, err := countLooseRootArticles(kbID); err == nil {
		resp["loose"] = loose
	} else {
		resp["loose"] = -1
	}
	if st.Last != nil {
		resp["last"] = st.Last
	}
	if st.LastErr != "" {
		resp["last_error"] = st.LastErr
	}
	if !st.LastAt.IsZero() {
		resp["last_at"] = st.LastAt.Format(time.RFC3339)
	}
	h.WriteJSON(w, resp, http.StatusOK)
}
