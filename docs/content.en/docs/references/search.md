---
title: "Search"
weight: 50
---

# Search API


## Search API Reference


### Get Query Suggestions

```shell
//request
curl  -XGET http://localhost:9000/query/_suggest\?query\=buss

//response
{
  "query": "buss",
  "suggestions": [
    {
      "suggestion": "Q3 Business Report",
      "score": 0.99,
      "source": {
         "type":"connector",
         "name":"google_drive",
         "id":"e806831dacc3",
       }
    }
  ]
}
```

### Get Search Results

```shell
//request
curl  -XGET http://localhost:9000/query/_search\?query\=Business

//response
{"took":15,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":1,"relation":"eq"},"max_score":3.0187376,"hits":[{"_index":"coco_document","_type":"_doc","_id":"csstf6rq50k5sqipjaa0","_score":3.0187376,"_source":{"id":"csstf6rq50k5sqipjaa0", ...OMITTED...}}}]}}
```

### Parameters

| Parameter       | Type   | Default   | Description                                                                                   |
|-----------------|--------|-----------|-----------------------------------------------------------------------------------------------|
| `query`         | string | `""`      | The search query string.                                                                      |
| `from`          | int    | `0`       | The starting offset of the search results (used for pagination).                              |
| `size`          | int    | `10`      | The number of search results to return.                                                       |