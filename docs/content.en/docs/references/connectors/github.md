---
title: "GitHub"
weight: 60
---
# GitHub Connector

## Register GitHub Connector

```shell
curl -XPUT "http://localhost:9000/connector/github?replace=true" -d '{
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
      "default" : "/assets/icons/connector/github/icon.png",
      "repository" : "/assets/icons/connector/github/repository.png",
      "issue" : "/assets/icons/connector/github/issue.png",
      "pull_request" : "/assets/icons/connector/github/pull_request.png"
    }
  }
}'
```

> Use `github` as a unique identifier, as it is a builtin connector.

## Update coco-server's config

Below is an example configuration for enabling the GitHub Connector in coco-server:

```yaml
connector:
  github:
    enabled: true
    queue:
      name: indexing_documents
    interval: 30s
```

### Explanation of Config Parameters

| **Field**    | **Type**  | **Description**                                                                           |
|--------------|-----------|-------------------------------------------------------------------------------------------|
| `enabled`    | `boolean` | Enables or disables the GitHub connector. Set to`true` to activate it.                    |
| `interval`   | `string`  | Specifies the time interval (e.g., `30s`) at which the connector will check for updates.  |
| `queue.name` | `string`  | Defines the name of the queue where indexing tasks will be added.                         |

## Use the GitHub Connector

The GitHub Connector allows you to index repositories, issues, and pull requests from your GitHub account or organization.

### Configure GitHub Datasource

To configure your GitHub connection, you'll need to provide a Personal Access Token (PAT) and specify the scope of the data you wish to index.

`token`: A GitHub Personal Access Token (PAT) with at least the `repo` scope. This is required for authentication.

`owner`: The username or organization name that owns the repositories you want to index (e.g., `infinilabs`).

`repos`: (Optional) A list of specific repository names to index. If left empty, the connector will index all repositories belonging to the `owner` that the `token` has access to.

`index_issues`: (Optional) A boolean (`true` or `false`) to enable indexing of issues. Defaults to `true`.

`index_pull_requests`: (Optional) A boolean (`true` or `false`) to enable indexing of pull requests. Defaults to `true`.

### Example Request

Here is an example request to configure the GitHub Connector:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
    "name":"My Organization Repos",
    "type":"connector",
    "connector":{
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
    }
}'
```

## Supported Config Parameters for GitHub Connector

| **Field**             | **Type**   | **Description**                                                                                                |
|-----------------------|------------|----------------------------------------------------------------------------------------------------------------|
| `token`               | `string`   | Required. Your GitHub Personal Access Token (PAT) with `repo` scope.                                           |
| `owner`               | `string`   | Required. The username or organization name to scan.                                                           |
| `repos`               | `[]string` | Optional. A list of repository names to index. If empty, all repositories for the owner will be indexed.       |
| `index_issues`        | `boolean`  | Optional. Whether to index issues. Defaults to `true`.                                                         |
| `index_pull_requests` | `boolean`  | Optional. Whether to index pull requests. Defaults to `true`.                                                  |

