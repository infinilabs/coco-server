/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common/fingerprint"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Chunk-index projection (W3 方案 c): documents carry their chunks as
// nested objects (the processing timeline's view); this projection
// mirrors them into the standalone KnowledgeChunk index where the
// retrieval legs can hit them directly. One mom row per section
// (Available=false, the context target), one child row per chunk
// (Available=true, the retrievable unit), MomID linking the two.
//
// Fired from the W11 liaison (document processed) and from _reprocess —
// every path that re-chunks also re-projects. Idempotent: projection
// clears the document's old rows first (deterministic doc-scoped query).

const chunkIndexBatchSize = 200

// ProjectDocumentChunksFn is the swappable seam (import direction:
// processors → document is one-way; tests swap it out).
var ProjectDocumentChunksFn = ProjectDocumentChunks

// ProjectDocumentChunks mirrors a document's nested chunks into the
// KnowledgeChunk index. Destructive-then-constructive per document:
// old rows die, new rows land — reprocessing never stacks ghosts.
func ProjectDocumentChunks(ctx context.Context, doc *core.Document) error {
	if doc == nil || doc.ID == "" {
		return nil
	}

	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	octx.Refresh = orm.WaitForRefresh
	orm.WithModel(octx, &core.KnowledgeChunk{})

	// clear this document's previous projection
	delCtx := orm.NewContextWithParent(ctx)
	delCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(delCtx, &core.KnowledgeChunk{})
	if _, err := orm.DeleteByQuery(delCtx, orm.NewQuery().Size(10000).
		Filter(orm.TermQuery("doc_id", doc.ID))); err != nil {
		log.Warnf("chunk index: clearing old rows for [%s] failed: %v", doc.ID, err)
	}

	if len(doc.Chunks) == 0 {
		return nil
	}

	// group consecutive chunks by breadcrumb — each group gets one mom
	type sectionGroup struct {
		breadcrumb string
		start, end int // chunk index range [start, end)
		momRowID   string
	}
	var groups []sectionGroup
	for i, c := range doc.Chunks {
		bc := c.Breadcrumb
		if len(groups) > 0 && groups[len(groups)-1].breadcrumb == bc {
			groups[len(groups)-1].end = i + 1
			continue
		}
		groups = append(groups, sectionGroup{breadcrumb: bc, start: i, end: i + 1})
	}

	rows := make([]*core.KnowledgeChunk, 0, len(doc.Chunks)+len(groups))
	for _, g := range groups {
		// mom row: the section's full text as context, not retrievable
		var sb strings.Builder
		for i := g.start; i < g.end; i++ {
			sb.WriteString(doc.Chunks[i].Text)
			sb.WriteString("\n")
		}
		mom := &core.KnowledgeChunk{
			DocID:      doc.ID,
			Source:     doc.Source,
			Seq:        g.start,
			ChunkType:  core.ChunkTypeMom,
			Breadcrumb: g.breadcrumb,
			Text:       strings.TrimSpace(sb.String()),
			Available:  false,
			ModelID:    doc.EmbeddingModel,
		}
		mom.ID = chunkRowID(doc.ID, "mom", g.start)
		g.momRowID = mom.ID
		rows = append(rows, mom)

		for i := g.start; i < g.end; i++ {
			c := doc.Chunks[i]
			child := &core.KnowledgeChunk{
				DocID:      doc.ID,
				Source:     doc.Source,
				Seq:        i,
				ChunkType:  chunkTypeOf(c),
				Breadcrumb: c.Breadcrumb,
				Text:       c.Text,
				Quote:      truncateRunes(c.Text, 300),
				Locators:   locatorOf(c),
				MomID:      g.momRowID,
				Available:  true,
				Embedding:  c.Embedding,
				ModelID:    doc.EmbeddingModel,
			}
			if h, _, ok := fingerprint.Compute(c.Text); ok {
				child.ContentHash = h
			}
			child.ID = chunkRowID(doc.ID, "c", i)
			rows = append(rows, child)
		}
	}

	for from := 0; from < len(rows); from += chunkIndexBatchSize {
		end := from + chunkIndexBatchSize
		if end > len(rows) {
			end = len(rows)
		}
		for _, row := range rows[from:end] {
			if err := orm.Create(octx, row); err != nil {
				log.Warnf("chunk index: row create failed for [%s/%d]: %v", doc.ID, row.Seq, err)
			}
		}
	}
	log.Infof("chunk index: projected %d rows (%d chunks + %d moms) for [%s]",
		len(rows), len(doc.Chunks), len(groups), doc.Title)
	return nil
}

func chunkRowID(docID, kind string, seq int) string {
	return fmt.Sprintf("%s_%s_%06d", docID, kind, seq)
}

func chunkTypeOf(c core.DocumentChunk) string {
	// the structured splitter doesn't tag chunk types yet — every chunk
	// is text until table detection lands; FAQ docs tag their own chunks
	if strings.HasPrefix(strings.TrimSpace(c.Text), "|") {
		return core.ChunkTypeTable
	}
	return core.ChunkTypeText
}

func locatorOf(c core.DocumentChunk) map[string]interface{} {
	if c.Range.Start == 0 && c.Range.End == 0 {
		return nil
	}
	return map[string]interface{}{
		"pages": map[string]interface{}{
			"start": c.Range.Start,
			"end":   c.Range.End,
		},
	}
}

// chunkIndexRoute is the retrieval face: BM25 over child chunks, each
// hit carrying its mom link and locator for citation. Fallback when the
// index has no rows for the query's scope: the caller (hybrid_rrf text
// leg) falls back to document-level BM25.
func (h *APIHandler) chunkIndexRoute(req *http.Request, query string, window int) (*elastic.SearchResponseWithMeta[core.Document], error) {
	octx := orm.NewContextWithParent(req.Context())
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.KnowledgeChunk{})

	builder := orm.NewQuery().From(0).Size(window)
	builder.Query(query)
	builder.DefaultQueryField("text^2", "breadcrumb^4", "doc_id")
	builder.Filter(
		orm.TermQuery("available", true),
		orm.MustNotQuery(orm.TermQuery("chunk_type", core.ChunkTypeMom)),
	)

	res, err := orm.SearchV2(octx, builder)
	if err != nil {
		return nil, err
	}
	chunks, _, err := elastic.DecodeHits[core.KnowledgeChunk](res)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, nil // signal fallback to document-level
	}

	// wrap as document hits: group by doc_id, best chunk wins (the RRF
	// math is doc-level; chunk metadata rides in metadata for citations)
	byDoc := map[string]int{} // doc_id → index in out
	out := &elastic.SearchResponseWithMeta[core.Document]{}
	for i := range chunks {
		c := &chunks[i]
		idx, ok := byDoc[c.DocID]
		if !ok {
			hit := chunkToDocumentHit(c, float32(window-i))
			out.Hits.Hits = append(out.Hits.Hits, hit)
			byDoc[c.DocID] = len(out.Hits.Hits) - 1
			continue
		}
		// earlier hits rank higher (window-i descends); nothing to do
		_ = idx
	}
	out.Hits.Total = map[string]interface{}{"value": int64(len(out.Hits.Hits)), "relation": "eq"}
	return out, nil
}

func chunkToDocumentHit(c *core.KnowledgeChunk, score float32) elastic.DocumentWithMeta[core.Document] {
	quote := c.Quote
	if quote == "" {
		quote = truncateRunes(c.Text, 300)
	}
	meta := util.MapStr{
		"chunk":       true,
		"chunk_id":    c.ID,
		"chunk_type":  c.ChunkType,
		"breadcrumb":  c.Breadcrumb,
		"quote":       quote,
		"mom_id":      c.MomID,
		"seq":         c.Seq,
		"chunk_model": c.ModelID,
	}
	if len(c.Locators) > 0 {
		meta["locators"] = c.Locators
	}
	doc := core.Document{
		Title:   c.Breadcrumb,
		Type:    "chunk",
		Content: c.Text,
		Source:  c.Source,
	}
	doc.Metadata = meta
	return elastic.DocumentWithMeta[core.Document]{
		ID:     c.DocID,
		Index:  "knowledge_chunk",
		Source: doc,
		Score:  score,
	}
}
