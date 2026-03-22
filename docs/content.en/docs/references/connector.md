---
title: "Connector"
weight: 85
---

# Connector

## Work with *Connector*

A Connector defines how to access and fetch data from external services such as Google Drive, GitHub, Confluence, databases, and more. Connectors are used by datasources to pull data into Coco Server.

## Connector API

Below is the field description for the connector.

| **Field**                    | **Type**   | **Description**                                                                        |
|------------------------------|------------|----------------------------------------------------------------------------------------|
| `name`                       | `string`   | The connector's name.                                                                  |
| `description`                | `string`   | A brief description of the connector.                                                  |
| `category`                   | `string`   | The category of the connector, e.g., `cloud_storage`, `version_control`.               |
| `icon`                       | `string`   | The icon representing the connector in the UI.                                         |
| `tags`                       | `array`    | Tags associated with the connector.                                                    |
| `url`                        | `string`   | Direct link associated with the connector.                                             |
| `path_hierarchy`             | `boolean`  | Whether the connector supports hierarchical path-based access.                         |
| `assets.icons`               | `object`   | Map of icon keys to icon URLs for different connector states/types.                    |
| `builtin`                    | `boolean`  | Indicates whether the connector is built-in (built-in connectors cannot be deleted).   |
| `config`                     | `object`   | Connector-specific configuration (varies by connector type).                           |
| `oauth_connect_implemented`  | `boolean`  | Whether the connector supports OAuth-based connection flow.                            |
| `processor.enabled`          | `boolean`  | Whether the data processing pipeline is enabled for this connector.                    |
| `processor.name`             | `string`   | Name of the processor to use.                                                          |
| `enabled`                    | `boolean`  | Enables or disables the connector.                                                     |

### Create a Connector

```shell
//request
curl -H 'Content-Type: application/json' -XPOST http://localhost:9000/connector/ -d'{
  "name": "My GitHub Connector",
  "description": "Connects to GitHub repositories",
  "category": "version_control",
  "icon": "font_github",
  "tags": ["github", "code"],
  "config": {
    "token": "ghp_xxxxxxxxxxxx",
    "repos": ["infinilabs/coco-server"]
  },
  "processor": {
    "enabled": true
  },
  "enabled": true
}'

//response
{
  "_id": "d07abc123def456ghi",
  "result": "created"
}
```

### View a Connector

```shell
//request
curl -XGET http://localhost:9000/connector/d07abc123def456ghi

//response
{
  "_id": "d07abc123def456ghi",
  "_source": {
    "id": "d07abc123def456ghi",
    "created": "2025-01-15T10:00:00+08:00",
    "updated": "2025-01-15T10:00:00+08:00",
    "name": "My GitHub Connector",
    "description": "Connects to GitHub repositories",
    "category": "version_control",
    "icon": "font_github",
    "tags": ["github", "code"],
    "config": {
      "token": "******",
      "repos": ["infinilabs/coco-server"]
    },
    "processor": {
      "enabled": true
    },
    "enabled": true
  },
  "found": true
}
```

### Update a Connector

```shell
//request
curl -H 'Content-Type: application/json' -XPUT http://localhost:9000/connector/d07abc123def456ghi -d'{
  "name": "My GitHub Connector - Updated",
  "description": "Updated connector for GitHub repositories",
  "enabled": true
}'

//response
{
  "_id": "d07abc123def456ghi",
  "result": "updated"
}
```

> By default, the update merges with the existing connector configuration. Use `?replace=true` to fully replace the connector configuration instead of merging.

### Delete a Connector

```shell
//request
curl -XDELETE http://localhost:9000/connector/d07abc123def456ghi

//response
{
  "_id": "d07abc123def456ghi",
  "result": "deleted"
}
```

> **Note:** Built-in connectors cannot be deleted. Attempting to delete a built-in connector will return a `403 Forbidden` error.

### Search Connectors

```shell
//request
curl -XGET http://localhost:9000/connector/_search

//response
{
  "took": 3,
  "hits": {
    "total": { "value": 5, "relation": "eq" },
    "hits": [
      {
        "_id": "d07abc123def456ghi",
        "_source": {
          "id": "d07abc123def456ghi",
          "name": "My GitHub Connector",
          "category": "version_control",
          "icon": "font_github",
          "enabled": true,
          "created": "2025-01-15T10:00:00+08:00"
        }
      }
    ]
  }
}
```

You can filter connectors by name:

```shell
curl -XGET "http://localhost:9000/connector/_search?query=github"
```

> **Note:** Sensitive fields in connector configurations (such as API keys and tokens) are automatically masked in search results.

For detailed connector-specific configurations, refer to the [Connectors Reference](./connectors/).
