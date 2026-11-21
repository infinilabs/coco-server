/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://www.infinilabs.com
 * Email: hello#infini.ltd */

package wiki

// Contribution inverted index doc → page (DESIGN-incremental-compile.md
// 3.4). Articles carry their evidence as Sources[].doc_id; the compile lane
// needs the reverse — which pages a changed document feeds — without the
// in-memory scan of every article (capped at 50 in the liaison). Rows are
// maintained best-effort at the write choke points (crud hooks +
// snapshotPipelineWrite) and healed once at startup; the consumer treats a
// stale row as a wider, still-correct scope, never a wrong one.

import (
	"context"
	"time"

	log "github.com/cihub/seelog"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// contributionID is deterministic so re-syncs upsert instead of duplicating.
func contributionID(kbID, docID, pageID string) string {
	return util.MD5digest("contrib|" + kbID + "|" + docID + "|" + pageID)
}

func contributionCtx(parent context.Context) *orm.Context {
	if parent == nil {
		parent = context.Background()
	}
	ctx := orm.NewContextWithParent(parent)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiContribution{})
	return ctx
}

// syncArticleContributions re-indexes one article's Sources.
func syncArticleContributions(article *core.WikiArticle) {
	if article == nil {
		return
	}
	docIDs := make([]string, 0, len(article.Sources))
	for _, s := range article.Sources {
		if s.DocID != "" {
			docIDs = append(docIDs, s.DocID)
		}
	}
	syncPageContributions(nil, article.KbID, article.ID, docIDs)
}

// syncPageContributions diff-syncs the page's rows to the desired doc set:
// rows whose doc is no longer cited are deleted, newly cited docs get rows,
// the rest are touched with a fresh Updated stamp.
func syncPageContributions(parent context.Context, kbID, pageID string, docIDs []string) {
	if pageID == "" {
		return
	}
	want := make(map[string]bool, len(docIDs))
	for _, d := range docIDs {
		if d != "" {
			want[d] = true
		}
	}

	ctx := contributionCtx(parent)
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("page_id", pageID)))
	if err != nil {
		log.Debugf("wiki contrib: page query failed for [%s]: %v", pageID, err)
		return
	}
	rows, _, derr := elastic.DecodeHits[core.WikiContribution](res)
	if derr != nil {
		log.Debugf("wiki contrib: page decode failed for [%s]: %v", pageID, derr)
		return
	}

	now := time.Now()
	for _, r := range rows {
		if want[r.DocID] {
			delete(want, r.DocID)
			r.Updated = &now
			_ = orm.Update(ctx, &r)
			continue
		}
		_ = orm.Delete(ctx, &r)
	}
	for d := range want {
		row := core.WikiContribution{KbID: kbID, DocID: d, PageID: pageID}
		row.SetID(contributionID(kbID, d, pageID))
		row.Updated = &now
		if err := orm.Create(ctx, &row); err != nil {
			log.Debugf("wiki contrib: create failed for page [%s] doc [%s]: %v", pageID, d, err)
		}
	}
}

// dropPageContributions removes every row of a deleted page.
func dropPageContributions(parent context.Context, pageID string) {
	if pageID == "" {
		return
	}
	ctx := contributionCtx(parent)
	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("page_id", pageID)))
	if err != nil {
		return
	}
	rows, _, derr := elastic.DecodeHits[core.WikiContribution](res)
	if derr != nil {
		return
	}
	for _, r := range rows {
		_ = orm.Delete(ctx, &r)
	}
}

func contributionsForDocs(parent context.Context, kbID string, docIDs []string) []core.WikiContribution {
	if len(docIDs) == 0 {
		return nil
	}
	return contributionsQuery(parent, kbID, orm.TermsQuery("doc_id", docIDs))
}

func contributionsForPages(parent context.Context, kbID string, pageIDs []string) []core.WikiContribution {
	if len(pageIDs) == 0 {
		return nil
	}
	return contributionsQuery(parent, kbID, orm.TermsQuery("page_id", pageIDs))
}

func contributionsQuery(parent context.Context, kbID string, clause *orm.Clause) []core.WikiContribution {
	ctx := contributionCtx(parent)
	q := orm.NewQuery().Size(1000).Filter(clause)
	if kbID != "" {
		q = q.Filter(orm.TermQuery("kb_id", kbID))
	}
	res, err := orm.SearchV2(ctx, q)
	if err != nil {
		return nil
	}
	rows, _, derr := elastic.DecodeHits[core.WikiContribution](res)
	if derr != nil {
		return nil
	}
	return rows
}

// incrementalScope is what an incremental pass maps and reconciles.
type incrementalScope struct {
	OpDocs        []string // docs the ops name (may include deleted ones)
	AffectedPages []string // pages citing any op doc
	MapDocs       []string // docs to MAP: op docs ∪ docs cited by affected pages
}

// planIncrementalScope folds the contribution graph into the pass scope
// (pure function over the two query results, for tests).
//
// Re-mapping the cited-but-unchanged docs is what keeps claim pools whole:
// a topic page citing A+B, when A changes, must be re-proposed with BOTH
// docs' claims — B's extracts come from the fingerprint cache at zero LLM
// cost. Topics absent from the mapped set are never touched.
func planIncrementalScope(ops []CompileOp, byDoc, byPage []core.WikiContribution) incrementalScope {
	scope := incrementalScope{}
	opDocs := map[string]bool{}
	wholeCorpus := false
	for _, op := range ops {
		if op.DocID == "" {
			wholeCorpus = true
			continue
		}
		if !opDocs[op.DocID] {
			opDocs[op.DocID] = true
			scope.OpDocs = append(scope.OpDocs, op.DocID)
		}
	}
	if wholeCorpus {
		return scope // caller falls back to the full pipeline
	}

	pages := map[string]bool{}
	for _, r := range byDoc {
		if opDocs[r.DocID] && !pages[r.PageID] {
			pages[r.PageID] = true
			scope.AffectedPages = append(scope.AffectedPages, r.PageID)
		}
	}
	mapSet := map[string]bool{}
	scope.MapDocs = append(scope.MapDocs, scope.OpDocs...)
	for _, d := range scope.OpDocs {
		mapSet[d] = true
	}
	for _, r := range byPage {
		if pages[r.PageID] && !mapSet[r.DocID] {
			mapSet[r.DocID] = true
			scope.MapDocs = append(scope.MapDocs, r.DocID)
		}
	}
	return scope
}

/* ---------------- startup reconciliation ---------------- */

// scheduleReconcileContributions heals index drift once at boot: every
// article's rows are rebuilt from its stored Sources, so writes that
// bypassed the choke points self-correct before the lane first runs.
func scheduleReconcileContributions() {
	global.RegisterFuncAfterSetup(func() {
		reconcileContributions()
	})
}

func reconcileContributions() {
	ctx := orm.NewContextWithParent(context.Background())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).Include("id", "kb_id", "sources"))
	if err != nil {
		log.Warnf("wiki contrib: reconcile search failed: %v", err)
		return
	}
	articles, _, derr := elastic.DecodeHits[core.WikiArticle](res)
	if derr != nil {
		log.Warnf("wiki contrib: reconcile decode failed: %v", derr)
		return
	}
	for i := range articles {
		syncArticleContributions(&articles[i])
	}
	log.Infof("wiki contrib: reconcile pass over %d articles done", len(articles))
}
