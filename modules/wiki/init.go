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

	// TOC tree, one per KB (kb-level permissions guard the object); the
	// param name must match the crud-generated :id of /wiki/kb/:id
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/toc", handler.getToc,
		api.RequireLogin(), api.RequirePermission(readKbPermission))
	api.HandleUIMethod(api.PUT, "/wiki/kb/:id/toc", handler.updateToc,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))

	// read-only version history (article-level read)
	api.HandleUIMethod(api.GET, "/wiki/article/:id/versions", handler.articleVersions,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))
	api.HandleUIMethod(api.GET, "/wiki/article/:id/versions/:version", handler.articleVersionDetail,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))

	// status machine, see statusTransitions
	api.HandleUIMethod(api.PUT, "/wiki/article/:id/status", handler.updateArticleStatus,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))

	// ontology query face (design doc B5): exact-name resolution and
	// one-hop graph traversal, exposed as MCP tools for assistants
	api.HandleUIMethod(api.GET, "/wiki/entity/lookup", handler.entityLookup,
		api.RequireLogin(), api.RequirePermission(searchEntityPermission),
		api.MCPTool("entity_lookup", "Resolve an ontology entity by exact name or alias; returns the entity with relations, or not_found"))
	api.HandleUIMethod(api.GET, "/wiki/entity/:id/neighbors", handler.entityNeighbors,
		api.RequireLogin(), api.RequirePermission(readEntityPermission),
		api.MCPTool("entity_neighbors", "One-hop traversal of an entity's relations; returns edges and the expanded neighbor entities"))

	// KM agent generation, SSE (design doc §5.1, stage C): the event
	// contract is progress/article/done; output stops at draft — the
	// review gate stays on PUT /wiki/article/:id/status (D1)
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/ai/generate", handler.aiGenerate,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))

	// instruction-based AI edit, SSE (design doc §6.1/C4): rewrites content
	// and records an ai-generated version, status machine untouched
	api.HandleUIMethod(api.POST, "/wiki/article/:id/ai/edit", handler.aiEdit,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))
}
