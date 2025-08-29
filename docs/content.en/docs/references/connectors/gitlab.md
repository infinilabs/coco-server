---
title: "GitLab"
weight: 61
---
# GitLab Connector

## Register GitLab Connector

```shell
curl -XPUT "http://localhost:9000/connector/gitlab?replace=true" -d '{
  "name": "GitLab Connector",
  "description": "Fetch repositories, issues, merge requests, wikis, and snippets from GitLab.",
  "icon": "/assets/icons/connector/gitlab/icon.png",
  "category": "website",
  "tags": [
    "git", 
    "code", 
    "vcs", 
    "website"
  ],
  "url": "http://coco.rs/connectors/gitlab",
  "assets": {
    "icons": {
      "default" : "/assets/icons/connector/gitlab/icon.png",
      "repository" : "/assets/icons/connector/gitlab/repository.png",
      "issue" : "/assets/icons/connector/gitlab/issue.png",
      "merge_request" : "/assets/icons/connector/gitlab/merge_request.png",
      "wiki" : "/assets/icons/connector/gitlab/wiki.png",
      "snippet" : "/assets/icons/connector/gitlab/snippet.png"
    }
  }
}'
```

> Use `gitlab` as a unique identifier, as it is a builtin connector.

## Update coco-server's config

Below is an example configuration for enabling the GitLab Connector in coco-server:

```yaml
connector:
  gitlab:
    enabled: true
    queue:
      name: indexing_documents
    interval: 30s
```

### Explanation of Config Parameters

| **Field**    | **Type**  | **Description**                                                                           |
|--------------|-----------|-------------------------------------------------------------------------------------------|
| `enabled`    | `boolean` | Enables or disables the GitLab connector. Set to`true` to activate it.                    |
| `interval`   | `string`  | Specifies the time interval (e.g., `30s`) at which the connector will check for updates.  |
| `queue.name` | `string`  | Defines the name of the queue where indexing tasks will be added.                         |

## Use the GitLab Connector

The GitLab Connector allows you to index repositories, issues, merge requests, wikis, and snippets from your GitLab account or group.

### Configure GitLab Datasource

To configure your GitLab connection, you'll need to provide a Personal Access Token (PAT) and specify the scope of the data you wish to index.

`base_url`: (Optional) The base URL of your self-hosted GitLab instance (e.g., `https://gitlab.example.com`). If left empty, it will default to `https://gitlab.com`.

`token`: A GitLab Personal Access Token (PAT) with at least the `api` scope. This is required for authentication.

`owner`: The username or group name that owns the repositories you want to index (e.g., `infinilabs`).

`repos`: (Optional) A list of specific repository names to index. If left empty, the connector will index all repositories belonging to the `owner` that the `token` has access to.

`index_issues`: (Optional) A boolean (`true` or `false`) to enable indexing of issues. Defaults to `true`.

`index_merge_requests`: (Optional) A boolean (`true` or `false`) to enable indexing of merge requests. Defaults to `true`.

`index_wikis`: (Optional) A boolean (`true` or `false`) to enable indexing of wikis. Defaults to `true`.

`index_snippets`: (Optional) A boolean (`true` or `false`) to enable indexing of snippets. Defaults to `true`.

### Example Request

Here is an example request to configure the GitLab Connector:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
    "name":"My Organization Repos",
    "type":"connector",
    "connector":{
        "id": "gitlab",
        "config": {
            "base_url": "https://gitlab.com",
            "token": "YourPersonalAccessToken",
            "owner": "infinilabs",
            "repos": [
                "coco-server",
                "console"
            ],
            "index_issues": true,
            "index_merge_requests": true,
            "index_wikis": true,
            "index_snippets": true
        }
    }
}'
```

## Supported Config Parameters for GitLab Connector

| **Field**               | **Type**   | **Description**                                                                                          |
|-------------------------|------------|----------------------------------------------------------------------------------------------------------|
| `base_url`              | `string`   | Optional. The base URL of your self-hosted GitLab instance.                                              |
| `token`                 | `string`   | Required. Your GitLab Personal Access Token (PAT) with `api` scope.                                      |
| `owner`                 | `string`   | Required. The username or group name to scan.                                                            |
| `repos`                 | `[]string` | Optional. A list of repository names to index. If empty, all repositories for the owner will be indexed. |
| `index_issues`          | `boolean`  | Optional. Whether to index issues. Defaults to `true`.                                                   |
| `index_merge_requests`  | `boolean`  | Optional. Whether to index merge requests. Defaults to `true`.                                           |
| `index_wikis`           | `boolean`  | Optional. Whether to index wikis. Defaults to `true`.                                                    |
| `index_snippets`        | `boolean`  | Optional. Whether to index snippets. Defaults to `true`.                                                 |
