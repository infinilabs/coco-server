/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package document

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// Virtual folders (W4): documents carry a normalized folder_path — the
// tree is an aggregation, moves are field updates, nothing re-parses.
// Paths look like "/部门/研发/规范", depth-capped, segment-capped; the
// root documents live with an empty path.

const (
	folderMaxDepth      = 16
	folderMaxSegment    = 128
	foldersAggSize      = 200
	folderMoveBatchSize = 100
)

// NormalizeFolderPath validates and canonicalizes a folder path:
// slash-separated, no "."/".." segments, depth and segment capped,
// leading slash added, trailing slash dropped. The empty string is the
// root and stays empty.
func NormalizeFolderPath(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" || p == "/" {
		return "", nil
	}
	p = strings.Trim(p, "/")
	if p == "" {
		return "", nil
	}
	segments := strings.Split(p, "/")
	if len(segments) > folderMaxDepth {
		return "", fmt.Errorf("folder path deeper than %d levels", folderMaxDepth)
	}
	for _, seg := range segments {
		if seg == "" {
			return "", fmt.Errorf("empty segment in folder path %q", path)
		}
		if seg == "." || seg == ".." {
			return "", fmt.Errorf("relative segment in folder path %q", path)
		}
		if len([]rune(seg)) > folderMaxSegment {
			return "", fmt.Errorf("folder segment longer than %d runes in %q", folderMaxSegment, path)
		}
	}
	return "/" + p, nil
}

// foldersHandler aggregates the folder tree of a datasource: distinct
// folder_path values with document counts — the tree IS the aggregation.
func (h *APIHandler) foldersHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	datasourceID := ps.ByName("id")

	octx := orm.NewContextWithParent(req.Context())
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})

	builder := orm.NewQuery().Size(0).
		AddAgg("folders", &orm.TermsAggregation{Field: "folder_path", Size: foldersAggSize}).
		Filter(orm.TermQuery("source.id", datasourceID))
	res, err := orm.Aggregate(octx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	folders := []util.MapStr{{"path": "", "name": "/", "count": 0}} // root always present
	if node := res.Aggs["folders"]; node != nil {
		for _, b := range node.Buckets {
			folders = append(folders, util.MapStr{"path": b.Key, "count": b.DocCount})
		}
	}
	h.WriteOKJSON(w, util.MapStr{"folders": folders, "total": len(folders)})
}

// moveFolderHandler moves documents to a folder: POST /document/_move_folder
// {ids: [...], folder: "/部门/研发"} — a field update, never a reparse.
func (h *APIHandler) moveFolderHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := util.MapStr{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	folder, err := NormalizeFolderPath(fmt.Sprintf("%v", body["folder"]))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	ids := stringSliceOf(body["ids"])
	if len(ids) == 0 {
		h.WriteError(w, "ids required", http.StatusBadRequest)
		return
	}

	moved := 0
	for from := 0; from < len(ids); from += folderMoveBatchSize {
		end := from + folderMoveBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[from:end]
		ctx := orm.NewContextWithParent(req.Context())
		ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
		ctx.Refresh = orm.WaitForRefresh
		orm.WithModel(ctx, &core.Document{})
		for _, id := range batch {
			doc := core.Document{}
			doc.ID = id
			if exists, err := orm.GetV2(ctx, &doc); err != nil || !exists {
				continue
			}
			doc.FolderPath = folder
			if err := orm.Save(ctx, &doc); err == nil {
				moved++
			}
		}
	}
	h.WriteOKJSON(w, util.MapStr{"moved": moved, "folder": folder})
}

// applyTagsFilter adds the ?tags= filter to a query builder — the
// retrieval-side half of the controlled vocabulary (W4): tags only ever
// come from the vocabulary, so the filter matches real facets.
func applyTagsFilter(builder *orm.QueryBuilder, tagsParam string) {
	if strings.TrimSpace(tagsParam) == "" {
		return
	}
	tags := strings.Split(tagsParam, ",")
	cleaned := make([]string, 0, len(tags))
	for _, t := range tags {
		if t = strings.TrimSpace(t); t != "" {
			cleaned = append(cleaned, t)
		}
	}
	if len(cleaned) > 0 {
		builder.Filter(orm.TermsQuery("tags", cleaned))
	}
}

// capabilitiesHandler reports what a datasource's corpus can do (W4):
// cheap index-state signals — document count, vector coverage, FAQ
// presence, and whether a wiki KB binds it. Assistant editors and the
// MCP search schema consume this so agents filter sources by ability.
func (h *APIHandler) capabilitiesHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	datasourceID := ps.ByName("id")
	ctx := req.Context()

	caps := util.MapStr{
		"datasource_id": datasourceID,
		"documents":     countDocs(ctx, datasourceID, nil),
		"vector":        countDocs(ctx, datasourceID, orm.ExistsQuery(documentEmbeddingField())) > 0,
		"faq":           countDocs(ctx, datasourceID, orm.TermQuery("type", faqDocType)) > 0,
		"wiki":          wikiKBBindsDatasource(ctx, datasourceID),
	}
	h.WriteOKJSON(w, caps)
}

func countDocs(ctx context.Context, datasourceID string, extra *orm.Clause) int64 {
	octx := orm.NewContextWithParent(ctx)
	octx.DirectReadAccess()
	orm.WithModel(octx, &core.Document{})
	builder := orm.NewQuery().Size(0)
	if datasourceID != "" {
		builder.Filter(orm.TermQuery("source.id", datasourceID))
	}
	if extra != nil {
		builder.Filter(extra)
	}
	res, err := orm.SearchV2(octx, builder)
	if err != nil || res == nil {
		return 0
	}
	out := &elastic.SearchResponseWithMeta[core.Document]{}
	if raw, ok := res.Payload.([]byte); ok && len(raw) > 0 {
		util.MustFromJSONBytes(raw, out)
	}
	return out.GetTotal()
}

// wikiKBBindsDatasource reports whether any wiki KB lists this datasource
// in its bound set (WikiKnowledgeBase.DatasourceIDs).
func wikiKBBindsDatasource(ctx context.Context, datasourceID string) bool {
	octx := orm.NewContextWithParent(ctx)
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiKnowledgeBase{})
	res, err := orm.SearchV2(octx, orm.NewQuery().Size(1).
		Filter(orm.TermQuery("datasource_ids", datasourceID)))
	if err != nil {
		return false
	}
	hits, _, _ := elastic.DecodeHits[core.WikiKnowledgeBase](res)
	return len(hits) > 0
}

// documentChunksHandler lists a document's chunks for the artifact
// visualization (W13b read side): breadcrumb, page range and a display
// excerpt per chunk — the parsing result becomes inspectable per block.
// GET /document/:doc_id/_chunks?from=0&size=50
func (h *APIHandler) documentChunksHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.ByName("doc_id")

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")

	doc := core.Document{}
	doc.ID = id
	exists, err := orm.GetV2(ctx, &doc)
	if err != nil || !exists {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	from := h.GetIntOrDefault(req, "from", 0)
	if from < 0 {
		from = 0
	}
	size := h.GetIntOrDefault(req, "size", 50)
	if size < 1 {
		size = 50
	}
	if size > maxSearchPageSize {
		size = maxSearchPageSize
	}

	total := len(doc.Chunks)
	end := from + size
	if end > total {
		end = total
	}
	items := make([]util.MapStr, 0, end-from)
	for i := from; i < end; i++ {
		c := doc.Chunks[i]
		items = append(items, util.MapStr{
			"index":      i,
			"breadcrumb": c.Breadcrumb,
			"pages":      util.MapStr{"start": c.Range.Start, "end": c.Range.End},
			"excerpt":    truncateRunes(c.Text, 500),
			"vectorized": len(c.Embedding.Embedding1024) > 0,
		})
	}
	h.WriteOKJSON(w, util.MapStr{
		"id":     id,
		"total":  total,
		"from":   from,
		"size":   size,
		"chunks": items,
	})
}
