/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"fmt"
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"time"

	log "github.com/cihub/seelog"
)

// Deletion reconciliation (soft tombstones): after a FULL collect run
// closes, compare the document IDs the connector observed against the
// documents present in the engine for that datasource. Engine documents
// absent from the source listing were deleted at the source — they are
// MARKED (source_deleted=true, source_deleted_at=now) rather than removed.
// The copies stay searchable; the datasource document list surfaces the
// flag as a badge so a human decides whether to delete (which then fans
// out through the normal deletion path incl. W11 stale marking). Marks
// clear automatically once the source lists the document again.
//
// Safety rails:
//   - only datasources whose connector performs a full source listing each
//     run (fullScanConnectors) participate — cursor/watermark incremental
//     runs observe a PARTIAL source and must never reconcile;
//   - an empty seen set (connector listed nothing) skips reconciliation:
//     this is the empty-API-response protection against mass false deletes;
//   - when more than markRatioCap of the engine documents are missing in
//     one run, marking is downgraded to log-only: a source outage or API
//     glitch must not tombstone a whole dataset; clearing still runs;
//   - engine listing failures skip the run silently.

// fullScanConnectors lists connector IDs whose collect phase always walks
// the complete source listing, so the seen-ID set is complete for deletion
// detection. Cursor-based and watermark-based connectors are excluded.
var fullScanConnectors = map[string]bool{
	"box":           true,
	"dropbox":       true,
	"google_drive":  true,
	"s3":            true,
	"network_drive": true,
	"local_fs":      true,
}

// markRatioCap bounds tombstone marking per run: when the missing share of
// engine documents exceeds it, the run logs the candidates but marks
// nothing — consistent with the empty-seen protection, just graduated.
const markRatioCap = 0.30

// ShouldReconcile reports whether a closed run of this datasource is
// eligible for deletion reconciliation.
func ShouldReconcile(ds *core.DataSource) bool {
	return ds != nil && fullScanConnectors[ds.Connector.ConnectorID]
}

// ReconcileDeletedDocs applies source-deletion tombstones: documents the
// engine has but the source listing no longer shows get the
// source_deleted flag; previously flagged documents that reappeared get
// it cleared. Returns (seenCount, markedCount, clearedCount).
func ReconcileDeletedDocs(ds *core.DataSource, seenIDs map[string]struct{}) (seenCount, markedCount, clearedCount int) {
	if ds == nil || len(seenIDs) == 0 {
		return 0, 0, 0 // empty source protection: nothing observed, nothing to judge
	}
	seenCount = len(seenIDs)

	// id → currently tombstoned; presence in the map = presence in engine
	marked, err := listEngineDocMarks(ds.ID)
	if err != nil {
		log.Errorf("reconcile: failed to list engine documents for datasource [%s(%s)]: %v — skipped", ds.Name, ds.ID, err)
		return seenCount, 0, 0
	}
	engineCount := len(marked)

	var missing []string
	for id := range marked {
		if _, ok := seenIDs[id]; !ok {
			missing = append(missing, id)
		}
	}
	missingCount := len(missing)

	// clear first: reappearance always un-tombstones, even under the cap
	for id := range marked {
		if _, seen := seenIDs[id]; seen {
			doc := core.Document{}
			doc.ID = id
			if err := patchDocTombstone(&doc, false); err != nil {
				log.Warnf("reconcile: failed to clear source_deleted on document [%s]: %v", id, err)
				continue
			}
			clearedCount++
		}
	}
	if clearedCount > 0 {
		log.Infof("reconcile: datasource [%s(%s)] cleared source_deleted on %d reappeared document(s)", ds.Name, ds.ID, clearedCount)
	}

	if missingCount == 0 {
		log.Debugf("reconcile: datasource [%s(%s)] no deletions detected (source: %d, engine: %d)", ds.Name, ds.ID, seenCount, engineCount)
		return seenCount, markedCount, clearedCount
	}

	// graduated mass-deletion guard: report only when too much vanished
	if engineCount > 0 && float64(missingCount)/float64(engineCount) > markRatioCap {
		// cap the ID sample so a mass-deletion event cannot flood the log
		const sampleCap = 20
		sample := missing
		if len(sample) > sampleCap {
			sample = sample[:sampleCap]
		}
		log.Warnf("reconcile: datasource [%s(%s)] %d of %d engine docs missing at source (over %.0f%% cap) — tombstones withheld, review required. candidates: %v%s",
			ds.Name, ds.ID, missingCount, engineCount, markRatioCap*100, sample,
			map[bool]string{true: fmt.Sprintf(" … (+%d more)", missingCount-sampleCap), false: ""}[missingCount > sampleCap])
		return seenCount, markedCount, clearedCount
	}

	now := time.Now().Format(time.RFC3339)
	for _, id := range missing {
		doc := core.Document{}
		doc.ID = id
		if err := patchDocTombstone(&doc, true, now); err != nil {
			log.Warnf("reconcile: failed to mark source_deleted on document [%s]: %v", id, err)
			continue
		}
		markedCount++
	}
	log.Infof("reconcile: datasource [%s(%s)] marked %d/%d missing document(s) as source_deleted (source: %d, engine: %d)",
		ds.Name, ds.ID, markedCount, missingCount, seenCount, engineCount)

	return seenCount, markedCount, clearedCount
}

// patchDocTombstone writes the source_deleted delta as a partial update —
// a full overwrite would race with concurrent enrichment writes on the
// same document. With marked=false the timestamp is cleared too. The
// reconciler runs in a background goroutine without a session user, so
// the write bypasses the security ownership hook (DirectWrite...), same
// as the other system-level jobs.
func patchDocTombstone(doc *core.Document, marked bool, markedAt ...string) error {
	ctx := orm.NewContext()
	// both flags: the update hook checks DirectWrite..., but saveOrUpdate
	// also fetches the previous object (OpGet) with the same context,
	// which checks DirectRead... — missing either panics in the hook
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	delta := util.MapStr{"source_deleted": marked}
	if marked && len(markedAt) > 0 {
		delta["source_deleted_at"] = markedAt[0]
	} else {
		delta["source_deleted_at"] = ""
	}
	return orm.UpdatePartialFields(ctx, doc, delta)
}

// listEngineDocMarks pages through all documents of one datasource and
// returns id → whether the document currently carries the source_deleted
// tombstone. Presence in the result = presence in the engine.
func listEngineDocMarks(datasourceID string) (map[string]bool, error) {
	marks := map[string]bool{}
	const pageSize = 1000
	for from := 0; ; from += pageSize {
		q := orm.Query{}
		q.From = from
		q.Size = pageSize
		q.Filter = &orm.Cond{
			BoolType:  orm.Filter,
			QueryType: orm.QueryTerms,
			Field:     "source.id",
			Value:     []interface{}{datasourceID},
		}

		var results []core.Document
		err, _ := orm.SearchWithJSONMapper(&results, &q)
		if err != nil {
			return nil, err
		}

		for i := range results {
			if results[i].ID != "" {
				marks[results[i].ID] = results[i].SourceDeleted
			}
		}
		if len(results) < pageSize {
			break
		}
	}
	return marks, nil
}
