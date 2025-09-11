/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/security"
	"net/http"
	"time"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func (h *APIHandler) create(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &common.Connector{}
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

func (h *APIHandler) get(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := common.Connector{}
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
	obj := common.Connector{}

	replace := h.GetBoolOrDefault(req, "replace", false)
	ctx := orm.NewContextWithParent(req.Context())

	var err error
	var create *time.Time
	var builtin bool
	if !replace {
		obj.ID = id

		//can't remove, since we need it for update
		exists, err := orm.GetWithSystemFields(ctx, &obj)
		if !exists || err != nil {
			h.WriteJSON(w, util.MapStr{
				"_id":    id,
				"result": "not_found",
			}, http.StatusNotFound)
			return
		}
		id = obj.ID
		create = obj.Created
		builtin = obj.Builtin
	} else {
		t := time.Now()
		create = &t
	}

	obj = common.Connector{}
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	obj.ID = id
	obj.Created = create
	obj.Builtin = builtin

	ctx.Refresh = orm.WaitForRefresh
	ctx.DirectReadAccess() //TODO platform permission, rather user level permission
	err = orm.Save(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "updated",
	}, 200)
}

func (h *APIHandler) delete(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := common.Connector{}
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
	if obj.Builtin {
		h.WriteError(w, "builtin connector cannot be deleted", http.StatusForbidden)
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

// ?query=keyword&filter=fieldA:efg&filter=fieldB=abc&filter=url_escape( a:B AND c:A OR(abc AND efg) )&sort=a:desc,b:asc&
func (h *APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var err error
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	searchRequest := elastic.SearchRequest{}
	bodyBytes, err := h.GetRawBody(req)
	if err == nil && len(bodyBytes) > 0 {
		err = util.FromJSONBytes(bodyBytes, &searchRequest)
		if err != nil {
			h.Error(w, err)
			return
		}
		builder.SetRequestBodyBytes(bodyBytes)
	}

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &common.Connector{})
	ctx.Set(orm.ReadPermissionCheckingScope, []int{security.PermissionScopePlatform})

	appConfig := common.AppConfig()
	var connectors []common.Connector

	itemMapFunc := func(source map[string]interface{}, targetRef interface{}) error {
		if !appConfig.ServerInfo.EncodeIconToBase64 {
			return nil
		}

		// Modify icons in-place
		if assets, ok := source["assets"].(map[string]interface{}); ok {
			if icons, ok := assets["icons"].(map[string]interface{}); ok {
				for k, v := range icons {
					if iconStr, ok := v.(string); ok {
						link := common.AutoGetFullIconURL(&appConfig, iconStr)
						icons[k] = common.ConvertIconToBase64(&appConfig, link)
					}
				}
			}
		}

		if iconRef, ok := source["icon"].(string); ok {
			if assets, ok := source["assets"].(map[string]interface{}); ok {
				if icons, ok := assets["icons"].(map[string]interface{}); ok {
					if iconValue, ok := icons[iconRef].(string); ok {
						source["icon"] = common.ConvertIconToBase64(&appConfig, common.AutoGetFullIconURL(&appConfig, iconValue))
					} else {
						source["icon"] = common.ConvertIconToBase64(&appConfig, common.AutoGetFullIconURL(&appConfig, iconRef))
					}
				}
			} else {
				source["icon"] = common.ConvertIconToBase64(&appConfig, common.AutoGetFullIconURL(&appConfig, "icons"))
			}
		}

		return nil
	}

	err, res := core.SearchV2WithResultItemMapper(ctx, &connectors, builder, itemMapFunc)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}

}
