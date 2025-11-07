/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/connector"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
)

func (h *APIHandler) createDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &core.Document{}
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

	h.WriteCreatedOKJSON(w, obj.ID)
}

func (h *APIHandler) getDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")

	obj := core.Document{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")
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
	ctx := orm.NewContextWithParent(req.Context())

	obj := core.Document{}
	err := h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//protect
	obj.ID = id
	ctx.Refresh = orm.WaitForRefresh
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")

	//update share context
	ctx.Set(orm.SharingCheckingResourceCategoryEnabled, true)
	ctx.Set(orm.SharingResourceCategoryType, "datasource")
	ctx.Set(orm.SharingResourceCategoryFilterField, "source.id")
	ctx.Set(orm.SharingResourceCategoryID, obj.Source.ID)
	ctx.Set(orm.SharingResourceParentPath, obj.Category)
	ctx.Set(orm.SharingCheckingInheritedRulesEnabled, true)

	err = orm.Save(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteUpdatedOKJSON(w, obj.ID)
}

func (h *APIHandler) deleteDoc(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("doc_id")

	obj := core.Document{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())

	ctx.Refresh = orm.WaitForRefresh
	err := orm.Delete(ctx, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteDeletedOKJSON(w, obj.ID)
}

func (h *APIHandler) searchDocs(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "title", "summary", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	builder.EnableBodyBytes()
	if len(builder.Sorts()) == 0 {
		builder.SortBy(orm.Sort{Field: "created", SortType: orm.DESC})
	}

	ctx := orm.NewContextWithParent(req.Context())
	view := h.GetParameter(req, "view")
	//view := "list"
	sourceIDs := builder.GetFilterValues("source.id")

	pathHierarchy := false
	//apply datasource filter //TODO datasource may support multi ids
	if len(sourceIDs) == 1 {
		ctx1 := orm.NewContext()
		ctx1.DirectReadAccess()
		sourceIDArray, ok := sourceIDs[0].([]interface{})
		if ok {
			sourceID, ok := sourceIDArray[0].(string)
			if ok {
				ds, err := common.GetDatasourceConfig(ctx1, sourceID)
				if err != nil {
					panic(err)
				}
				if ds != nil {
					conn, err := connector.GetConnectorByID(ds.Connector.ConnectorID)
					if err != nil {
						panic(err)
					}
					if conn.PathHierarchy {
						pathHierarchy = true
					}

					ctx.Set(orm.SharingCheckingResourceCategoryEnabled, true)
					ctx.Set(orm.SharingResourceCategoryType, "datasource")
					ctx.Set(orm.SharingResourceCategoryFilterField, "source.id")
					ctx.Set(orm.SharingResourceCategoryID, ds.ID)
				}
			}
		}
	}

	//TODO cache
	var pathStr = "/"
	pathFilterStr := h.GetParameter(req, "path")
	if pathFilterStr != "" {
		array := []string{}
		err = util.FromJson(pathFilterStr, &array)
		if err != nil {
			panic(err)
		}
		if len(array) > 0 {
			pathStr = common.GetFullPathForCategories(array)
			//builder.Filter(orm.TermQuery("_system.parent_path", pathStr))
			//log.Trace("adding path hierarchy filter: ", pathStr)
			//ctx.Set(orm.SharingResourceParentPath, pathStr)
		}
	}

	//path str
	if view != "list" && pathHierarchy && pathStr != "" {
		builder.Filter(orm.TermQuery("_system.parent_path", pathStr))
		log.Trace("adding path hierarchy filter: ", pathStr)
		ctx.Set(orm.SharingResourceParentPath, pathStr)
	} else {
		//apply path filter to list view too
		if pathStr != "/" {
			builder.Filter(orm.TermQuery("_system.parent_path", pathStr))
			log.Trace("adding path hierarchy filter: ", pathStr)
			ctx.Set(orm.SharingResourceParentPath, pathStr)
		}
	}

	orm.WithModel(ctx, &core.Document{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")
	ctx.Set(orm.SharingCheckingInheritedRulesEnabled, true)

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

	builder, err := orm.NewQueryBuilderFromRequest(req, "title", "summary", "combined_fulltext")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	builder.Filter(orm.TermsQuery("id", ids))

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.Document{})

	_, err = orm.DeleteByQuery(ctx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.WriteAckOKJSON(w)
}
