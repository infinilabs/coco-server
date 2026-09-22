/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package wiki

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/security"
)

const Category = "coco"

// resources and their permission keys (design doc §4.1)
var (
	workspaceResource = "wiki_workspace"
	kbResource        = "wiki_kb"
	articleResource   = "wiki_article"
	bookmarkResource  = "wiki_bookmark"
	notificationRes   = "wiki_notification"
	entityResource    = "wiki_entity"
)

type APIHandler struct {
	api.Handler
}

func init() {

	permKeys := make([]security.PermissionKey, 0, 30)
	for _, resource := range []string{workspaceResource, kbResource, articleResource, bookmarkResource, notificationRes, entityResource} {
		permKeys = append(permKeys,
			security.GetSimplePermission(Category, resource, string(security.Create)),
			security.GetSimplePermission(Category, resource, string(security.Update)),
			security.GetSimplePermission(Category, resource, string(security.Read)),
			security.GetSimplePermission(Category, resource, string(security.Delete)),
			security.GetSimplePermission(Category, resource, string(security.Search)),
		)
	}
	security.GetOrInitPermissionKeys(permKeys...)

	registerWorkspaceCRUD()
	registerKbCRUD()
	registerArticleCRUD()
	registerBookmarkCRUD()
	registerNotificationCRUD()
	registerEntityCRUD()

	handler := APIHandler{}

	// TOC tree, one per KB (kb-level permissions guard the object)
	api.HandleUIMethod(api.GET, "/wiki/kb/:kbId/toc", handler.getToc,
		api.RequireLogin(), api.RequirePermission(readKbPermission))
	api.HandleUIMethod(api.PUT, "/wiki/kb/:kbId/toc", handler.updateToc,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))

	// read-only version history (article-level read)
	api.HandleUIMethod(api.GET, "/wiki/article/:id/versions", handler.articleVersions,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))
	api.HandleUIMethod(api.GET, "/wiki/article/:id/versions/:version", handler.articleVersionDetail,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))

	// status machine, see statusTransitions
	api.HandleUIMethod(api.PUT, "/wiki/article/:id/status", handler.updateArticleStatus,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))
}
