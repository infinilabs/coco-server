/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package assistant

import (
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"net/http"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func (h *APIHandler) createAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var obj = &core.Assistant{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	obj.Builtin = false
	err = orm.Create(ctx, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	common.ClearAssistantsCache()

	h.WriteCreatedOKJSON(w, obj.ID)

}

func (h *APIHandler) getAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj, exists, err := common.GetAssistant(req, id)
	if !exists || err != nil {
		_ = log.Error(err)
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	h.WriteGetOKJSON(w, id, obj)
}

func (h *APIHandler) updateAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	//clear cache
	common.GeneralObjectCache.Delete(common.AssistantCachePrimary, id)

	obj := core.Assistant{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		_ = log.Error(err)
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	newObj := core.Assistant{}
	err = h.DecodeJSON(req, &newObj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	newObj.ID = id
	if obj.Builtin {
		newObj.Name = obj.Name
	}
	newObj.Builtin = obj.Builtin
	newObj.Created = obj.Created

	ctx.Refresh = orm.WaitForRefresh
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "assistant")

	err = orm.Update(ctx, &newObj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteUpdatedOKJSON(w, obj.ID)
}

func (h *APIHandler) deleteAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	//clear cache
	common.GeneralObjectCache.Delete(common.AssistantCachePrimary, id)
	common.ClearAssistantsCache()

	obj := core.Assistant{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteOpRecordNotFoundJSON(w, id)

		return
	}
	if obj.Builtin {
		h.WriteError(w, "Built-in model providers cannot be deleted", http.StatusForbidden)

		return
	}

	ctx.Refresh = orm.WaitForRefresh

	err = orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)

		return
	}

	h.WriteDeletedOKJSON(w, obj.ID)
}

func (h *APIHandler) searchAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var err error
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	builder.EnableBodyBytes()
	if len(builder.Sorts()) == 0 {
		builder.SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.Assistant{})
	docs := []core.Assistant{}
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "assistant")
	err, res := elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

func (h *APIHandler) cloneAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := core.Assistant{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "assistant")

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		_ = log.Error(err)
		h.WriteOpRecordNotFoundJSON(w, id)

		return
	}

	obj.ID = util.GetUUID()
	obj.Name = obj.Name + "_copy"
	now := time.Now()
	obj.Created = &now
	obj.Updated = &now
	obj.Builtin = false

	ctx.Refresh = orm.WaitForRefresh

	err = orm.Create(ctx, &obj)
	if err != nil {
		h.Error(w, err)
		return
	}
	h.WriteCreatedOKJSON(w, obj.ID)
}
