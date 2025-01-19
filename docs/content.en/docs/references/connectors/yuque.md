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
    "icon": "/assets/connector/yuque/icon.png", 
    "category": "website", 
    "tags": [
        "static_site", 
        "hugo", 
        "web"
    ], 
    "url": "http://coco.rs/connectors/hugo_site", 
    "assets": {
        "icons": {
            "default": "/assets/connector/yuque/icon.png", 
            "book": "/assets/connector/yuque/book.png", 
            "board": "/assets/connector/yuque/board.png", 
            "sheet": "/assets/connector/yuque/sheet.png", 
            "table": "/assets/connector/yuque/table.png", 
            "doc": "/assets/connector/yuque/doc.png"
        }
    }
}'
```


> Use `yuque` as a unique identifier, as it is a builtin connector.