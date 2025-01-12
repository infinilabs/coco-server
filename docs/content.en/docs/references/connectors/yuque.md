---
title: "Yuque"
weight: 30
---

# Yuque Connector

## Register Yuque Connector

```shell
curl -XPUT http://localhost:9000/connector/yuque?replace=true -d '{
    "name": "Yuque Docs Connector", 
    "description": "Fetch the docs metadata for yuque.", 
    "icon": "https://coco.infini.cloud/assets/connector/yuque/icon.png", 
    "category": "website", 
    "tags": [
        "static_site", 
        "hugo", 
        "web"
    ], 
    "url": "http://coco.rs/connectors/hugo_site", 
    "assets": {
        "icons": {
            "default": "https://coco.infini.cloud/assets/connector/yuque/icon.png", 
            "book": "https://coco.infini.cloud/assets/connector/yuque/book.png", 
            "board": "https://coco.infini.cloud/assets/connector/yuque/icon.png", 
            "sheet": "https://coco.infini.cloud/assets/connector/yuque/sheet.png", 
            "table": "https://coco.infini.cloud/assets/connector/yuque/table.png", 
            "doc": "https://coco.infini.cloud/assets/connector/yuque/icon.png"
        }
    }
}'
```


> Use `yuque` as a unique identifier, or substitute it with any ID of your choice.
>
> Replace `https://coco.infini.cloud` to your coco-server's endpoint.
