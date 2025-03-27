/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/security"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
)

func (h *APIHandler) create(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//user already login
	reqUser, err := core.UserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}

	var obj = &common.Integration{}
	err = h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret, err := security.CreateAPIToken(reqUser, "")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	obj.Token = ret["access_token"].(string)
	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
	err = orm.Create(ctx, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if obj.Enabled && obj.Cors.Enabled && len(obj.Cors.AllowedOrigins) > 0 {
		allowOriginFn := getAllowOriginFunc(obj.Cors.AllowedOrigins, obj.ID)
		api.RegisterAllowOriginFunc(obj.ID, allowOriginFn)
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "created",
	}, 200)

}

func (h *APIHandler) get(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := common.Integration{}
	obj.ID = id

	exists, err := orm.Get(&obj)
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

func (h *APIHandler) update(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	obj := common.Integration{}
	obj.ID = id
	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	newObj := common.Integration{}
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	newObj.ID = id
	newObj.Created = obj.Created
	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
	err = orm.Update(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// first remove the existing allow origin function
	api.RemoveAllowOriginFunc(obj.ID)
	// then register the new allow origin function
	if obj.Enabled && obj.Cors.Enabled && len(obj.Cors.AllowedOrigins) > 0 {
		allowOriginFn := getAllowOriginFunc(obj.Cors.AllowedOrigins, obj.ID)
		api.RegisterAllowOriginFunc(obj.ID, allowOriginFn)
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "updated",
	}, 200)
}

func (h *APIHandler) delete(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := common.Integration{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
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
	// remove the allow origin function
	api.RemoveAllowOriginFunc(obj.ID)

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "deleted",
	}, 200)
}

func (h *APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var err error
	q := orm.Query{}
	q.RawQuery, err = h.GetRawBody(req)

	err, res := orm.Search(&common.Integration{}, &q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

func getAllowOriginFunc(allowedOrigins []string, integrationID string) func(origin string, req *http.Request) bool {
	return func(origin string, req *http.Request) bool {
		appIntegrationID := req.Header.Get("APP-INTEGRATION-ID")
		for _, allowedOrigin := range allowedOrigins {
			if (allowedOrigin == "*" || origin == allowedOrigin) && (appIntegrationID == integrationID || req.Method == http.MethodOptions) {
				return true
			}
		}
		return false
	}
}

func RegisterAllowOriginFuncs() {
	integrations := []common.Integration{}
	err, _ := orm.SearchWithJSONMapper(&integrations, &orm.Query{
		Size:  100,
		Conds: orm.And(orm.Eq("enabled", true), orm.Eq("cors.enabled", true)),
	})
	if err != nil {
		log.Error(err)
		return
	}
	for _, integration := range integrations {
		if integration.Enabled && integration.Cors.Enabled && len(integration.Cors.AllowedOrigins) > 0 {
			allowOriginFn := getAllowOriginFunc(integration.Cors.AllowedOrigins, integration.ID)
			api.RegisterAllowOriginFunc(integration.ID, allowOriginFn)
		}
	}
}
