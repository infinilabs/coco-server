/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package connector

import (
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"time"
)

func (h *APIHandler) create(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &common.Connector{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
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
	obj := common.Connector{}

	replace := h.GetBoolOrDefault(req, "replace", false)

	var err error
	var create *time.Time
	var builtin bool
	if !replace {
		obj.ID = id
		exists, err := orm.Get(&obj)
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
	ctx := &orm.Context{
		Refresh: orm.WaitForRefresh,
	}
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

func (h *APIHandler) delete(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := common.Connector{}
	obj.ID = id

	exists, err := orm.Get(&obj)
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

	err = orm.Delete(nil, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "deleted",
	}, 200)
}

func (h *APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var err error
	q := orm.Query{}
	q.RawQuery, err = h.GetRawBody(req)
	//TODO handle url query args

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

	err, res := orm.SearchWithResultItemMapper(&connectors, itemMapFunc, &q)

	//err, res := orm.SearchWithResultItemMapper(&connectors, func(source map[string]interface{}, targetRef interface{}) error {
	//	if !appConfig.ServerInfo.EncodeIconToBase64 {
	//		return nil
	//	}
	//
	//	// Ensure it's a pointer to common.Connector
	//	connPtr, ok := targetRef.(*common.Connector)
	//	if !ok {
	//		return errors.New("targetRef must be *common.Connector")
	//	}
	//
	//	// Unmarshal source into the pointer
	//	sourceBytes, err := util.ToJSONBytes(source)
	//	if err != nil {
	//		return err
	//	}
	//	if err := util.FromJSONBytes(sourceBytes, connPtr); err != nil {
	//		return err
	//	}
	//
	//	// Process icons
	//	newIcons := map[string]string{}
	//	for k, icon := range connPtr.Assets.Icons {
	//		link := common.AutoGetFullIconURL(&appConfig, icon)
	//		newIcons[k] = common.ConvertIconToBase64(&appConfig, link)
	//	}
	//	connPtr.Assets.Icons = newIcons
	//
	//	if connPtr.Icon != "" {
	//		connPtr.Icon = common.ParseAndGetIcon(connPtr, connPtr.Icon)
	//	}
	//
	//	return nil
	//}, &q)

	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//bytes := res.Raw
	//if common.AppConfig().ServerInfo.EncodeIconToBase64 {
	//
	//	//fmt.Println(util.MustToJSON(connectors))
	//	//for _, connector := range connectors {
	//		data := elastic.DocumentWithMeta[common.Connector]{}
	//		data.ID = connector.ID
	//	}
	//}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}
