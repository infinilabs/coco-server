/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/security/share"
)

var sharingService = share.NewSharingService()

// helper function to build the Elasticsearch field path for the document-level
// embedding used by semantic search. It must stay in sync with the
// RequiredEmbeddingDimension constant.
func documentEmbeddingField() string {
	return "ai_insights.embedding.embedding" + strconv.Itoa(core.RequiredEmbeddingDimension)
}

//func QueryDocuments(ctx1 context.Context, builder *orm.QueryBuilder, query string, datasource, integrationID, category, subcategory, richCategory string, outputDocs *[]core.Document) (*orm.SimpleResult, error) {
//	filters := BuildFilters(category, subcategory, richCategory)
//	builder.Filter(filters...)
//	return InternalQueryDocuments(ctx1, builder, query, datasource, integrationID, outputDocs)
//}

// searchSettingsSnapshot is the operator-configured retrieval strategy
// (search_settings): field weights, default sort, min score, match operator
// and minimum_should_match. All accessors are nil-safe; unconfigured values
// fall back to the historic behavior.
// searchSettingsSnapshot is the operator-configured retrieval strategy
// (indirection kept so tests can swap the source).
var searchSettingsSnapshotFn = func() *core.SearchSettings {
	return common.AppConfig().SearchSettings
}

func searchSettingsSnapshot() *core.SearchSettings {
	return searchSettingsSnapshotFn()
}

// applyMatchSettings pushes the operator-configured match behavior onto the
// keyword text clauses: the AND/OR operator on every leaf match, and the raw
// ES minimum_should_match spec on the clause that owns the should logic
// (the wrapper bool when several clauses, the match itself when one).
func applyMatchSettings(cfg *core.SearchSettings, clauses []*orm.Clause) {
	if cfg == nil || len(clauses) == 0 {
		return
	}
	if cfg.MatchOperatorVal() == "AND" {
		for _, c := range clauses {
			if c.IsLeaf() && c.Operator == orm.QueryMatch {
				c.Parameter("operator", "AND")
			}
		}
	}
	msm := strings.TrimSpace(cfg.MinimumShouldMatch)
	if msm == "" {
		return
	}
	if len(clauses) == 1 {
		clauses[0].Parameter("minimum_should_match", msm)
	}
	// >1 clauses: the caller wraps them into a should bool — msm is applied
	// on that wrapper (see applyTextClause below).
}

// applyTextClause attaches msm to the wrapper should clause.
func applyTextClause(cfg *core.SearchSettings, clause *orm.Clause, multi bool) {
	if cfg == nil || clause == nil {
		return
	}
	if msm := strings.TrimSpace(cfg.MinimumShouldMatch); msm != "" && multi {
		clause.Parameter("minimum_should_match", msm)
	}
}

func QueryDocuments(ctx1 context.Context, builder *orm.QueryBuilder, query string,
	datasource, integrationID, category, subcategory, richCategory, searchType string, fuzziness int,
	outputDocs *[]core.Document) (*orm.SimpleResult, error) {
	log.Trace("old datasource:", datasource, ",integrationID:", integrationID)

	reqUser := security.MustGetUserFromContext(ctx1)
	userID := reqUser.MustGetUserID()
	teamsID, _ := reqUser.GetStringArray(orm.TeamsIDKey)

	// Operator-configured retrieval strategy overrides the built-in field
	// boosts when present (search_settings.field_weights).
	searchCfg := searchSettingsSnapshot()
	defaultFields := searchCfg.TextFieldWeights()
	if len(defaultFields) == 0 {
		defaultFields = []string{"title.keyword^100", "title^10", "title.pinyin^4", "combined_fulltext"}
	}

	builder.Query(query)
	builder.DefaultQueryField(defaultFields...)
	// Omit these fields. The frontend does not need them, and they are large enough
	// to slow us down. Role-based field restrictions stack on top (field_access.go).
	builder.Exclude(documentSourceExcludes(reqUser.Roles)...)
	// 搜索设置 · 结果呈现: the operator's result-field whitelist trims every
	// hit's _source (ES _source.includes) on top of the role excludes.
	if incs := searchCfg.ResultSourceIncludes; len(incs) > 0 {
		builder.Include(incs...)
	}
	// Let framework skip the buildFuzzinessQuery() call as we did it here.
	builder.SkipFuzziness()

	semanticEmbeddingField := documentEmbeddingField()

	// Self-healing semantic leg: when the engine cannot run the semantic
	// query (typically: no engine-side embedding service), degrade to
	// keyword recall here so every caller keeps working. The HTTP handlers
	// additionally run the client-side cosine rerank route on top of this
	// recall (semantic_client.go); this guard is the safety net for callers
	// that go straight to QueryDocuments.
	effectiveSearchType := searchType
	if (searchType == "semantic" || searchType == "hybrid") && !engineSemanticReady(ctx1) {
		log.Warnf("search: engine semantic query unavailable (%s), %s search degrades to keyword", engineVectorCapability(ctx1).Reason, searchType)
		effectiveSearchType = "keyword"
	}

	/*
		Search type support:
	*/
	// Modify the query based on search_type
	switch effectiveSearchType {
	case "semantic":
		semanticClause := orm.SemanticQuery(semanticEmbeddingField, query, 0, "")
		builder.Must(semanticClause)
	case "hybrid":
		textClauses, err := orm.BuildFuzzinessQueryClauses(query, fuzziness, defaultFields)
		if err != nil {
			return nil, err
		}
		applyMatchSettings(searchCfg, textClauses)
		var textClause *orm.Clause
		if len(textClauses) == 1 {
			textClause = textClauses[0]
		} else {
			textClause = orm.ShouldQuery(textClauses...)
			applyTextClause(searchCfg, textClause, true)
		}

		// Semantic clause on ai_insights.embedding
		semanticClause := orm.SemanticQuery(semanticEmbeddingField, query, 0, "")

		// Combine with HybridQuery
		hybridClause := orm.HybridQuery(textClause, semanticClause)
		builder.Must(hybridClause)
	case "keyword":
		// empty query = pure filter query (the pinned-rule fetch rides this:
		// curated docs must surface regardless of text match — only the
		// permission / datasource / blocked filters gate them). Callers that
		// search real text always pass a non-empty query.
		if strings.TrimSpace(query) != "" {
			textClauses, err := orm.BuildFuzzinessQueryClauses(query, fuzziness, defaultFields)
			if err != nil {
				return nil, err
			}
			applyMatchSettings(searchCfg, textClauses)
			if len(textClauses) == 1 {
				builder.Must(textClauses[0])
			} else {
				wrapper := orm.ShouldQuery(textClauses...)
				applyTextClause(searchCfg, wrapper, true)
				builder.Must(wrapper)
			}
		}
	default:
		return nil, fmt.Errorf("invalid search_type: %s, must be one of: semantic, hybrid, keyword", searchType)
	}

	filters := BuildFilters(category, subcategory, richCategory)

	rules, err := sharingService.GetDirectResourceRulesByResourceTypeAndUserID(userID, teamsID, "datasource", nil, share.View)
	if err != nil {
		panic(err)
	}
	log.Trace("rules: ", util.ToJson(rules, true))

	directAccessDatasources := []string{}
	checkingScopeDatasources := []string{}
	for _, rule := range rules {
		checkingScopeDatasources = append(checkingScopeDatasources, rule.ResourceID)
		directAccessDatasources = append(directAccessDatasources, rule.ResourceID)
	}

	rules, err = sharingService.GetAllCategoryVisibleWithChildrenSharedObjects(userID, teamsID, "datasource")
	if err != nil {
		panic(err)
	}

	for _, rule := range rules {
		checkingScopeDatasources = append(checkingScopeDatasources, rule.ResourceCategoryID)
	}
	//get all directly assigned rules assign to document level
	queryDatasourceIDs := []string{}
	if datasource == "" || util.ContainStr(datasource, "*") {
	} else {
		queryDatasourceIDs = strings.Split(datasource, ",")
	}

	//(user own datasource + shared datasource) intersect query datasource
	checkingScopeDatasources, mergedFullAccessDatasourceIDS, disabledIDs := BuildDatasourceFilter(userID, checkingScopeDatasources, directAccessDatasources, queryDatasourceIDs, integrationID, true)

	// User has no accessible datasources, return empty result directly to avoid unfiltered search.
	if len(checkingScopeDatasources) == 0 && len(mergedFullAccessDatasourceIDS) == 0 {
		return &orm.SimpleResult{Raw: []byte(`{"hits":{"total":{"value":0,"relation":"eq"},"hits":[]}}`), Total: 0}, nil
	}

	if len(disabledIDs) > 0 {
		filters = append(filters, orm.MustNotQuery(orm.TermsQuery("source.id", disabledIDs)))
	}

	// 搜索设置 · 结果干预: operator-blocked documents/datasources never
	// surface in any leg that runs through here (keyword/semantic/hybrid,
	// every hybrid_rrf route, the search-box suggest)
	if len(searchCfg.BlockedDocIDs) > 0 {
		filters = append(filters, orm.MustNotQuery(orm.TermsQuery("id", searchCfg.BlockedDocIDs)))
	}
	if len(searchCfg.BlockedDatasourceIDs) > 0 {
		filters = append(filters, orm.MustNotQuery(orm.TermsQuery("source.id", searchCfg.BlockedDatasourceIDs)))
	}

	log.Trace("CheckingScopeDatasources:", checkingScopeDatasources, ",directAccessDatasources:", directAccessDatasources, ",queryDatasourceIDs:", queryDatasourceIDs, ",new mergedFullAccessDatasourceIDS:", mergedFullAccessDatasourceIDS, ",disabledIDs:", disabledIDs, ",integrationID:", integrationID)

	shouldClauses := []*orm.Clause{}

	if len(mergedFullAccessDatasourceIDS) > 0 {
		//make sure checking scope include user's own datasource ids
		checkingScopeDatasources = append(checkingScopeDatasources, mergedFullAccessDatasourceIDS...)
		shouldClauses = append(shouldClauses, orm.TermsQuery("source.id", mergedFullAccessDatasourceIDS)) //shared with user
	}

	if len(checkingScopeDatasources) > 0 {
		filters = append(filters, orm.MustQuery(orm.TermsQuery("source.id", checkingScopeDatasources)))
	}

	//filter enabled doc
	filters = append(filters, orm.BoolQuery(orm.Should, orm.TermQuery("disabled", false), orm.MustNotQuery(orm.ExistsQuery("disabled"))).Parameter("minimum_should_match", 1))

	if effectiveSearchType == "semantic" || effectiveSearchType == "hybrid" {
		filters = append(filters, orm.ExistsQuery(semanticEmbeddingField))
	}

	builder.Filter(filters...)

	rules, err = sharingService.GetDirectResourceRulesByResourceCategoryAndUserID(userID, teamsID, "document", "datasource", checkingScopeDatasources, share.None)
	if err != nil {
		panic(err)
	}
	log.Trace("doc rules:", util.ToJson(rules, true))
	allowdDocs := []string{}
	deniedDocs := []string{}
	for _, rule := range rules {
		if rule.Permission > share.None {
			allowdDocs = append(allowdDocs, rule.ResourceID)
		} else {
			deniedDocs = append(deniedDocs, rule.ResourceID)
		}
	}

	shouldClauses = append(shouldClauses, orm.TermQuery(core.SystemOwnerQueryField, userID)) //user is the owner

	if len(allowdDocs) > 0 {
		shouldClauses = append(shouldClauses, orm.TermsQuery("id", allowdDocs)) //direct shared with user
	}

	builder.Must(orm.BoolQuery(orm.Should, shouldClauses...).Parameter("minimum_should_match", 1))

	if len(deniedDocs) > 0 {
		builder.Must(orm.BoolQuery(orm.MustNot, orm.TermsQuery("id", deniedDocs)))
	}

	ctx := orm.NewContextWithParent(ctx1)
	ctx.DirectReadAccess()

	// Retrieval strategy knobs that ride on the final query body. hybrid_rrf
	// never reaches QueryDocuments (it runs through the engine search
	// pipeline), but guard anyway so a future caller can't corrupt fused
	// ranking with a BM25 sort or score cutoff.
	if effectiveSearchType != "hybrid_rrf" {
		// 搜索设置 · 检索策略/语义与AI: min_score is a search-body-level
		// parameter — builder.Parameter would render it INSIDE the bool query
		// and ES rejects that with 400 unknown field [min_score]; use
		// BodyParameter so it sits next to query/size at the body root. The
		// semantic threshold rides the same key — cosine scores are far
		// smaller than BM25, so the larger of the two wins
		ms := searchCfg.MinScore
		if (effectiveSearchType == "semantic" || effectiveSearchType == "hybrid") && searchCfg.SemanticMinScore > ms {
			ms = searchCfg.SemanticMinScore
		}
		if ms > 0 {
			builder.BodyParameter("min_score", ms)
		}
		// The operator's default sort only applies when the client didn't
		// pass an explicit sort (the request args parser fills Sorts()).
		if field, order, ok := searchCfg.DefaultSortSpec(); ok && len(builder.Sorts()) == 0 {
			builder.SortBy(orm.Sort{Field: field, SortType: orm.SortType(order)})
		}
	}

	// The engine-side semantic leg runs through the Coco-managed search
	// pipeline (query enrichment + RRF fusion); attach it whenever that leg
	// is actually taken. 搜索设置 · 高级: the operator's per-query engine
	// timeout rides the same URL args (ES ?timeout=) so a slow query gives
	// up early instead of pinning the shard.
	extraArgs := []util.KV{}
	if searchCfg.QueryTimeoutMS > 0 {
		extraArgs = append(extraArgs, util.KV{Key: "timeout", Value: fmt.Sprintf("%dms", searchCfg.QueryTimeoutMS)})
	}
	if effectiveSearchType == "semantic" || effectiveSearchType == "hybrid" {
		if name := engineSearchPipelineName(); name != "" {
			extraArgs = append(extraArgs, util.KV{Key: "search_pipeline", Value: name})
		}
	}
	if len(extraArgs) > 0 {
		orm.WithQueryArgs(ctx, &extraArgs)
	}

	orm.WithModel(ctx, &core.Document{})
	log.Trace(builder.ToString())

	err, resp := elastic.SearchV2WithResultItemMapper(ctx, outputDocs, builder, nil)
	if err != nil || resp == nil {
		return nil, err
	}

	return resp, nil
}
