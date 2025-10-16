---
title: "Hugo Site"
weight: 20
---

# Hugo Site Connector

## Register Hugo Site Connector

```shell
curl -XPOST "http://localhost:9000/connector" -d '{
    "name": "Hugo Site Connector", 
    "description": "Fetch the index.json file from a specified Hugo site.", 
    "icon": "/assets/connector/hugo_site/icon.png", 
    "category": "website", 
    "tags": [
        "static_site", 
        "hugo", 
        "web"
    ], 
    "url": "http://coco.rs/connectors/hugo_site", 
    "assets": {
        "icons": {
            "default": "/assets/connector/hugo_site/web.png", 
            "blog": "/assets/connector/hugo_site/blog.png", 
            "web": "/assets/connector/hugo_site/web.png", 
            "web_page": "/assets/connector/hugo_site/web_page.png", 
            "news": "/assets/connector/hugo_site/news.png"
        }
    },
  "processor":{
     "enabled":true,
     "name":"hugo_site"
  }
}'
```

Make sure hugo generated the json format with this:
```
[
    {
    "category": "Product",
    "content": "INFINI Console v1.28 Released We’re excited to announce INFINI Console v1.28, the latest update from INFINI Labs! This release brings the powerful TopN feature to help you identify key metrics efficiently, alongside other performance improvements and bug fixes. Read on for all the details and enhancements in this release.\nWhat is INFINI Console? Great question! INFINI Console is a lightweight, cross-version, unified management platform designed specifically for search infrastructures. It empowers enterprises to:\nManage multiple search clusters across different versions seamlessly. Gain centralized control for efficient cluster monitoring and maintenance. INFINI Console – The Choice of Elasticsearch Professionals. Be an Elasticsearch Pro Today!\nWith INFINI Console, you can streamline the management of your search ecosystem like never before! 🚀\nLearn more here: ",
    "created": "2025-01-11T17:00:00+08:00",
    "lang": "en",
    "subcategory": "Released",
    "summary": "Discover the new TopN feature and other enhancements in INFINI Console v1.28.",
    "tags": [
        "Console",
        "TopN",
        "Release"
    ],
    "title": "INFINI Console v1.28 Released",
    "updated": null,
    "url": "/posts/2025/01-11-produc-released-console-topn/"
    }
]
```

## Use the Hugo Site Connector

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST "http://localhost:9000/datasource/" -d'
{
    "name":"My Hugo Site",
    "type":"connector",
    "connector":{
        "id":"hugo_site's connector id",
         "config":{
            "urls": [ "https://pizza.rs/index.json" ]
        }
    }
}'
```

Below is the config parameters supported by this connector.

| **Field**              | **Type**           | **Description**                                                                                     |
|-------------------------|--------------------|-----------------------------------------------------------------------------------------------------|
| `urls`               |  []string          | The array list of the hugo's site, support more than one url.                                                  |
