---
title: "Search"
weight: 50
---

# Search API


## Search API Reference

### Parameters

| Parameter       | Type   | Default     | Description                                                                                   |
|-----------------|--------|-------------|-----------------------------------------------------------------------------------------------|
| `query`         | string | `""`        | The search query string.                                                                      |
| `from`          | int    | `0`         | The starting offset of the search results (used for pagination).                              |
| `size`          | int    | `10`        | The number of search results to return.                                                       |
| `datasource`    | string | `""`        | Filter results by datasource ID.                                                              |
| `category`      | string | `""`        | Filter results by primary category.                                                           |
| `subcategory`   | string | `""`        | Filter results by secondary category.                                                         |
| `rich_category` | string | `""`        | Filter results by rich category key.                                                          |
| `search_type`   | string | `"keyword"` | Type of search to perform. Options: `keyword`, `semantic`, `hybrid`.                          |
| `fuzziness`     | string | `"3"`       | Fuzziness level for fuzzy matching (0-5).                                                     |
| `filter`        | string | `""`        | Additional filter criteria.                                                                   |


### Get Search Results

```shell
//request
curl  -XGET http://localhost:9000/query/_search\?query\=Business

//response
{
  "hits": {
    "total": { "value": 1, "relation": "eq" },
    "max_score": 3.0187376,
    "hits": [
      {
        "_index": "coco_document",
        "_id": "csstf6rq50k5sqipjaa0",
        "_score": 3.0187376,
        "_source": {
          "id": "csstf6rq50k5sqipjaa0",
          "title": "Q3 Business Report",
          "category": "report",
          "source": {
            "type": "connector",
            "name": "google_drive",
            "id": "e806831dacc3"
          },
          "url": "https://drive.google.com/file/d/abc123/view"
        }
      }
    ]
  }
}
```

You can also filter by datasource, category, or use different search types:

```shell
//semantic search filtered by datasource
curl -XGET "http://localhost:9000/query/_search?query=quarterly+revenue&datasource=cu1rf03q50k43nn2pi6g&search_type=semantic"

//hybrid search with a specific category
curl -XGET "http://localhost:9000/query/_search?query=report&category=business&search_type=hybrid&size=20"
```

### Get Query Suggestions

The suggestion API supports three modes based on the `tag` parameter.

#### Document Suggestions (default)

```shell
//request
curl  -XGET http://localhost:9000/query/_suggest\?query\=buss

//response
{
  "query": "buss",
  "suggestions": [
    {
      "suggestion": "Q3 Business Report",
      "highlighted_suggestion": "Q3 <em>Bus</em>iness Report",
      "score": 0.99,
      "icon": "pdf",
      "source": {
         "type":"connector",
         "name":"google_drive",
         "id":"e806831dacc3"
       },
      "url": "http://localhost:9000/#/preview/document/csstf6rq50k5sqipjaa0"
    }
  ],
  "recent_searches": [],
  "banner": {
    "icon": "",
    "name": "INFINI Labs",
    "url": "http://infinilabs.com"
  }
}
```

#### Field Name Suggestions

Returns available field names matching the query, used for building advanced search filters.

```shell
//request
curl -XGET http://localhost:9000/query/_suggest/field_names\?query\=cat

//response
{
  "query": "cat",
  "suggestions": [
    {
      "suggestion": "category",
      "score": 1,
      "payload": {
        "field_icon": "folder",
        "field_label": "Category",
        "field_description": "Primary category of the document",
        "field_name": "category",
        "field_data_type": "keyword",
        "support_multi_select": true
      }
    }
  ]
}
```

#### Field Value Suggestions

Returns distinct values for a specific field, used for filter auto-completion.

```shell
//request
curl -XGET "http://localhost:9000/query/_suggest/field_values?field_name=category&query=rep"

//response
{
  "query": "rep",
  "suggestions": [
    {
      "suggestion": "report",
      "score": 1
    }
  ]
}
```

| Parameter    | Type   | Required | Description                                                      |
|--------------|--------|----------|------------------------------------------------------------------|
| `tag`        | string | No       | Suggestion type: `documents` (default), `field_names`, `field_values`. |
| `query`      | string | No       | Search prefix or query term.                                     |
| `from`       | int    | No       | Pagination offset (default: `0`).                                |
| `size`       | int    | No       | Number of suggestions to return (default: `10`).                 |
| `field_name` | string | Yes*     | Required when `tag=field_values`. The field to aggregate values from. |
| `datasource` | string | No       | Filter by datasource ID (document suggestions only).             |
| `category`   | string | No       | Filter by category (document suggestions only).                  |


### Get Recommendations

Returns recommended content based on configured recommendation profiles.

```shell
//request
curl -XGET http://localhost:9000/query/_recommend

//response
{
  "recommendations": [
    {
      "title": "Getting Started Guide",
      "description": "Learn how to set up and use Coco Server",
      "score": 1.0,
      "icon": "book",
      "url": "https://docs.infinilabs.com/coco-server/main/",
      "category": "documentation",
      "breadcrumbs": [
        {
          "icon": "home",
          "name": "Docs",
          "url": "https://docs.infinilabs.com/"
        }
      ]
    }
  ],
  "total": 1
}
```

You can also specify a recommendation profile tag:

```shell
curl -XGET http://localhost:9000/query/_recommend/hot
```

### Get Field Metadata

Returns metadata for document fields, useful for building dynamic search UIs.

```shell
//request
curl -XGET http://localhost:9000/field_meta/category

//response
{
  "category": {
    "field_icon": "folder",
    "field_label": "Category",
    "field_description": "Primary category of the document",
    "field_name": "category",
    "field_data_type": "keyword",
    "support_multi_select": true
  }
}
```

Multiple fields can be requested by separating them with commas:

```shell
//request
curl -XGET http://localhost:9000/field_meta/category,tags,source.name

//response
{
  "category": {
    "field_icon": "folder",
    "field_label": "Category",
    "field_name": "category",
    "field_data_type": "keyword",
    "support_multi_select": true
  },
  "tags": {
    "field_icon": "tag",
    "field_label": "Tags",
    "field_name": "tags",
    "field_data_type": "keyword",
    "support_multi_select": true
  }
}
```