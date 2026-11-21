/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common/fingerprint"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

/* KB health report — the whole-KB counterpart of the per-page lint
 * (LLM-wiki 体检): one scan tells maintainers where attention pays off
 * next, without opening pages one by one. Advisory only, recomputed
 * live on request so it never drifts from stored warnings.
 *
 * Findings and where they come from:
 *   orphan pages     — no inbound wikilink from any live page (article
 *                      titles, aliases, entity pages and KeyEntities all
 *                      count as inbound anchors)
 *   knowledge gaps   — red links aggregated across the KB: a name that
 *                      pages keep referencing but neither an entity nor a
 *                      page answers. Red links are a feature: each one is
 *                      the next page worth writing.
 *   thin pages       — body under the shared readability floor
 *   unbacked pages   — concept/topic pages with zero cited sources,
 *                      skill pages with no provenance (same rules as
 *                      computeLintWarnings)
 *   hub pages        — highest inbound degree; the pages holding the
 *                      graph together, most expensive to break
 *
 * Archived pages are retired: excluded from findings and their links no
 * longer count as inbound anchors. */

const (
	healthMaxArticles = 1000
	healthHubFloor    = 3 // minimum inbound links before a page reads as a hub
	healthMaxHubs     = 10
	healthMaxGaps     = 20
	healthMaxGapRefs  = 5 // cited-by samples per gap; count carries the rest
)

type KBHealthRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type KBHealthArticle struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	PageType     string   `json:"page_type,omitempty"`
	Status       string   `json:"status,omitempty"`
	Codes        []string `json:"codes,omitempty"`
	InboundLinks int      `json:"inbound_links"`
	ContentLen   int      `json:"content_len"`
}

type KBHealthGap struct {
	Name    string        `json:"name"`
	Count   int           `json:"count"`
	CitedBy []KBHealthRef `json:"cited_by"`
}

type KBHealthReport struct {
	KbID          string            `json:"kb_id"`
	GeneratedAt   time.Time         `json:"generated_at"`
	TotalPages    int               `json:"total_pages"`
	ArchivedPages int               `json:"archived_pages"`
	Counts        map[string]int    `json:"counts"`
	Gaps          []KBHealthGap     `json:"knowledge_gaps,omitempty"`
	Orphans       []KBHealthArticle `json:"orphans,omitempty"`
	Hubs          []KBHealthArticle `json:"hubs,omitempty"`
	Flagged       []KBHealthArticle `json:"flagged,omitempty"`
}

// kbHealth serves GET /wiki/kb/:id/_health.
func (h *APIHandler) kbHealth(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})

	var kb core.WikiKnowledgeBase
	kb.SetID(kbID)
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil && strings.Contains(err.Error(), "record not found") {
		// sqlite reports missing rows as an error (ES returns exists=false)
		err, exists = nil, false
	}
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, kbID)
		return
	}

	report, err := buildKBHealthReport(req.Context(), kbID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, report)
}

func kbHealthArticles(ctx *orm.Context, kbID string) ([]core.WikiArticle, error) {
	orm.WithModel(ctx, &core.WikiArticle{})
	builder := orm.NewQuery().Size(healthMaxArticles).
		Filter(orm.TermQuery("kb_id", kbID)).
		SortBy(orm.Sort{Field: "title", SortType: orm.ASC})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		return nil, err
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	return articles, err
}

func buildKBHealthReport(ctx context.Context, kbID string) (*KBHealthReport, error) {
	readCtx := orm.NewContextWithParent(ctx)
	readCtx.Set(orm.DirectReadWithoutPermissionCheck, true)

	articles, err := kbHealthArticles(readCtx, kbID)
	if err != nil {
		return nil, err
	}
	report := &KBHealthReport{
		KbID:        kbID,
		GeneratedAt: time.Now(),
		TotalPages:  len(articles),
		Counts: map[string]int{
			"orphan_pages":       0,
			"thin_pages":         0,
			"no_source_pages":    0,
			"missing_provenance": 0,
			"knowledge_gaps":     0,
		},
	}

	// inbound anchors: title, aliases, entity page and KeyEntities all
	// make an article reachable from a wikilink
	titleOwner := map[string]string{}
	aliasOwner := map[string]string{}
	entityPage := map[string]string{}
	keyEntityPages := map[string][]string{}
	live := make([]*core.WikiArticle, 0, len(articles))
	// KeyEntities store entity NAMES (keyEntityNames), not IDs — index
	// them normalized so wikilinks can resolve by name. Raw keys stay in
	// a separate map for ID-shaped links (older data).
	keyEntityNamePages := map[string][]string{}
	for i := range articles {
		a := &articles[i]
		if a.Status == "archived" {
			report.ArchivedPages++
			continue
		}
		live = append(live, a)
		if a.Title != "" {
			titleOwner[a.Title] = a.ID
		}
		for _, alias := range a.Aliases {
			if alias != "" {
				aliasOwner[alias] = a.ID
			}
		}
		if a.EntityID != "" {
			entityPage[a.EntityID] = a.ID
		}
		for _, id := range a.KeyEntities {
			if id != "" {
				keyEntityPages[id] = append(keyEntityPages[id], a.ID)
				if n := fingerprint.Normalize(id); n != "" {
					keyEntityNamePages[n] = append(keyEntityNamePages[n], a.ID)
				}
			}
		}
	}

	// one pass over every live page's wikilinks: inbound degrees and the
	// KB-wide red-link gap list (a link that resolved to an entity is
	// never a gap even without a curated page — the entity node answers it)
	inbound := map[string]int{}
	gapByName := map[string]*KBHealthGap{}
	for _, a := range live {
		for _, link := range a.LinkedPages {
			targets := map[string]bool{}
			if link.EntityID != "" {
				if id := entityPage[link.EntityID]; id != "" {
					targets[id] = true
				}
				for _, id := range keyEntityPages[link.EntityID] {
					targets[id] = true
				}
			}
			if link.Name != "" {
				if id := titleOwner[link.Name]; id != "" {
					targets[id] = true
				}
				if id := aliasOwner[link.Name]; id != "" {
					targets[id] = true
				}
				// entity links carry the entity name too; topic pages
				// advertise their key entities by name — resolve through
				// the normalized name index
				if n := fingerprint.Normalize(link.Name); n != "" {
					for _, id := range keyEntityNamePages[n] {
						targets[id] = true
					}
				}
			}
			for id := range targets {
				if id != a.ID {
					inbound[id]++
				}
			}
			if link.EntityID == "" && link.Name != "" && len(targets) == 0 {
				gap := gapByName[link.Name]
				if gap == nil {
					gap = &KBHealthGap{Name: link.Name}
					gapByName[link.Name] = gap
				}
				gap.Count++
				if len(gap.CitedBy) < healthMaxGapRefs {
					gap.CitedBy = append(gap.CitedBy, KBHealthRef{ID: a.ID, Title: a.Title})
				}
			}
		}
	}

	card := func(a *core.WikiArticle, codes []string) KBHealthArticle {
		return KBHealthArticle{
			ID:           a.ID,
			Title:        a.Title,
			PageType:     a.PageType,
			Status:       a.Status,
			Codes:        codes,
			InboundLinks: inbound[a.ID],
			ContentLen:   len([]rune(strings.TrimSpace(a.Content))),
		}
	}

	for _, a := range live {
		codes := make([]string, 0, 3)
		contentLen := len([]rune(strings.TrimSpace(a.Content)))
		if len(live) > 1 && inbound[a.ID] == 0 {
			codes = append(codes, "orphan")
			report.Counts["orphan_pages"]++
		}
		if contentLen < minReadableContent {
			codes = append(codes, core.WikiLintThinContent)
			report.Counts["thin_pages"]++
		}
		// same page rules as computeLintWarnings — the report must never
		// contradict the per-page lint
		if (a.PageType == core.WikiPageTypeConcept || a.PageType == core.WikiPageTypeTopic) && len(a.Sources) == 0 {
			codes = append(codes, core.WikiLintNoSources)
			report.Counts["no_source_pages"]++
		}
		if a.PageType == core.WikiPageTypeSkill && len(a.Sources) == 0 && len(a.LinkedPages) == 0 {
			codes = append(codes, core.WikiLintMissingProvenance)
			report.Counts["missing_provenance"]++
		}
		if len(codes) > 0 {
			report.Flagged = append(report.Flagged, card(a, codes))
			if inbound[a.ID] == 0 && len(live) > 1 {
				report.Orphans = append(report.Orphans, card(a, codes))
			}
		}
		if inbound[a.ID] >= healthHubFloor {
			report.Hubs = append(report.Hubs, card(a, nil))
		}
	}

	// deterministic order: hubs by degree, gaps by demand, flagged by
	// finding count then title (articles arrive title-sorted)
	sort.SliceStable(report.Hubs, func(i, j int) bool {
		return report.Hubs[i].InboundLinks > report.Hubs[j].InboundLinks
	})
	if len(report.Hubs) > healthMaxHubs {
		report.Hubs = report.Hubs[:healthMaxHubs]
	}
	for _, gap := range gapByName {
		report.Gaps = append(report.Gaps, *gap)
	}
	sort.SliceStable(report.Gaps, func(i, j int) bool {
		return report.Gaps[i].Count > report.Gaps[j].Count
	})
	if len(report.Gaps) > healthMaxGaps {
		report.Gaps = report.Gaps[:healthMaxGaps]
	}
	report.Counts["knowledge_gaps"] = len(gapByName)

	return report, nil
}

/* ---------------- GET /wiki/article/:id/backlinks ---------------- */

// articleBacklinks lists the live pages that reference this article through
// wikilinks — the per-page "What links here" face of the KB health scan
// (entity pages already had one via /wiki/entity/:id/backlinks). Anchors
// mirror buildKBHealthReport exactly: title, aliases, entity page and
// KeyEntities all make an article reachable; archived referrers don't
// count. The scan is in-memory — linked_pages is stored unmapped.
func (h *APIHandler) articleBacklinks(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	var target core.WikiArticle
	target.SetID(id)
	exists, err := orm.GetV2(ctx, &target)
	if err != nil && !strings.Contains(err.Error(), "record not found") {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err != nil || !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	articles, err := kbHealthArticles(ctx, target.KbID)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// anchor indexes — same construction as buildKBHealthReport so the
	// panel can never disagree with the health report's inbound count.
	// The TARGET stays in these indexes: it is typically its own entity's
	// entity page, and excluding it here would erase the entity→page
	// mapping the inbound links resolve through. Self-references are
	// filtered in the scan loop below instead.
	titleOwner := map[string]string{}
	aliasOwner := map[string]string{}
	entityPage := map[string]string{}
	keyEntityNamePages := map[string][]string{}
	for i := range articles {
		a := &articles[i]
		if a.Status == "archived" {
			continue
		}
		if a.Title != "" {
			titleOwner[a.Title] = a.ID
		}
		for _, alias := range a.Aliases {
			if alias != "" {
				aliasOwner[alias] = a.ID
			}
		}
		if a.EntityID != "" {
			entityPage[a.EntityID] = a.ID
		}
		for _, ke := range a.KeyEntities {
			if ke == "" {
				continue
			}
			if n := fingerprint.Normalize(ke); n != "" {
				keyEntityNamePages[n] = append(keyEntityNamePages[n], a.ID)
			}
		}
	}

	items := []util.MapStr{}
	for i := range articles {
		a := &articles[i]
		if a.Status == "archived" || a.ID == target.ID {
			continue
		}
		for _, link := range a.LinkedPages {
			matched := false
			if link.EntityID != "" && entityPage[link.EntityID] == target.ID {
				matched = true
			}
			if !matched && link.Name != "" {
				if titleOwner[link.Name] == target.ID || aliasOwner[link.Name] == target.ID {
					matched = true
				} else if n := fingerprint.Normalize(link.Name); n != "" {
					for _, pid := range keyEntityNamePages[n] {
						if pid == target.ID {
							matched = true
						}
					}
				}
			}
			if matched {
				items = append(items, util.MapStr{
					"article_id": a.ID,
					"title":      a.Title,
					"kb_id":      a.KbID,
					"page_type":  a.PageType,
					"status":     a.Status,
					"updated":    a.Updated,
					"link_name":  link.Name,
					"snippet":    backlinkSnippet(a.Content, link.Name),
				})
				break // one row per referrer, even when it links twice
			}
		}
	}

	h.WriteOKJSON(w, util.MapStr{"items": items, "total": len(items)})
}
