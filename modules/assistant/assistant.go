// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package assistant

import (
	"infini.sh/coco/core"
	"net/http"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func (h *APIHandler) createAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var obj = &common.Assistant{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
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

	obj, exists, err := common.GetAssistant(id)
	if !exists || err != nil {
		log.Error(err)
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	h.WriteGetOKJSON(w, id, obj)
}

func (h *APIHandler) updateAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	//clear cache
	common.GeneralObjectCache.Delete(common.AssistantCachePrimary, id)

	obj := common.Assistant{}
	obj.ID = id
	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		log.Error(err)
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	newObj := common.Assistant{}
	err = h.DecodeJSON(req, &obj)
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
	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
	err = orm.Save(ctx, &obj)
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

	obj := common.Assistant{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}
	if obj.Builtin {
		h.WriteError(w, "Built-in model providers cannot be deleted", http.StatusForbidden)
		return
	}

	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
	err = orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteDeletedOKJSON(w, obj.ID)
}

func (h *APIHandler) searchAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	body, err := h.GetRawBody(req)

	//for backward compatibility
	if err == nil && body != nil { //TODO remove legacy code
		var err error
		q := orm.Query{}
		q.RawQuery = body

		err, res := orm.Search(&common.Assistant{}, &q)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = h.Write(w, res.Raw)
		if err != nil {
			h.Error(w, err)
		}
	} else {
		var err error
		//handle url query args, convert to query builder
		builder, err := orm.NewQueryBuilderFromRequest(req, "name", "combined_fulltext")
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx := orm.NewModelContext(&common.Assistant{})
		docs := []common.Assistant{}

		err, res := core.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = h.Write(w, res.Raw)
		if err != nil {
			h.Error(w, err)
		}
	}
}

func (h *APIHandler) cloneAssistant(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := common.Assistant{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		log.Error(err)
		h.WriteOpRecordNotFoundJSON(w, id)
		return
	}

	obj.ID = util.GetUUID()
	obj.Name = obj.Name + "_copy"
	now := time.Now()
	obj.Created = &now
	obj.Updated = &now
	obj.Builtin = false
	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
	orm.Create(ctx, &obj)
	h.WriteCreatedOKJSON(w, obj.ID)
}
