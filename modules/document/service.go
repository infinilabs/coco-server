/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/plugins/enterprise/security/share"
)

func QueryDocuments(ctx1 context.Context, builder *orm.QueryBuilder, query string, datasource, category, subcategory, richCategory string) ([]core.Document, int64, error) {

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
	orm.WithModel(ctx, &core.Document{})
	ctx.Set(orm.SharingEnabled, true)
	ctx.Set(orm.SharingResourceType, "document")
	ctx.Set(orm.SharingCheckingInheritedRulesEnabled, true)

	docs := []core.Document{}
	err, resp := elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	if err != nil || resp == nil {
		return nil, 0, err
	}

	return docs, resp.Total, nil
}

func QueryDocumentsAndFilter(ctx1 context.Context, userID string, builder *orm.QueryBuilder, integrationID string, query string, datasource, category, subcategory, richCategory string) ([]elastic.DocumentWithMeta[core.Document], int64, error) {
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

func SecondStageFilter(ctx1 context.Context, userID string, query string, docs []core.Document) ([]elastic.DocumentWithMeta[core.Document], error) {
	//double check the document permission
	var docs1 []elastic.DocumentWithMeta[core.Document]
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
			doc := elastic.DocumentWithMeta[core.Document]{
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
