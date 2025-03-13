---
title: "How to integrate Coco AI with your own datasource"
weight: 30
---

# How to integrate Coco AI with your own datasource

This guide will walk you through the process of creating a custom datasource and building a simple connector to integrate it with the system.

## Obtain API Token

Make sure you have a valid API token before continue.

{{% load-img "/img/create-api-token.png" "Search via App" %}}

## Create a simple connector

```
curl -H'X-API-TOKEN: cv9bb63q50k5eikkq43gp07lmg60gn2nor7uf650xpxazd6a6nns8uttymz52uvnplhmtaf829x71lph51li' -XPOST http://localhost:9000/connector/ -d'{
    "name": "My Connector"
}'
```

Response
```
{
  "_id": "cv9bc23q50k5eikkq4c0",
  "result": "created"
}
```

## Create a datasource for this connector

```
curl -H'X-API-TOKEN: cv9bb63q50k5eikkq43gp07lmg60gn2nor7uf650xpxazd6a6nns8uttymz52uvnplhmtaf829x71lph51li'  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/datasource/ -d'
{
    "name":"My Datasource",
    "type":"connector",
    "connector":{
        "id":"cv9bc23q50k5eikkq4c0"
    }
}'
```

Response
```
{
  "_id": "cv9bcjrq50k5eikkq4h0",
  "result": "created"
}
```

## Insert document to this datasource

```
curl -H'X-API-TOKEN: cv9bb63q50k5eikkq43gp07lmg60gn2nor7uf650xpxazd6a6nns8uttymz52uvnplhmtaf829x71lph51li'  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/datasource/cv9bcjrq50k5eikkq4h0/_doc -d'
{
   "title": "I am just a Coco doc that you can search",
   "summary": "Nothing but great start",
   "content": "Coco is a unified private search engien that you can trust."
}'
```
Response
```
{
  "_id": "cv9be7jq50k5eikkq4vg",
  "result": "created"
}
```

## Search via App

Now you are ready to consume your document through Coco AI app.

{{% load-img "/img/search-custom-data-source.png" "Search via App" %}}


## Search via API

You may search via API.

```
curl -H'X-API-TOKEN: cv9bb63q50k5eikkq43gp07lmg60gn2nor7uf650xpxazd6a6nns8uttymz52uvnplhmtaf829x71lph51li'  -XGET http://localhost:9000/query/_search\?query\=coco&datasource=cv9bcjrq50k5eikkq4h0
```

Response
```
{"took":4,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":1,"relation":"eq"},"max_score":33.31594,"hits":[{"_index":"coco_document","_type":"_doc","_id":"cv9be7jq50k5eikkq4vg","_score":33.31594,"_source":{"summary":"Nothing but great start","created":"2025-03-13T18:44:46.600896+08:00","id":"cv9be7jq50k5eikkq4vg","source":{"name":"My Datasource","id":"cv9bcjrq50k5eikkq4h0","type":"connector"},"title":"I am just a Coco doc that you can search","updated":"2025-03-13T18:44:46.600908+08:00"}}]}}
```