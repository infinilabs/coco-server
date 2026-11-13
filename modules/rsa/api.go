/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package rsa

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"infini.sh/coco/core"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/elastic"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

type APIHandler struct {
	api.Handler
}

const (
	Category  = "coco"
	Resource  = "rsa"
	SearchHit = 3 // _test fetches a small window, not a full page
)

// createRemoteSource POST /rsa/
func (h *APIHandler) createRemoteSource(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	obj := &core.RemoteSource{}
	if err := h.DecodeJSON(req, obj); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	obj.ID = ""
	if err := h.validateSource(obj); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh
	if err := orm.Create(ctx, obj); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, util.MapStr{"_id": obj.ID, "result": "created"}, 200)
}

// updateRemoteSource PUT /rsa/:id — masked credentials round-trip back
// as "******" and are restored from the stored row before saving.
func (h *APIHandler) updateRemoteSource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	ctx := orm.NewContextWithParent(req.Context())
	ctx.DirectReadAccess()

	stored := core.RemoteSource{}
	stored.ID = id
	exists, err := orm.GetV2(ctx, &stored)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{"_id": id, "result": "not_found"}, http.StatusNotFound)
		return
	}

	obj := &core.RemoteSource{}
	if err := h.DecodeJSON(req, obj); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	CleanMaskedCredentials(obj, &stored)
	if err := h.validateSource(obj); err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	obj.ID = id
	obj.Created = stored.Created
	ctx2 := orm.NewContextWithParent(req.Context())
	ctx2.Refresh = orm.WaitForRefresh
	if err := orm.Save(ctx2, obj); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	InvalidateSource(id) // config changed — stale cached results must die
	h.WriteJSON(w, util.MapStr{"_id": id, "result": "updated"}, 200)
}

// getRemoteSource GET /rsa/:id — credentials masked.
func (h *APIHandler) getRemoteSource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	src, exists, err := GetSource(req.Context(), id)
	if err != nil || !exists {
		h.WriteJSON(w, util.MapStr{"_id": id, "found": false}, http.StatusNotFound)
		return
	}
	h.WriteJSON(w, util.MapStr{"_id": id, "found": true, "_source": MaskSource(&src)}, 200)
}

// deleteRemoteSource DELETE /rsa/:id
func (h *APIHandler) deleteRemoteSource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.RemoteSource{})

	obj := core.RemoteSource{}
	obj.ID = id
	if err := orm.Delete(ctx, &obj); err != nil {
		h.WriteJSON(w, util.MapStr{"_id": id, "result": "error", "error": err.Error()}, http.StatusInternalServerError)
		return
	}
	InvalidateSource(id)
	h.WriteJSON(w, util.MapStr{"_id": id, "result": "deleted"}, 200)
}

// searchRemoteSource GET/POST /rsa/_search
func (h *APIHandler) searchRemoteSource(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	builder, err := orm.NewQueryBuilderFromRequest(req, "name", "description", "adapter")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(builder.Sorts()) == 0 {
		builder.SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.RemoteSource{})
	ctx.PermissionScope(security.PermissionScopePlatform)

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sources, total, err := elastic.DecodeHits[core.RemoteSource](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	masked := make([]core.RemoteSource, 0, len(sources))
	for i := range sources {
		masked = append(masked, MaskSource(&sources[i]))
	}
	h.WriteJSON(w, util.MapStr{
		"total": total,
		"hits":  masked,
	}, 200)
}

// validateSource enforces adapter shape at every write.
func (h *APIHandler) validateSource(obj *core.RemoteSource) error {
	if strings.TrimSpace(obj.Name) == "" {
		return fmt.Errorf("name is required")
	}
	adapter, err := GetAdapter(obj.Adapter)
	if err != nil {
		return err
	}
	return adapter.Validate(obj)
}

// testRemoteSource POST /rsa/:id/_test — one tiny live query; proves
// connectivity, auth and mapping in a single round trip.
func (h *APIHandler) testRemoteSource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	src, exists, err := GetSource(req.Context(), id)
	if err != nil || !exists {
		h.WriteJSON(w, util.MapStr{"_id": id, "found": false}, http.StatusNotFound)
		return
	}
	src.Controls.CacheTTLSeconds = 0 // tests must hit the live source

	body := struct {
		Query string `json:"query"`
	}{}
	_ = h.DecodeJSON(req, &body)
	qtext := strings.TrimSpace(body.Query)
	if qtext == "" {
		qtext = "*" // match-anything probe for connectivity
	}

	cctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()

	res, diag := ExecuteQuery(cctx, &src, Query{Text: qtext, Size: SearchHit})
	out := util.MapStr{
		"_id":     id,
		"status":  diag.Status,
		"took_ms": diag.TookMS,
		"hits":    diag.Hits,
		"diag":    diag,
	}
	if diag.Err != "" {
		out["error"] = diag.Err
	}
	if res != nil && len(res.Hits) > 0 {
		sample := res.Hits[0].Document
		out["sample"] = util.MapStr{
			"title":   sample.Title,
			"url":     sample.URL,
			"snippet": truncateTail(sample.Content, 200),
		}
	}
	h.WriteJSON(w, out, 200)
}

// debugRemoteSource POST /rsa/_debug — the dry-run: render the exact
// native query, run it once, and show the raw→document mapping side by
// side (DESIGN-RSA.md §8).
func (h *APIHandler) debugRemoteSource(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	body := struct {
		SourceID string `json:"source_id"`
		Query    string `json:"query"`
		From     int    `json:"from"`
		Size     int    `json:"size"`
	}{}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(body.SourceID) == "" || strings.TrimSpace(body.Query) == "" {
		h.WriteError(w, "source_id and query are required", http.StatusBadRequest)
		return
	}
	src, exists, err := GetSource(req.Context(), body.SourceID)
	if err != nil || !exists {
		h.WriteJSON(w, util.MapStr{"_id": body.SourceID, "found": false}, http.StatusNotFound)
		return
	}
	src.Controls.CacheTTLSeconds = 0 // dry-runs bypass the cache by design

	adapter, err := GetAdapter(src.Adapter)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
		return
	}

	from, size := ClampWindow(&src, body.From, body.Size)
	if size == 0 {
		size = SearchHit
	}
	q := Query{Text: strings.TrimSpace(body.Query), From: from, Size: size}

	rendered := util.MapStr{"adapter": src.Adapter}
	if es, ok := adapter.(*AdapterEsCompatible); ok {
		if b, berr := es.buildBody(&src, q); berr == nil {
			rendered["es_search_body"] = b
			rendered["es_search_url"] = strings.Join(src.Connection.Endpoints, ",") + "/" + src.Query.Indices + "/_search"
		} else {
			rendered["render_error"] = berr.Error()
		}
	}

	res, diag := ExecuteQuery(req.Context(), &src, q)
	out := util.MapStr{
		"source_id": src.ID,
		"status":    diag.Status,
		"took_ms":   diag.TookMS,
		"cache_hit": diag.CacheHit,
		"native":    rendered,
		"diag":      diag,
	}
	if diag.Err != "" {
		out["error"] = diag.Err
	}
	if res != nil {
		rows := make([]util.MapStr, 0, len(res.Hits))
		for i := range res.Hits {
			d := res.Hits[i].Document
			md, _ := d.Metadata["rsa"].(util.MapStr)
			rows = append(rows, util.MapStr{
				"rank":    res.Hits[i].Rank,
				"doc_id":  d.ID,
				"title":   d.Title,
				"url":     d.URL,
				"content": truncateTail(d.Content, 300),
				"raw_id":  md["raw_id"],
				"index":   md["index"],
			})
		}
		out["total"] = res.Total
		out["mapped_hits"] = rows
	}
	h.WriteJSON(w, out, 200)
}

// statsHandler GET /rsa/_stats — live per-source counters.
func (h *APIHandler) statsHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	h.WriteJSON(w, util.MapStr{"stats": SnapshotStats()}, 200)
}

func truncateTail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

