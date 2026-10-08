/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"net/http"
	"sync"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
)

// isBuiltInIntegration reports whether the integration id is reserved for the
// app's own search page.
func isBuiltInIntegration(id string) bool {
	return core.IsBuiltInIntegration(id)
}

func (h *APIHandler) create(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	ctx := orm.NewContextWithParent(req.Context())
	ctx.Refresh = orm.WaitForRefresh

	var obj = &core.Integration{}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isBuiltInIntegration(obj.ID) {
		h.WriteError(w, "integration id is reserved for the built-in search integration", http.StatusBadRequest)
		return
	}
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
	// the built-in integration stays editable (its payload is the in-app
	// search's config surface), but disabling it would take the app's own
	// search page down — reject that one field
	if isBuiltInIntegration(id) {
		if enabled, ok := delta["enabled"].(bool); ok && !enabled {
			h.WriteError(w, "the built-in search integration cannot be disabled", http.StatusBadRequest)
			return
		}
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

	if isBuiltInIntegration(id) {
		h.WriteError(w, "the built-in search integration cannot be deleted", http.StatusBadRequest)
		return
	}

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
	integrationOrigins.Delete(id)

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

		// NOTICE: i don't think we should modify anything when do search!

		// the built-in search integration is app infrastructure, not a
		// manageable item — keep it out of every management list (the direct
		// GET /integration/:id the search page uses is unaffected)
		core.StripBuiltInIntegration(&searchRes)

	}

	h.WriteJSON(w, searchRes, http.StatusOK)
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
