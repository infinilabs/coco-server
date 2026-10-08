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
func onDocumentDeleted(docID string) {
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
		fileProposal(kb, &a, core.WikiGovernanceStale,
			"cited document was deleted — this page lost a piece of evidence",
			util.MapStr{
				"doc_id":  docID,
				"cascade": "document-deleted",
			})
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
