/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"net/http"
	"strings"
	"time"

	connectorcommon "infini.sh/coco/modules/common"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/kv"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Processing task management (W13a ops face): the ingestion pipeline now
// stamps a lifecycle on every document (W1) — this surface turns those
// stamps into an operable task list. One endpoint answers "what is
// in-flight, what failed and why", one button retries the failures, and
// the sync-status view says when every datasource last pulled.

const (
	processingFailedListSize = 50
	dispatcherLastAccessKV   = "/datasource/lastAccessTime"
	connectorCursorKV        = "/datasource/increment/lastModifiedTime"
)

// processingOverviewHandler reports the lifecycle distribution plus the
// most recent failures with their reasons: GET /search/ops/processing
// ?datasource= optional scope.
func (h *APIHandler) processingOverviewHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	ctx := req.Context()
	datasource := strings.TrimSpace(req.URL.Query().Get("datasource"))

	statuses := util.MapStr{
		"indexing":  int64(0),
		"completed": int64(0),
		"failed":    int64(0),
		"legacy":    int64(0), // written before W1: no status stamp
	}
	if agg := documentStatusDistribution(ctx, datasource); agg != nil {
		var bucketSum int64
		for k, v := range agg {
			if _, known := statuses[k]; known {
				statuses[k] = v
			}
			bucketSum += v
		}
		// empty status is omitempty'd out of the document entirely — no
		// aggregation bucket can ever see it; legacy is the arithmetic
		// remainder (total minus the stamped buckets), backend-agnostic
		if total := countDocs(ctx, datasource, nil); total > bucketSum {
			statuses["legacy"] = total - bucketSum
		}
	}

	failed := failedDocuments(ctx, datasource, processingFailedListSize)
	failedList := make([]util.MapStr, 0, len(failed))
	for i := range failed {
		failedList = append(failedList, util.MapStr{
			"id":           failed[i].ID,
			"title":        failed[i].Title,
			"datasource":   failed[i].Source.Name,
			"error":        failed[i].ErrorMessage,
			"updated":      failed[i].Updated,
			"reprocessurl": "/document/" + failed[i].ID + "/_reprocess?force=true",
		})
	}

	h.WriteOKJSON(w, util.MapStr{
		"statuses":   statuses,
		"failed":     failedList,
		"failed_top": len(failedList),
	})
}

// documentStatusDistribution aggregates the status field (empty bucket
// carries the legacy count separately).
func documentStatusDistribution(ctx context.Context, datasource string) map[string]int64 {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})
	builder := orm.NewQuery().Size(0).AddAgg("status", &orm.TermsAggregation{Field: "status", Size: 10})
	if datasource != "" {
		builder.Filter(orm.TermQuery("source.id", datasource))
	}
	res, err := orm.Aggregate(octx, builder)
	if err != nil || res == nil {
		return nil
	}
	out := map[string]int64{}
	if node := res.Aggs["status"]; node != nil {
		for _, b := range node.Buckets {
			out[b.Key] = b.DocCount
		}
	}
	return out
}

// failedDocuments lists the most recent failed documents with reasons.
func failedDocuments(ctx context.Context, datasource string, size int) []core.Document {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})
	builder := orm.NewQuery().From(0).Size(size).
		Filter(orm.TermQuery("status", core.DocumentStatusFailed))
	if datasource != "" {
		builder.Filter(orm.TermQuery("source.id", datasource))
	}
	res, err := orm.SearchV2(octx, builder)
	if err != nil {
		return nil
	}
	hits, _, err := elastic.DecodeHits[core.Document](res)
	if err != nil {
		return nil
	}
	return hits
}

// retryFailedHandler reprocesses every failed document:
// POST /document/_retry_failed {datasource?: "ds"} — the W1 reset+enqueue
// for each, same semantics as the single-doc _reprocess (failed documents
// always rerun, no fingerprint guard applies).
func (h *APIHandler) retryFailedHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := util.MapStr{}
	_ = h.DecodeJSON(req, &body)
	datasource, _ := body["datasource"].(string)

	ctx := orm.NewContextWithParent(req.Context())
	ctx.DirectReadAccess()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.Document{})

	builder := orm.NewQuery().From(0).Size(reprocessBatchCap).
		Filter(orm.TermQuery("status", core.DocumentStatusFailed))
	if datasource != "" {
		builder.Filter(orm.TermQuery("source.id", datasource))
	}
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	docs, _, err := elastic.DecodeHits[core.Document](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	retried, still := 0, 0
	for i := range docs {
		doc := docs[i]
		doc.Chunks = nil
		doc.Summary = ""
		doc.AiInsights = core.AiInsights{}
		doc.Status = core.DocumentStatusIndexing
		doc.ErrorMessage = ""
		ctx.Refresh = orm.WaitForRefresh
		if err := orm.Save(ctx, &doc); err != nil {
			log.Warnf("failed to reset document [%s] for retry: %v", doc.ID, err)
			still++
			continue
		}
		if err := enqueueForIndexing(&doc); err != nil {
			log.Warnf("failed to enqueue document [%s] for retry: %v", doc.ID, err)
			still++
			continue
		}
		retried++
	}

	h.WriteOKJSON(w, util.MapStr{
		"result":       "ok",
		"failed_seen":  len(docs),
		"retried":      retried,
		"still_failed": still,
		"truncated":    len(docs) >= reprocessBatchCap,
	})
}

// syncStatusHandler reports per-datasource ingestion state:
// GET /datasource/_sync_status — sync config, the dispatcher's last tick
// and the connector's incremental cursor. The kv stamps are raw strings
// (timestamps or API cursors), surfaced verbatim.
func (h *APIHandler) syncStatusHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	ctx := orm.NewContextWithParent(req.Context())
	ctx.DirectReadAccess()
	orm.WithModel(ctx, &core.DataSource{})

	builder := orm.NewQuery().Size(500).
		Filter(orm.MustNotQuery(orm.TermQuery("type", "connector"))).
		SortBy(orm.Sort{Field: "name", SortType: orm.ASC})
	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sources, _, err := elastic.DecodeHits[core.DataSource](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]util.MapStr, 0, len(sources))
	for i := range sources {
		ds := &sources[i]
		item := util.MapStr{
			"id":               ds.ID,
			"name":             ds.Name,
			"type":             ds.Type,
			"enabled":          ds.Enabled,
			"sync_enabled":     ds.SyncConfig.Enabled,
			"sync_interval":    ds.SyncConfig.Interval,
			"last_dispatch":    kvString(dispatcherLastAccessKV, ds.ID),
			"increment_cursor": kvString(connectorCursorKV, ds.ID),
		}
		// sync run journal (W8): last run's state + progress; a running
		// record untouched for 30+ minutes reads as stale (crash evidence,
		// kept verbatim until the next run supersedes it)
		if run, rerr := connectorcommon.ReadSyncRun(ds.ID); rerr == nil && run != nil {
			item["run"] = util.MapStr{
				"run_id":     run.RunID,
				"state":      run.State,
				"started_at": run.StartedAt,
				"updated_at": run.UpdatedAt,
				"batches":    run.Batches,
				"documents":  run.Documents,
				"stale":      connectorcommon.SyncRunStale(run, syncRunStaleAfter),
			}
		}
		items = append(items, item)
	}
	h.WriteOKJSON(w, util.MapStr{"items": items, "total": len(items)})
}

// syncRunStaleAfter is the crash-evidence window: a running record
// untouched this long is almost certainly a dead run (the dispatcher's
// pipeline timeout is minutes; this leaves headroom).
const syncRunStaleAfter = 30 * time.Minute

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}

func kvString(bucket, key string) string {
	v, err := kv.GetValue(bucket, []byte(key))
	if err != nil || len(v) == 0 {
		return ""
	}
	return string(v)
}
