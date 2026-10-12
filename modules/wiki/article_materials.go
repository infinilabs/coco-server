/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"io"
	"net/http"
	"sort"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// SourceMaterial is one cited source document with its enrichment
// metadata and collected attachments.
type SourceMaterial struct {
	DocID       string               `json:"doc_id"`
	Title       string               `json:"title"`
	SourceName  string               `json:"source_name,omitempty"`
	SourceType  string               `json:"source_type,omitempty"`
	Excerpt     string               `json:"excerpt,omitempty"`
	Locator     string               `json:"locator,omitempty"`
	Updated     string               `json:"updated,omitempty"`
	Created     string               `json:"created,omitempty"`
	Missing     bool                 `json:"missing,omitempty"` // cited doc no longer exists
	Attachments []AttachmentMaterial `json:"attachments,omitempty"`
}

// AttachmentMaterial is one attachment of a cited source document.
type AttachmentMaterial struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Size     int64  `json:"size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Created  string `json:"created,omitempty"`
}

// GET /wiki/article/:id/_materials — everything behind the page: for each
// cited source document its title, datasource and update time, plus the
// attachments collected during its enrichment (name/size/time, download
// ready). Feeds the article page's 来源与附件 panel.
func (h *APIHandler) articleMaterials(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	article := core.WikiArticle{}
	article.ID = articleID
	exists, err := orm.GetV2(ctx, &article)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, articleID)
		return
	}

	type SourceMaterial struct {
		DocID       string               `json:"doc_id"`
		Title       string               `json:"title"`
		SourceName  string               `json:"source_name,omitempty"`
		SourceType  string               `json:"source_type,omitempty"`
		Excerpt     string               `json:"excerpt,omitempty"`
		Locator     string               `json:"locator,omitempty"`
		Updated     string               `json:"updated,omitempty"`
		Created     string               `json:"created,omitempty"`
		Missing     bool                 `json:"missing,omitempty"` // cited doc no longer exists
		Attachments []AttachmentMaterial `json:"attachments,omitempty"`
	}

	materials := make([]SourceMaterial, 0, len(article.Sources))
	attachmentIDs := map[string]string{} // attachment id -> source doc id
	for _, src := range article.Sources {
		m := SourceMaterial{
			DocID:      src.DocID,
			Title:      src.Title,
			SourceName: src.SourceName,
			SourceType: src.SourceType,
			Excerpt:    src.Excerpt,
			Locator:    src.Locator,
		}
		if src.DocID == "" {
			m.Missing = true
			materials = append(materials, m)
			continue
		}
		doc := core.Document{}
		doc.ID = src.DocID
		found, err := orm.GetV2(ctx, &doc)
		if err != nil {
			return
		}
		if !found {
			m.Missing = true
			materials = append(materials, m)
			continue
		}
		m.Title = doc.Title
		if doc.Updated != nil {
			m.Updated = doc.Updated.Format("2006-01-02 15:04")
		}
		if doc.Created != nil {
			m.Created = doc.Created.Format("2006-01-02")
		}
		for _, aid := range doc.Attachments {
			attachmentIDs[aid] = src.DocID
		}
		materials = append(materials, m)
	}

	// attachment metadata grouped under their source document
	if len(attachmentIDs) > 0 {
		ids := make([]string, 0, len(attachmentIDs))
		for id := range attachmentIDs {
			ids = append(ids, id)
		}
		orm.WithModel(ctx, &core.Attachment{})
		if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
			Filter(orm.TermsQuery("id", ids))); err == nil && res != nil {
			if atts, _, derr := elastic.DecodeHits[core.Attachment](res); derr == nil {
				attByID := map[string]core.Attachment{}
				for _, a := range atts {
					attByID[a.ID] = a
				}
				for i := range materials {
					for _, aid := range docAttachmentIDs(materials[i].DocID, attachmentIDs) {
						if a, ok := attByID[aid]; ok {
							materials[i].Attachments = append(materials[i].Attachments, AttachmentMaterial{
								ID:       a.ID,
								Name:     a.Name,
								Size:     a.Size,
								MimeType: a.MimeType,
								Created:  a.Created.Format("2006-01-02"),
							})
						}
					}
					sort.Slice(materials[i].Attachments, func(x, y int) bool {
						return materials[i].Attachments[x].Name < materials[i].Attachments[y].Name
					})
				}
			}
		}
	}

	h.WriteOKJSON(w, util.MapStr{"sources": materials})
}

func docAttachmentIDs(docID string, attachmentToDoc map[string]string) []string {
	out := make([]string, 0, 4)
	for aid, owner := range attachmentToDoc {
		if owner == docID {
			out = append(out, aid)
		}
	}
	sort.Strings(out)
	return out
}

/* ---------------- page-level attachments ---------------- */

// resolveArticleAttachments loads metadata for the page's own attachment ids;
// ids whose blob row vanished are skipped (a pruned attachment must not
// render as a dead row).
func resolveArticleAttachments(ctx *orm.Context, ids []string) []core.WikiArticleAttachment {
	out := make([]core.WikiArticleAttachment, 0, len(ids))
	if len(ids) == 0 {
		return out
	}
	orm.WithModel(ctx, &core.Attachment{})
	if res, err := orm.SearchV2(ctx, orm.NewQuery().Size(1000).
		Filter(orm.TermsQuery("id", ids))); err == nil && res != nil {
		if atts, _, derr := elastic.DecodeHits[core.Attachment](res); derr == nil {
			for _, a := range atts {
				out = append(out, core.WikiArticleAttachment{
					ID:       a.ID,
					Name:     a.Name,
					Size:     a.Size,
					MimeType: a.MimeType,
					Created:  a.Created.Format("2006-01-02"),
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// GET /wiki/article/:id/_attachments — the page's own files with resolved
// metadata (name/size/mime/date). Rendered even when empty: the section is
// the page's attach point, not a conditional extra.
func (h *APIHandler) articleAttachments(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	article := core.WikiArticle{}
	article.ID = articleID
	exists, err := orm.GetV2(ctx, &article)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, articleID)
		return
	}
	h.WriteOKJSON(w, util.MapStr{"attachments": resolveArticleAttachments(ctx, article.Attachments)})
}

// PUT /wiki/article/:id/_attachments — replace the page's attachment id
// list wholesale (the client renders upload-append and row-remove as new
// arrays). Unknown ids are dropped and reported, never stored.
func (h *APIHandler) updateArticleAttachments(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	articleID := ps.ByName("id")
	var body struct {
		AttachmentIDs []string `json:"attachment_ids"`
	}
	raw, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil || util.FromJSONBytes(raw, &body) != nil {
		h.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(ctx, &core.WikiArticle{})

	article := core.WikiArticle{}
	article.ID = articleID
	exists, err := orm.GetV2(ctx, &article)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		h.WriteOpRecordNotFoundJSON(w, articleID)
		return
	}

	// keep only ids that actually resolve — a stale or forged id would
	// otherwise render as a dead row forever
	resolved := resolveArticleAttachments(ctx, body.AttachmentIDs)
	kept := make([]string, 0, len(resolved))
	for _, a := range resolved {
		kept = append(kept, a.ID)
	}

	writeCtx := orm.NewContextWithParent(req.Context())
	writeCtx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	orm.WithModel(writeCtx, &core.WikiArticle{})
	article.Attachments = kept
	if err := orm.Update(writeCtx, &article); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteOKJSON(w, util.MapStr{
		"attachments": resolved,
		"dropped":     len(body.AttachmentIDs) - len(kept),
	})
}
