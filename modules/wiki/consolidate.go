/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// Context organize & filter (POST /wiki/kb/:id/_consolidate): the compile
// files a proposal per extracted key, so a small corpus can still produce
// dozens of one-claim stub pages that hollow out the navigation. This pass
// re-organizes them against the whole-KB context:
//
//   - thin pages (a single 来源主张, little else) fold into their dimension
//     overview page as a short section — the fragment stays readable and
//     cited, but stops occupying the tree
//   - near-duplicate titles ("Maxim's Cake" vs "Maxim's Cakes") merge into
//     the fullest page of the group
//   - the TOC is rewritten from the surviving pages
//
// Everything is draft-level; published pages are never touched.
var consolidateNoisePattern = regexp.MustCompile(`[\s\p{P}]+`)

type consolidateStats struct {
	Scanned    int      `json:"scanned"`
	Merged     int      `json:"merged"`
	Deduped    int      `json:"deduped"`
	Kept       int      `json:"kept"`
	MergedInto []string `json:"merged_into,omitempty"`
	RemovedIDs []string `json:"removed_ids,omitempty"`
}

func (h *APIHandler) consolidateKB(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh

	stats, err := consolidateKBContent(ctx, kbID)
	if err != nil {
		log.Warnf("wiki: consolidate kb [%s] failed: %v", kbID, err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{"result": "ok", "stats": stats})
}

// thinPage: a draft whose body carries at most one 来源主张 and almost no
// prose beyond it — the page adds nothing its citation doesn't already say.
func thinPageBody(content string) bool {
	claims := strings.Count(content, "(来源:")
	if claims > 1 {
		return false
	}
	// strip structure (banner, headings, bullets, citations) and measure
	// what's actually left
	lines := strings.Split(content, "\n")
	prose := 0
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, ">") || strings.HasPrefix(t, "#") ||
			strings.HasPrefix(t, "-") || strings.Contains(t, "(来源:") ||
			strings.HasPrefix(t, "*") || strings.Contains(t, "自动起草") ||
			strings.Contains(t, "自动汇编") {
			continue
		}
		prose += len([]rune(t))
	}
	return prose < 120
}

func normalizeTitleKey(title string) string {
	t := strings.ToLower(strings.TrimSpace(title))
	return consolidateNoisePattern.ReplaceAllString(t, "")
}

func consolidateKBContent(ctx *orm.Context, kbID string) (*consolidateStats, error) {
	stats := &consolidateStats{}
	orm.WithModel(ctx, &core.WikiArticle{})

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("kb_id", kbID), orm.TermQuery("status", core.WikiArticleDraft)))
	if err != nil || res == nil {
		return nil, fmt.Errorf("draft scan failed: %v", err)
	}
	articles, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		return nil, derr
	}
	stats.Scanned = len(articles)
	if len(articles) == 0 {
		return stats, nil
	}

	// dimension per page: entity type via linked pages (resolved at save)
	// or the entity referenced by the page
	etypeByEntity := map[string]string{}
	if eids := articleEntityIDs(articles); len(eids) > 0 {
		orm.WithModel(ctx, &core.WikiEntity{})
		if eres, eerr := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
			Filter(orm.TermsQuery("id", eids))); eerr == nil && eres != nil {
			if entities, _, dd := elastic.DecodeHits[core.WikiEntity](eres); dd == nil {
				for i := range entities {
					etypeByEntity[entities[i].ID] = entities[i].Type
				}
			}
		}
	}

	dimension := func(a *core.WikiArticle) string {
		for _, lp := range a.LinkedPages {
			if t := etypeByEntity[lp.EntityID]; t != "" {
				if t == "organization" || t == "person" {
					return "org"
				}
				if t == "product" {
					return "product"
				}
				return "topic"
			}
		}
		if t := etypeByEntity[a.EntityID]; t != "" {
			if t == "organization" || t == "person" {
				return "org"
			}
			if t == "product" {
				return "product"
			}
		}
		return "topic"
	}
	overviewTitle := map[string]string{
		"org": "客户与组织概览", "product": "产品与服务概览", "topic": "专题与概念概览",
	}

	// group thin pages by dimension; group near-duplicate titles overall.
	// Impact protection (LLM-wiki P3): a thin page the answer pipeline
	// actually uses is doing its job — never absorb it into an overview.
	thinByDim := map[string][]*core.WikiArticle{}
	byTitleKey := map[string][]*core.WikiArticle{}
	for i := range articles {
		a := &articles[i]
		title := strings.TrimSpace(a.Title)
		if title == "" {
			continue
		}
		if thinPageBody(a.Content) && !impactProtected(a) {
			thinByDim[dimension(a)] = append(thinByDim[dimension(a)], a)
		}
		key := normalizeTitleKey(title)
		if key == "" {
			continue
		}
		byTitleKey[key] = append(byTitleKey[key], a)
	}

	// 1) fold thin pages into their dimension overview (kept as a cited
	//    fragment under 已收录片段), so the context keeps them readable
	overviewByDim := map[string]*core.WikiArticle{}
	for dim, thin := range thinByDim {
		if len(thin) == 0 {
			continue
		}
		ovTitle, ok := overviewTitle[dim]
		if !ok {
			continue
		}
		ov := core.WikiArticle{}
		ov.ID = overviewPageID(ovTitle)
		found, err := orm.GetV2(ctx, &ov)
		if err != nil || !found {
			continue // no overview page in this KB — leave the thin pages be
		}
		overviewByDim[dim] = &ov
		var sb strings.Builder
		for _, a := range thin {
			claim := firstClaim(a.Content)
			sb.WriteString(fmt.Sprintf("- **%s** — %s\n", a.Title, claim))
			stats.Merged++
			stats.RemovedIDs = append(stats.RemovedIDs, a.ID)
		}
		section := fmt.Sprintf("\n## 已收录片段(%s 更新)\n\n%s", time.Now().Format("2006-01-02 15:04"), sb.String())
		ov.Content = strings.TrimRight(ov.Content, "\n") + "\n" + section
		if err := orm.Update(ctx, &ov); err != nil {
			log.Warnf("wiki: consolidate overview [%s] update failed: %v", ovTitle, err)
			stats.Merged -= len(thin)
			stats.RemovedIDs = stats.RemovedIDs[:len(stats.RemovedIDs)-len(thin)]
		}
	}
	stats.MergedInto = []string{"客户与组织概览", "产品与服务概览", "专题与概念概览"}

	// 2) merge near-duplicate title groups into the fullest page
	for _, group := range byTitleKey {
		if len(group) < 2 {
			continue
		}
		sort.Slice(group, func(i, j int) bool { return len(group[i].Content) > len(group[j].Content) })
		keep := group[0]
		for _, other := range group[1:] {
			if other.ID == keep.ID || other.Status != core.WikiArticleDraft {
				continue
			}
			for _, lp := range other.LinkedPages {
				known := false
				for _, klp := range keep.LinkedPages {
					if klp.EntityID == lp.EntityID && klp.Name == lp.Name {
						known = true
						break
					}
				}
				if !known {
					keep.LinkedPages = append(keep.LinkedPages, lp)
				}
			}
			if strings.TrimSpace(keep.Summary) == "" {
				keep.Summary = other.Summary
			}
			if err := orm.Update(ctx, keep); err != nil {
				log.Debugf("wiki: consolidate merge update failed: %v", err)
				continue
			}
			if err := orm.Delete(ctx, other); err != nil {
				log.Debugf("wiki: consolidate duplicate delete failed [%s]: %v", other.ID, err)
				continue
			}
			stats.Deduped++
			stats.RemovedIDs = append(stats.RemovedIDs, other.ID)
		}
	}

	// 3) delete the folded thin pages + rewrite the TOC from survivors
	for _, id := range stats.RemovedIDs {
		del := core.WikiArticle{}
		del.ID = id
		if err := orm.Delete(ctx, &del); err != nil {
			log.Debugf("wiki: consolidate delete failed [%s]: %v", id, err)
		}
	}
	if len(stats.RemovedIDs) > 0 {
		if err := rewriteTocWithout(ctx, kbID, stats.RemovedIDs); err != nil {
			log.Warnf("wiki: consolidate toc rewrite failed: %v", err)
		}
	}
	stats.Kept = stats.Scanned - len(stats.RemovedIDs)
	return stats, nil
}

func articleEntityIDs(articles []core.WikiArticle) []string {
	out := make([]string, 0, len(articles))
	for i := range articles {
		if articles[i].EntityID != "" {
			out = append(out, articles[i].EntityID)
		}
		for _, lp := range articles[i].LinkedPages {
			if lp.EntityID != "" {
				out = append(out, lp.EntityID)
			}
		}
	}
	return out
}

func firstClaim(content string) string {
	for _, l := range strings.Split(content, "\n") {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, ">") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "*") {
			continue
		}
		t = strings.TrimPrefix(t, "- ")
		r := []rune(t)
		if len(r) > 160 {
			t = string(r[:160]) + "…"
		}
		return t
	}
	return ""
}

// rewriteTocWithout drops every node whose article id is in removedIDs.
func rewriteTocWithout(ctx *orm.Context, kbID string, removedIDs []string) error {
	gone := map[string]bool{}
	for _, id := range removedIDs {
		gone[id] = true
	}
	toc, found, err := findToc(ctx, kbID)
	if err != nil || !found {
		return err
	}
	var prune func(nodes []core.WikiTocNode) []core.WikiTocNode
	prune = func(nodes []core.WikiTocNode) []core.WikiTocNode {
		out := nodes[:0]
		for _, n := range nodes {
			if n.ArticleID != "" && gone[n.ArticleID] {
				continue
			}
			n.Children = prune(n.Children)
			out = append(out, n)
		}
		return out
	}
	toc.Nodes = prune(toc.Nodes)
	if len(toc.Nodes) == 0 {
		return nil
	}
	return orm.Update(ctx, toc)
}

// overviewPageID resolves the overview page id by title (the consolidate
// pass merges fragments into it).
func overviewPageID(title string) string {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("title", title)))
	if err != nil || res == nil {
		return ""
	}
	articles, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil || len(articles) == 0 {
		return ""
	}
	return articles[0].ID
}
