/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	log "github.com/cihub/seelog"
	"github.com/emirpasic/gods/maps/treemap"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/env"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

type APIHandler struct {
	api.Handler
	recommendConfigs map[string]core.RecommendResponse
	fieldMetadata    *treemap.Map
}

const Category = "coco"
const Resource = "document"
const Search = "search"
const Assistant = "assistant"
const QuickAISearchAction = "quick_ai_access"

func init() {
	handler := APIHandler{}

	createPermission := security.GetSimplePermission(Category, Resource, string(security.Create))
	updatePermission := security.GetSimplePermission(Category, Resource, string(security.Update))
	readPermission := security.GetSimplePermission(Category, Resource, string(security.Read))
	deletePermission := security.GetSimplePermission(Category, Resource, string(security.Delete))
	searchPermission := security.GetSimplePermission(Category, Resource, string(security.Search))
	security.GetOrInitPermissionKeys(createPermission, updatePermission, readPermission, deletePermission, searchPermission)

	// stamp content fingerprints on every document write, whichever path
	// it takes (document API, datasource API, pipeline, MCP) — D1.5
	registerFingerprintHook()

	//for internal document management, security should be enabled
	api.HandleUIMethod(api.POST, "/document/", handler.createDoc, api.RequirePermission(createPermission))
	api.HandleUIMethod(api.GET, "/document/:doc_id", handler.getDoc, api.RequirePermission(readPermission),
		api.MCPTool("get_document", "Get one document by id, including its content — the end of the citation chain: search hits and wiki pages carry source doc ids, this tool reads them."))
	api.HandleUIMethod(api.GET, "/document/:doc_id/raw_content/:hint", handler.getDocRawContent, api.RequirePermission(readPermission), api.AllowOPTIONSS(), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.PUT, "/document/:doc_id", handler.updateDoc, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.DELETE, "/document/:doc_id", handler.deleteDoc, api.RequirePermission(deletePermission))
	api.HandleUIMethod(api.GET, "/document/_search", handler.searchDocs, api.RequirePermission(searchPermission),
		//distinct from the /query/_search "search_documents" tool: the MCP
		//registry keys tools by name, a duplicate silently shadows one of
		//the two and leaves it with the wrong permission metadata
		api.MCPTool("search_documents_bm25", "Search documents (BM25). Pass query, optional filters and size; returns ids, titles, summaries — then get_document reads the full content."))
	api.HandleUIMethod(api.DELETE, "/document/", handler.batchDeleteDoc, api.RequirePermission(deletePermission))

	//content dedup: deterministic fingerprints, human review, no auto-delete;
	//the report scans cross-datasource with direct reads, so it shares the
	//operator-grade update gate with the dismiss/action endpoints instead
	//of the every-user document:read
	api.HandleUIMethod(api.GET, "/document/dedup/report", handler.dedupReportHandler, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.POST, "/document/dedup/dismiss", handler.dedupDismissHandler, api.RequirePermission(updatePermission))
	api.HandleUIMethod(api.POST, "/document/dedup/action", handler.dedupActionHandler, api.RequirePermission(updatePermission))

	querySearchPermission := security.GetSimplePermission(Category, Search, string(security.Search))
	assistantSearchPermission := security.GetSimplePermission(Category, Assistant, QuickAISearchAction)
	searchStudioPermission := security.GetSimplePermission(Category, Search, "studio")
	searchOpsPermission := security.GetSimplePermission(Category, Search, "ops")
	security.GetOrInitPermissionKeys(querySearchPermission, assistantSearchPermission, searchStudioPermission, searchOpsPermission)
	security.AssignPermissionsToRoles(querySearchPermission, core.WidgetRole)

	//live tuning surface for the dual-engine recall: runs both routes and
	//returns the RRF fusion math behind the hybrid_rrf search mode
	api.HandleUIMethod(api.POST, "/search/studio/test", handler.searchStudioTest, api.RequirePermission(searchStudioPermission))
	//golden-query evaluation set (D9): human-annotated query → expected-doc
	//pairs, scored against the live pipeline — the before/after gate for any
	//recall-stack change
	api.HandleUIMethod(api.GET, "/search/studio/eval/_cases", handler.searchEvalCases, api.RequirePermission(searchStudioPermission))
	api.HandleUIMethod(api.POST, "/search/studio/eval/_cases", handler.createSearchEvalCase, api.RequirePermission(searchStudioPermission))
	api.HandleUIMethod(api.DELETE, "/search/studio/eval/_cases/:id", handler.deleteSearchEvalCase, api.RequirePermission(searchStudioPermission))
	api.HandleUIMethod(api.POST, "/search/studio/eval/_run", handler.runSearchEval, api.RequirePermission(searchStudioPermission),
		api.MCPTool("run_search_eval", "Run the golden-query evaluation set against the live search pipeline and return the top-4 hit rate, MRR and per-query outcomes — the before/after evidence for any recall-stack change."))
	api.HandleUIMethod(api.GET, "/search/studio/eval/_runs", handler.listSearchEvalRuns, api.RequirePermission(searchStudioPermission))
	//what the search stack can do right now: resolved semantic route, engine
	//probe verdict, embedding model, vectorized document coverage
	api.HandleUIMethod(api.GET, "/search/engine-capability", handler.engineCapabilityReport, api.RequirePermission(searchStudioPermission))
	//operator's view of the iteration loop: what people searched, where
	//recall came up empty, what it cost in latency (P2/D5)
	api.HandleUIMethod(api.GET, "/search/ops/overview", handler.searchOpsOverview, api.RequirePermission(searchOpsPermission))
	//index health sweep (P4): every knowledge-hub store, one count each
	api.HandleUIMethod(api.GET, "/search/ops/index-health", handler.indexHealthHandler, api.RequirePermission(searchOpsPermission))
	//engine AI curation: read back the managed pipelines and their drift, and
	//push the current config to the engine on demand
	api.HandleUIMethod(api.GET, "/search/engine-ai", handler.engineAIStatus, api.RequirePermission(searchStudioPermission))
	api.HandleUIMethod(api.POST, "/search/engine-ai/sync", handler.engineAISync, api.RequirePermission(searchStudioPermission))

	api.HandleUIMethod(api.OPTIONS, "/field_meta/:field_name", handler.getFieldMeta, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.GET, "/field_meta/:field_name", handler.getFieldMeta, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.POST, "/field_meta/:field_name", handler.getFieldMeta, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.OPTIONS, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.GET, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS),
		api.MCPTool("search_documents", "Search the enterprise content indexed by Coco AI — files, wiki pages, chat messages, web pages and more, across all connected data sources. Returns matched documents with title, summary, source and deep link."),
		api.Label(api.MCPToolInputSchema, common.MCPQueryEnvelopeSchema(util.MapStr{
			"query":       util.MapStr{"type": "string", "description": "Keywords to search for; Lucene query_string syntax is supported."},
			"search_type": util.MapStr{"type": "string", "enum": []string{"keyword", "semantic", "hybrid", "hybrid_rrf"}, "description": "Search strategy override; when omitted the operator-configured default runs (keyword unless changed in settings)."},
			"datasource":  util.MapStr{"type": "string", "description": "Restrict the search to one datasource ID (list IDs with search_datasources)."},
			"category":    util.MapStr{"type": "string", "description": "Document category filter, e.g. file, page, message."},
			"size":        util.MapStr{"type": "integer", "description": "Page size, default 10."},
			"from":        util.MapStr{"type": "integer", "description": "Pagination offset."},
		}, []string{"query"})))
	api.HandleUIMethod(api.POST, "/query/_search", handler.search, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_suggest", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_suggest", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_suggest/:tag", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_suggest/:tag", handler.suggest, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_recommend", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_recommend", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	api.HandleUIMethod(api.GET, "/query/_recommend/:tag", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))
	api.HandleUIMethod(api.OPTIONS, "/query/_recommend/:tag", handler.recommend, api.RequirePermission(querySearchPermission), api.Feature(core.FeatureCORS))

	global.RegisterFuncAfterSetup(func() {

		fieldMetadataMap := map[string]FieldMetadata{}
		ok, err := env.ParseConfig("field_metadata", &fieldMetadataMap)
		if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		}
		// Convert to TreeMap for sorted iteration by key
		handler.fieldMetadata = treemap.NewWithStringComparator()
		for k, v := range fieldMetadataMap {
			handler.fieldMetadata.Put(k, v)
		}
		log.Trace(util.ToJson(fieldMetadataMap, true))

		cfg1 := map[string]string{}
		ok, err = env.ParseConfig("recommend", &cfg1)
		if ok && err != nil && global.Env().SystemConfig.Configs.PanicOnConfigError {
			panic(err)
		}

		recommends := map[string]core.RecommendResponse{}
		for k, v := range cfg1 {
			recommend := core.RecommendResponse{}
			err := util.FromJson(v, &recommend)
			if err != nil {
				panic(err)
			}
			recommends[k] = recommend
		}
		handler.recommendConfigs = recommends
	})

}
