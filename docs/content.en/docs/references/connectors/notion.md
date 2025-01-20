---
title: "Notion"
weight: 30
---

# Notion Connector

## Register Notion Connector

```shell
curl -XPUT "http://localhost:9000/connector/notion?replace=true" -d '{
    "name": "Notion Docs Connector",
    "description": "Fetch the docs metadata for notion.",
    "icon": "/assets/connector/notion/icon.png",
    "category": "website", 
    "tags": [
        "docs",
        "notion",
        "web"
    ], 
    "url": "http://coco.rs/connectors/notion",
    "assets": {
        "icons": {
            "default": "/assets/connector/notion/icon.png",
            "web_page": "/assets/connector/notion/icon.png",
            "database": "/assets/connector/notion/database.png",
            "page": "/assets/connector/notion/page.png"
        }
    }
}'
```


> Use `notion` as a unique identifier, as it is a builtin connector.


## Update coco-server's config

Below is an example configuration for enabling the Notion Connector in coco-server:

```shell
connector:
  notion:
    enabled: true
    queue:
      name: indexing_documents
    interval: 10s
```

### Explanation of Config Parameters

| **Field**      | **Type**  | **Description**                                                                 |
|-----------------|-----------|---------------------------------------------------------------------------------|
| `enabled`      | `boolean` | Enables or disables the Hugo Site connector. Set to `true` to activate it.      |
| `interval`     | `string`  | Specifies the time interval (e.g., `60s`) at which the connector will check for updates. |
| `queue.name`   | `string`  | Defines the name of the queue where indexing tasks will be added.               |

## Use the Notion Connector

The Notion Connector allows you to index data from your notion account into your system. Follow these steps to set it up:

### Obtain Notion API Token

Before using this connector, you need to obtain your Notion API token. Refer to the official [Notion integrations documentation](https://www.notion.so/profile/integrations) for instructions.

{{% load-img "/img/notion-create-app.png" "Notion integrations" %}}


### Example Request

Here is an example request to configure the Notion Connector:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
    "name": "My Notion",
    "type": "connector",
    "connector": {
        "id": "notion",
        "config": {
            "token": "your_notion_api_token"
        }
    }
}'
```

## Supported Config Parameters for Notion Connector

Below are the configuration parameters supported by the Notion Connector:

| **Field**               | **Type**  | **Description**                                                                                  |
|--------------------------|-----------|--------------------------------------------------------------------------------------------------|
| `token`                 | `string`  | Your Notion API token. This is required to access Notion's API.                                    |

### Notes

- Set `token` to your valid Notion API token to enable the connector.
