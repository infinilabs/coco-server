---
title: "Confluence"
weight: 30
---
# Confluence Connector

## Register Confluence Connector

```shell
curl -XPOST "http://localhost:9000/connector/" -d '{
  "name": "Confluence wiki Connector",
  "description": "Fetch Confluence Wiki pages and blogposts.",
  "category": "website",
  "icon": "/assets/icons/connector/confluence/icon.png",
  "tags": [
    "wiki",
    "storage",
    "docs",
    "web"
  ],
  "url": "http://coco.rs/connectors/confluence",
  "assets": {
    "icons": {
      "default": "/assets/icons/connector/confluence/icon.png"
    }
  },
  "processor": {
    "enabled": true,
    "name": "confluence"
  }
}'
```

## Use the Confluence Connector

The Confluence Connector allows you to index data from your Confluence wiki into your system. Follow these steps to set it up:

### Configure Confluence Client

To configure your Confluence connection, you need to provide several key parameters. The authentication method differs slightly between Confluence Cloud and Confluence Data Center/Server.

#### For Confluence Cloud
- endpoint: The URL of your Confluence Cloud instance (e.g., https://your-company.atlassian.net/wiki).
- username: The email address associated with your Atlassian account.
- token: The API Token you generate from your Atlassian account settings.
- 
#### For Confluence Data Center and Server
- endpoint: The URL of your self-hosted Confluence instance (e.g., https://confluence.your-domain.com).
- token: The Personal Access Token (PAT) generated from your user profile in Confluence. The username field is not required when using a PAT.

#### Common Parameters (for both versions)
- space: The key of the Confluence space you want to index (e.g., for a space named "Documentation", the key might be "DOCS").
- enable_blogposts: (Optional) A boolean (true or false) to enable indexing of blog posts within the space. Defaults to false.
- enable_attachments: (Optional) A boolean (true or false) to enable indexing of attachments (like PDFs, Word documents) within the space. Defaults to false.

### Datasource Configuration

Each datasource has its own sync configuration and Confluence settings:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "My Confluence Wiki",
    "type": "connector",
    "enabled": true,
    "connector": {
        "id": "confluence",
        "config": {
            "endpoint": "https://your-company.atlassian.net/wiki",
            "username": "your-email@example.com",
            "token": "your-confluence-api-token",
            "space": "DOCS",
            "enable_blogposts": false,
            "enable_attachments": false
        }
    },
    "sync": {
        "enabled": true,
        "interval": "5m"
    }
}'
```

## Supported Config Parameters for Confluence Connector

Below are the configuration parameters supported by the Confluence Connector:

### Datasource Config Parameters

| **Field**          | **Type**  | **Description**                                                                    |
|--------------------|-----------|------------------------------------------------------------------------------------|
| `endpoint`         | `string`  | The base URL of your Confluence instance (required).                               |
| `username`         | `string`  | Your Confluence username (email for Cloud). Not required for Data Center with PAT. |
| `token`            | `string`  | Your Confluence API Token (Cloud) or Personal Access Token (Data Center) (required). |
| `space`            | `string`  | The key of the Confluence space to index (required).                               |
| `enable_blogposts` | `boolean` | Optional. Set to true to index blog posts within the space. Defaults to false.    |
| `enable_attachments` | `boolean` | Optional. Set to true to index attachments within the space. Defaults to false. |
| `sync.enabled`     | `boolean` | Enable/disable syncing for this datasource.                                       |
| `sync.interval`    | `string`  | Sync interval for this datasource (e.g., "5m", "1h", "30s").                       |