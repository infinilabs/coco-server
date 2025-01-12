---
title: "Connector"
weight: 80
---

# Connector

## Work with *Connector*

### Register a Connector

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:2900/connector/ -d'{
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
    }
}'

//response
{
  "_id": "cu0caqt3q95r66at41o0",
  "result": "created"
}
```

### Update a Connector
```shell
curl -XPUT http://localhost:9000/connector/cu0caqt3q95r66at41o0?replace=true -d '{
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
