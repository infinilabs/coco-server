/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"fmt"
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"

	log "github.com/cihub/seelog"
)

// Deletion reconciliation (DRY-RUN): after a FULL collect run closes,
// compare the document IDs the connector observed against the IDs present
// in the engine for that datasource. Engine documents absent from the
// source listing were deleted at the source — today this only LOGS the
// candidates; acting on them (orm delete + downstream W11 stale marking)
// comes after the report has been observed in real deployments.
//
// Safety rails:
//   - only datasources whose connector performs a full source listing each
//     run (fullScanConnectors) participate — cursor/watermark incremental
//     runs observe a PARTIAL source and must never reconcile;
//   - an empty seen set (connector listed nothing) skips reconciliation:
//     this is the empty-API-response protection against mass false deletes;
//   - engine listing failures skip the run silently.

// fullScanConnectors lists connector IDs whose collect phase always walks
// the complete source listing, so the seen-ID set is complete for deletion
// detection. Cursor-based and watermark-based connectors are excluded.
var fullScanConnectors = map[string]bool{
	"box":          true,
	"dropbox":      true,
	"google_drive": true,
	"s3":           true,
	"network_drive": true,
	"local_fs":     true,
}

// ShouldReconcile reports whether a closed run of this datasource is
// eligible for deletion reconciliation.
func ShouldReconcile(ds *core.DataSource) bool {
	return ds != nil && fullScanConnectors[ds.Connector.ConnectorID]
}

// ReconcileDeletedDryRun compares the seen-ID set against the engine and
// logs the deletion candidates. Safe to call with nil seenIDs (no-op).
func ReconcileDeletedDryRun(ds *core.DataSource, seenIDs map[string]struct{}) (seenCount, engineCount, missingCount int) {
	if ds == nil || len(seenIDs) == 0 {
		return 0, 0, 0 // empty source protection: nothing observed, nothing to judge
	}
	seenCount = len(seenIDs)

	engineIDs, err := listEngineDocIDs(ds.ID)
	if err != nil {
		log.Errorf("reconcile: failed to list engine documents for datasource [%s(%s)]: %v — skipped", ds.Name, ds.ID, err)
		return seenCount, 0, 0
	}
	engineCount = len(engineIDs)

	var missing []string
	for id := range engineIDs {
		if _, ok := seenIDs[id]; !ok {
			missing = append(missing, id)
		}
	}
	missingCount = len(missing)

	if missingCount == 0 {
		log.Debugf("reconcile: datasource [%s(%s)] no deletions detected (source: %d, engine: %d)", ds.Name, ds.ID, seenCount, engineCount)
		return seenCount, engineCount, 0
	}

	// dry-run: report only. Cap the ID sample so a mass-deletion event
	// cannot flood the log.
	const sampleCap = 20
	sample := missing
	if len(sample) > sampleCap {
		sample = sample[:sampleCap]
	}
	log.Warnf("reconcile (dry-run): datasource [%s(%s)] source has %d docs, engine has %d docs, %d engine docs no longer seen at source. candidates: %v%s",
		ds.Name, ds.ID, seenCount, engineCount, missingCount, sample,
		map[bool]string{true: fmt.Sprintf(" … (+%d more)", missingCount-sampleCap), false: ""}[missingCount > sampleCap])

	return seenCount, engineCount, missingCount
}

// listEngineDocIDs pages through all document IDs of one datasource.
func listEngineDocIDs(datasourceID string) (map[string]struct{}, error) {
	ids := map[string]struct{}{}
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
				ids[results[i].ID] = struct{}{}
			}
		}
		if len(results) < pageSize {
			break
		}
	}
	return ids, nil
}
