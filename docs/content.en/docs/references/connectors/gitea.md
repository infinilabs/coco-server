---
title: "Gitea"
weight: 62
---
# Gitea Connector

## Register Gitea Connector

```shell
cURL -XPUT "http://localhost:9000/connector/gitea?replace=true" -d '{
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
      "default" : "/assets/icons/connector/gitea/icon.png",
      "repository" : "/assets/icons/connector/gitea/repository.png",
      "issue" : "/assets/icons/connector/gitea/issue.png",
      "pull_request" : "/assets/icons/connector/gitea/pull_request.png"
    }
  }
}'
```

> Use `gitea` as a unique identifier, as it is a builtin connector.

## Update coco-server's config

Below is an example configuration for enabling the Gitea Connector in coco-server:

```yaml
connector:
  gitea:
    enabled: true
    queue:
      name: indexing_documents
    interval: 30s
```

### Explanation of Config Parameters

| **Field**    | **Type**  | **Description**                                                                            |
|--------------|-----------|--------------------------------------------------------------------------------------------|
| `enabled`    | `boolean` | Enables or disables the Gitea connector. Set to`true` to activate it.                      |
| `interval`   | `string`  | Specifies the time interval (e.g., `30s`) at which the connector will check for updates.   |
| `queue.name` | `string`  | Defines the name of the queue where indexing tasks will be added.                          |

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

### Example Request

Here is an example request to configure the Gitea Connector:

```shell
cURL -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
    "name":"My Gitea Projects",
    "type":"connector",
    "connector":{
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
            "index_pull_requests": true,
            "index_wikis": true
        }
    }
}'
```

## Supported Config Parameters for Gitea Connector

| **Field**               | **Type**   | **Description**                                                                                          |
|-------------------------|------------|----------------------------------------------------------------------------------------------------------|
| `base_url`              | `string`   | Optional. The base URL of your self-hosted Gitea instance.                                               |
| `token`                 | `string`   | Required. Your Gitea Personal Access Token (PAT).                                                        |
| `owner`                 | `string`   | Required. The username or organization name to scan.                                                     |
| `repos`                 | `[]string` | Optional. A list of repository names to index. If empty, all repositories for the owner will be indexed. |
| `index_issues`          | `boolean`  | Optional. Whether to index issues. Defaults to `true`.                                                   |
| `index_pull_requests`   | `boolean`  | Optional. Whether to index pull requests. Defaults to `true`.                                            |
