/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package search

import (
	"errors"
	"infini.sh/coco/core"
	"infini.sh/coco/modules/common"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	ccache "infini.sh/framework/lib/cache"
	"net/http"
	"net/url"
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
	var (
		query        = h.GetParameterOrDefault(req, "query", "")
		from         = h.GetIntOrDefault(req, "from", 0)
		size         = h.GetIntOrDefault(req, "size", 10)
		datasource   = h.GetParameterOrDefault(req, "datasource", "")
		category     = h.GetParameterOrDefault(req, "category", "")
		username     = h.GetParameterOrDefault(req, "username", "")
		userid       = h.GetParameterOrDefault(req, "userid", "")
		tags         = h.GetParameterOrDefault(req, "tags", "")
		subcategory  = h.GetParameterOrDefault(req, "subcategory", "")
		richCategory = h.GetParameterOrDefault(req, "rich_category", "")
		field        = h.GetParameterOrDefault(req, "search_field", "title")
		source       = h.GetParameterOrDefault(req, "source_fields", "*")
	)

	mustClauses := BuildMustClauses(category, subcategory, richCategory, username, userid)
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
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.Raw != nil {
		v2 := SearchResponse{}
		err := util.FromJSONBytes(res.Raw, &v2)
		if err != nil {
			panic(err)
		}

		// Loop over the hits and ensure Source is modified correctly
		for i, doc := range v2.Hits.Hits {
			// Get the pointer to doc.Source to make sure you're modifying the original
			datasourceConfig, err := getDatasourceConfig(doc.Source.Source.ID)
			if err == nil && datasourceConfig != nil && datasourceConfig.Connector.ConnectorID != "" {
				connectorConfig, err := getConnectorConfig(datasourceConfig.Connector.ConnectorID)

				if connectorConfig != nil && err == nil {
					icon, err := getIcon(connectorConfig, doc.Source.Icon)
					if err == nil && icon != "" {
						v2.Hits.Hits[i].Source.Icon = icon
					}

					if doc.Source.Source.Icon != "" {
						icon, err = getIcon(connectorConfig, doc.Source.Source.Icon)
						if err == nil && icon != "" {
							v2.Hits.Hits[i].Source.Source.Icon = icon
						}
					} else {
						//try connector's icon
						icon, err = getIcon(connectorConfig, connectorConfig.Icon)
						if err == nil && icon != "" {
							v2.Hits.Hits[i].Source.Source.Icon = icon
						}
					}
				}
			}
		}

		h.WriteJSON(w, v2, 200)
		return
	}

	_, err = h.Write(w, res.Raw)
	if err != nil {
		h.Error(w, err)
	}
}

var datasourceCacheKey = "Datasource"
var connectorCacheKey = "Datasource"

func getDatasourceConfig(id string) (*common.DataSource, error) {
	v := configCache.Get(datasourceCacheKey, id)
	if v != nil {
		if !v.Expired() {
			x, ok := v.Value().(*common.DataSource)
			if ok && x != nil {
				return x, nil
			}
		}
	}

	obj := common.DataSource{}
	obj.ID = id
	exists, err := orm.Get(&obj)
	if err == nil && exists {
		configCache.Set(datasourceCacheKey, id, &obj, util.GetDurationOrDefault("30m", time.Duration(30)*time.Minute))
		return &obj, nil
	}

	return nil, errors.New("not found")
}

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

func getIcon(connector *common.Connector, icon string) (string, error) {
	appCfg := common.AppConfig()
	baseEndpoint := appCfg.ServerInfo.Endpoint
	link, ok := connector.Assets.Icons[icon]
	if ok {
		if util.PrefixStr(link, "/") && baseEndpoint != "" {
			link, err := url.JoinPath(baseEndpoint, link)
			if err == nil && link != "" {
				return link, nil
			}
		}
		// return the direct key to the font icon
		return link, nil
	} else {
		if util.PrefixStr(icon, "/") {
			link, err := url.JoinPath(baseEndpoint, icon)
			if err == nil && link != "" {
				return link, nil
			}
		}
	}
	return icon, nil
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

func BuildMustClauses(category string, subcategory string, richCategory string, username string, userid string) []interface{} {
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

	if len(clauses) > 0 {

	}

	clause := util.MapStr{}
	clause["bool"] = util.MapStr{
		"should": clauses,
		"boost":  100,
	}

	return clause
}
