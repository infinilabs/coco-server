/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/common/fingerprint"
)

// KB-level files: the raw material behind the knowledge — documents from
// the KB's bound datasources plus files uploaded straight into the KB.
// Uploads flow through the same enrichment pipeline as connector content
// (chunking → summarization → entities → compile proposals), so a manual
// file updates the KB exactly like a synced one would.

func uploadDatasourceID(kbID string) string {
	return util.MD5digest("wiki-upload-" + kbID)
}

// ensureUploadDatasource returns the KB's manual-upload datasource,
// creating it lazily: a disabled connector datasource that exists so
// uploaded documents have a home and the search facets stay honest.
func ensureUploadDatasource(ctx *orm.Context, kb *core.WikiKnowledgeBase) (*core.DataSource, error) {
	id := uploadDatasourceID(kb.ID)
	ds := core.DataSource{}
	ds.SetID(id)
	exists, err := orm.GetV2(ctx, &ds)
	if err != nil && !strings.Contains(err.Error(), "record not found") {
		return nil, err
	}
	if exists {
		return &ds, nil
	}
	ds.Name = "知识库上传 · " + kb.Name
	ds.Type = "connector"
	ds.Connector = core.ConnectorConfig{ConnectorID: "manual_upload"}
	ds.Enabled = true
	// no sync config: the dispatcher never touches this source, files
	// arrive exclusively through the upload endpoint
	if err := orm.Create(ctx, &ds); err != nil {
		return nil, err
	}
	return &ds, nil
}

// GET /wiki/kb/:id/_files — every document behind the KB: bound-datasource
// content plus manual uploads, each with resolved attachments and the
// articles that cite it.
func (h *APIHandler) kbFiles(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)

	kb := core.WikiKnowledgeBase{}
	kb.SetID(kbID)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, kbID)
		return
	}

	upDS, err := ensureUploadDatasource(ctx, &kb)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dsIDs := append([]string{}, kb.DatasourceIDs...)
	dsIDs = append(dsIDs, upDS.ID)

	// documents of all sources feeding this KB
	dctx := orm.NewContextWithParent(ctx)
	dctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(dctx, &core.Document{})
	dres, err := orm.SearchV2(dctx, orm.NewQuery().Size(1000).
		Filter(orm.TermsQuery("source.id", dsIDs)).
		SortBy(orm.Sort{Field: "updated", SortType: "desc"}))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	docs, _, derr := elastic.DecodeHits[core.Document](dres)
	if derr != nil {
		h.WriteError(w, derr.Error(), http.StatusInternalServerError)
		return
	}

	// doc → citing articles: one pass over the KB's articles
	artRes, err := orm.SearchV2(dctx, orm.NewQuery().Size(1000).
		Filter(orm.TermQuery("kb_id", kbID)).
		Include("id", "title", "sources"))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type articleRef struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	articlesByDoc := map[string][]articleRef{}
	if artRes != nil {
		arts, _, aerr := elastic.DecodeHits[core.WikiArticle](artRes)
		if aerr == nil {
			for _, a := range arts {
				seen := map[string]bool{}
				for _, s := range a.Sources {
					if s.DocID == "" || seen[s.DocID] {
						continue
					}
					seen[s.DocID] = true
					articlesByDoc[s.DocID] = append(articlesByDoc[s.DocID], articleRef{ID: a.ID, Title: a.Title})
				}
			}
		}
	}

	// resolve attachment metadata for every doc in one query
	attIDs := []string{}
	for _, d := range docs {
		attIDs = append(attIDs, d.Attachments...)
	}
	attMeta := map[string]core.WikiArticleAttachment{}
	if len(attIDs) > 0 {
		orm.WithModel(ctx, &core.Attachment{})
		if ares, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
			Filter(orm.TermsQuery("id", attIDs))); err == nil && ares != nil {
			if atts, _, aerr := elastic.DecodeHits[core.Attachment](ares); aerr == nil {
				for _, a := range atts {
					attMeta[a.ID] = core.WikiArticleAttachment{ID: a.ID, Name: a.Name, Size: a.Size, MimeType: a.MimeType, Created: a.Created.Format("2006-01-02")}
				}
			}
		}
	}

	type kbFile struct {
		ID          string                       `json:"id"`
		Title       string                       `json:"title"`
		SourceID    string                       `json:"source_id,omitempty"`
		SourceName  string                       `json:"source_name,omitempty"`
		Uploaded    bool                         `json:"uploaded,omitempty"`
		Size        int                          `json:"size,omitempty"`
		Status      string                       `json:"status,omitempty"`
		Processed   bool                         `json:"processed"`
		Updated     string                       `json:"updated,omitempty"`
		URL         string                       `json:"url,omitempty"`
		Attachments []core.WikiArticleAttachment `json:"attachments,omitempty"`
		Articles    []articleRef                 `json:"articles,omitempty"`
	}
	files := make([]kbFile, 0, len(docs))
	for _, d := range docs {
		f := kbFile{
			ID:         d.ID,
			Title:      d.Title,
			SourceID:   d.Source.ID,
			SourceName: d.Source.Name,
			Uploaded:   d.Source.ID == upDS.ID,
			Size:       d.Size,
			Status:     d.Status,
			Processed:  d.Processed,
			URL:        d.URL,
		}
		if d.Updated != nil {
			f.Updated = d.Updated.Format("2006-01-02 15:04")
		}
		for _, aid := range d.Attachments {
			if m, ok := attMeta[aid]; ok {
				f.Attachments = append(f.Attachments, m)
			}
		}
		f.Articles = articlesByDoc[d.ID]
		files = append(files, f)
	}
	h.WriteOKJSON(w, util.MapStr{"files": files, "upload_source_id": upDS.ID})
}

// textExtensions are the uploads whose bytes go straight into the
// document content — the enrichment chain (entities, summary, compile)
// consumes them with zero pipeline changes. Binary uploads index through
// their attachment instead.
var textExtensions = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".text": true,
	".html": true, ".htm": true, ".xhtml": true,
	".json": true, ".csv": true, ".xml": true, ".yaml": true, ".yml": true,
	".log": true, ".svg": true, ".rst": true,
}

// POST /wiki/kb/:id/_files/upload — manual raw material: each file lands
// as a document in the KB's upload datasource and enters the same
// enrichment pipeline as synced content, so entities/articles update from
// it (auto-approve KBs without any further clicks).
func (h *APIHandler) uploadKBFiles(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	kbID := ps.ByName("id")
	req.Body = http.MaxBytesReader(w, req.Body, 20<<20)
	if err := req.ParseMultipartForm(20 << 20); err != nil {
		h.WriteError(w, "failed to parse multipart form (20MB cap)", http.StatusBadRequest)
		return
	}
	files := req.MultipartForm.File["files"]
	if len(files) == 0 {
		h.WriteError(w, "no files uploaded", http.StatusBadRequest)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)

	kb := core.WikiKnowledgeBase{}
	kb.SetID(kbID)
	orm.WithModel(ctx, &core.WikiKnowledgeBase{})
	exists, err := orm.GetV2(ctx, &kb)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, kbID)
		return
	}
	upDS, err := ensureUploadDatasource(ctx, &kb)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userID := ""
	if user, err := security.GetUserFromRequest(req); err == nil && user != nil {
		userID = user.UserID
	}
	now := time.Now()

	created := 0
	type uploadErr struct {
		File string `json:"file"`
		Err  string `json:"error"`
	}
	var failures []uploadErr
	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			failures = append(failures, uploadErr{File: fh.Filename, Err: err.Error()})
			continue
		}
		attID, err := attachment.UploadToBlobStore(ctx, "", file, fh, fh.Filename, userID, nil, "", false)
		file.Close()
		if err != nil {
			failures = append(failures, uploadErr{File: fh.Filename, Err: err.Error()})
			continue
		}

		ext := strings.ToLower(strings.TrimPrefix(fh.Filename[strings.LastIndex(fh.Filename, "."):], "."))
		var content string
		if textExtensions["."+ext] {
			// reopen: text files feed their bytes straight into the
			// document content so enrichment has material on the first pass
			file2, err := fh.Open()
			if err == nil {
				buf := make([]byte, 2<<20)
				if n, _ := file2.Read(buf); n > 0 && utf8.Valid(buf[:n]) {
					content = string(buf[:n])
				}
				file2.Close()
			}
		}

		doc := core.Document{
			Source:      core.DataSourceReference{ID: upDS.ID, Type: "connector", Name: upDS.Name},
			Type:        "file",
			Title:       fh.Filename,
			Content:     content,
			URL:         "/attachment/" + attID,
			Size:        int(fh.Size),
			Icon:        "file",
			Attachments: []string{attID},
		}
		doc.Created = &now
		doc.Updated = &now
		doc.SetID(util.MD5digest(upDS.ID + "-" + fh.Filename))
		doc.Status = core.DocumentStatusIndexing
		doc.System = util.MapStr{}
		if h, s, ok := fingerprint.Compute(content); ok {
			doc.ContentHash, doc.ContentSimhash = h, int64(s)
		}
		// content hash lets the stale-retry gate drop duplicate queues;
		// the source fingerprint keys future skip decisions on this upload
		doc.SourceFingerprint = fmt.Sprintf("upload:%d|%s", fh.Size, attID)

		// upsert semantics: re-uploading the same filename refreshes the
		// document in place instead of stacking twins
		existing := core.Document{}
		existing.SetID(doc.ID)
		if found, gerr := orm.GetV2(ctx, &existing); gerr == nil && found {
			doc.Created = existing.Created
		}
		wctx := orm.NewContextWithParent(ctx)
		wctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
		wctx.Refresh = orm.WaitForRefresh
		orm.WithModel(wctx, &core.Document{})
		if err := orm.Update(wctx, &doc); err != nil {
			failures = append(failures, uploadErr{File: fh.Filename, Err: err.Error()})
			continue
		}
		if err := common.EnqueueForIndexing(&doc); err != nil {
			failures = append(failures, uploadErr{File: fh.Filename, Err: err.Error()})
			continue
		}
		created++
		// the new raw material should reach the knowledge: mark the KB for
		// compile so entities/pages pick it up (auto-approve applies them)
		MarkKBForCompile(kbID)
	}
	h.WriteOKJSON(w, util.MapStr{"created": created, "failed": failures})
}
