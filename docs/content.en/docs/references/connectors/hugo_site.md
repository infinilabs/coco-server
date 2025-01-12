---
title: "Hugo Site"
weight: 20
---

# Hugo Site Connector

## Register Hugo Site Connector

```shell
curl -XPUT http://localhost:9000/connector/hugo_site?replace=true -d '{
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
            "web": "http://coco.infini.cloud/assets/hugo/web.png", 
            "web_page": "http://coco.infini.cloud/assets/hugo/web_page.png", 
            "news": "http://coco.infini.cloud/assets/hugo/news.png"
        }
    }
}'
```