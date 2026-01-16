/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"strings"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/security/share"
)

var sharingService = share.NewSharingService()

func QueryDocuments(ctx1 context.Context, userID string, teamsID []string, builder *orm.QueryBuilder, query string, datasource, integrationID, category, subcategory, richCategory string, outputDocs *[]core.Document) (*orm.SimpleResult, error) {

	log.Trace("old datasource:", datasource, ",integrationID:", integrationID)

	builder.Query(query)
	builder.DefaultQueryField("title.keyword^100", "title^10", "title.pinyin^4", "combined_fulltext")
	// Omit these fields. The frontend does not need them, and they are large enough
	// to slow us down.
	builder.Exclude("payload.*", "document_chunk", "ai_insights.embedding")

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
	if len(disabledIDs) > 0 {
		filters = append(filters, orm.MustNotQuery(orm.TermsQuery("source.id", disabledIDs)))
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

	orm.WithModel(ctx, &core.Document{})
	log.Trace(builder.ToString())

	err, resp := elastic.SearchV2WithResultItemMapper(ctx, outputDocs, builder, nil)
	if err != nil || resp == nil {
		return nil, err
	}

	return resp, nil
}
