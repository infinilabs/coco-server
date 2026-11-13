/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
)

// Document lifecycle liaison (W11): the indexing pipeline and the document
// API fire these events; the wiki module registers handlers that reconcile
// the curated layer (refresh proposals for articles citing a changed
// document, stale proposals when a cited document is deleted). The registry
// keeps the modules decoupled — processors and the document module never
// import the wiki package.
//
// Contract: handlers are synchronous, best-effort and panic-isolated; a
// failing handler logs and gives way, the indexing flow never waits on
// curation and never fails because of it.

var (
	documentProcessedHandlers []DocumentProcessedHandler
	documentDeletedHandlers   []DocumentDeletedHandler
)

// DocumentProcessedHandler receives the post-pipeline state of a document.
// ContentHash is the FRESH fingerprint; comparing it against the stored
// document (still pre-merge when this fires) tells changed from resynced.
type DocumentProcessedHandler func(docID, contentHash string)

// DocumentDeletedHandler receives the id of a deleted document.
type DocumentDeletedHandler func(docID string)

// RegisterDocumentProcessedHandler subscribes to pipeline completions.
func RegisterDocumentProcessedHandler(h DocumentProcessedHandler) {
	documentProcessedHandlers = append(documentProcessedHandlers, h)
}

// RegisterDocumentDeletedHandler subscribes to document deletions.
func RegisterDocumentDeletedHandler(h DocumentDeletedHandler) {
	documentDeletedHandlers = append(documentDeletedHandlers, h)
}

// FireDocumentProcessed notifies every subscriber. Callers pass the doc as
// it leaves the enrichment pipeline (status/error already stamped).
func FireDocumentProcessed(doc *core.Document) {
	if doc == nil || doc.ID == "" {
		return
	}
	for _, h := range documentProcessedHandlers {
		runLiaison(func() { h(doc.ID, doc.ContentHash) })
	}
}

// FireDocumentDeleted notifies every subscriber.
func FireDocumentDeleted(docID string) {
	if docID == "" {
		return
	}
	for _, h := range documentDeletedHandlers {
		runLiaison(func() { h(docID) })
	}
}

// runLiaison isolates one handler: the liaison is advisory, a panic in a
// subscriber must never break indexing or deletion.
func runLiaison(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("liaison: document lifecycle handler skipped (subsystem unavailable): %v", r)
		}
	}()
	fn()
}
