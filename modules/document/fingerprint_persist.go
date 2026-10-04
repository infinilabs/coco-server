/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Fingerprint persistence (D1.5): every document write stamps the content
// fingerprint the dedup engine already computes — via one orm pre-hook, so
// the document API, the datasource API, the pipeline and MCP writes all
// pass through it. Retrieval then folds visible duplicates: same hash in
// one result page collapses onto the highest-ranked copy with a "there are
// N more copies" note. Deep cleanup stays in the dedup report where a human
// decides.

// registerFingerprintHook stamps documents on create and update. Recomputed
// every write: same content yields the same fingerprint (idempotent),
// changed content must never keep the stale one.
func registerFingerprintHook() {
	orm.RegisterDataOperationPreHook(100, func(ctx *orm.Context, _ orm.Operation, model interface{}) (*orm.Context, interface{}, error) {
		if doc, ok := model.(*core.Document); ok {
			ensureDocumentFingerprint(doc)
		}
		return ctx, model, nil
	}, orm.OpCreate, orm.OpUpdate)
}

// ensureDocumentFingerprint computes hash+simhash from the content when
// there is enough of it; blank or too-short content leaves the fields empty.
func ensureDocumentFingerprint(doc *core.Document) {
	if doc == nil {
		return
	}
	hash, sim, ok := contentFingerprint(doc.Content)
	if !ok {
		return
	}
	// int64 reinterpretation: the persisted field is signed so high-bit
	// simhash values stay storable in the engine's long type
	doc.ContentHash, doc.ContentSimhash = hash, int64(sim)
}

// foldDuplicateHits collapses same-content-hash hits within one result
// page onto the first (highest-ranked) occurrence and annotates it with the
// copies; hits without a fingerprint (legacy docs, wiki/assistant pseudo
// hits) pass through untouched. The total stays as reported by the engine —
// this is presentation-level folding, the dedup report remains the deep
// cleanup surface.
func foldDuplicateHits(hits []elastic.DocumentWithMeta[core.Document]) []elastic.DocumentWithMeta[core.Document] {
	seen := map[string]int{} // content_hash → index of the representative
	out := make([]elastic.DocumentWithMeta[core.Document], 0, len(hits))
	for i := range hits {
		hash := hits[i].Source.ContentHash
		if hash == "" {
			out = append(out, hits[i])
			continue
		}
		if first, ok := seen[hash]; ok {
			if out[first].Source.Metadata == nil {
				out[first].Source.Metadata = util.MapStr{}
			}
			dupes, _ := out[first].Source.Metadata["fingerprint_duplicates"].(util.MapStr)
			if dupes == nil {
				dupes = util.MapStr{}
				out[first].Source.Metadata["fingerprint_duplicates"] = dupes
			}
			dupes["count"] = toInt(dupes["count"]) + 1
			ids, _ := dupes["ids"].([]string)
			dupes["ids"] = append(ids, hits[i].ID)
			continue
		}
		seen[hash] = len(out)
		out = append(out, hits[i])
	}
	return out
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}
