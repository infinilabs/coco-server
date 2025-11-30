---
title: "Jira"
weight: 31
---
# Jira Connector

## Register Jira Connector

```shell
curl -XPUT "http://localhost:9000/connector/" -d '{
  "name": "Jira Connector",
  "description": "Fetch Jira issues, comments, and attachments from your Jira projects.",
  "category": "project_management",
  "icon": "/assets/icons/connector/jira/icon.png",
  "tags": [
    "project",
    "issue_tracking",
    "agile",
    "collaboration"
  ],
  "url": "http://coco.rs/connectors/jira",
  "assets": {
    "icons": {
      "default": "/assets/icons/connector/jira/icon.png"
    }
  },
  "processor": {
    "enabled": true,
    "name": "jira"
  }
}'
```

## Use the Jira Connector

The Jira Connector allows you to index issues, comments, and attachments from your Jira projects into your system. Follow these steps to set it up:

### Configure Jira Client

To configure your Jira connection, you need to provide several key parameters. The authentication method differs between Jira Cloud and Jira Data Center/Server.

#### For Jira Cloud
- **endpoint**: The URL of your Jira Cloud instance (e.g., https://your-company.atlassian.net).
- **username**: The email address associated with your Atlassian account.
- **token**: The API Token you generate from your Atlassian account settings (go to https://id.atlassian.com/manage-profile/security/api-tokens).

#### For Jira Data Center and Server
- **endpoint**: The URL of your self-hosted Jira instance (e.g., https://jira.your-domain.com or https://your-domain.com/jira for Apache Jira).
- **token**: The Personal Access Token (PAT) generated from your user profile in Jira. The username field is not required when using a PAT.

#### For Anonymous Access (Public Jira Instances)
- **endpoint**: The URL of the public Jira instance.
- Leave **username** and **token** empty for anonymous access (only works if the Jira instance allows public access).

#### Common Parameters (for all versions)
- **project_key**: The key of the Jira project you want to index (e.g., for a project named "Development", the key might be "DEV").
- **index_comments**: (Optional) A boolean (true or false) to include issue comments in the indexed content. Defaults to false.
- **index_attachments**: (Optional) A boolean (true or false) to index attachments as separate documents. Defaults to false.

### Datasource Configuration

Each datasource has its own sync configuration and Jira settings:

#### Example: Jira Cloud with Basic Authentication
```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "My Jira Cloud Project",
    "type": "connector",
    "enabled": true,
    "connector": {
        "id": "jira",
        "config": {
            "endpoint": "https://your-company.atlassian.net",
            "username": "your-email@example.com",
            "token": "your-jira-api-token",
            "project_key": "DEV",
            "index_comments": true,
            "index_attachments": false
        }
    },
    "sync": {
        "enabled": true,
        "interval": "5m"
    }
}'
```

#### Example: Jira Data Center/Server with Personal Access Token
```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "My Self-Hosted Jira",
    "type": "connector",
    "enabled": true,
    "connector": {
        "id": "jira",
        "config": {
            "endpoint": "https://jira.your-domain.com",
            "token": "your-personal-access-token",
            "project_key": "PROJ",
            "index_comments": true,
            "index_attachments": true
        }
    },
    "sync": {
        "enabled": true,
        "interval": "10m"
    }
}'
```

## Supported Config Parameters for Jira Connector

Below are the configuration parameters supported by the Jira Connector:

### Datasource Config Parameters

| **Field**              | **Type**  | **Description**                                                                                                    |
|------------------------|-----------|--------------------------------------------------------------------------------------------------------------------|
| `endpoint`             | `string`  | The base URL of your Jira instance (required). Examples: https://company.atlassian.net or https://jira.company.com |
| `username`             | `string`  | Your Jira username (email for Cloud). Not required for Data Center/Server with PAT or anonymous access.            |
| `token`                | `string`  | Your Jira API Token (Cloud) or Personal Access Token (Data Center/Server). Optional for anonymous access.          |
| `project_key`          | `string`  | The key of the Jira project to index (required). Example: "DEV", "PROJ", "INFRA".                                  |
| `index_comments`       | `boolean` | Optional. Set to true to include issue comments in the indexed content. Defaults to false.                         |
| `index_attachments`    | `boolean` | Optional. Set to true to index attachments as separate documents. Defaults to false.                               |
| `sync.enabled`         | `boolean` | Enable/disable syncing for this datasource.                                                                        |
| `sync.interval`        | `string`  | Sync interval for this datasource (e.g., "5m", "1h", "30s").                                                       |