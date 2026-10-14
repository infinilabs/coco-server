/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"context"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Published-article projection (W16a, "knowledge base as a built-in
// datasource"): every published wiki article is mirrored into the document
// index as a type=wiki_article row with the same mapping the raw documents
// use. The wiki_article index remains the governance source of truth
// (versions, proposals, review status); the document row is a read-only
// retrieval projection — one-way, never synced back.
//
// The noise gate lives at projection time, not query time: a low-confidence
// page or an under-sourced concept page is simply never projected, so every
// route that reads the unified index inherits the gate for free.

// ProjectionDocID is the deterministic projection row id — upserts are
// idempotent and removal never needs a lookup.
func ProjectionDocID(articleID string) string {
	return "wikiproj_" + articleID
}

// passesProjectionGate mirrors the retrieval noise gate at projection time:
// published-only callers plus confidence floor. Concept-page minimum
// sources apply when rules demand it.
func passesProjectionGate(article *core.WikiArticle, conceptMinSources int) bool {
	if article.Confidence == "low" {
		return false
	}
	if article.PageType == core.WikiPageTypeConcept && conceptMinSources > 0 && len(article.Sources) < conceptMinSources {
		return false
	}
	return true
}

// projectArticleToDocument upserts the projection row for a published
// article. Best-effort: a failed projection logs and returns the error —
// callers decide whether to surface it.
func projectArticleToDocument(article *core.WikiArticle) error {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	// GetV2 below is an OpGet — the get/search hooks bypass on the read flag
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.Document{})

	kbName := "Wiki"
	if kb := loadKBName(article.KbID); kb != "" {
		kbName = kb
	}

	doc := projectionDocument(article, kbName)
	// upsert by hand: the sqlite backend's Upsert errors on a missing row,
	// and the deterministic id makes the existence check cheap
	existing := core.Document{}
	existing.ID = doc.ID
	if exists, _ := orm.GetV2(ctx, &existing); exists {
		doc.Created = existing.Created
		return orm.Update(ctx, doc)
	}
	return orm.Create(ctx, doc)
}

// projectionDocument builds the projection row. The id is deterministic
// (ProjectionDocID) so re-publishing updates in place instead of stacking.
func projectionDocument(article *core.WikiArticle, kbName string) *core.Document {
	sources := make([]interface{}, 0, len(article.Sources))
	endpoint := ""
	if cfg := projectionConfigFn(); cfg.ServerInfo != nil {
		endpoint = cfg.ServerInfo.Endpoint
	}
	for _, s := range article.Sources {
		sources = append(sources, util.MapStr{
			"doc_id":  s.DocID,
			"title":   s.Title,
			"url":     s.URL,
			"excerpt": s.Excerpt,
			"locator": s.Locator,
		})
	}
	doc := &core.Document{
		Title:   article.Title,
		Summary: article.Summary,
		Content: article.Content,
		Type:    "wiki_article",
		Tags:    article.Tags,
		URL:     endpoint + "/#/wiki/article/" + article.ID,
		Source: core.DataSourceReference{
			ID:   "wiki_kb_" + article.KbID,
			Name: kbName,
			Type: "wiki",
			Icon: "font_book",
		},
		Status:    core.DocumentStatusCompleted,
		Processed: true,
	}
	doc.ID = ProjectionDocID(article.ID)
	doc.Metadata = util.MapStr{
		"wiki_article": true,
		"kb_id":        article.KbID,
		"page_type":    article.PageType,
		"confidence":   article.Confidence,
		"source_count": len(article.Sources),
		"wiki_sources": sources,
		"article_id":   article.ID,
	}
	return doc
}

// removeArticleProjection deletes the projection row (article unpublished,
// archived or deleted).
func removeArticleProjection(articleID string) error {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.Document{})

	doc := &core.Document{}
	doc.ID = ProjectionDocID(articleID)
	return orm.Delete(ctx, doc)
}

// SyncArticleProjection reconciles one article's projection with its
// current state — the single entry point for status transitions, content
// edits and deletions. Idempotent: projecting an already-projected article
// upserts; removing a missing row is a no-op. Best-effort by design: a
// projection subsystem failure (missing config store in a test binary, a
// hiccup) logs and gives way — the article write itself must never fail
// because its projection could not be maintained.
func SyncArticleProjection(article *core.WikiArticle) {
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("wiki: article projection skipped (subsystem unavailable): %v", r)
		}
	}()
	if article == nil || article.ID == "" {
		return
	}
	if article.Status == core.WikiArticlePublished && passesProjectionGate(article, projectionConceptMinSources()) {
		if err := projectArticleToDocument(article); err != nil {
			log.Warnf("wiki: projection upsert failed for article [%s]: %v", article.ID, err)
		}
		return
	}
	if err := removeArticleProjection(article.ID); err != nil {
		log.Debugf("wiki: projection removal failed for article [%s]: %v", article.ID, err)
	}
}

// projectionConfigFn is swapped out in tests (AppConfig touches the kv store).
var projectionConfigFn = common.AppConfig

// projectionConceptMinSources reads the compile rule from the tenant-level
// ontology schema; zero keeps the projection gate permissive (the
// wiki_article route keeps enforcing its own gate as today).
func projectionConceptMinSources() int {
	doc := loadOntologySchema(context.Background(), "")
	if doc == nil || doc.Rules == nil {
		return 0
	}
	return doc.Rules.ConceptMinSources
}

func loadKBName(kbID string) string {
	if kbID == "" {
		return ""
	}
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})
	kb := core.WikiKnowledgeBase{}
	kb.ID = kbID
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil || !exists {
		return ""
	}
	return kb.Name
}

// backfillPublishedProjections projects every published article that has
// no projection row yet — the one-off migration before the retrieval leg
// switches to the unified index (W16a). Idempotent by construction:
// existing projections are skipped, counted, never rewritten.
func backfillPublishedProjections() (total, projected, skipped int, err error) {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("status", core.WikiArticlePublished)).
		SortBy(orm.Sort{Field: "updated", SortType: orm.DESC}))
	if err != nil {
		return 0, 0, 0, err
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		return 0, 0, 0, err
	}

	haveCtx := orm.NewContext()
	haveCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(haveCtx, &core.Document{})

	total = len(articles)
	for i := range articles {
		a := &articles[i]
		if !passesProjectionGate(a, projectionConceptMinSources()) {
			skipped++ // gated: stays unprojected, the leg keeps its noise floor
			continue
		}
		existing := core.Document{}
		existing.ID = ProjectionDocID(a.ID)
		if exists, _ := orm.GetV2(haveCtx, &existing); exists {
			skipped++
			continue
		}
		if perr := projectArticleToDocument(a); perr != nil {
			log.Warnf("wiki: backfill projection failed for [%s]: %v", a.ID, perr)
			continue
		}
		projected++
	}
	return total, projected, skipped, nil
}

// wikiProjectionListBody is the backfill endpoint response.
func wikiProjectionListBody(total, projected, skipped int) util.MapStr {
	return util.MapStr{
		"total":     total,
		"projected": projected,
		"skipped":   skipped,
		"note":      strings.TrimSpace("one-off projection of published articles into the document index; flip search_settings.wiki_projection afterwards"),
	}
}

// backfillProjection is the one-off migration endpoint (W16a): project
// every published article lacking a projection row. Idempotent.
func (h *APIHandler) backfillProjection(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	total, projected, skipped, err := backfillPublishedProjections()
	if err != nil {
		h.Error500(w, err.Error())
		return
	}
	h.WriteOKJSON(w, wikiProjectionListBody(total, projected, skipped))
}
