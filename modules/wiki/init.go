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
	// semantic grouping of the loose root pages (LLM categories); manual
	// folders are preserved — the AI only re-files what was left unfiled
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/toc/_organize", handler.organizeToc,
		api.RequireLogin(), api.RequirePermission(updateKbPermission),
		api.MCPTool("wiki_organize_toc", "Trigger a background semantic grouping of a knowledge base's loose TOC pages: proposes/uses topic categories and files every unfiled root page (manual folders untouched, existing folders reused by name). Returns {started, loose}; poll wiki_toc_organize_status until running is false"))
	// polling target for the running pass — read-only, so read permission
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/toc/_organize", handler.organizeTocStatus,
		api.RequireLogin(), api.RequirePermission(readKbPermission),
		api.MCPTool("wiki_toc_organize_status", "Check a knowledge base's semantic toc-organize state: {running, loose, last pass stats, last_error, last_at}"))

	// evolution log (LLM-wiki P0): the KB's append-only decision record —
	// applied/dismissed proposals with actor and outcome
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_evolution_log", handler.evolutionLog,
		api.RequireLogin(), api.RequirePermission(readKbPermission),
		api.MCPTool("wiki_evolution_log", "Read a knowledge base's evolution log: the append-only record of what was applied, dismissed, distilled, proposed and gate-rejected — read it before proposing so the same fix is not re-proposed"))

	// experience layer (LLM-wiki P1): raw traces the distiller mines —
	// capture (API/MCP callers; the chat vote button lands here) and list
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_experience", handler.captureExperience,
		api.RequireLogin(), api.RequirePermission(updateKbPermission),
		api.MCPTool("record_wiki_experience", "Record how an answer went in the wild for a knowledge base (question, answer excerpt, cited doc ids, feedback). Only lands when the KB opted into experience capture; the maintainer mines these to distill failure patterns."))
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_experience", handler.listExperience,
		api.RequireLogin(), api.RequirePermission(readKbPermission))

	// chat-surface vote capture: the widget knows the assistant, not the
	// KB — targets resolve through the assistant↔KB binding
	api.HandleUIMethod(api.POST, "/wiki/experience/_vote", handler.captureExperienceVote,
		api.RequireLogin(),
		api.MCPTool("vote_answer_experience", "Record a thumbs-up/down on an assistant answer so knowledge bases bound to that assistant (with experience capture on) can learn from it"))

	// EINO compile agent (DESIGN-compile-agent.md): iterative ReAct loop
	// over the KB's topic pages — evidence-grounded page revisions land as
	// compile_patch governance proposals
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_agent_compile", handler.runCompileAgentHandler,
		api.RequireLogin(), api.RequirePermission(updateKbPermission),
		api.MCPTool("run_compile_agent", "Run the iterative knowledge-compilation agent over a knowledge base: assesses topic pages against the corpus with tools (search documents, read pages, read the evolution log) and files at most one compile_patch governance proposal per iteration"))

	// maintainer distillation (LLM-wiki P1): mine experience records into
	// pattern proposals; the scheduled sweep runs the same loop
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_distill", handler.distillKBHandler,
		api.RequireLogin(), api.RequirePermission(updateKbPermission),
		api.MCPTool("distill_wiki_experience", "Run one maintainer pass over a knowledge base's experience records: samples recent traces, distills failure patterns via LLM and files pattern_create/pattern_update governance proposals. Returns the run stats."))

	// skill proposer (LLM-wiki P2): one atomic skill proposal per run from
	// pattern pages + experiences + evolution log; reviewer-gated at file
	// time; skills export to the Agent Skills SKILL.md format
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_propose", handler.proposeSkillHandler,
		api.RequireLogin(), api.RequirePermission(updateKbPermission),
		api.MCPTool("propose_wiki_skill", "Run one proposer pass over a knowledge base: reads pattern pages, recent experiences and the evolution log, then files at most one reviewer-gated skill_create/skill_patch governance proposal (or records no_action)."))
	api.HandleUIMethod(api.GET, "/wiki/article/:id/_skill_md", handler.exportSkillMd,
		api.RequireLogin(), api.RequirePermission(readArticlePermission),
		api.MCPTool("export_skill_md", "Export a wiki skill page as an Agent Skills SKILL.md document (frontmatter name/description plus steps)"))
	// import half: an external SKILL.md lands as a draft skill page — the
	// review gate stays human
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_import_skill", handler.importSkillMd,
		api.RequireLogin(), api.RequirePermission(updateKbPermission),
		api.MCPTool("import_skill_md", "Import an Agent Skills SKILL.md document into a knowledge base as a draft skill page; same-title pages report instead of duplicating"))
	// package half (P3): SKILL.md + references/ as one zip — reference
	// files land as the page's tier-3 load attachments
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_import_skill_pack", handler.importSkillPack,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))

	// per-KB validation set (LLM-wiki P3): golden queries scored against
	// KB-scoped published-page retrieval; a skill landing in a KB with
	// cases fires a replay whose scores land in the evolution log
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_evalset", handler.listWikiEvalCasesHandler,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_evalset", handler.upsertWikiEvalCaseHandler,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))
	api.HandleUIMethod(api.DELETE, "/wiki/kb/:id/_evalset/:cid", handler.deleteWikiEvalCaseHandler,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_evalset/_run", handler.runWikiEvalHandler,
		api.RequireLogin(), api.RequirePermission(updateKbPermission))
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_evalset/_runs", handler.listWikiEvalRunsHandler,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))

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
	api.HandleUIMethod(api.PUT, "/wiki/entity/_batch_status", handler.batchUpdateEntityStatus,
		api.RequireLogin(), api.RequirePermission(security.GetSimplePermission(Category, entityResource, string(security.Update))))
	// merge same-concept entities: absorbed rows' names become aliases,
	// sources/relations union, references retarget to the canonical row
	api.HandleUIMethod(api.POST, "/wiki/entity/_merge", handler.mergeEntities,
		api.RequireLogin(), api.RequirePermission(security.GetSimplePermission(Category, entityResource, string(security.Update))))
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

	// KB health report (LLM-wiki lint, whole-KB 体检): orphan pages,
	// red-link knowledge gaps, thin/unbacked pages and hub stats — one
	// scan tells maintainers where attention pays off next
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_health", handler.kbHealth,
		api.RequireLogin(), api.RequirePermission(readKbPermission),
		api.MCPTool("kb_health", "Get a knowledge base health report: orphan pages, red-link knowledge gaps worth writing next, thin pages, unbacked concept pages and hub pages"))

	// KB-level files: raw material listing + manual uploads (same pipeline
	// as synced content, manual delivery)
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_files", handler.kbFiles,
		api.RequireLogin(), api.RequirePermission(readKbPermission))
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_files/upload", handler.uploadKBFiles,
		api.RequireLogin(), api.RequirePermission(security.GetSimplePermission(Category, kbResource, string(security.Update))))

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
	// batch review: same transition rules per id, tree_organize refused
	api.HandleUIMethod(api.PUT, "/wiki/governance/_batch_status", handler.batchUpdateGovernanceStatus,
		api.RequireLogin(), api.RequirePermission(security.GetSimplePermission(Category, governanceResource, string(security.Update))))

	// the queue's closed loop: execute the fix a proposal implies (tree the
	// orphan, merge the duplicate, rebuild the stale page, adopt the type,
	// materialize the compiled draft, apply the suggested TOC), then retire
	// it as resolved — one click instead of a note that changes nothing
	api.HandleUIMethod(api.POST, "/wiki/governance/:id/_apply", handler.applyGovernanceProposal,
		api.RequireLogin(), api.RequirePermission(security.GetSimplePermission(Category, governanceResource, string(security.Update))))
	// batch apply: background worker + polled progress
	api.HandleUIMethod(api.POST, "/wiki/governance/_batch_apply", handler.startBatchApply,
		api.RequireLogin(), api.RequirePermission(security.GetSimplePermission(Category, governanceResource, string(security.Update))))
	api.HandleUIMethod(api.GET, "/wiki/governance/_batch_apply", handler.batchApplyProgressHandler,
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

	// knowledge compiler (W9): four-stage pipeline over the KB's source
	// documents — every output is a governance proposal
	api.HandleUIMethod(api.GET, "/wiki/article/:id/_materials", handler.articleMaterials,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))
	// page-level attachments: files hung on the wiki page itself (N per
	// page), independent of the cited source documents
	api.HandleUIMethod(api.GET, "/wiki/article/:id/_attachments", handler.articleAttachments,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))
	api.HandleUIMethod(api.PUT, "/wiki/article/:id/_attachments", handler.updateArticleAttachments,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))
	// REFINE rewrite: the writing pass for drafts applied before it existed
	api.HandleUIMethod(api.POST, "/wiki/article/:id/_rewrite", handler.rewriteArticle,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_consolidate", handler.consolidateKB,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_compile", handler.compileKBHandler,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))
	// compile console: progress/logs of the current or last run, and cancel
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_compile", handler.compileJobStatus,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))
	api.HandleUIMethod(api.POST, "/wiki/kb/:id/_compile/_cancel", handler.cancelCompile,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))

	// compile op dead letter (compileops P1): ops that exhausted their
	// retries — the KB overview shows the count, acknowledge to clear
	api.HandleUIMethod(api.GET, "/wiki/kb/:id/_opdead", handler.compileOpsDeadHandler,
		api.RequireLogin(), api.RequirePermission(readArticlePermission))
	api.HandleUIMethod(api.DELETE, "/wiki/kb/:id/_opdead", handler.compileOpsDeadClearHandler,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))

	// one-off published-article projection backfill (W16a): run before
	// flipping search_settings.wiki_projection so the unified leg never
	// serves a half-empty index
	api.HandleUIMethod(api.POST, "/wiki/_backfill_projection", handler.backfillProjection,
		api.RequireLogin(), api.RequirePermission(updateArticlePermission))

	scheduleSeedOntologySchema()
	scheduleBackfillDraftEntityContent()
	scheduleBackfillLinkedPages()
	scheduleReconcileContributions()
}
