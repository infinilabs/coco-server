---
title: "RSS"
weight: 20
---

# Rss Connector

## Register RSS Connector

```shell
curl -XPOST "http://localhost:9000/connector/" -d '{
  "name" : "RSS Connector",
  "description" : "Fetch items from a specified RSS feed.",
  "category" : "website",
  "icon" : "/assets/icons/connector/rss/icon.png",
  "tags" : [
    "rss",
    "feed",
    "web"
  ],
  "url" : "http://coco.rs/connectors/rss",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/rss/icon.png"
    }
  },
     "processor":{
        "enabled":true,
        "name":"rss"
     }
}'
```

## Use the RSS Connector

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST "http://localhost:9000/datasource/" -d'
{
    "name":"My RSS feed",
    "type":"connector",
    "connector":{
        "id":"rss's connector id",
         "config":{
            "urls": [ "The RSS link" ]
        }
    }
}'
```

Below is the config parameters supported by this connector.

| **Field**              | **Type**           | **Description**                                            |
|-------------------------|--------------------|------------------------------------------------------------|
| `urls`               |  []string          | The array list of the rss feet, support more than one url. |
