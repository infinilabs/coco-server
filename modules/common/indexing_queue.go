/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package common

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
)

// IndexingQueueName is the ingestion entry the connector dispatcher, the
// webhook ingester and (since W0) the document/datasource APIs all feed.
// Its consumer runs the per-datasource enrichment pipeline and the merge
// stage upserts by document id, so re-queuing an existing document is
// idempotent — the safe way to re-chunk/re-embed after an edit.
const IndexingQueueName = "indexing_documents"

// EnqueueForIndexing pushes a document onto the indexing queue so the
// enrichment pipeline runs on it. Used by the write-path closure (W0):
// a content edit and an API/datasource-API create must go through the
// same chunking/summarizing/embedding as connector-sourced documents,
// instead of silently landing chunk-less and vector-less.
func EnqueueForIndexing(doc *core.Document) error {
	if doc == nil {
		return nil
	}
	cfg := queue.SmartGetOrInitConfig(&queue.QueueConfig{Name: IndexingQueueName})
	return queue.Push(cfg, util.MustToJSONBytes(doc))
}
