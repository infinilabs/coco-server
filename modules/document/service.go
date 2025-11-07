/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"github.com/cihub/seelog"
	log "github.com/cihub/seelog"
	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	"infini.sh/framework/plugins/enterprise/security/share"
	"strings"
)

func QueryDocuments(ctx1 context.Context, builder *orm.QueryBuilder, query string, datasource, category, subcategory, richCategory string) ([]common.Document, int64, error) {

	builder.Query(query)
	builder.DefaultQueryField("title.keyword^100", "title^10", "title.pinyin^4", "combined_fulltext")
	builder.Exclude("payload.*")

	filters := BuildFilters(category, subcategory, richCategory)

	datasourceFilter := BuildDatasourceFilter(datasource, true)
	if datasourceFilter != nil {
		filters = append(filters, datasourceFilter...)
	}

	//filter enabled doc
	filters = append(filters, orm.BoolQuery(orm.Should, orm.TermQuery("disabled", false), orm.MustNotQuery(orm.ExistsQuery("disabled"))).Parameter("minimum_should_match", 1))

	builder.Filter(filters...)

	ctx := orm.NewContextWithParent(ctx1)
	orm.WithModel(ctx, &common.Document{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")
	ctx.Set(orm.SharingCheckingInheritedRulesEnabled, true)

	docs := []common.Document{}
	err, resp := elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	if err != nil || resp == nil {
		return nil, 0, err
	}

	return docs, resp.Total, nil
}

func QueryDocumentsAndFilter(ctx1 context.Context, userID string, builder *orm.QueryBuilder, integrationID string, query string, datasource, category, subcategory, richCategory string) ([]elastic.DocumentWithMeta[common.Document], int64, error) {

	if integrationID != "" {
		cfg, _ := common.InternalGetIntegration(integrationID)
		if cfg != nil {
			if cfg.Guest.Enabled && cfg.Guest.RunAs != "" {
				ctx1 = orm.NewContextWithParent(security.RunAs(ctx1, cfg.Guest.RunAs))
				userID = cfg.Guest.RunAs
				log.Error("run as: ", cfg.Guest.RunAs, ",old datasource:", datasource)
			}

			// get datasource by integration id
			datasourceIDs, hasAll, err := common.GetDatasourceByIntegration1(cfg)
			if err != nil {
				return nil, 0, err
			}

			if !hasAll {
				if len(datasourceIDs) == 0 {
					return nil, 0, err
				}
				// update datasource filter
				if datasource == "" {
					datasource = strings.Join(datasourceIDs, ",")
				} else {
					// calc intersection with datasource and datasourceIDs
					queryDatasource := strings.Split(datasource, ",")
					queryDatasource = util.StringArrayIntersection(queryDatasource, datasourceIDs)
					if len(queryDatasource) == 0 {
						return nil, 0, err
					}
					datasource = strings.Join(queryDatasource, ",")
					log.Error("run as: ", cfg.Guest.RunAs, ", new datasource:", datasource)
				}
			}
		}
	}

	docs, total, err := QueryDocuments(ctx1, builder, query, datasource, category, subcategory, richCategory)
	if err != nil {
		return nil, 0, err
	}
	docs1, err := SecondStageFilter(ctx1, userID, query, docs)
	if err != nil {
		return nil, 0, err
	}
	return docs1, total, nil
}

func SecondStageFilter(ctx1 context.Context, userID string, query string, docs []common.Document) ([]elastic.DocumentWithMeta[common.Document], error) {
	//double check the document permission
	var docs1 []elastic.DocumentWithMeta[common.Document]
	for _, v := range docs {

		valid := false
		if v.GetOwnerID() == userID {
			seelog.Info("the user is owner")
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
			doc := elastic.DocumentWithMeta[common.Document]{
				ID:     v.ID,
				Source: v,
			}
			RefineIcon(ctx1, &v)
			docs1 = append(docs1, doc)
		} else {
			seelog.Info("invalid permission to access doc:", v.Title, ",", query)
		}
	}
	seelog.Infof("hit %v->%v docs for query: %v", len(docs), len(docs1), query)
	return docs1, nil
}

var sharingService = share.NewSharingService()
