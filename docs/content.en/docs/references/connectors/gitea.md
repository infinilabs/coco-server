---
title: "Gitea"
weight: 62
---
# Gitea Connector

## Register Gitea Connector

```shell
curl -XPUT "http://localhost:9000/connector/gitea?replace=true" -d '{
  "name": "Gitea Connector",
  "description": "Fetch repositories, issues, pull requests, and wikis from Gitea.",
  "icon": "/assets/icons/connector/gitea/icon.png",
  "category": "website",
  "tags": [
    "git",
    "code",
    "vcs",
    "website"
  ],
  "url": "http://coco.rs/connectors/gitea",
  "assets": {
    "icons": {
      "default": "/assets/icons/connector/gitea/icon.png",
      "repository": "/assets/icons/connector/gitea/repository.png",
      "issue": "/assets/icons/connector/gitea/issue.png",
      "pull_request": "/assets/icons/connector/gitea/pull_request.png"
    }
  },
  "processor": {
    "enabled": true,
    "name": "gitea"
  }
}'
```

> Use `gitea` as a unique identifier, as it is a builtin connector.

> **Note**: Starting from version **0.4.0**, the Gitea connector uses a **pipeline-based architecture** for better performance and flexibility. The `processor` configuration is required for the connector to work properly.

## Pipeline Architecture

Starting from version **0.4.0**, the Gitea connector uses a **pipeline-based architecture** instead of the legacy scheduled task approach. This provides:

- **Better Performance**: Centralized dispatcher manages all connector sync operations
- **Per-Datasource Configuration**: Each datasource can have its own sync interval
- **Enrichment Pipeline Support**: Optional data enrichment pipelines per datasource
- **Resource Efficiency**: Optimized scheduling and resource management

### Pipeline Configuration (coco.yml)

The connector is managed by the centralized dispatcher pipeline:

```yaml
pipeline:
  - name: connector_dispatcher
    auto_start: true
    keep_running: true
    singleton: true
    retry_delay_in_ms: 10000
    processor:
      - connector_dispatcher:
          max_running_timeout_in_seconds: 1200
```

> **Important**: This pipeline configuration replaces the old connector-level config. The dispatcher automatically manages all enabled connectors.

### Connector Configuration

The Gitea connector is configured via the management interface or API:

```json
{
  "id": "gitea",
  "name": "Gitea Connector",
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "gitea"
  }
}
```

### Explanation of Connector Config Parameters

| **Field**           | **Type**  | **Description**                                                      |
|---------------------|-----------|----------------------------------------------------------------------|
| `processor.enabled` | `boolean` | Enables the pipeline processor (required).                           |
| `processor.name`    | `string`  | Processor name, must be "gitea" (required).                          |

## Use the Gitea Connector

The Gitea Connector allows you to index repositories, issues, pull requests, and wikis from your Gitea instance.

### Configure Gitea Datasource

To configure your Gitea connection, you'll need to provide the URL of your instance, a Personal Access Token (PAT), and specify the scope of the data you wish to index.

`base_url`: (Optional) The base URL of your self-hosted Gitea instance (e.g., `https://gitea.example.com`).

`token`: A Gitea Personal Access Token (PAT). This is required for authentication.

`owner`: The username or organization name that owns the repositories you want to index (e.g., `infinilabs`).

`repos`: (Optional) A list of specific repository names to index. If left empty, the connector will index all repositories belonging to the `owner` that the `token` has access to.

`index_issues`: (Optional) A boolean (`true` or `false`) to enable indexing of issues. Defaults to `true`.

`index_pull_requests`: (Optional) A boolean (`true` or `false`) to enable indexing of pull requests. Defaults to `true`.

### Datasource Configuration

Each datasource has its own sync configuration and Gitea settings:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "My Gitea Projects",
    "type": "connector",
    "enabled": true,
    "connector": {
        "id": "gitea",
        "config": {
            "base_url": "https://gitea.example.com",
            "token": "YourPersonalAccessToken",
            "owner": "infinilabs",
            "repos": [
                "coco-server",
                "console"
            ],
            "index_issues": true,
            "index_pull_requests": true
        }
    },
    "sync": {
        "enabled": true,
        "interval": "30s"
    }
}'
```

## Supported Config Parameters for Gitea Connector

Below are the configuration parameters supported by the Gitea Connector:

### Datasource Config Parameters

| **Field**               | **Type**   | **Description**                                                                                          |
|-------------------------|------------|----------------------------------------------------------------------------------------------------------|
| `base_url`              | `string`   | Optional. The base URL of your self-hosted Gitea instance.                                               |
| `token`                 | `string`   | Your Gitea Personal Access Token (PAT) (required).                                                        |
| `owner`                 | `string`   | The username or organization name to scan (required).                                                     |
| `repos`                 | `[]string` | Optional. A list of repository names to index. If empty, all repositories for the owner will be indexed. |
| `index_issues`          | `boolean`  | Optional. Whether to index issues. Defaults to `true`.                                                   |
| `index_pull_requests`   | `boolean`  | Optional. Whether to index pull requests. Defaults to `true`.                                            |
| `sync.enabled`          | `boolean`  | Enable/disable syncing for this datasource.                                                              |
| `sync.interval`         | `string`   | Sync interval for this datasource (e.g., "30s", "5m", "1h").                                             |
