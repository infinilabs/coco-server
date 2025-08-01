/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
)

func (h *APIHandler) createDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &common.Document{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh
	err = orm.Create(ctx, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "created",
	}, 200)

}

func (h *APIHandler) getDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")

	obj := common.Document{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
}

func (h *APIHandler) updateDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")
	obj := common.Document{}

	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	id = obj.ID
	create := obj.Created

	obj = common.Document{}
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	obj.ID = id
	obj.Created = create
	ctx.Refresh = orm.WaitForRefresh
	err = orm.Update(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "updated",
	}, 200)
}

func (h *APIHandler) deleteDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")

	obj := common.Document{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	ctx.Refresh = orm.WaitForRefresh
	err = orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "deleted",
	}, 200)
}

func (h *APIHandler) searchDocs(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "title", "summary", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &common.Document{})

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Payload.([]byte))
	if err != nil {
		h.Error(w, err)
	}
}

func (h *APIHandler) batchDeleteDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var ids []string
	err := h.DecodeJSON(req, &ids)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(ids) == 0 {
		h.WriteError(w, "document ids can not be empty", http.StatusBadRequest)
		return
	}
	query := util.MapStr{
		"query": util.MapStr{
			"terms": util.MapStr{
				"id": ids,
			},
		},
	}
	err = orm.DeleteBy(common.Document{}, util.MustToJSONBytes(query))
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteAckOKJSON(w)
}
