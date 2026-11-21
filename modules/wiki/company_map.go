/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package wiki

import (
	"net/http"

	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"

	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
)

// Company map (D6): one hand-maintained page (page_type=map) per KB is the
// entry point for humans and agents — what we do, current priorities, the
// source-priority table and navigation. More context creates more
// confusion; the answer is navigation, not more memory. Exposed as the
// get_company_map MCP tool so external agents read the map before anything
// else (J.B.'s "company brain" entry ritual).

// companyMapConfigFn is swapped out in tests (AppConfig touches the kv store).
var companyMapConfigFn = common.AppConfig

// getCompanyMap serves GET /wiki/company-map?kb=<id>.
func (h *APIHandler) getCompanyMap(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	per := security.GetSimplePermission(Category, articleResource, string(security.Search))
	perID := security.GetOrInitPermissionKey(per)
	reqUser, err := security.GetUserFromRequest(req)
	if err != nil || reqUser == nil {
		h.WriteError(w, "login required", http.StatusUnauthorized)
		return
	}
	if !(reqUser.Roles != nil && util.AnyInArrayEquals(reqUser.Roles, security.RoleAdmin)) &&
		!reqUser.UserAssignedPermission.ValidateFor(perID) {
		h.WriteError(w, "permission denied", http.StatusForbidden)
		return
	}

	kbID := req.URL.Query().Get("kb")

	octx := orm.NewContextWithParent(req.Context())
	octx.Set(orm.DirectReadWithoutPermissionCheck, true)
	orm.WithModel(octx, &core.WikiArticle{})

	builder := orm.NewQuery().Size(1).
		Filter(orm.TermQuery("page_type", core.WikiPageTypeMap),
			orm.TermQuery("status", core.WikiArticlePublished)).
		SortBy(orm.Sort{Field: "updated", SortType: orm.DESC})
	if kbID != "" {
		builder.Filter(orm.TermQuery("kb_id", kbID))
	}
	res, err := orm.SearchV2(octx, builder)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	articles, _, err := elastic.DecodeHits[core.WikiArticle](res)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(articles) == 0 {
		h.WriteOKJSON(w, util.MapStr{
			"found": false,
			"note":  "no published company map page (page_type=map) — create one so agents get an entry point",
		})
		return
	}

	article := &articles[0]
	endpoint := ""
	if cfg := companyMapConfigFn(); cfg.ServerInfo != nil {
		endpoint = cfg.ServerInfo.Endpoint
	}
	h.WriteOKJSON(w, util.MapStr{
		"found": true,
		"id":    article.ID,
		"map": util.MapStr{
			"kb_id":   article.KbID,
			"title":   article.Title,
			"summary": article.Summary,
			"content": article.Content,
			"updated": article.Updated,
			"url":     endpoint + "/#/wiki/article/" + article.ID,
			"tags":    article.Tags,
			"sources": article.Sources,
		},
	})
}
