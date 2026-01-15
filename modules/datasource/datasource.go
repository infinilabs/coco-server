/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package datasource

import (
	"net/http"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/document"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

func (h *APIHandler) createDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &core.DataSource{}
	h.MustDecodeJSON(req, obj)

	if obj.Type == "connector" {

		if obj.Connector.ConnectorID == "" {
			panic("invalid connector")
		}
		ctx := orm.NewContextWithParent(req.Context())

		//check connector
		connector := core.Connector{}
		connector.ID = obj.Connector.ConnectorID
		exists, err := orm.GetV2(ctx, &connector)
		if !exists || err != nil {
			panic("invalid connector")
		}

		ctx.Refresh = orm.WaitForRefresh

		err = orm.Create(ctx, obj)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !obj.Enabled {
			common.ClearDatasourcesCache()
			common.ClearDatasourceCache(obj.ID)
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

	obj := core.DataSource{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	exists, err := orm.GetV2(ctx, &obj)
	if err != nil {
		panic(err)
	}
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	//mark deleted in cache
	common.MarkDatasourceDeleted(id)

	// clear cache
	common.ClearDatasourcesCache()
	common.ClearDatasourceCache(obj.ID)

	//deleting related documents
	builder, err := orm.NewQueryBuilderFromRequest(req)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	builder.Filter(orm.TermQuery("source.id", id))

	ctx1 := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx1, &core.Document{})

	_, err = orm.DeleteByQuery(ctx1, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.WriteDeletedOKJSON(w, obj.ID)
}

func (h *APIHandler) getDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")

	obj := core.DataSource{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "datasource")
	ctx.Set(orm.SharingCategoryCheckingChildrenEnabled, true)

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

func (h *APIHandler) updateDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("id")
	if id == "" {
		panic("invalid id")
	}

	replace := h.GetBoolOrDefault(req, "replace", false)
	var err error

	obj := core.DataSource{}
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "datasource")

	if replace {
		err = orm.Upsert(ctx, &obj)
	} else {
		err = orm.Update(ctx, &obj)
	}

	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//clear cache
	common.ClearDatasourcesCache()
	common.ClearDatasourceCache(obj.ID)

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "updated",
	}, 200)
}

func GetDatasourceByID(id []string) ([]core.DataSource, error) {
	var err error
	q := orm.Query{}
	q.RawQuery, err = core.RewriteQueryWithFilter(q.RawQuery, util.MapStr{
		"terms": util.MapStr{
			"id": id,
		},
	})

	docs := []core.DataSource{}
	err, _ = orm.SearchWithJSONMapper(&docs, &q)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (h *APIHandler) searchDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name", "name.pinyin", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	builder.EnableBodyBytes()

	integrationID := req.Header.Get(core.HeaderIntegrationID)
	if integrationID != "" {
		ids, all, err := document.GetDatasourceByIntegration(integrationID)
		if err != nil {
			panic(err)
		}
		if !all {
			builder.Filter(orm.TermsQuery("id", ids))
		}
	}

	if len(builder.Sorts()) == 0 {
		builder.SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	}

	ctx := orm.NewContextWithParent(req.Context())

	orm.WithModel(ctx, &core.DataSource{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "datasource")
	ctx.Set(orm.SharingCategoryCheckingChildrenEnabled, true)

	docs := []core.DataSource{}

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

func (h *APIHandler) createDocInDatasource(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &core.Document{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		panic(err)
	}

	//TODO cache for speed
	datasourceID := ps.MustGetParameter("id")
	datasourceObj := core.DataSource{}
	datasourceObj.ID = datasourceID
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &datasourceObj)
	if !exists || err != nil {
		panic("invalid datasource")
	}

	//replace datasource info
	sourceRefer := core.DataSourceReference{}
	sourceRefer.ID = datasourceObj.ID
	sourceRefer.Type = datasourceObj.Type
	sourceRefer.Name = datasourceObj.Name
	obj.Source = sourceRefer

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

func (h *APIHandler) createDocInDatasourceWithID(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &core.Document{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		panic(err)
	}

	//TODO cache for speed
	datasourceID := ps.MustGetParameter("id")
	docID := ps.MustGetParameter("doc_id")
	obj.ID = docID
	datasourceObj := core.DataSource{}
	datasourceObj.ID = datasourceID
	ctx := orm.NewContextWithParent(req.Context())

	exists, err := orm.GetV2(ctx, &datasourceObj)
	if !exists || err != nil {
		panic("invalid datasource")
	}

	//replace datasource info
	sourceRefer := core.DataSourceReference{}
	sourceRefer.ID = datasourceObj.ID
	sourceRefer.Type = datasourceObj.Type
	sourceRefer.Name = datasourceObj.Name
	obj.Source = sourceRefer

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
