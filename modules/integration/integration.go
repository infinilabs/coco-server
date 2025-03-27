/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/security"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"sync"
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
		integrationOrigins.Store(obj.ID, stringArrayToMap(obj.Cors.AllowedOrigins))
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
	// first related origins check
	integrationOrigins.Delete(obj.ID)
	// then register the new check
	if obj.Enabled && obj.Cors.Enabled && len(obj.Cors.AllowedOrigins) > 0 {
		integrationOrigins.Store(obj.ID, stringArrayToMap(obj.Cors.AllowedOrigins))
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
	// remove related origins check
	integrationOrigins.Delete(obj.ID)

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

func IntegrationAllowOrigin(origin string, req *http.Request) bool {
	appIntegrationID := req.Header.Get("APP-INTEGRATION-ID")
	if v, ok := integrationOrigins.Load(appIntegrationID); ok {
		if allowedOrigins, ok := v.(map[string]struct{}); ok {
			if _, allowed := allowedOrigins[origin]; allowed {
				return true
			}
			if _, allowedAll := allowedOrigins["*"]; allowedAll {
				return true
			}
		}
	}
	return false
}

var (
	integrationOrigins sync.Map
)

func InitIntegrationOrigins() {
	integrations := []common.Integration{}
	err, _ := orm.SearchWithJSONMapper(&integrations, &orm.Query{
		Size:  100,
		Conds: orm.And(orm.Eq("enabled", true), orm.Eq("cors.enabled", true)),
	})
	if err != nil {
		panic(err)
	}
	for _, integration := range integrations {
		integrationOrigins.Store(integration.ID, stringArrayToMap(integration.Cors.AllowedOrigins))
	}
}

func stringArrayToMap(arr []string) map[string]struct{} {
	if len(arr) == 0 {
		return nil
	}
	ret := make(map[string]struct{}, len(arr))
	for _, v := range arr {
		ret[v] = struct{}{}
	}
	return ret
}
