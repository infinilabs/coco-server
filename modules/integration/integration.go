/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package integration

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"

	"infini.sh/coco/core"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
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
	err = validateAlias(obj.Tenant, obj.Alias, "")
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusBadRequest)
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
	_, tenantInDelta := delta["tenant"]
	_, aliasInDelta := delta["alias"]
	if tenantInDelta || aliasInDelta {
		if err := validateAliasDelta(req, id, delta); err != nil {
			h.WriteError(w, err.Error(), http.StatusBadRequest)
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

		// NOTICE: i don't think we should modify anything when do search!

		//for _, hit := range searchRes.Hits.Hits {
		//	if token, ok := hit.Source["token"].(string); ok && token != "" {
		//		tokenObj, err := security.GetToken(token)
		//		if tokenObj == nil && err == nil {
		//			// token is not found in the kv, here we set it as expired
		//			hit.Source["token_expire_in"] = time.Time{}.Unix()
		//		}
		//		if tokenObj != nil {
		//			hit.Source["token_expire_in"] = tokenObj.ExpireIn
		//		}
		//	}
		//}

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

// aliasSegmentPattern constrains each part of the (tenant, alias) pair so both
// can be safely embedded in the public URL path segment "tenant:alias"; ":"
// itself is excluded so the separator cannot be ambiguous.
var aliasSegmentPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Helper function to validate the (tenant, alias) pair of an integration:
// format-check both parts, require them to be set or cleared together, and
// reject a pair that is already used by another integration. excludeID is
// skipped so re-saving a document does not trip the check on its own alias.
func validateAlias(tenant, alias, excludeID string) error {
	if tenant == "" && alias == "" {
		return nil
	}
	if tenant == "" || alias == "" {
		return fmt.Errorf("tenant and alias must be set together")
	}
	if !aliasSegmentPattern.MatchString(tenant) {
		return fmt.Errorf("invalid tenant [%s]: must start with a letter or digit, and only contain letters, digits, dot, underscore or hyphen", tenant)
	}
	if !aliasSegmentPattern.MatchString(alias) {
		return fmt.Errorf("invalid alias [%s]: must start with a letter or digit, and only contain letters, digits, dot, underscore or hyphen", alias)
	}
	integrations := []core.Integration{}
	err, _ := orm.SearchWithJSONMapper(&integrations, &orm.Query{
		Size:  10,
		Conds: orm.And(orm.Eq("tenant", tenant), orm.Eq("alias", alias)),
	})
	if err != nil {
		return err
	}
	for _, item := range integrations {
		if item.ID != excludeID {
			return fmt.Errorf("alias [%s:%s] is already taken", tenant, alias)
		}
	}
	return nil
}

// Helper function to validate the effective (tenant, alias) pair of a partial
// update: merges the delta over the stored document first, because the request
// body may carry only one of the two fields.
func validateAliasDelta(req *http.Request, id string, delta util.MapStr) error {
	obj := core.Integration{}
	obj.ID = id
	ctx := orm.NewContextWithParent(req.Context())
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "integration")
	ctx.DirectReadAccess()
	exists, err := orm.GetV2(ctx, &obj)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("integration [%s] not found", id)
	}
	tenant, alias := obj.Tenant, obj.Alias
	if v, ok := delta["tenant"].(string); ok {
		tenant = v
	}
	if v, ok := delta["alias"].(string); ok {
		alias = v
	}
	return validateAlias(tenant, alias, id)
}
