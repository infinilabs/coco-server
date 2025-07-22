---
title: "RSS"
weight: 20
---

# Rss Connector

## Register RSS Connector

```shell
curl -XPUT "http://localhost:9000/connector/rss?replace=true" -d '{
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
  }
}'
```

> Use `rss` as a unique identifier, as it is a builtin connector.


## Update coco-server's config

Below is an example configuration for enabling the RSS Connector in coco-server:

```shell
connector:
  rss:
    enabled: true
    interval: 30s
    queue:
      name: indexing_documents
```
### Explanation of Config Parameters

| **Field**      | **Type**  | **Description**                                                                         |
|-----------------|-----------|-----------------------------------------------------------------------------------------|
| `enabled`      | `boolean` | Enables or disables the RSS connector. Set to `true` to activate it.                |
| `interval`     | `string`  | Specifies the time interval (e.g., `60s`) at which the connector will check for updates. |
| `queue.name`   | `string`  | Defines the name of the queue where indexing tasks will be added.                       |

## Use the RSS Connector

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST "http://localhost:9000/datasource/" -d'
{
    "name":"My RSS feed",
    "type":"connector",
    "connector":{
        "id":"rss",
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
