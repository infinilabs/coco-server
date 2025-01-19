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
    }
}'
```


> Use `hugo_site` as a unique identifier, as it is a builtin connector.


## Use the Hugo Site Connector

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/datasource/ -d'
{
    "name":"My Hugo Site",
    "type":"connector",
    "connector":{
        "id":"hugo_site",
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
