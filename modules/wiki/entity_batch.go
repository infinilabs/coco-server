/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"fmt"
	"net/http"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
)

// entityBatchStatuses are the target states 批量设置状态 may assign — the
// full lifecycle, since the operator picks the destination explicitly
// (unlike the single-row one-step advance).
var entityBatchStatuses = map[string]bool{
	core.WikiEntityProposed:  true,
	core.WikiEntityReviewed:  true,
	core.WikiEntityPublished: true,
	"retired":                true,
}

// PUT /wiki/entity/_batch_status {ids, status} — review N entities in one
// action: proposed→reviewed→published gates are per-item human decisions,
// and a bulk import cleanup wants the whole set moved at once. Entities are
// retired via status (design §4.1), so retired here is the batch "delete".
func (h *APIHandler) batchUpdateEntityStatus(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var body struct {
		IDs    []string `json:"ids"`
		Status string   `json:"status"`
	}
	if err := h.DecodeJSON(req, &body); err != nil {
		h.Error400(w, err.Error())
		return
	}
	if !entityBatchStatuses[body.Status] {
		h.Error400(w, fmt.Sprintf("invalid entity status: %s", body.Status))
		return
	}
	if len(body.IDs) == 0 {
		h.Error400(w, "ids is empty")
		return
	}
	if len(body.IDs) > 2000 {
		h.Error400(w, "too many ids (cap 2000 per batch)")
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.DirectReadWithoutPermissionCheck, true)
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiEntity{})

	updated, skipped := 0, 0
	for _, id := range body.IDs {
		if id == "" {
			skipped++
			continue
		}
		var entity core.WikiEntity
		entity.SetID(id)
		exists, err := orm.GetV2(ctx, &entity)
		if err != nil || !exists {
			skipped++
			continue
		}
		if entity.Status == body.Status {
			continue // idempotent: already there, not a skip
		}
		entity.Status = body.Status
		if err := orm.Update(ctx, &entity); err != nil {
			skipped++
			continue
		}
		updated++
	}
	h.WriteOKJSON(w, util.MapStr{"updated": updated, "skipped": skipped})
}
