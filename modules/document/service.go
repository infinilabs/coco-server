/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	orm2 "infini.sh/framework/plugins/enterprise/security/orm"
	"infini.sh/framework/plugins/enterprise/security/share"
	"strings"
)

func QueryDocuments(ctx1 context.Context, userID string, builder *orm.QueryBuilder, query string, datasource, integrationID, category, subcategory, richCategory string) ([]core.Document, *orm.SimpleResult, error) {

	log.Error("old datasource:", datasource, ",integrationID:", integrationID)

	builder.Query(query)
	builder.DefaultQueryField("title.keyword^100", "title^10", "title.pinyin^4", "combined_fulltext")
	builder.Exclude("payload.*")

	filters := BuildFilters(category, subcategory, richCategory)

	//resource level share rules, get other user's shared for this user //TODO, get parent datasource by doc level sharing rules
	//rules, err := sharingService.GetDirectResourceRulesByResourceTypeAndUserID(userID, "datasource", userOwnDatasourceIDs, share.View)
	rules, err := sharingService.GetDirectResourceRulesByResourceTypeAndUserID(userID, "datasource", nil, share.View)
	if err != nil {
		panic(err)
	}
	log.Error("rules: ", util.ToJson(rules, true))
	checkingScopeDatasources := []string{}
	fullAccessSharedDatasources := []string{}
	for _, rule := range rules {
		fullAccessSharedDatasources = append(fullAccessSharedDatasources, rule.ResourceID)
		checkingScopeDatasources = append(checkingScopeDatasources, rule.ResourceID)
	}
	log.Error("fullAccessSharedDatasources: ", fullAccessSharedDatasources)

	rules, err = sharingService.GetAllCategoryVisibleWithChildrenSharedObjects(userID, "datasource")
	for _, rule := range rules {
		checkingScopeDatasources = append(checkingScopeDatasources, rule.ResourceCategoryID)
	}
	log.Error("AGAIN checkingScopeDatasources: ", checkingScopeDatasources)

	//var mergedDatasourceIDS []string

	//get all directly assigned rules assign to document level
	queryDatasourceIDs := []string{}
	if datasource == "" || util.ContainStr(datasource, "*") {
		//return finalDatasourceIDs
		//queryDatasourceIDs
	} else {
		queryDatasourceIDs = strings.Split(datasource, ",")
	}

	//(user own datasource + shared datasource) intersect query datasource
	//if datasource != "" && !util.ContainStr(datasource, "*") {
	mergedDatasourceIDS, datasourceFilter := BuildDatasourceFilter(userID, checkingScopeDatasources, queryDatasourceIDs, integrationID, true)
	if datasourceFilter != nil {
		filters = append(filters, datasourceFilter...)
	}
	//mergedDatasourceIDS = ds1
	log.Error("AAA checkingScopeDatasources:", checkingScopeDatasources, "new mergedDatasourceIDS:", mergedDatasourceIDS, ",integrationID:", integrationID)
	//}

	//if len(mergedDatasourceIDS) == 0 {
	//	panic("no datasources allow to access")
	//}

	//filter enabled doc
	filters = append(filters, orm.BoolQuery(orm.Should, orm.TermQuery("disabled", false), orm.MustNotQuery(orm.ExistsQuery("disabled"))).Parameter("minimum_should_match", 1))

	builder.Filter(filters...)

	log.Error("queryDatasourceIDs:", queryDatasourceIDs)

	rules, err = sharingService.GetDirectResourceRulesByResourceCategoryAndUserID(userID, "document", "datasource", checkingScopeDatasources, share.None)
	if err != nil {
		panic(err)
	}
	log.Error("doc rules:", util.ToJson(rules, true))
	allowdDocs := []string{}
	deniedDocs := []string{}
	for _, rule := range rules {
		if rule.Permission > share.None {
			allowdDocs = append(allowdDocs, rule.ResourceID)
		} else {
			deniedDocs = append(deniedDocs, rule.ResourceID)
		}
	}

	shouldClauses := []*orm.Clause{}
	shouldClauses = append(shouldClauses, orm.TermQuery(orm2.SystemOwnerQueryField, userID)) //user is the owner

	if len(fullAccessSharedDatasources) > 0 {
		shouldClauses = append(shouldClauses, orm.TermsQuery("source.id", fullAccessSharedDatasources)) //shared with user
	}

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
	//ctx.Set(orm.SharingEnabled, true)
	//ctx.Set(orm.SharingResourceType, "document")
	//ctx.Set(orm.SharingCheckingResourceCategoryEnabled, true)
	//ctx.Set(orm.SharingCheckingInheritedRulesEnabled, true)

	//if datasource != "" {
	//	ctx.Set(orm.SharingResourceCategoryType, "datasource")
	//	ctx.Set(orm.SharingResourceCategoryFilterField, "source.id")
	//	ctx.Set(orm.SharingResourceCategoryID, datasource)
	//}

	log.Error(builder.ToString())

	docs := []core.Document{}
	err, resp := elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	if err != nil || resp == nil {
		return nil, nil, err
	}

	return docs, resp, nil
}

func QueryDocumentsAndFilter(ctx1 context.Context, user *security.UserSessionInfo, builder *orm.QueryBuilder, integrationID string, query string, datasource, category, subcategory, richCategory string) ([]elastic.DocumentWithMeta[core.Document], *orm.SimpleResult, error) {
	docs, resp, err := QueryDocuments(ctx1, user.MustGetUserID(), builder, query, datasource, integrationID, category, subcategory, richCategory)
	if err != nil {
		return nil, nil, err
	}

	////bypass admin
	//if !util.AnyInArrayEquals(user.Roles, security.RoleAdmin) {
	//	docs, err = SecondStageFilter(ctx1, user.MustGetUserID(), query, docs)
	//	if err != nil {
	//		return nil, nil, err
	//	}
	//}

	var final []elastic.DocumentWithMeta[core.Document]
	for _, v := range docs {
		doc := elastic.DocumentWithMeta[core.Document]{
			ID:     v.ID,
			Source: v,
		}
		RefineIcon(ctx1, &v)
		final = append(final, doc)
	}

	return final, resp, nil
}

func SecondStageFilter(ctx1 context.Context, userID string, query string, docs []core.Document) ([]core.Document, error) {
	//double check the document permission
	var docs1 = []core.Document{}
	for _, v := range docs {
		valid := false
		if v.GetOwnerID() == userID {
			valid = true
		} else {
			shareEntity := share.NewResourceEntity("document", v.ID, "")
			shareEntity.ResourceCategoryType = "datasource"
			shareEntity.ResourceCategoryID = v.Source.ID
			shareEntity.ResourceParentPath = v.Category
			per, _ := sharingService.GetUserExplicitEffectivePermission(userID, shareEntity)
			if per > share.None {
				valid = true
			}
		}

		if valid {
			docs1 = append(docs1, v)
		} else {
			log.Info("invalid permission to access doc:", v.Title, ",", query)
		}
	}
	log.Infof("hit %v->%v docs for query: %v", len(docs), len(docs1), query)
	return docs1, nil
}

var sharingService = share.NewSharingService()

//
//func QueryForMultiDatasource(ctx *orm.Context, user *security.UserSessionInfo, qb *orm.QueryBuilder, integrationID string, query string, datasource, category, subcategory, richCategory string)  {
//	if ctx == nil {
//		panic("invalid data access")
//	}
//
//	sessionUser := security.MustGetUserFromContext(ctx.Context)
//	userID := sessionUser.MustGetUserID()
//
//	var bq *orm.Clause
//
//	bq = orm.ShouldQuery(
//		//orm.MustNotQuery(orm.ExistsQuery(SystemFieldsKey + "." + OwnerIDKey)),
//	)
//
//	var globalShareMustFilters = []*orm.Clause{}
//	////apply sharing rules
//	if ctx.GetBool(orm.SharingEnabled, false) {
//
//		//log.Error("hit SharingEnabled")
//		resourceType := ctx.MustGetString(orm.SharingResourceType)
//
//		//var bypassByCategoryFilter = false //TODO support multi category filter and bypass
//		//check category level filter first!
//		//apply parent sharing rules, like if the parent object is shared, eg: datasource level, all docs will be marked as shared
//		if ctx.GetBool(orm.SharingCheckingResourceCategoryEnabled, false) {
//			//log.Error("hit SharingCheckingResourceCategoryEnabled")
//			resourceCategoryType := "datasource"
//			resourceCategoryID := ctx.MustGetString(orm.SharingResourceCategoryID)
//			filterField := ctx.MustGetString(orm.SharingResourceCategoryFilterField)
//
//			ids:=strings.Split(resourceCategoryID,",")
//			if resourceCategoryID=="*" || util.ContainStr(resourceCategoryID,"*"){
//				ids=common.GetUserDatasource(ctx) //TODO, should reflect sharing rules
//				log.Error("hit * by datasource ids:",ids)
//			}
//
//			log.Error("final ids:",ids)
//
//			boolQuery:=orm.BooleanQuery()
//
//			for _,resourceID:=range ids{
//				globalShareMustFilters = append(globalShareMustFilters, orm.TermQuery("resource_category_type", resourceCategoryType))
//				globalShareMustFilters = append(globalShareMustFilters, orm.TermQuery("resource_category_id", resourceID))
//
//				//check if the current user have access to this resource
//				//log.Error("check if the current user have access to this resource")
//				perm, err := sharingService.GetUserExplicitEffectivePermission(userID, share.NewResourceEntity(resourceCategoryType, resourceID, ""))
//				//log.Error("user have access to this parent object", perm, err)
//				if err == nil {
//					//TODO, not permission, just 403
//					//self or not inherit any permission, we should throw a permission error
//					if perm >= 1 {
//						//bypassByCategoryFilter = true
//						categoryFilter := orm.TermQuery(filterField, resourceID)
//						bq.MustClauses = append(bq.MustClauses, categoryFilter)
//					}
//				}
//				globalShareMustFilters=append(globalShareMustFilters,boolQuery)
//			}
//		}else{
//			//for none-documents search
//			ids, err := sharingService.GetResourceIDsByResourceTypeAndUserID(userID, resourceType)
//			//log.Error("user have access to this parent object", ids, err)
//			if err == nil {
//				//TODO, not permission, just 403
//				//self or not inherit any permission, we should throw a permission error
//				if len(ids) >= 1 {
//					bq.ShouldClauses = append(bq.ShouldClauses, orm.TermsQuery("id", ids))
//				}
//			}
//		}
//
//		//only enable this for documents search
//		if ctx.GetBool(orm.SharingCheckingInheritedRulesEnabled, false){
//				var rules []share.SharingRecord
//				rules, _ = share.GetSharingRules(rbac.PrincipalTypeUser, userID, resourceType, "", "", globalShareMustFilters)
//				log.Error("get all shared rules: ", resourceType, " => ", util.MustToJSON(rules))
//
//				if len(rules) > 0 {
//					allowedIDs := []string{}
//					allowedFolderPaths := []string{}
//					deniedIDs := []string{}
//					deniedFolderPaths := []string{}
//
//					for _, v := range rules {
//						switch {
//						// ✅ Allow rules
//						case v.Permission > share.None:
//							allowedIDs = append(allowedIDs, v.ResourceID)
//							if v.ResourceIsFolder {
//								allowedFolderPaths = append(allowedFolderPaths, v.ResourceFullPath)
//							}
//
//						// ❌ Deny rules
//						case v.Permission == share.None:
//							if v.ResourceIsFolder {
//								deniedFolderPaths = append(deniedFolderPaths, v.ResourceFullPath)
//							} else {
//								deniedIDs = append(deniedIDs, v.ResourceID)
//							}
//
//						default:
//							log.Error("invalid permission rule: ", util.ToJson(v, true))
//						}
//					}
//
//					// --- Build final boolean query ---
//					// ✅ allow items or folders
//					shouldClauses := []*orm.Clause{}
//					if len(allowedIDs) > 0 {
//						shouldClauses = append(shouldClauses, orm.TermsQuery("id", allowedIDs))
//					}
//					for _, path := range allowedFolderPaths {
//						shouldClauses = append(shouldClauses, orm.PrefixQuery("_system.parent_path", path))
//					}
//					if len(shouldClauses) > 0 {
//						bq.ShouldClauses = append(bq.ShouldClauses, shouldClauses...)
//					}
//
//					// ❌ deny rules
//					mustNotClauses := []*orm.Clause{}
//					if len(deniedIDs) > 0 {
//						mustNotClauses = append(mustNotClauses, orm.TermsQuery("id", deniedIDs))
//					}
//					for _, path := range deniedFolderPaths {
//						// exclude docs under this path except explicitly allowed IDs
//						folderExclude := orm.BooleanQuery()
//						folderExclude.MustClauses = append(folderExclude.MustClauses,
//							orm.PrefixQuery("_system.parent_path", path))
//						if len(allowedIDs) > 0 {
//							folderExclude.MustNotClauses = append(folderExclude.MustNotClauses,
//								orm.TermsQuery("id", allowedIDs))
//						}
//						mustNotClauses = append(mustNotClauses, folderExclude)
//					}
//
//					if len(mustNotClauses) > 0 {
//						bq.MustNotClauses = append(bq.MustNotClauses, mustNotClauses...)
//					}
//				}
//
//		}
//	}
//
//	bq.ShouldClauses = append(bq.ShouldClauses, orm.TermQuery(orm2.SystemOwnerQueryField, userID))
//
//	if len(bq.ShouldClauses) > 1 {
//		bq.Parameter("minimum_should_match", 1)
//	}
//
//	//	}
//	//}
//
//	if bq != nil {
//		qb.Must(bq)
//	} else {
//		qb.Filter(orm.MustQuery(orm.TermQuery(orm2.SystemOwnerQueryField, userID)))
//	}
//}
