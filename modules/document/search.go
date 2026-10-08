/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/attachment"
	"infini.sh/coco/modules/common"
	"infini.sh/coco/modules/connector"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
)

// maxSearchPageSize bounds one interactive search page: search serves
// humans and widgets; bulk export has dedicated APIs.
const maxSearchPageSize = 100

// Multimodal search: attachments posted alongside the query (images the
// vision pipeline described, documents with extracted text) join the query
// text so a picture can be searched by what it shows.
const (
	// multimodalAttachmentTextCap limits each attachment's contribution —
	// vision descriptions can run long and the keyword leg turns every term
	// into a clause
	multimodalAttachmentTextCap = 400
	// multimodalQueryTextCap bounds the combined attachment text
	multimodalQueryTextCap = 1000
	// multimodalWaitTimeout bounds how long the search waits for an
	// attachment's background text extraction (vision model) to finish;
	// chat waits far longer, a search must stay interactive
	multimodalWaitTimeout = 10 * time.Second
)

// enrichQueryFromAttachments merges the extracted text of the given
// attachment IDs into the user query (multimodal search). It waits briefly
// for in-flight extraction, skips attachments with no usable text (pending,
// failed, or owned by someone else — attachment IDs are unguessable UUIDs,
// this is a cheap consistency guard, not the security boundary), and returns
// the enriched query plus a human-readable note for the Warning header when
// something was skipped.
func enrichQueryFromAttachments(ctx context.Context, query string, attachmentsParam string, userID string) (string, string) {
	return enrichQuery(ctx, query, attachmentsParam, userID,
		func(ctx context.Context, ids []string) error {
			_, err := attachment.WaitForAttachmentsCompletion(ctx, ids, multimodalWaitTimeout, nil)
			return err
		},
		attachment.LoadAttachmentsForChat)
}

// enrichQuery is enrichQueryFromAttachments with the extraction wait and the
// attachment loader as seams, so the merge rules are unit-testable without a
// live engine.
func enrichQuery(ctx context.Context, query string, attachmentsParam string, userID string,
	wait func(ctx context.Context, ids []string) error,
	load func(ids []string) []*core.Attachment) (string, string) {

	ids := make([]string, 0, 4)
	for _, id := range strings.Split(attachmentsParam, ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return query, ""
	}

	// an image submitted right after upload may still be describing; wait a
	// bounded moment so its text makes it into this search. A timeout or
	// client cancel is not fatal: search with whatever text is ready.
	if err := wait(ctx, ids); err != nil {
		log.Debugf("multimodal search: attachment wait ended early: %v", err)
	}

	var parts []string
	skipped := 0
	for _, att := range load(ids) {
		if owner := att.GetOwnerID(); owner != "" && owner != userID {
			skipped++
			continue
		}
		text := strings.TrimSpace(att.Text)
		if text == "" {
			skipped++
			continue
		}
		parts = append(parts, util.SubString(text, 0, multimodalAttachmentTextCap))
	}

	note := ""
	if skipped > 0 {
		note = fmt.Sprintf("multimodal: %d of %d attachments had no usable text (extraction pending, failed, or unavailable); searched without them", skipped, len(ids))
	}
	if len(parts) == 0 {
		return query, note
	}

	combined := strings.Join(parts, " ")
	combined = util.SubString(combined, 0, multimodalQueryTextCap)
	if strings.TrimSpace(query) == "" {
		return combined, note
	}
	return query + " " + combined, note
}

// stripHighlightWithoutQuery drops the highlight section a client posted in
// the request body when the search carries no query term. Pure-filter
// browsing has nothing to mark, and the empty text clauses the keyword leg
// builds for an empty query (an empty prefix matches every term) would
// light whole fragments up instead.
func stripHighlightWithoutQuery(builder *orm.QueryBuilder, query string) {
	if strings.TrimSpace(query) != "" {
		return
	}
	body := builder.RequestBodyBytesVal()
	if len(body) == 0 {
		return
	}
	var dsl map[string]interface{}
	if err := util.FromJSONBytes(body, &dsl); err != nil || dsl == nil {
		return
	}
	if _, ok := dsl["highlight"]; !ok {
		return
	}
	delete(dsl, "highlight")
	builder.SetRequestBodyBytes(util.MustToJSONBytes(dsl))
}

func (h APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	// connector facet filters arrive as source.connector_id terms; documents
	// carry no such field — rewrite them into datasource terms before any
	// query builder (keyword/semantic legs, RRF routes, aggregations) reads
	// the request
	translateConnectorFilters(req)

	var (
		query        = h.GetParameterOrDefault(req, "query", "")
		datasource   = h.GetParameterOrDefault(req, "datasource", "")
		category     = h.GetParameterOrDefault(req, "category", "")
		subcategory  = h.GetParameterOrDefault(req, "subcategory", "")
		richCategory = h.GetParameterOrDefault(req, "rich_category", "")
		searchType   = h.GetParameterOrDefault(req, "search_type", "")
		fuzzinessStr = h.GetParameterOrDefault(req, "fuzziness", "3")
		// multimodal search: uploaded attachments (images/documents) whose
		// extracted text joins the query, comma-separated IDs
		attachmentsParam = h.GetParameterOrDefault(req, "attachments", "")
	)

	// Parse fuzziness
	var fuzziness = 3 // default to 3
	if fuzzinessStr != "" {
		parsed, err := strconv.Atoi(fuzzinessStr)
		if err == nil && parsed >= 0 && parsed <= 5 {
			fuzziness = parsed
		}
	}

	if searchType == "" {
		// no explicit strategy: the operator's configured default decides
		// (search_settings.search_type) — this is the switch that puts the
		// hybrid pipeline the studio evaluates in front of the app search
		// and the widget without URL surgery; keyword when unset
		searchType = common.AppConfig().SearchSettings.DefaultType()
	}

	reqUser := security.MustGetUserFromRequest(req)

	// multimodal search: enrich the query with attachment text BEFORE the
	// empty-query gate below so an image-only search (no typed query) still
	// searches — the enrichment output is what "the query" means from here on
	multimodalNote := ""
	if attachmentsParam != "" {
		query, multimodalNote = enrichQueryFromAttachments(req.Context(), query, attachmentsParam, reqUser.UserID)
	}

	query = util.CleanUserQuery(query)
	searchStarted := time.Now()

	//try to collect assistants
	if query != "" || h.GetParameter(req, "filter") != "" {
		integrationID := req.Header.Get(core.HeaderIntegrationID)

		result := elastic.SearchResponseWithMeta[core.Document]{}
		rerankNote := ""
		// note travels to the client in the Warning header: the semantic
		// leg is transparent about which route actually ran.
		note := ""
		// rewritten: the D8 query-rewrite leg fired for this search
		rewritten := false
		if searchType == "hybrid_rrf" {
			fused, fusedNote, rwApplied, err := h.queryWithRRF(req, query, datasource, integrationID, category, subcategory, richCategory, fuzziness)
			if err != nil {
				// with hybrid_rrf as the configured default, an engine
				// hiccup here would otherwise crash every plain search
				h.WriteError(w, fmt.Sprintf("hybrid search failed: %v", err), http.StatusInternalServerError)
				return
			}
			result = *fused
			rerankNote = fusedNote
			rewritten = rwApplied
		} else {
			builder, err := orm.NewQueryBuilderFromRequest(req)
			if err != nil {
				h.WriteError(w, fmt.Sprintf("invalid search filter: %v", err), http.StatusBadRequest)
				return
			}
			builder.EnableBodyBytes()
			stripHighlightWithoutQuery(builder, query)
			// ?tags= controlled-vocabulary facet filter (W4): the legs'
			// recall respects the same facets the UI offers
			applyTagsFilter(builder, h.GetParameterOrDefault(req, "tags", ""))
			// one interactive page can never drag unbounded rows out of
			// the engine — bulk export has dedicated APIs
			if reqSize := h.GetIntOrDefault(req, "size", 10); reqSize > maxSearchPageSize {
				builder.Size(maxSearchPageSize)
			}

			writeResult := func(resp *orm.SimpleResult) {
				util.MustFromJSONBytes(resp.Raw, &result)
			}

			if searchType == "semantic" || searchType == "hybrid" {
				from := h.GetIntOrDefault(req, "from", 0)
				if from < 0 {
					from = 0
				}
				size := h.GetIntOrDefault(req, "size", 10)
				if size < 1 {
					size = 10
				}
				if size > maxSearchPageSize {
					size = maxSearchPageSize
				}

				switch plan := planSemantic(req.Context()); plan.Route {
				case semanticRouteEngine:
					resp, err := QueryDocuments(req.Context(), builder, query, datasource, integrationID, category, subcategory, richCategory, searchType, fuzziness, nil)
					if err != nil {
						h.WriteError(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
						return
					}
					writeResult(resp)
				case semanticRouteClient:
					builder.From(0)
					builder.Size(rrfRecallWindow(from, size))
					reranked, clientNote, err := clientSemanticRecall(req.Context(), builder, query, datasource, integrationID, category, subcategory, richCategory, fuzziness)
					if err != nil {
						h.WriteError(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
						return
					}
					paginateHits(reranked, from, size)
					result = *reranked
					note = clientNote
				default:
					note = plan.Reason
					resp, err := QueryDocuments(req.Context(), builder, query, datasource, integrationID, category, subcategory, richCategory, "keyword", fuzziness, nil)
					if err != nil {
						h.WriteError(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
						return
					}
					writeResult(resp)
				}
			} else {
				resp, err := QueryDocuments(req.Context(), builder, query, datasource, integrationID, category, subcategory, richCategory, searchType, fuzziness, nil)
				if err != nil {
					h.WriteError(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
					return
				}
				writeResult(resp)
			}

		}

		// both branches may carry a note: the semantic plan (which route
		// actually ran), the rerank verdict (applied/degraded), and the
		// multimodal verdict (attachments actually searched or not)
		for _, n := range []string{note, rerankNote, multimodalNote} {
			if n != "" {
				w.Header().Add("Warning", n)
			}
		}

		// same-content copies collapse onto the highest-ranked hit with a
		// "N more copies" note (D1.5); deep cleanup stays in the dedup report
		result.Hits.Hits = foldDuplicateHits(result.Hits.Hits, dismissedPairsCached(req.Context()))

		docsSize := len(result.Hits.Hits)
		//update icon
		if docsSize > 0 {
			for i := range result.Hits.Hits {
				// wiki hits are curated pages: they carry their own article
				// URL and provenance, the document refinement would clobber both
				if result.Hits.Hits[i].Source.Source.ID == "wiki" {
					continue
				}
				RefineDocument(req.Context(), &result.Hits.Hits[i].Source)
			}
		}

		// the datasource facet labels its buckets via top_hits carrying the
		// indexed source.name — stale after a datasource rename, so refresh
		// it the same way the hits above are refreshed
		if agg, ok := result.Aggregations["source.id"]; ok {
			refreshDatasourceFacetLabels(req.Context(), agg)
		}

		size := h.GetIntOrDefault(req, "size", 10)
		if size > maxSearchPageSize {
			size = maxSearchPageSize
		}
		assistantSearchPermission := security.GetSimplePermission(Category, Assistant, string(QuickAISearchAction))
		perID := security.GetOrInitPermissionKey(assistantSearchPermission)

		//only for app search, not for widget integration or AI search portal
		if datasource == "" && integrationID == "" && ((reqUser.Roles != nil && util.AnyInArrayEquals(reqUser.Roles, security.RoleAdmin)) || reqUser.UserAssignedPermission.ValidateFor(perID)) {
			assistantSize := 2
			if docsSize < 5 {
				assistantSize = size - (docsSize)
			}

			assistants := searchAssistant(req, query, assistantSize)
			if len(assistants) > 0 {
				newHits := make([]elastic.DocumentWithMeta[core.Document], 0, len(assistants))
				for i, assistant := range assistants {
					doc := core.Document{}
					doc.ID = assistant.ID
					doc.Type = "AI Assistant"
					doc.Icon = assistant.Icon
					doc.Title = assistant.Name
					doc.Summary = assistant.Description
					doc.URL = fmt.Sprintf("coco://extenstions/infinilabs/ask_assistant/%v", assistant.ID)
					doc.Source = core.DataSourceReference{
						ID:   "assistant",
						Name: "Assistant",
						Icon: "font_robot",
					}
					newHit := elastic.DocumentWithMeta[core.Document]{
						ID:     assistant.ID,
						Index:  "assistant",
						Source: doc,
						Score:  result.Hits.MaxScore + float32(size-i),
					}
					newHits = append(newHits, newHit)
				}
				result.Hits.Hits = append(newHits, result.Hits.Hits...)
			}
		}

		// telemetry (P2/D5): what was asked, which strategy ran, what came
		// back and how long it took — the overview and the knowledge-gap
		// loop are built on it; async so the search never waits on it.
		// WithoutCancel: the goroutine outlives the request, a canceled
		// request context would kill the log write and the gap check
		go recordSearchLog(context.WithoutCancel(req.Context()), query, searchType, reqUser.UserID,
			result.GetTotal(), time.Since(searchStarted), rewritten)

		api.WriteJSON(w, result, 200)
	} else {
		h.WriteJSON(w, elastic.SearchResponse{Hits: elastic.Hits{Total: elastic.TotalHits{Value: 0, Relation: "eq"}}}, http.StatusOK)
	}
}

func RefineDocument(ctx context.Context, doc *core.Document) {
	RefineIcon(ctx, doc)
	RefineCoverThumbnail(ctx, doc)
	RefineURL(ctx, doc)
	RefineRawContentURL(ctx, doc)
}

// ResolveIcon runs the icon fallback chain:
// 1. currentIcon
// 2. datasource.Icon
// 3. connector.Icon
func ResolveIcon(
	connectorConfig *core.Connector,
	datasourceConfig *core.DataSource,
	currentIcon string,
) string {

	// 1. Try current field's icon
	if icon := common.ParseAndGetIcon(connectorConfig, currentIcon); icon != "" {
		return icon
	}

	// 2. Try datasource icon
	if datasourceConfig.Icon != "" {
		if icon := common.ParseAndGetIcon(connectorConfig, datasourceConfig.Icon); icon != "" {
			return icon
		}
	}

	// 3. Try connector default icon
	if icon := common.ParseAndGetIcon(connectorConfig, connectorConfig.Icon); icon != "" {
		return icon
	}

	return ""
}

// refreshDatasourceFacetLabels rewrites the top_hits-carried source.name of a
// "source.id" terms aggregation with the live datasource name. Buckets are
// keyed by datasource id, so each label is one cached config lookup.
func refreshDatasourceFacetLabels(ctx context.Context, agg elastic.AggregationResponse) {
	if len(agg.Buckets) == 0 {
		return
	}
	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectReadAccess()
	ctx1.PermissionScope(security.PermissionScopePlatform)

	for _, bucket := range agg.Buckets {
		datasourceID, ok := bucket["key"].(string)
		if !ok || datasourceID == "" {
			continue
		}
		datasourceConfig, err := common.GetDatasourceConfig(ctx1, datasourceID)
		if err != nil || datasourceConfig == nil || datasourceConfig.Name == "" {
			continue
		}
		rewriteBucketDatasourceName(bucket, datasourceConfig.Name)
	}
}

// rewriteBucketDatasourceName walks a terms bucket's top_hits payload and
// replaces the indexed source.name snapshot with the given live name. The
// payload is generic JSON, so every level is guarded; a bucket without the
// expected shape is left alone.
func rewriteBucketDatasourceName(bucket elastic.BucketBase, name string) {
	top, ok := bucket["top"].(map[string]interface{})
	if !ok {
		return
	}
	hitsObj, ok := top["hits"].(map[string]interface{})
	if !ok {
		return
	}
	hits, ok := hitsObj["hits"].([]interface{})
	if !ok || len(hits) == 0 {
		return
	}
	hit, ok := hits[0].(map[string]interface{})
	if !ok {
		return
	}
	source, ok := hit["_source"].(map[string]interface{})
	if !ok {
		return
	}
	if docSource, ok := source["source"].(map[string]interface{}); ok {
		docSource["name"] = name
	}
}

func RefineIcon(ctx context.Context, doc *core.Document) {
	ctx1 := orm.NewContextWithParent(ctx)
	ctx1.DirectReadAccess()
	ctx1.PermissionScope(security.PermissionScopePlatform)

	datasourceConfig, err := common.GetDatasourceConfig(ctx1, doc.Source.ID)
	if err != nil || datasourceConfig == nil {
		return
	}

	// The indexed source.name is an ingestion-time snapshot: after a rename,
	// every already-indexed document still carries the previous name and the
	// results list keeps showing it. Refresh from the live config (cached,
	// invalidation happens on datasource update) — same read-time pattern as
	// the connector fields below.
	if datasourceConfig.Name != "" {
		doc.Source.Name = datasourceConfig.Name
	}

	if datasourceConfig.Connector.ConnectorID == "" {
		return
	}

	connectorConfig, err := connector.GetConnectorConfig(datasourceConfig.Connector.ConnectorID)
	if err != nil || connectorConfig == nil {
		return
	}

	// Update doc.Icon
	if icon := ResolveIcon(connectorConfig, datasourceConfig, doc.Icon); icon != "" {
		doc.Icon = icon
	}

	// Update doc.Source.Icon
	if icon := ResolveIcon(connectorConfig, datasourceConfig, doc.Source.Icon); icon != "" {
		doc.Source.Icon = icon
	}

	// Stamp the connector reference (read-time only, never persisted): the
	// detail views render the datasource and its connector as filterable
	// entity cards, and both configs are already in hand here — no extra
	// lookups. Guarded so a doc that ever carries the values (e.g. stamped at
	// ingestion in the future) keeps its own.
	if doc.Source.ConnectorID == "" {
		doc.Source.ConnectorID = connectorConfig.ID
	}
	if doc.Source.ConnectorName == "" {
		doc.Source.ConnectorName = connectorConfig.Name
	}
	if doc.Source.ConnectorIcon == "" {
		if icon := common.ParseAndGetIcon(connectorConfig, connectorConfig.Icon); icon != "" {
			doc.Source.ConnectorIcon = icon
		}
	}
}

// RefineCoverThumbnail converts Cover and Thumbnail from "attachment://UUID"
// to full attachment URL for preview capability.
func RefineCoverThumbnail(ctx context.Context, doc *core.Document) {
	appCfg := common.AppConfig()
	baseEndpoint := appCfg.ServerInfo.Endpoint

	if doc.Cover != "" && strings.HasPrefix(doc.Cover, "attachment://") {
		uuid := strings.TrimPrefix(doc.Cover, "attachment://")
		relativePath := fmt.Sprintf("/attachment/%s", uuid)
		if fullURL, err := url.JoinPath(baseEndpoint, relativePath); err == nil {
			doc.Cover = fullURL
		}
	}
	if doc.Thumbnail != "" && strings.HasPrefix(doc.Thumbnail, "attachment://") {
		uuid := strings.TrimPrefix(doc.Thumbnail, "attachment://")
		relativePath := fmt.Sprintf("/attachment/%s", uuid)
		if fullURL, err := url.JoinPath(baseEndpoint, relativePath); err == nil {
			doc.Thumbnail = fullURL
		}
	}
}

// RefineURL converts [doc.URL] to [ENDPOINT/#/preview/document/DOC_ID]
func RefineURL(ctx context.Context, doc *core.Document) {
	appCfg := common.AppConfig()
	baseEndpoint := appCfg.ServerInfo.Endpoint

	doc.URL = fmt.Sprintf("%s/#/preview/document/%s", baseEndpoint, doc.ID)
}

// RefineRawContentURL adds a "raw_content" key to doc.Metadata with the
// full URL to the document's raw content endpoint.
func RefineRawContentURL(ctx context.Context, doc *core.Document) {
	appCfg := common.AppConfig()
	baseEndpoint := appCfg.ServerInfo.Endpoint

	rawContentURL := fmt.Sprintf("%s/document/%s/raw_content/%s", baseEndpoint, doc.ID, url.PathEscape(doc.Title))

	if doc.Metadata == nil {
		doc.Metadata = make(map[string]interface{})
	}
	doc.Metadata["raw_content"] = rawContentURL
}

func searchAssistant(req *http.Request, query string, size int) []core.Assistant {
	docs := []core.Assistant{}
	if size <= 0 {
		size = 2
	}

	//handle url query args, convert to query builder
	builder, err := orm.NewQueryBuilderFromRequest(req, "name^10", "name.pinyin^5", "combined_fulltext^1")
	if err != nil {
		return docs
	}
	builder.Query(query)
	builder.Must(orm.TermQuery("enabled", true))
	builder.Size(size)
	builder.Fuzziness(3)

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &core.Assistant{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "assistant")
	err, _ = elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	if err != nil {
		return docs
	}

	return docs
}

func BuildFilters(category string, subcategory string, richCategory string) []*orm.Clause {
	mustClauses := []*orm.Clause{}

	if category != "" {
		mustClauses = append(mustClauses, orm.TermQuery("category", category))
	}

	if subcategory != "" {
		mustClauses = append(mustClauses, orm.TermQuery("subcategory", subcategory))
	}

	if richCategory != "" {
		mustClauses = append(mustClauses, orm.TermQuery("rich_categories.key", richCategory))
	}

	return mustClauses
}

// GetDatasourceByIntegration returns the datasource IDs that the integration is allowed to access
func GetDatasourceByIntegration(integrationID string) ([]string, bool, error) {
	var items = []core.Integration{}
	q := orm.Query{
		Size:  1,
		Conds: orm.And(orm.Eq("id", integrationID), orm.Eq("enabled", true)),
	}
	err, _ := orm.SearchWithJSONMapper(&items, &q)
	if err != nil {
		return nil, false, err
	}
	if len(items) == 0 {
		return nil, false, nil
	}
	var ret = make([]string, 0, len(items))
	for _, item := range items {
		for _, datasourceID := range item.EnabledModule.Search.Datasource {
			if datasourceID == "*" {
				return nil, true, nil
			}
			ret = append(ret, datasourceID)
		}
	}
	return ret, false, nil
}

// BuildDatasourceFilter computes the final set of datasource IDs a user is allowed to search,
// by merging the user's own datasources, shared datasources, the query-requested datasources,
// and the integration-scoped datasources.
//
// Returns:
//   - checkingScopeDatasources: datasource IDs that require further document-level permission checks
//     (e.g. datasources shared at category level where individual documents may still be denied).
//   - finalDatasourceIDs: datasource IDs the user has full direct access to (user-owned + directly shared).
//   - disabledIDs: datasource IDs that are currently disabled and should be excluded from search results.
//
// If the user has no accessible datasources at all, all three slices are returned as nil.
func BuildDatasourceFilter(userID string, checkingScopeDatasources, directAccessDatasources []string, queryDatasourceIDs []string, integrationID string, filterDisabled bool) ([]string, []string, []string) {

	//merge user's own datasource, other shareable datasource, within user's query datasource, within integration's datasource

	//fist, merge the user's accessable datasource
	userOwnDatasourceIDs := common.GetUsersOwnDatasource(userID)
	directAccessDatasources = append(directAccessDatasources, userOwnDatasourceIDs...)

	log.Trace("userID:", userID, "user's own", userOwnDatasourceIDs, ",queryDatasource:", queryDatasourceIDs, ",integrationID:", integrationID, ",merged datasources:", directAccessDatasources)

	finalDatasourceIDs := directAccessDatasources
	if len(queryDatasourceIDs) > 0 && !util.ContainsAnyInArray("*", queryDatasourceIDs) {
		//only merge if the query are specify datasources
		finalDatasourceIDs = util.StringArrayIntersection(queryDatasourceIDs, finalDatasourceIDs)
		checkingScopeDatasources = util.StringArrayIntersection(queryDatasourceIDs, checkingScopeDatasources)
	}

	if integrationID != "" {
		// get queryDatasource by integration id
		datasourceIDs, hasAll, err := GetDatasourceByIntegration(integrationID)
		if err != nil {
			panic(err)
		}

		if len(datasourceIDs) == 0 {
			log.Warnf("empty datasource for integration: %v", integrationID)
		}

		log.Trace("integration:", integrationID, ", datasource:", datasourceIDs, ",has all:", hasAll)

		//finalDatasourceIDs = datasourceIDs
		if !hasAll {
			if len(datasourceIDs) > 0 {
				finalDatasourceIDs = util.StringArrayIntersection(datasourceIDs, finalDatasourceIDs)
				checkingScopeDatasources = util.StringArrayIntersection(datasourceIDs, checkingScopeDatasources)

				//log.Error("finalDatasourceIDs:", finalDatasourceIDs, ",checkingScopeDatasources", checkingScopeDatasources)

			}
		}
	}

	if len(finalDatasourceIDs) == 0 && len(checkingScopeDatasources) == 0 {
		return nil, nil, nil
	}

	log.Trace("userID:", userID, "user's own", userOwnDatasourceIDs, ",queryDatasource:", queryDatasourceIDs, ",integrationID:", integrationID, ",final merged directAccess datasources:", finalDatasourceIDs)

	if !filterDisabled {
		return checkingScopeDatasources, finalDatasourceIDs, []string{}
	}

	disabledIDs, err := common.GetDisabledDatasourceIDs()
	if err != nil {
		panic(err)
	}

	return checkingScopeDatasources, finalDatasourceIDs, disabledIDs
}
