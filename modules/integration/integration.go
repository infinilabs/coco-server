/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"infini.sh/coco/core"
	"infini.sh/coco/plugins/security"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"sync"
	"time"
)

func (h *APIHandler) create(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	var obj = &core.Integration{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//if obj.Guest.Enabled && obj.Guest.RunAs != "" {
	//	//get permissions for this token
	//	ret, err := security.CreateAPIToken(obj.Guest.RunAs, "", "widget", security2.MustGetPermissionKeysByRole([]string{"widget"}))
	//	if err != nil {
	//		h.WriteError(w, err.Error(), http.StatusInternalServerError)
	//		return
	//	}
	//	obj.Token = ret["access_token"].(string)
	//}

	err = orm.Create(ctx, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if obj.Enabled && obj.Cors.Enabled && len(obj.Cors.AllowedOrigins) > 0 {
		integrationOrigins.Store(obj.ID, stringArrayToMap(obj.Cors.AllowedOrigins))
	}

	h.WriteCreatedOKJSON(w, obj.ID)

}

func (h *APIHandler) get(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := core.Integration{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "integration")
	ctx.DirectReadAccess()
	exists, err := orm.GetV2(ctx, &obj)
	if !exists || err != nil {
		h.WriteGetMissingJSON(w, id)
		return
	}

	h.WriteGetOKJSON(w, id, obj)
}

func (h *APIHandler) update(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	obj := core.Integration{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	delta := util.MapStr{}
	err := h.DecodeJSON(req, &delta)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "integration")
	ctx.Refresh = orm.WaitForRefresh
	err = orm.UpdatePartialFields(ctx, &obj, delta)
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

	h.WriteUpdatedOKJSON(w, obj.ID)
}

func (h *APIHandler) delete(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := core.Integration{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	ctx.Refresh = orm.WaitForRefresh
	err := orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// remove related origins check
	integrationOrigins.Delete(obj.ID)

	h.WriteDeletedOKJSON(w, id)
}

func (h *APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
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
	orm.WithModel(ctx, &core.Integration{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "integration")
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
					hit.Source["token_expire_in"] = tokenObj.ExpireIn
				}
			}
		}

	}

	h.WriteJSON(w, searchRes, http.StatusOK)
}

//func (h *APIHandler) renewAPIToken(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
//	ctx := orm.NewContextWithParent(req.Context())
//	ctx.Refresh = orm.WaitForRefresh
//
//	reqUser, err := security2.GetUserFromContext(req.Context())
//	if reqUser == nil || err != nil {
//		panic(err)
//	}
//	id := ps.MustGetParameter("id")
//
//	obj := core.Integration{}
//	obj.ID = id
//
//	ctx.Set(orm.SharingEnabled, true)
//	ctx.Set(orm.SharingResourceType, "integration")
//
//	exists, err := orm.GetV2(ctx, &obj)
//	if !exists || err != nil {
//		h.WriteJSON(w, util.MapStr{
//			"_id":   id,
//			"found": false,
//		}, http.StatusNotFound)
//		return
//	}
//	if obj.Token != "" {
//		// clear old token
//		kv.DeleteKey(core.KVAccessTokenBucket, []byte(obj.Token))
//	}
//	//create new token form this integration
//	if obj.Guest.Enabled && obj.Guest.RunAs != "" {
//		ret, err := security.CreateAPIToken(obj.Guest.RunAs, "", "widget", security2.MustGetPermissionKeysByRole([]string{"widget"}))
//		if err != nil {
//			h.WriteError(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		obj.Token = ret["access_token"].(string)
//		err = orm.Update(ctx, &obj)
//		if err != nil {
//			h.WriteError(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//	}
//	h.WriteAckOKJSON(w)
//}

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
	integrations := []core.Integration{}
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
