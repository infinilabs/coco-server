/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Document→wiki liaison (W11 realtime propagation): the "signal is
// realtime, effecting is human" loop. When an indexing pipeline finishes,
// articles citing that document get an article_refresh proposal — but only
// when the content fingerprint actually changed: dispatcher re-syncs push
// every document every cycle, and an unchanged re-ingestion must not
// flood the governance queue. When a document is deleted, citing articles
// get a stale proposal — evidence loss is marked, never auto-removed.

// liaisonArticleCap bounds the citing-article scan (Sources is stored
// unmapped, the scan is in-memory like the backlink one).
const liaisonArticleCap = 50

func init() {
	common.RegisterDocumentProcessedHandler(onDocumentProcessed)
	common.RegisterDocumentDeletedHandler(onDocumentDeleted)
}

// onDocumentProcessed compares the fresh fingerprint with the STORED
// document (the merge stage has not run yet when this fires — the stored
// row still carries the previous fingerprint) and files refresh proposals
// for the articles citing a genuinely changed document.
func onDocumentProcessed(docID, freshHash string) {
	if freshHash == "" {
		return // no fingerprint to compare — nothing decisive to say
	}

	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.Document{})

	stored := core.Document{}
	stored.ID = docID
	exists, err := orm.GetV2(ctx, &stored)
	if err != nil {
		return
	}
	if exists && stored.ContentHash == freshHash {
		return // resync without change: no signal, no proposal
	}

	articles := articlesCitingDocument(docID)
	if len(articles) == 0 {
		return
	}
	if len(articles) > liaisonArticleCap {
		log.Warnf("wiki: document [%s] changed with %d citing articles, capping at %d", docID, len(articles), liaisonArticleCap)
		articles = articles[:liaisonArticleCap]
	}

	for i := range articles {
		a := articles[i]
		kb := kbForLiaison(a.KbID)
		if kb == nil {
			continue
		}
		// W9 incremental trigger (compileops P1): the corpus change lands as
		// a persistent per-document op — the lane coalesces bursts and
		// survives restarts
		EnqueueCompileOp(kb.ID, "ingest", docID, "content-changed")
		fileProposal(kb, &a, core.WikiGovernanceArticleRefresh,
			"cited document changed (re-ingested with new content fingerprint)",
			util.MapStr{
				"doc_id":       docID,
				"content_hash": freshHash,
				"cascade":      "document-change",
			})
	}
}

// onDocumentDeleted marks the articles that cited the deleted document as
// stale — the human gate decides what happens to evidence-less pages.
// Entities are retracted automatically (WeKnora discipline): entity Sources
// are pipeline-managed provenance, so the ghost citation is stripped at once
// instead of surviving as a dead reference; an entity left with no sources
// at all is demoted back to proposed for the queue to re-examine.
func onDocumentDeleted(docID string) {
	retractEntitySources(docID)

	articles := articlesCitingDocument(docID)
	if len(articles) == 0 {
		return
	}
	if len(articles) > liaisonArticleCap {
		articles = articles[:liaisonArticleCap]
	}
	for i := range articles {
		a := articles[i]
		kb := kbForLiaison(a.KbID)
		if kb == nil {
			continue
		}
		// the loss also reaches the compiler as a retract op (compileops P1)
		EnqueueCompileOp(kb.ID, "retract", docID, "document-deleted")
		fileProposal(kb, &a, core.WikiGovernanceStale,
			"cited document was deleted — this page lost a piece of evidence",
			util.MapStr{
				"doc_id":  docID,
				"cascade": "document-deleted",
			})
	}
}

// retractEntitySources strips the deleted document from every entity's
// Sources (in-memory scan: sources is stored unmapped, the same trade-off
// articlesCitingDocument makes). Entities with no remaining sources keep
// their page but drop back to proposed — their claims no longer trace to
// any document.
func retractEntitySources(docID string) {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiEntity{})

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(graphMaxArticles))
	if err != nil {
		log.Warnf("wiki: entity retract scan failed for [%s]: %v", docID, err)
		return
	}
	entities, _, err := elastic.DecodeHits[core.WikiEntity](res)
	if err != nil {
		return
	}

	retracted := 0
	for i := range entities {
		e := entities[i]
		kept := e.Sources[:0:0]
		for _, s := range e.Sources {
			if s.DocID != docID {
				kept = append(kept, s)
			}
		}
		if len(kept) == len(e.Sources) {
			continue
		}
		e.Sources = kept
		if len(kept) == 0 && e.Status == "published" {
			e.Status = "proposed"
		}
		if err := orm.Update(ctx, &e); err != nil {
			log.Warnf("wiki: failed to retract entity [%s] source [%s]: %v", e.ID, docID, err)
			continue
		}
		retracted++
	}
	if retracted > 0 {
		log.Infof("wiki: retracted deleted document [%s] from %d entities", docID, retracted)
	}
}

// articlesCitingDocument scans published-curated articles for Sources
// entries pointing at the document (in-memory scan: sources is stored
// unmapped, the same trade-off the backlinks endpoint makes).
func articlesCitingDocument(docID string) []core.WikiArticle {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	res, err := orm.SearchV2(ctx, orm.NewQuery().Size(graphMaxArticles).
		Filter(orm.MustNotQuery(orm.TermQuery("status", "archived"))))
	if err != nil {
		return nil
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		return nil
	}

	out := make([]core.WikiArticle, 0, 4)
	for i := range articles {
		for _, s := range articles[i].Sources {
			if s.DocID == docID {
				out = append(out, articles[i])
				break
			}
		}
	}
	return out
}

func kbForLiaison(kbID string) *core.WikiKnowledgeBase {
	ctx := orm.NewContext()
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})
	kb := core.WikiKnowledgeBase{}
	kb.ID = kbID
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil || !exists {
		return nil
	}
	return &kb
}
