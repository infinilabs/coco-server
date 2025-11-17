---
title: "Connectors"
weight: 100
bookCollapseSection: true
---

# Connectors

## Work with *Connector*

### Register a Connector

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/connector/ -d'{
    "name": "Hugo Site Connector",
    "description": "Fetch the index.json file from a specified Hugo site.",
    "icon": "http://coco.infini.cloud/assets/hugo.png",
    "category": "website",
    "tags": [
        "static_site",
        "hugo",
        "web"
    ],
    "url": "http://coco.rs/connectors/hugo_site",
    "assets": {
        "icons": {
            "default": "http://coco.infini.cloud/assets/web.png",
            "blog": "http://coco.infini.cloud/assets/hugo/blog.png",
            "web": "http://coco.infini.cloud/assets/hugo/web_page.png",
            "news": "http://coco.infini.cloud/assets/hugo/news.png"
        }
    },
    "processor":{
       "enabled":true,
       "name":"hugo"
    }
}'

//response
{
  "_id": "cu0caqt3q95r66at41o0",
  "result": "created"
}
```

> Note: Every connector is expected to implement a processor that can be registered through the API. For instance, the built-in `hugo` processor is demonstrated in the example above.

### Update a Connector
```shell
curl -XPUT http://localhost:9000/connector/cu0caqt3q95r66at41o0  -d '{
    "name": "Hugo Site Connector",
    "description": "Fetch the index.json file from a specified Hugo site.",
    "icon": "http://coco.infini.cloud/assets/hugo.png",
    "category": "website",
    "tags": [
        "static_site",
        "hugo",
        "web"
    ],
    "url": "http://coco.rs/connectors/hugo_site",
    "assets": {
        "icons": {
            "default": "http://coco.infini.cloud/assets/web.png",
            "blog": "http://coco.infini.cloud/assets/hugo/blog.png",
            "web": "http://coco.infini.cloud/assets/hugo/web_page.png",
            "news": "http://coco.infini.cloud/assets/hugo/news.png"
        }
    },
    "processor":{
       "enabled":true,
       "name":"hugo"
    }
}'

//response
{
  "_id": "cu0caqt3q95r66at41o0",
  "result": "updated"
}
```

> `?replace=true` can safely ignore errors for non-existent items.

### View a Connector
```shell
curl -XGET http://localhost:9000/connector/cu0caqt3q95r66at41o0
```

### Delete the Connector
```shell
curl -XDELETE http://localhost:9000/connector/cu0caqt3q95r66at41o0

//response
{
  "_id": "cu0caqt3q95r66at41o0",
  "result": "deleted"
}
```

### Search Connectors
```shell
curl -XGET http://localhost:9000/connector/_search
```
