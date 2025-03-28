/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package datasource

import (
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	log "src/github.com/cihub/seelog"
	"time"
)

func (h *APIHandler) createDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &common.DataSource{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if obj.Type == "connector" {

		if obj.Connector.ConnectorID == "" {
			panic("invalid connector")
		}

		//check connector
		connector := common.Connector{}
		connector.ID = obj.Connector.ConnectorID
		exists, err := orm.Get(&connector)
		if !exists || err != nil {
			panic("invalid connector")
		}

		ctx := orm.Context{
			Refresh: "wait_for",
		}
		err = orm.Create(&ctx, obj)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !obj.Enabled {
			common.DisabledDatasourceIDsCache.Delete(common.DatasourceCachePrimary, common.DisabledDatasourceIDsCacheKey)
		}

		h.WriteJSON(w, util.MapStr{
			"_id":    obj.ID,
			"result": "created",
		}, 200)
		return
	}

	h.WriteError(w, "invalid datasource", http.StatusInternalServerError)
}

func (h *APIHandler) deleteDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := common.DataSource{}
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
		Refresh: "wait_for",
	}
	err = orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// clear cache
	common.DisabledDatasourceIDsCache.Delete(common.DatasourceCachePrimary, common.DisabledDatasourceIDsCacheKey)
	// deleting related documents
	query := util.MapStr{
		"query": util.MapStr{
			"term": util.MapStr{
				"source.id": id,
			},
		},
	}
	err = orm.DeleteBy(&common.Document{}, util.MustToJSONBytes(query))
	if err != nil {
		log.Errorf("delete related documents with datasource [%s] error: %v", obj.Name, err)
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "deleted",
	}, 200)
}

func (h *APIHandler) getDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := common.DataSource{}
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

func (h *APIHandler) updateDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	obj := common.DataSource{}

	replace := h.GetBoolOrDefault(req, "replace", false)

	var err error
	var create *time.Time
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
	} else {
		t := time.Now()
		create = &t
	}

	obj = common.DataSource{}
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	obj.ID = id
	obj.Created = create
	ctx := &orm.Context{
		Refresh: "wait_for",
	}
	err = orm.Update(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//clear cache
	common.DisabledDatasourceIDsCache.Delete(common.DatasourceCachePrimary, common.DisabledDatasourceIDsCacheKey)

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "updated",
	}, 200)
}

func (h *APIHandler) searchDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var err error
	q := orm.Query{}
	q.RawQuery, err = h.GetRawBody(req)

	//TODO handle url query args
	docs := []common.DataSource{}
	err, res := orm.SearchWithJSONMapper(&docs, &q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

func (h *APIHandler) createDocInDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &common.Document{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		panic(err)
	}

	//TODO cache for speed
	datasourceID := ps.MustGetParameter("id")
	datasourceObj := common.DataSource{}
	datasourceObj.ID = datasourceID

	exists, err := orm.Get(&datasourceObj)
	if !exists || err != nil {
		panic("invalid datasource")
	}

	//replace datasource info
	sourceRefer := common.DataSourceReference{}
	sourceRefer.ID = datasourceObj.ID
	sourceRefer.Type = datasourceObj.Type
	sourceRefer.Name = datasourceObj.Name
	obj.Source = sourceRefer

	ctx := orm.Context{
		Refresh: orm.WaitForRefresh,
	}
	err = orm.Create(&ctx, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "created",
	}, 200)
}
