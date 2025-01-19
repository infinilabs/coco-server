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


## Update coco-server's config

Below is an example configuration for enabling the Yuque Connector in coco-server:

```shell
connector:
  yuque:
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

## Use the Yuque Connector

The Yuque Connector allows you to index data from your Yuque account into your system. Follow these steps to set it up:

### Obtain Yuque API Token

Before using this connector, you need to obtain your Yuque API token. Refer to the official [Yuque API documentation](https://www.yuque.com/yuque/developer/api) for instructions.

### Example Request

Here is an example request to configure the Yuque Connector:

```shell
curl -H 'Content-Type: application/json' -XPOST http://localhost:9000/datasource/ -d '
{
    "name": "My Yuque",
    "type": "connector",
    "connector": {
        "id": "yuque",
        "config": {
            "token": "your_yuque_api_token",
            "include_private_book": false,
            "include_private_doc": false,
            "indexing_books": true,
            "indexing_docs": true,
            "indexing_users": true,
            "indexing_groups": true
        }
    }
}'
```

## Supported Config Parameters for Yuque Connector

Below are the configuration parameters supported by the Yuque Connector:

| **Field**               | **Type**  | **Description**                                                                                  |
|--------------------------|-----------|--------------------------------------------------------------------------------------------------|
| `token`                 | `string`  | Your Yuque API token. This is required to access Yuque's API.                                    |
| `include_private_book`  | `bool`    | Whether to include private books in indexing. Defaults to `false`.                              |
| `include_private_doc`   | `bool`    | Whether to include private documents in indexing. Defaults to `false`.                          |
| `indexing_books`        | `bool`    | Whether to index books in Yuque. Defaults to `false`.                                           |
| `indexing_docs`         | `bool`    | Whether to index documents in Yuque. Defaults to `false`.                                       |
| `indexing_users`        | `bool`    | Whether to index user data from Yuque. Defaults to `false`.                                     |
| `indexing_groups`       | `bool`    | Whether to index group data from Yuque. Defaults to `false`.                                    |

### Notes

- Set `token` to your valid Yuque API token to enable the connector.
- Boolean parameters like `include_private_book`, `indexing_books`, etc., allow you to customize the scope of indexing.