/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package search

import (
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
	"infini.sh/framework/core/util"
	ccache "infini.sh/framework/lib/cache"
	"net/http"
	"strings"
	"time"
)

// MatchQuery represents a match query in Elasticsearch
type MatchQuery struct {
	Field string `json:"field"`
	Query string `json:"query"`
}

// TermQuery represents a term query in Elasticsearch
type TermQuery struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type QueryStringQuery struct {
	Field string `json:"field"`
	Value string `json:"query"`
}

// RangeQuery represents a range query in Elasticsearch
type RangeQuery struct {
	Field string `json:"field"`
	GTE   *int   `json:"gte,omitempty"` // Greater than or equal
	LTE   *int   `json:"lte,omitempty"` // Less than or equal
}

// BoolQuery supports combining multiple queries
type BoolQuery struct {
	Must    []interface{} `json:"must,omitempty"`
	Should  []interface{} `json:"should,omitempty"`
	MustNot []interface{} `json:"must_not,omitempty"`
	Filter  []interface{} `json:"filter,omitempty"`
}

// Aggregation represents a basic metric aggregation
type Aggregation struct {
	Field string `json:"field"`
}

// SortOption defines sorting for search results
type SortOption struct {
	Field string `json:"field"`
	Order string `json:"order"` // "asc" or "desc"
}

// TotalHits represents the total number of hits in the search response
type TotalHits struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"` // "eq" (exact) or "gte" (greater than or equal)
}

// IndexDocument used to construct indexing document
type IndexDocument struct {
	Index     string                   `json:"_index,omitempty"`
	Type      string                   `json:"_type,omitempty"`
	ID        string                   `json:"_id,omitempty"`
	Routing   string                   `json:"_routing,omitempty"`
	Score     float32                  `json:"_score,omitempty"`
	Source    common.Document          `json:"_source,omitempty"`
	Highlight map[string][]interface{} `json:"highlight,omitempty"`
}

type SearchResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     struct {
		Total    interface{}     `json:"total"`
		MaxScore float32         `json:"max_score"`
		Hits     []IndexDocument `json:"hits,omitempty"`
	} `json:"hits"`
	//Aggregations map[string]AggregationResponse `json:"aggregations,omitempty"`
}

var configCache = ccache.Layered(ccache.Configure().MaxSize(10000).ItemsToPrune(100))

func (h APIHandler) search(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	reqUser := security.MustGetUserFromRequest(req)
	log.Error(util.MustToJSON(reqUser))

	var (
		query = h.GetParameterOrDefault(req, "query", "")

		//TODO tobe removed
		datasource = h.GetParameterOrDefault(req, "datasource", "")
		category   = h.GetParameterOrDefault(req, "category", "")

		//TODO tobe removed
		username = h.GetParameterOrDefault(req, "username", "")
		userid   = h.GetParameterOrDefault(req, "userid", "")

		tags         = h.GetParameterOrDefault(req, "tags", "")
		subcategory  = h.GetParameterOrDefault(req, "subcategory", "")
		richCategory = h.GetParameterOrDefault(req, "rich_category", "")

		//TODO tobe merged into query builder
		field = h.GetParameterOrDefault(req, "search_field", "title")

		source = h.GetParameterOrDefault(req, "source_fields", "*")
	)

	query = util.CleanUserQuery(query)

	newQuery := h.GetParameterOrDefault(req, "v2", "false")
	if newQuery != "true" {
		//TODO tobe removed
		from := h.GetIntOrDefault(req, "from", 0)
		size := h.GetIntOrDefault(req, "size", 10)

		mustClauses := BuildMustFilterClauses(category, subcategory, richCategory, username, userid)
		if integrationID := req.Header.Get(core.HeaderIntegrationID); integrationID != "" {
			// get datasource by integration id
			datasourceIDs, hasAll, err := common.GetDatasourceByIntegration(integrationID)
			if err != nil {
				panic(err)
			}
			if !hasAll {
				if len(datasourceIDs) == 0 {
					// return empty search result when no datasource found
					h.WriteJSON(w, elastic.SearchResponse{}, http.StatusOK)
					return
				}
				// update datasource filter
				if datasource == "" {
					datasource = strings.Join(datasourceIDs, ",")
				} else {
					// calc intersection with datasource and datasourceIDs
					queryDatasource := strings.Split(datasource, ",")
					queryDatasource = util.StringArrayIntersection(queryDatasource, datasourceIDs)
					if len(queryDatasource) == 0 {
						// return empty search result when intersection datasource ids is empty
						h.WriteJSON(w, elastic.SearchResponse{}, http.StatusOK)
						return
					}
					datasource = strings.Join(queryDatasource, ",")
				}
			}
		}
		datasourceClause := BuildDatasourceClause(datasource, true)
		if datasourceClause != nil {
			mustClauses = append(mustClauses, datasourceClause)
		}
		mustClauses = append(mustClauses, map[string]interface{}{
			"bool": map[string]interface{}{
				"minimum_should_match": 1,
				"should": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"disabled": false,
						},
					},
					map[string]interface{}{
						"bool": map[string]interface{}{
							"must_not": map[string]interface{}{
								"exists": map[string]interface{}{
									"field": "disabled",
								},
							},
						},
					},
				},
			},
		})

		var q *orm.Query
		if query != "" || len(mustClauses) > 0 {
			q = BuildTemplatedQuery(from, size, mustClauses, nil, field, query, source, tags)
		} else {
			body, err := h.GetRawBody(req)
			if err != nil {
				http.Error(w, "query must be provided", http.StatusBadRequest)
				return
			}
			if len(body) == 0 {
				//ignore empty query
				return
			}
			q = &orm.Query{}
			q.RawQuery = body
		}

		err, res := orm.Search(common.Document{}, q)
		if err != nil || res.Raw == nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		v2 := SearchResponse{}
		err = util.FromJSONBytes(res.Raw, &v2)
		if err != nil {
			panic(err)
		}

		hits := v2.Hits.Hits

		ctx := orm.NewContextWithParent(req.Context())

		// Loop over the hits and ensure Source is modified correctly
		for i, doc := range hits {
			// Get the pointer to doc.Source to make sure you're modifying the original
			datasourceConfig, err := common.GetDatasourceConfig(ctx, doc.Source.Source.ID)
			if err == nil && datasourceConfig != nil && datasourceConfig.Connector.ConnectorID != "" {
				connectorConfig, err := getConnectorConfig(datasourceConfig.Connector.ConnectorID)

				if connectorConfig != nil && err == nil {
					icon := common.ParseAndGetIcon(connectorConfig, doc.Source.Icon)
					if icon != "" {
						hits[i].Source.Icon = icon
					}

					if doc.Source.Source.Icon != "" {
						icon = common.ParseAndGetIcon(connectorConfig, doc.Source.Source.Icon)
						if icon != "" {
							hits[i].Source.Source.Icon = icon
						}
					} else {
						//try datasource's icon
						if datasourceConfig.Icon != "" {
							icon = common.ParseAndGetIcon(connectorConfig, datasourceConfig.Icon)
							if icon != "" {
								hits[i].Source.Source.Icon = icon
							}
						} else {
							//try connector's icon
							icon = common.ParseAndGetIcon(connectorConfig, connectorConfig.Icon)
							if icon != "" {
								hits[i].Source.Source.Icon = icon
							}
						}
					}
				}
			}
		}

		if query != "" {

			reqUser, err := security.GetUserFromContext(req.Context())
			if err == nil && reqUser != nil {
				assistantSearchPermission := security.GetSimplePermission(Category, Assistant, string(QuickAISearchAction))
				perID := security.GetOrInitPermissionKey(assistantSearchPermission)

				if (reqUser.Roles != nil && util.AnyInArrayEquals(reqUser.Roles, security.RoleAdmin)) || reqUser.UserAssignedPermission.ValidateFor(perID) {
					assistantSize := 2
					if len(hits) < 5 {
						assistantSize = size - (len(hits))
					}

					assistants := searchAssistant(query, assistantSize)
					if len(assistants) > 0 {
						for _, assistant := range assistants {
							doc := common.Document{}
							doc.ID = assistant.ID
							doc.Type = "AI Assistant"
							doc.Icon = assistant.Icon
							doc.Title = assistant.Name
							doc.Summary = assistant.Description
							doc.URL = fmt.Sprintf("coco://extenstions/infinilabs/ask_assistant/%v", assistant.ID)
							doc.Source = common.DataSourceReference{
								ID:   "assistant",
								Name: "Assistant",
								Icon: "font_robot",
							}
							newHit := IndexDocument{Index: "assistant", ID: assistant.ID, Source: doc, Score: v2.Hits.MaxScore + 500}
							hits = append(hits, newHit)
						}
					}
				}
			}
		}

		v2.Hits.Hits = hits

		h.WriteJSON(w, v2, 200)

		return
	}

	//NEW Query
	builder, err := orm.NewQueryBuilderFromRequest(req)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	builder.DefaultQueryField("title.keyword^100", "title^10", "title.pinyin^4", "combined_fulltext")
	builder.Exclude("payload.*")

	if source != "" && source != "*" {
		builder.Include(strings.Split(source, ",")...)
	}

	filters := BuildFilters(category, subcategory, richCategory, username, userid)
	if integrationID := req.Header.Get(core.HeaderIntegrationID); integrationID != "" {
		// get datasource by integration id
		datasourceIDs, hasAll, err := common.GetDatasourceByIntegration(integrationID)
		if err != nil {
			panic(err)
		}
		if !hasAll {
			if len(datasourceIDs) == 0 {
				// return empty search result when no datasource found
				h.WriteJSON(w, elastic.SearchResponse{}, http.StatusOK)
				return
			}
			// update datasource filter
			if datasource == "" {
				datasource = strings.Join(datasourceIDs, ",")
			} else {
				// calc intersection with datasource and datasourceIDs
				queryDatasource := strings.Split(datasource, ",")
				queryDatasource = util.StringArrayIntersection(queryDatasource, datasourceIDs)
				if len(queryDatasource) == 0 {
					// return empty search result when intersection datasource ids is empty
					h.WriteJSON(w, elastic.SearchResponse{}, http.StatusOK)
					return
				}
				datasource = strings.Join(queryDatasource, ",")
			}
		}
	}
	datasourceFilter := BuildDatasourceFilter(datasource, true)
	if datasourceFilter != nil {
		filters = append(filters, datasourceFilter...)
	}

	//filter enabled doc
	filters = append(filters, orm.BoolQuery(orm.Should, orm.TermQuery("disabled", false), orm.MustNotQuery(orm.ExistsQuery("disabled"))).Parameter("minimum_should_match", 1))

	builder.Filter(filters...)

	ctx := orm.NewContextWithParent(req.Context())
	orm.WithModel(ctx, &common.Document{})

	docs := []common.Document{}
	err, resp := elastic.SearchV2WithResultItemMapper(ctx, &docs, builder, nil)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.Write(w, resp.Raw)
	if err != nil {
		h.Error(w, err)
	}

}

func searchAssistant(query string, size int) []common.Assistant {

	if size <= 0 {
		size = 2
	}

	q := orm.Query{}
	q.Size = size
	q.Conds = orm.And(orm.QueryString("combined_fulltext", query))
	q.Filter = orm.Eq("enabled", true)
	docs := []common.Assistant{}
	err, _ := orm.SearchWithJSONMapper(&docs, &q)
	if err != nil {
		_ = log.Error(err)
	}
	return docs
}

var connectorCacheKey = "Datasource"

func getConnectorConfig(id string) (*common.Connector, error) {
	v := configCache.Get(connectorCacheKey, id)
	if v != nil {
		if !v.Expired() {
			x, ok := v.Value().(*common.Connector)
			if ok && x != nil {
				return x, nil
			}
		}
	}

	obj := common.Connector{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if err == nil && exists {
		configCache.Set(connectorCacheKey, id, &obj, util.GetDurationOrDefault("30m", time.Duration(30)*time.Minute))
		return &obj, nil
	}

	return nil, errors.New("not found")
}

func BuildTemplatedQuery(from int, size int, mustClauses []interface{}, shouldClauses interface{}, field string, query string, source string, tags string) *orm.Query {
	templatedQuery := orm.TemplatedQuery{}
	templatedQuery.TemplateID = "coco-query-string"
	if shouldClauses != nil {
		templatedQuery.TemplateID = "coco-query-string-extra-should"
	}

	templatedQuery.Parameters = util.MapStr{
		"from":                 from,
		"size":                 size,
		"must_clauses":         mustClauses,
		"extra_should_clauses": shouldClauses,
		"field":                field,
		"query":                query,
		"source":               strings.Split(source, ","),
		"tags":                 strings.Split(tags, ","),
	}
	q := orm.Query{}
	q.TemplatedQuery = &templatedQuery
	return &q
}

func BuildDatasourceClause(datasource string, filterDisabled bool) interface{} {
	var datasourceClause interface{}
	if datasource != "" {
		if strings.Contains(datasource, ",") {
			arr := strings.Split(datasource, ",")
			datasourceClause = map[string]interface{}{
				"terms": map[string]interface{}{
					"source.id": arr,
				},
			}
		} else {
			datasourceClause = map[string]interface{}{
				"term": map[string]interface{}{
					"source.id": datasource,
				},
			}
		}
	}
	if !filterDisabled {
		return datasourceClause
	}

	disabledIDs, err := common.GetDisabledDatasourceIDs()
	if err != nil {
		panic(err)
	}
	if len(disabledIDs) == 0 {
		return datasourceClause
	}
	mustNot := map[string]interface{}{
		"terms": map[string]interface{}{
			"source.id": disabledIDs,
		},
	}
	if datasourceClause == nil {
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": mustNot,
			},
		}
	}
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"must_not": mustNot,
			"must":     datasourceClause,
		},
	}
}

func BuildMustFilterClauses(category string, subcategory string, richCategory string, username string, userid string) []interface{} {
	mustClauses := []interface{}{}

	// Check and add conditions to mustClauses

	if category != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"category": category,
			},
		})
	}

	if subcategory != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"subcategory": subcategory,
			},
		})
	}

	if richCategory != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"rich_categories.key": richCategory,
			},
		})
	}

	if username != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"owner.username": username,
			},
		})
	}

	if userid != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"owner.userid": userid,
			},
		})
	}
	return mustClauses
}

func BuildFilters(category string, subcategory string, richCategory string, username string, userid string) []*orm.Clause {
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

	if username != "" {
		mustClauses = append(mustClauses, orm.TermQuery("owner.username", username))
	}

	if userid != "" {
		mustClauses = append(mustClauses, orm.TermQuery("owner.userid", userid))
	}
	return mustClauses
}

func BuildDatasourceFilter(datasource string, filterDisabled bool) []*orm.Clause {
	mustClauses := []*orm.Clause{}
	if datasource != "" {
		if strings.Contains(datasource, ",") {
			arr := strings.Split(datasource, ",")
			mustClauses = append(mustClauses, orm.TermsQuery("source.id", arr))
		} else {
			mustClauses = append(mustClauses, orm.TermQuery("source.id", datasource))
		}
	}
	if !filterDisabled {
		return mustClauses
	}

	disabledIDs, err := common.GetDisabledDatasourceIDs()
	if err != nil {
		panic(err)
	}

	//TODO verify this filter
	if len(disabledIDs) > 0 {
		mustClauses = append(mustClauses, orm.MustNotQuery(orm.TermsQuery("source.id", disabledIDs)))
	}

	return mustClauses
}

func BuildShouldClauses(query []string, keyword []string) interface{} {
	clauses := []interface{}{}

	if len(query) > 0 {
		for _, v := range query {
			clauses = append(clauses, map[string]interface{}{
				"match": map[string]interface{}{
					"combined_fulltext": v,
				},
			})
		}
	}

	if len(keyword) > 0 {
		clauses = append(clauses, map[string]interface{}{
			"terms": map[string]interface{}{
				"combined_fulltext": keyword,
			},
		})
	}

	clause := util.MapStr{}
	clause["bool"] = util.MapStr{
		"should": clauses,
		"boost":  1,
	}

	return clause
}
