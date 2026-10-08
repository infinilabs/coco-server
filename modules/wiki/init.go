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
	workspaceResource  = "wiki_workspace"
	kbResource         = "wiki_kb"
	articleResource    = "wiki_article"
	bookmarkResource   = "wiki_bookmark"
	notificationRes    = "wiki_notification"
	entityResource     = "wiki_entity"
	commentResource    = "wiki_comment"
	governanceResource = "wiki_governance"
	likeResource       = "wiki_like"
)

type APIHandler struct {
	api.Handler
}

func init() {

	permKeys := make([]security.PermissionKey, 0, 30)
	for _, resource := range []string{workspaceResource, kbResource, articleResource, bookmarkResource, notificationRes, entityResource, commentResource, governanceResource, likeResource} {
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
	registerCommentCRUD()
	registerGovernanceCRUD()
	registerLikeCRUD()

	handler := APIHandler{}

	// TOC tree, one per KB (kb-level permissions guard the object); the
	// param name must match the crud-generated :id of /wiki/kb/:id
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/toc", handler.getToc,
		api.RequireLogin(), api.RequirePermission(readKbPermission))
	api.HandleUIMethod(api.PUT, "/wiki/kb/:id/toc", handler.updateToc,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))

	// read-only version history (article-level read); exposed to agents so
	// they can see how a page's knowledge evolved (P3)
	api.HandleUIMethod(api.GET, "/wiki/article/:id/versions", handler.articleVersions,
		api.MCPTool("list_article_versions", "List the version history of a wiki article: version, change_type (ai-generated | human-edited | auto-updated), change_summary, when. Use it to see how the knowledge on a page evolved."),
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

	// wikilink back-references (ontology O4): which articles cite an entity
	api.HandleUIMethod(api.GET, "/wiki/entity/:id/backlinks", handler.entityBacklinks,
		api.RequireLogin(), api.RequirePermission(readEntityPermission),
		api.MCPTool("entity_backlinks", "List the articles that reference an entity through wikilinks, with a context snippet per article"))

	// entity create is hand-registered (not the generated CRUD route) so the
	// body may carry a kb_id passthrough that validates against that KB's
	// ontology vocabulary — KB-specific types become usable from the UI (W3)
	api.HandleUIMethod(api.POST, "/wiki/entity/", handler.createEntity,
		api.RequireLogin(), api.RequirePermission(createEntityPermission),
		api.MCPTool("wiki_entity_create", "Create an ontology entity; optional kb_id validates against that KB's vocabulary"))

	// whole-KB knowledge graph (ontology phase O3): articles, wikilink-
	// resolved entities, one-hop relation expansion and schema-resolved
	// edge labels — one request feeding the KbGraph canvas and cards
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_graph", handler.kbGraph,
		api.RequireLogin(), api.RequirePermission(readKbPermission),
		api.MCPTool("kb_graph", "Get the knowledge graph of a knowledge base: article and entity nodes, wikilink and typed-relation edges with schema labels"))

	// KM agent generation, SSE (design doc §5.1, stage C): the event
	// contract is progress/article/done; output stops at draft — the
	// review gate stays on PUT /wiki/article/:id/status (D1)
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/ai/generate", handler.aiGenerate,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))

	// instruction-based AI edit, SSE (design doc §6.1/C4): rewrites content
	// and records an ai-generated version, status machine untouched
	api.HandleUIMethod(api.POST, "/wiki/article/:id/ai/edit", handler.aiEdit,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))

	// knowledge execution: land a chat answer into a KB as a draft article
	// (D1 inherited from createDraftArticle — publish stays human)
	api.HandleUIMethod(api.POST, "/wiki/article/_from_chat", handler.createArticleFromChat,
		api.RequireLogin(), api.RequirePermission(createArticlePermission),
		api.MCPTool("save_chat_answer_to_wiki", "Save an assistant answer (content + citations) into a knowledge base as a draft article; returns the new article id"))

	// governance queue: the human gate — resolving/dismissing records the
	// decision, fixes themselves happen on the article face (D1)
	// company map (D6): the agent entry page; read-first ritual for
	// external agents (get_company_map MCP tool)
	api.HandleUIMethod(api.GET, "/wiki/company-map", handler.getCompanyMap,
		api.MCPTool("get_company_map", "Get the company map — the hand-maintained entry page (what we do, current priorities, source priority, navigation). Read this before deeper research: it says where knowledge lives and which source wins on conflicts. kb query param optional."),
		api.RequireLogin())

	// correction capture (D7): any logged-in user can report a wrong answer
	// from work; it lands in the same human-gated queue as scanner proposals
	api.HandleUIMethod(api.POST, "/wiki/governance/_correction", handler.createCorrection,
		api.MCPTool("report_correction", "Report that an answer was wrong or outdated so the knowledge base can learn: pass the query, an answer excerpt, cited doc ids, a route_hint (fact_missing | fact_outdated | preference | technique | source_conflict) and the correction. A human reviews it before anything changes."),
		api.RequireLogin())

	api.HandleUIMethod(api.PUT, "/wiki/governance/:id/status", handler.updateGovernanceStatus,
		api.RequireLogin(), api.RequirePermission(security.GetSimplePermission(Category, governanceResource, string(security.Update))))

	// ontology vocabulary (phase O1): the declared entity types, typed
	// properties and relation vocabulary entity writes validate against;
	// one schema per scope (tenant default, kb:<id> override)
	api.HandleUIMethod(api.GET, "/wiki/ontology/schema", handler.getOntologySchema,
		api.RequireLogin(), api.RequirePermission(readEntityPermission),
		api.MCPTool("get_ontology_schema", "Get the ontology vocabulary this knowledge base validates against: declared entity types, their typed properties and relation definitions"))
	api.HandleUIMethod(api.PUT, "/wiki/ontology/schema", handler.putOntologySchema,
		api.RequireLogin(), api.RequirePermission(updateEntityPermission))

	// dangling-wikilink repair (phase O2): after entities are proposed from
	// unresolved links, relink binds them without touching the content
	api.HandleUIMethod(api.POST, "/wiki/article/:id/_relink", handler.relinkArticle,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))

	scheduleSeedOntologySchema()
	scheduleSeedBuiltinKmAssistant()
}
