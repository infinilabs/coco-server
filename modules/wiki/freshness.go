/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/tmc/langchaingo/llms"

	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/task"
	"infini.sh/framework/core/util"
)

// Freshness loop (design doc §5.3, stage C5): a scheduled task watches the
// documents under each KB's bound datasources, and when a document that an
// article cites has changed, regenerates the page in single-page mode and
// records an auto-updated version for review — live content is never
// overwritten without a human accepting it.

const (
	freshnessTaskID            = "wiki-freshness-check"
	freshnessKVBucket          = "wiki-freshness"
	freshnessWatermarkKey      = "watermark"
	freshnessDefaultInterval   = "1h"
	freshnessDefaultLookback   = time.Hour
	freshnessMaxChangedDocs    = 500
	freshnessMaxArticlesPerRun = 50
)

func init() {
	registerFreshnessTask()
}

func registerFreshnessTask() {
	interval := strings.TrimSpace(os.Getenv("WIKI_FRESHNESS_INTERVAL"))
	switch strings.ToLower(interval) {
	case "":
		interval = freshnessDefaultInterval
	case "off", "disabled", "0":
		return
	}
	task.RegisterScheduleTask(task.ScheduleTask{
		ID:          freshnessTaskID,
		Group:       "wiki",
		Description: "Wiki freshness loop: regenerate pages whose cited source documents changed (design doc 5.3)",
		Interval:    interval,
		Singleton:   true,
		Task:        freshnessCheck,
	})
}

// freshnessLLM resolves the model used for regeneration; package var so
// tests can stub it.
var freshnessLLM = func() (llms.Model, error) {
	return resolveLanguageLLM("", "")
}

func freshnessCheck(ctx context.Context) {
	llm, err := freshnessLLM()
	if err != nil {
		// no model configured: nothing we can regenerate, retry next cycle
		log.Warnf("wiki: freshness check skipped, %v", err)
		return
	}

	since := readFreshnessWatermark()
	sweep := freshnessSweep(ctx, llm, since)
	log.Infof("wiki: freshness check found %d changed documents, regenerated %d pages, notified %d",
		sweep.changedDocs, sweep.regenerated, sweep.notified)

	writeFreshnessWatermark(sweep.highWater)
}

type freshnessStats struct {
	changedDocs int
	regenerated int
	notified    int
	highWater   time.Time
}

// freshnessSweep is the testable core of the loop: it walks changed
// documents since the watermark, maps them onto citing articles via the
// KB's datasources and each article's SourceReferences, and regenerates
// the hits as auto-updated versions.
func freshnessSweep(ctx context.Context, llm llms.Model, since time.Time) freshnessStats {
	stats := freshnessStats{highWater: time.Now()}

	changed := findDocumentsUpdatedSince(ctx, since, freshnessMaxChangedDocs)
	stats.changedDocs = len(changed)
	if len(changed) == 0 {
		return stats
	}
	for _, doc := range changed {
		if doc.Updated != nil && doc.Updated.After(stats.highWater) {
			stats.highWater = *doc.Updated
		}
	}

	kbs := knowledgeBasesWithDatasources(ctx)
	processed := 0
	for i := range kbs {
		if processed >= freshnessMaxArticlesPerRun {
			break
		}
		kb := &kbs[i]
		kbChanged := documentsForDatasources(changed, kb.DatasourceIDs)
		if len(kbChanged) == 0 {
			continue
		}
		articles := citingArticles(ctx, kb.ID, changedIDs(kbChanged))
		for j := range articles {
			if processed >= freshnessMaxArticlesPerRun {
				break
			}
			article := &articles[j]
			if err := regenerateStaleArticle(ctx, llm, article, mergeArticleDocs(ctx, article, kbChanged)); err != nil {
				log.Warnf("wiki: freshness regeneration of %q failed: %v", article.Title, err)
				continue
			}
			processed++
			stats.regenerated++
			notifyOwner(article.GetOwnerID(), "article", article.ID, "auto-updated",
				fmt.Sprintf("Sources of %q changed; an auto-updated draft version is awaiting review", article.Title))
			stats.notified++
		}
	}
	return stats
}

// findDocumentsUpdatedSince returns recently changed documents, freshest
// first, filtered client-side to `since` (portable across backends, no
// range query needed).
func findDocumentsUpdatedSince(ctx context.Context, since time.Time, limit int) []core.Document {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.Document{})

	builder := orm.NewQuery().Size(limit).
		SortBy(orm.Sort{Field: "updated", SortType: orm.DESC})
	res, err := orm.SearchV2(ormCtx, builder)
	if err != nil {
		log.Warnf("wiki: freshness document scan failed: %v", err)
		return nil
	}
	docs, _, err := elastic.DecodeHits[core.Document](res)
	if err != nil {
		log.Warnf("wiki: freshness document decode failed: %v", err)
		return nil
	}
	out := docs[:0]
	for _, doc := range docs {
		if doc.Updated != nil && doc.Updated.After(since) {
			out = append(out, doc)
		}
	}
	return out
}

// knowledgeBasesWithDatasources lists the KBs the sweep has to consider.
func knowledgeBasesWithDatasources(ctx context.Context) []core.WikiKnowledgeBase {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiKnowledgeBase{})

	res, err := orm.SearchV2(ormCtx, orm.NewQuery().Size(1000))
	if err != nil {
		log.Warnf("wiki: freshness kb scan failed: %v", err)
		return nil
	}
	kbs, _, err := elastic.DecodeHits[core.WikiKnowledgeBase](res)
	if err != nil {
		return nil
	}
	out := kbs[:0]
	for _, kb := range kbs {
		if len(kb.DatasourceIDs) > 0 {
			out = append(out, kb)
		}
	}
	return out
}

// documentsForDatasources keeps only the changed docs bound to one KB.
func documentsForDatasources(docs []core.Document, datasourceIDs []string) []core.Document {
	out := make([]core.Document, 0, len(docs))
	for _, doc := range docs {
		if doc.Source.ID != "" && util.AnyInArrayEquals(datasourceIDs, doc.Source.ID) {
			out = append(out, doc)
		}
	}
	return out
}

func changedIDs(docs []core.Document) map[string]bool {
	ids := make(map[string]bool, len(docs))
	for _, doc := range docs {
		ids[doc.ID] = true
	}
	return ids
}

// citingArticles returns the reviewed/published articles of a KB whose
// SourceReference backlinks hit any of the changed documents. Sources are
// not indexed (enabled:false), so matching runs client-side — which also
// keeps the sqlite backend honest.
func citingArticles(ctx context.Context, kbID string, changed map[string]bool) []core.WikiArticle {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ormCtx, &core.WikiArticle{})

	builder := orm.NewQuery().Size(10000).
		Filter(orm.TermQuery("kb_id", kbID))
	res, err := orm.SearchV2(ormCtx, builder)
	if err != nil {
		log.Warnf("wiki: freshness article scan failed: %v", err)
		return nil
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		return nil
	}
	return MatchArticlesToChangedDocs(articles, changed)
}

// MatchArticlesToChangedDocs filters reviewed/published articles down to
// those citing at least one changed document.
func MatchArticlesToChangedDocs(articles []core.WikiArticle, changed map[string]bool) []core.WikiArticle {
	out := make([]core.WikiArticle, 0, len(articles))
	for _, article := range articles {
		if article.Status != core.WikiArticleReviewed && article.Status != core.WikiArticlePublished {
			continue
		}
		for _, source := range article.Sources {
			if source.DocID != "" && changed[source.DocID] {
				out = append(out, article)
				break
			}
		}
	}
	return out
}

// mergeArticleDocs assembles the regeneration material: the article's own
// cited documents plus the changed documents of the KB it lives in.
func mergeArticleDocs(ctx context.Context, article *core.WikiArticle, kbChanged []core.Document) []core.Document {
	seen := map[string]bool{}
	docs := make([]core.Document, 0, len(article.Sources)+len(kbChanged))

	ids := make([]string, 0, len(article.Sources))
	for _, source := range article.Sources {
		if source.DocID != "" && !seen[source.DocID] {
			seen[source.DocID] = true
			ids = append(ids, source.DocID)
		}
	}
	if len(ids) > 0 {
		ormCtx := orm.NewContextWithParent(ctx)
		ormCtx.Set(orm.DirectReadWithoutPermissionCheck, true)
		orm.WithModel(ormCtx, &core.Document{})
		builder := orm.NewQuery().Size(len(ids)).Filter(orm.TermsQuery("id", ids))
		if res, err := orm.SearchV2(ormCtx, builder); err == nil {
			if found, _, err := elastic.DecodeHits[core.Document](res); err == nil {
				docs = append(docs, found...)
			}
		}
	}
	for _, doc := range kbChanged {
		if !seen[doc.ID] {
			seen[doc.ID] = true
			docs = append(docs, doc)
		}
	}
	return docs
}

// regenerateStaleArticle runs the single-page mode of the §5.1 pipeline
// and records an auto-updated version. The live article content and its
// status are deliberately left alone: the output is a pending-review
// version, publishing stays human (D1).
func regenerateStaleArticle(ctx context.Context, llm llms.Model, article *core.WikiArticle, docs []core.Document) error {
	if len(docs) == 0 {
		return fmt.Errorf("no source documents to regenerate from")
	}
	opts := generationOptions{}
	opts.applyDefaults()

	candidate := pageCandidate{
		Title:    article.Title,
		PageType: article.PageType,
	}
	if candidate.PageType == "" {
		candidate.PageType = core.WikiPageTypeConcept
	}
	for i := range docs {
		candidate.DocRefs = append(candidate.DocRefs, i+1)
	}

	usage := &llmUsage{}
	page, err := draftPage(ctx, llm, candidate, docs, opts, usage)
	if err != nil {
		return err
	}

	summary := fmt.Sprintf("sources changed: %d cited document(s), regenerated by freshness check; confidence %s",
		len(page.sources), page.confidence)
	return writeAutoUpdatedVersion(article, page.content, summary)
}

// writeAutoUpdatedVersion snapshots regenerated content without touching
// the live article (design doc §5.3).
func writeAutoUpdatedVersion(article *core.WikiArticle, content, summary string) error {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiVersion{})

	v := &core.WikiVersion{
		ArticleID:     article.ID,
		Version:       nextVersionNumber(article.ID),
		ChangeType:    core.WikiChangeAutoUpdated,
		ChangeSummary: summary,
		Content:       content,
		CreatedBy:     article.GetOwnerID(),
	}
	return orm.Create(ctx, v)
}

/* ---------------- watermark ---------------- */

// readFreshnessWatermark returns the last sweep time; the first run only
// looks back one interval so a fresh install doesn't regenerate history.
func readFreshnessWatermark() time.Time {
	value, err := kv.GetValue(freshnessKVBucket, []byte(freshnessWatermarkKey))
	if err != nil || len(value) == 0 {
		return time.Now().Add(-freshnessDefaultLookback)
	}
	var watermark time.Time
	if err := util.FromJSONBytes(value, &watermark); err != nil {
		return time.Now().Add(-freshnessDefaultLookback)
	}
	return watermark
}

func writeFreshnessWatermark(t time.Time) {
	if t.IsZero() {
		return
	}
	value, err := util.ToJSONBytes(t)
	if err != nil {
		return
	}
	// recover-guard: the kv handler is registered by the storage module at
	// boot; a missing handler must not take the task loop down
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("wiki: freshness watermark write skipped, kv unavailable: %v", r)
		}
	}()
	if err := kv.AddValue(freshnessKVBucket, []byte(freshnessWatermarkKey), value); err != nil {
		log.Warnf("wiki: freshness watermark write failed: %v", err)
	}
}
