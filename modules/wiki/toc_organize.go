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

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/assistant/langchain"
	llmmodule "infini.sh/coco/modules/llm"
	httprouter "infini.sh/framework/core/api/router"
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
// first is still classifying would race on the tree save.
type organizeRunGuard struct {
	mu sync.Mutex
	m  map[string]bool
}

func (g *organizeRunGuard) TryReserve(kbID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.m == nil {
		g.m = map[string]bool{}
	}
	if g.m[kbID] {
		return false
	}
	g.m[kbID] = true
	return true
}

func (g *organizeRunGuard) Release(kbID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.m, kbID)
}

var organizeRuns organizeRunGuard

const organizeCategoryPrompt = `You are organizing the table of contents of a knowledge base.
You will see a sample of page titles. Propose between 4 and 10 category names that group them meaningfully by topic.
Rules: short names (2-8 characters for Chinese), the same language as the titles, mutually exclusive, no numbering, no explanations.
Return ONLY a JSON array of strings, e.g. ["产品与服务","数据与检索","部署与运维"].`

const organizeAssignPrompt = `You are filing knowledge-base pages into fixed categories.
For every page id given, choose exactly ONE category from the allowed list. Judge by the title.
Rules: every id in the input must appear exactly once in the output; use only allowed categories; no explanations.
Return ONLY a JSON array of objects {"id":"...","category":"..."} .`

// organizeStats reports what one organize pass did — surfaced in the HTTP
// response, the evolution log and the UI toast.
type organizeStats struct {
	Loose       int      `json:"loose"`
	Moved       int      `json:"moved"`
	Categories  []string `json:"categories"`
	KeptFolders int      `json:"kept_folders"`
	Other       int      `json:"other"`
}

// parseCategoryList extracts the JSON array of category names from a model
// answer (bare array or fenced code block).
func parseCategoryList(s string) ([]string, error) {
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
		if len(out) >= organizeMaxCats {
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
// Manual folders (with their whole subtrees) keep their order at the top;
// category folders follow in proposal order; articles inside a folder are
// alphabetical; anything unassigned or filed into an invented category lands
// in the fallback folder (only created when non-empty).
func buildOrganizedToc(existing []core.WikiTocNode, assigns map[string]string, categories []string, folderIDSuffix string) ([]core.WikiTocNode, organizeStats) {
	stats := organizeStats{Categories: []string{}}
	out := make([]core.WikiTocNode, 0, len(existing))

	kept := []core.WikiTocNode{}
	buckets := map[string][]core.WikiTocNode{}
	catSet := map[string]bool{}
	for _, c := range categories {
		catSet[c] = true
	}
	var loose []core.WikiTocNode
	for _, n := range existing {
		if n.Type == "folder" {
			kept = append(kept, n)
			continue
		}
		loose = append(loose, n)
	}
	stats.Loose = len(loose)
	stats.KeptFolders = len(kept)

	for _, n := range loose {
		cat, ok := assigns[n.ArticleID]
		if !ok || !catSet[cat] {
			cat = organizeFallbackCategory
		}
		buckets[cat] = append(buckets[cat], n)
	}

	// deterministic article order inside every folder
	byTitle := func(ns []core.WikiTocNode) {
		sort.SliceStable(ns, func(i, j int) bool { return ns[i].Title < ns[j].Title })
	}

	out = append(out, kept...)
	for _, c := range categories {
		ns := buckets[c]
		if len(ns) == 0 {
			continue // a category nobody landed in must not leave an empty folder
		}
		byTitle(ns)
		out = append(out, core.WikiTocNode{
			ID:       organizeFolderID(c, folderIDSuffix),
			Title:    c,
			Type:     "folder",
			Children: ns,
		})
		stats.Categories = append(stats.Categories, c)
		stats.Moved += len(ns)
	}
	if ns := buckets[organizeFallbackCategory]; len(ns) > 0 {
		byTitle(ns)
		out = append(out, core.WikiTocNode{
			ID:       organizeFolderID(organizeFallbackCategory, folderIDSuffix),
			Title:    organizeFallbackCategory,
			Type:     "folder",
			Children: ns,
		})
		stats.Categories = append(stats.Categories, organizeFallbackCategory)
		stats.Moved += len(ns)
		stats.Other = len(ns)
	}
	return out, stats
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
	return fmt.Sprintf("folder-ai-%s-%s", s, util.MD5digest(kbID+"|"+category)[:8])
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

	categories, assigns, err := classifyForOrganize(parent, order)
	if err != nil {
		return nil, err
	}

	var stats *organizeStats
	saveErr := tocMutate(kbID, func(nodes []core.WikiTocNode) ([]core.WikiTocNode, bool) {
		next, s := buildOrganizedToc(nodes, assigns, categories, kbID)
		stats = &s
		return next, true
	})
	if saveErr != nil {
		return nil, saveErr
	}
	return stats, nil
}

// classifyForOrganize runs the two LLM phases: propose categories from a
// title sample, then assign every loose article to one of them.
func classifyForOrganize(ctx context.Context, order []core.WikiTocNode) ([]string, map[string]string, error) {
	model := llmmodule.ResolveModel(core.LLMTypeLanguage, nil)
	if model == nil {
		return nil, nil, fmt.Errorf("no default language model configured")
	}
	llm, err := langchain.SimplyGetLLM(model.ProviderID, model.ID, "")
	if err != nil {
		return nil, nil, err
	}

	sample := order
	if len(sample) > organizeSampleSize {
		sample = sample[:organizeSampleSize]
	}
	catResp, err := llm.GenerateContent(ctx, []llms.MessageContent{
		langchain.SystemTextParts(organizeCategoryPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, titlesBlock(sample)),
	}, llms.WithMaxTokens(400))
	if err != nil {
		return nil, nil, fmt.Errorf("category proposal failed: %w", err)
	}
	if catResp == nil || len(catResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("empty category proposal response")
	}
	categories, err := parseCategoryList(catResp.Choices[0].Content)
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
		userText := "Allowed categories: " + allowed + "\n\nPages:\n" + titlesBlock(batch)
		resp, err := llm.GenerateContent(ctx, []llms.MessageContent{
			langchain.SystemTextParts(organizeAssignPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, userText),
		}, llms.WithMaxTokens(4000))
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

// titlesBlock renders one "id. title" line per article — the id is what the
// assignment phase echoes back, the title is what the model judges by.
func titlesBlock(nodes []core.WikiTocNode) string {
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(n.ArticleID)
		b.WriteString(". ")
		b.WriteString(n.Title)
		b.WriteString("\n")
	}
	return b.String()
}

func (h *APIHandler) organizeToc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	userID := requestUserID(req)

	// read-side precheck for an honest immediate answer (no loose pages →
	// nothing to do) and the count the UI polls against
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiToc{})
	toc, _, err := findToc(ctx, kbID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	loose := 0
	for _, n := range toc.Nodes {
		if n.Type != "folder" && n.ArticleID != "" {
			loose++
		}
	}
	if loose == 0 {
		h.WriteJSON(w, util.MapStr{"started": false, "loose": 0}, http.StatusOK)
		return
	}

	// the two-phase classification makes 1 + N model calls and a slow model
	// pushes that past any sane HTTP timeout — run it in the background
	// (the eval-replay pattern) and let the UI poll the tree until the
	// loose count drops
	if !organizeRuns.TryReserve(kbID) {
		h.WriteJSON(w, util.MapStr{"started": false, "already_running": true, "loose": loose}, http.StatusTooManyRequests)
		return
	}
	go func() {
		defer organizeRuns.Release(kbID)
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("wiki: toc organize recovered: %v", r)
			}
		}()
		stats, err := OrganizeTocSemantically(context.Background(), kbID)
		if err != nil {
			log.Warnf("wiki: toc organize failed for kb %s: %v", kbID, err)
			return
		}
		AppendEvolutionEntry(orm.NewContext(), kbID, core.WikiEvolutionTocOrganize, userID,
			"kb", kbID,
			fmt.Sprintf("semantic toc organize: %d loose pages filed into %d categories (%d manual folders kept)", stats.Moved, len(stats.Categories), stats.KeptFolders),
			map[string]interface{}{"categories": stats.Categories, "moved": stats.Moved, "other": stats.Other, "kept_folders": stats.KeptFolders})
		log.Infof("wiki: toc organize kb %s: %d pages into %v", kbID, stats.Moved, stats.Categories)
	}()

	h.WriteJSON(w, util.MapStr{"started": true, "loose": loose}, http.StatusOK)
}
