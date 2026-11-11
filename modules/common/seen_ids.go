/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"sync"
)

// Seen-ID registry: during one datasource collect run, BatchCollect records
// every document ID the connector observed — BEFORE the fingerprint skip
// gate, so gate-dropped documents still count as seen. The dispatcher opens
// a collection when it dispatches a datasource and takes the set once the
// run closes, feeding the deletion reconciliation (reconcile.go).
//
// One datasource's runs never overlap (singleton dispatch pipeline), so a
// flat map keyed by datasource ID is safe. Recording is a no-op unless a
// collection is open, so non-dispatched writes (webhook push, direct _doc)
// cost nothing.

var (
	seenIDsMu  sync.Mutex
	seenIDSets = map[string]map[string]struct{}{}
)

// BeginSeenCollection starts (or resets) ID recording for a datasource run.
func BeginSeenCollection(datasourceID string) {
	seenIDsMu.Lock()
	defer seenIDsMu.Unlock()
	seenIDSets[datasourceID] = map[string]struct{}{}
}

// RecordSeenDocID records one observed document ID; no-op when the
// datasource has no open collection.
func RecordSeenDocID(datasourceID, docID string) {
	if docID == "" {
		return
	}
	seenIDsMu.Lock()
	defer seenIDsMu.Unlock()
	if set, ok := seenIDSets[datasourceID]; ok {
		set[docID] = struct{}{}
	}
}

// TakeSeenIDs returns the collected set and clears the recording state.
// nil when nothing was collected (no open run).
func TakeSeenIDs(datasourceID string) map[string]struct{} {
	seenIDsMu.Lock()
	defer seenIDsMu.Unlock()
	set, ok := seenIDSets[datasourceID]
	if !ok {
		return nil
	}
	delete(seenIDSets, datasourceID)
	return set
}
