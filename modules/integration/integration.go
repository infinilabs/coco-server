/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/plugins/security"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	security2 "infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"net/http"
	"sync"
	"time"
)

func (h *APIHandler) create(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//user already login
	reqUser, err := security2.GetUserFromRequest(req)
	if reqUser == nil || err != nil {
		panic(err)
	}

	var obj = &common.Integration{}
	err = h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//get permissions for this token
	ret, err := security.CreateAPIToken(reqUser, "", "widget", []string{"widget"})
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	obj.Token = ret["access_token"].(string)

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

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

func (h *APIHandler) update(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	obj := common.Integration{}
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

	newObj := common.Integration{}
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	newObj.ID = id
	newObj.Created = obj.Created

	ctx.Refresh = orm.WaitForRefresh

	err = orm.Save(ctx, &obj)
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
	// remove related origins check
	integrationOrigins.Delete(obj.ID)

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "deleted",
	}, 200)
}

func (h *APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &common.Integration{})

	res, err := orm.SearchV2(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes := res.Payload.([]byte)
	searchRes := elastic.SearchResponse{}
	if bytes != nil {
		err := util.FromJSONBytes(bytes, &searchRes)
		if err != nil {
			panic(err)
		}

		for _, hit := range searchRes.Hits.Hits {
			if token, ok := hit.Source["token"].(string); ok && token != "" {
				tokenObj, err := security.GetToken(token)
				if tokenObj == nil && err == nil {
					// token is not found in the kv, here we set it as expired
					hit.Source["token_expire_in"] = time.Time{}.Unix()
				}
				if tokenObj != nil {
					hit.Source["token_expire_in"] = tokenObj["expire_in"]
				}
			}
		}

	}

	h.WriteJSON(w, searchRes, http.StatusOK)
}

func (h *APIHandler) renewAPIToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	reqUser, err := security2.GetUserFromContext(req.Context())
	if reqUser == nil || err != nil {
		panic(err)
	}
	id := ps.MustGetParameter("id")

	obj := common.Integration{}
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
	if obj.Token != "" {
		// clear old token
		_ = security.DeleteAccessToken(reqUser.UserID, obj.Token)
	}
	//create new token form this integration
	ret, err := security.CreateAPIToken(reqUser, "", "widget", []string{"widget"})
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	obj.Token = ret["access_token"].(string)

	ctx.Refresh = orm.WaitForRefresh

	err = orm.Update(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteAckOKJSON(w)
}

func IntegrationAllowOrigin(origin string, req *http.Request) bool {
	appIntegrationID := req.Header.Get(core.HeaderIntegrationID)
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
