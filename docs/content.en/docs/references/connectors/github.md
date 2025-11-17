---
title: "GitHub"
weight: 60
---
# GitHub Connector

## Register GitHub Connector

```shell
curl -XPOST "http://localhost:9000/connector/" -d '{
  "name": "GitHub Connector",
  "description": "Fetch repositories, issues, and pull requests from GitHub.",
  "icon": "/assets/icons/connector/github/icon.png",
  "category": "website",
  "tags": [
    "git",
    "code",
    "vcs",
    "website"
  ],
  "url": "http://coco.rs/connectors/github",
  "assets": {
    "icons": {
      "default": "/assets/icons/connector/github/icon.png",
      "repository": "/assets/icons/connector/github/repository.png",
      "issue": "/assets/icons/connector/github/issue.png",
      "pull_request": "/assets/icons/connector/github/pull_request.png"
    }
  },
  "processor": {
    "enabled": true,
    "name": "github"
  }
}'
```

## Use the GitHub Connector

The GitHub Connector allows you to index repositories, issues, and pull requests from your GitHub account or organization.

### Configure GitHub Datasource

To configure your GitHub connection, you'll need to provide a Personal Access Token (PAT) and specify the scope of the data you wish to index.

`token`: A GitHub Personal Access Token (PAT) with at least the `repo` scope. This is required for authentication.

`owner`: The username or organization name that owns the repositories you want to index (e.g., `infinilabs`).

`repos`: (Optional) A list of specific repository names to index. If left empty, the connector will index all repositories belonging to the `owner` that the `token` has access to.

`index_issues`: (Optional) A boolean (`true` or `false`) to enable indexing of issues. Defaults to `true`.

`index_pull_requests`: (Optional) A boolean (`true` or `false`) to enable indexing of pull requests. Defaults to `true`.

### Datasource Configuration

Each datasource has its own sync configuration and GitHub settings:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "My Organization Repos",
    "type": "connector",
    "enabled": true,
    "connector": {
        "id": "github",
        "config": {
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

## Supported Config Parameters for GitHub Connector

Below are the configuration parameters supported by the GitHub Connector:

### Datasource Config Parameters

| **Field**             | **Type**   | **Description**                                                                                                  |
|-----------------------|------------|------------------------------------------------------------------------------------------------------------------|
| `token`               | `string`   | Your GitHub Personal Access Token (PAT) with `repo` scope (required).                                            |
| `owner`               | `string`   | The username or organization name to scan (required).                                                            |
| `repos`               | `[]string` | Optional. A list of repository names to index. If empty, all repositories for the owner will be indexed.         |
| `index_issues`        | `boolean`  | Optional. Whether to index issues. Defaults to `true`.                                                           |
| `index_pull_requests` | `boolean`  | Optional. Whether to index pull requests. Defaults to `true`.                                                    |
| `sync.enabled`        | `boolean`  | Enable/disable syncing for this datasource.                                                                      |
| `sync.interval`       | `string`   | Sync interval for this datasource (e.g., "30s", "5m", "1h").                                                     |

