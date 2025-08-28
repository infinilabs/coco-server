---
title: "Feishu"
weight: 35
---

# Feishu Connector

The Feishu connector indexes cloud documents from Feishu, including documents, spreadsheets, mind notes, multi-dimensional tables, and knowledge bases.

## Features

- 🔍 **Smart Search**: Keyword-based search for cloud documents
- 📚 **Multiple Document Types**: Support for doc, sheet, slides, mindnote, bitable, file, docx, folder, shortcut
- 🔐 **Dual Authentication**: OAuth 2.0 and user access token authentication (choose one)
- ⚡ **Efficient Sync**: Scheduled and manual synchronization
- 🔄 **Recursive Search**: Automatically search folder contents recursively

## Authentication Methods

The Feishu connector supports two authentication methods. **You must choose one**:

### 1. OAuth 2.0 Authentication (Recommended)

Uses OAuth flow to automatically obtain user access tokens with automatic refresh support.

#### Requirements
- `client_id`: Feishu app Client ID
- `client_secret`: Feishu app Client Secret
- `document_types`: List of document types to synchronize

#### Authentication Flow
1. User creates Feishu datasource with `client_id` and `client_secret`
2. Clicks "Connect" button, system redirects to Feishu authorization page
3. User completes authorization, system automatically obtains `access_token` and `refresh_token`
4. System automatically updates datasource configuration with complete OAuth information

#### Advantages
- High security, no manual token management required
- Automatic token refresh support
- Automatic user information retrieval
- Compliant with OAuth 2.0 standards

### 2. User Access Token Authentication (Alternative)

Directly uses user access tokens, suitable for scenarios with existing tokens.

#### Requirements
- `user_access_token`: User's access token
- `document_types`: List of document types to synchronize

#### Use Cases
- Already have valid user access tokens
- Don't want to use OAuth flow
- Testing or development environments

#### Considerations
- Manual token expiration management required
- Manual token updates needed after expiration
- Relatively lower security

## Configuration Architecture

### Connector Level (Fixed Configuration)
```yaml
connector:
  feishu:
    enabled: true
    interval: "30s"
    page_size: 100
    oauth:
      auth_url: "https://accounts.feishu.cn/open-apis/authen/v1/authorize"
      token_url: "https://open.feishu.cn/open-apis/authen/v2/oauth/token"
      redirect_uri: "/connector/feishu/oauth_redirect"
```

### Datasource Level (User Configuration)
```yaml
datasource:
  name: "Feishu Cloud Documents"
  connector:
    id: "feishu"
    config:
      # Method 1: OAuth Authentication (Recommended)
      client_id: "cli_xxxxxxxxxxxxxxxx"
      client_secret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
      document_types: ["doc", "sheet", "slides", "mindnote", "bitable"]
      
      # Method 2: User Access Token (Alternative)
      # user_access_token: "u-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
      # document_types: ["doc", "sheet", "slides", "mindnote", "bitable"]
```

## Register Feishu Connector

```shell
curl -XPUT "http://localhost:9000/connector/feishu?replace=true" -d '{
  "name": "Feishu Connector",
  "description": "Index Feishu cloud documents with OAuth 2.0 support.",
  "icon": "/assets/connector/feishu/icon.png",
  "category": "cloud_storage",
  "tags": ["feishu", "docs", "cloud"],
  "url": "http://coco.rs/connectors/feishu",
  "assets": {"icons": {"default": "/assets/connector/feishu/icon.png"}},
  "config": {
    "auth_url": "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
    "token_url": "https://open.feishu.cn/open-apis/authen/v2/oauth/token",
    "redirect_uri": "/connector/feishu/oauth_redirect"
  }
}'
```

> Use `feishu` as the unique connector ID.

## Update coco-server config

```yaml
connector:
  feishu:
    enabled: true
    queue:
      name: indexing_documents
    interval: "30s"
    page_size: 100
```

## Create a Datasource

### Method 1: OAuth Authentication (Recommended)

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
  "name": "Feishu Cloud Documents",
  "type": "connector",
  "connector": {
    "id": "feishu",
    "config": {
      "client_id": "cli_xxxxxxxxxxxxxxxx",
      "client_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "document_types": ["doc", "sheet", "slides", "mindnote", "bitable", "file", "docx", "folder", "shortcut"]
    }
  }
}'
```

### Method 2: User Access Token Authentication

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
  "name": "Feishu Cloud Documents",
  "type": "connector",
  "connector": {
    "id": "feishu",
    "config": {
      "user_access_token": "u-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "document_types": ["doc", "sheet", "slides", "mindnote", "bitable", "file", "docx", "folder", "shortcut"]
    }
  }
}'
```

## Configuration Parameters

### Required Parameters

| Field | Type | Description | Authentication Method |
|-------|------|-------------|---------------------|
| `client_id` | string | Feishu app Client ID | OAuth Authentication |
| `client_secret` | string | Feishu app Client Secret | OAuth Authentication |
| `user_access_token` | string | User access token | Token Authentication |
| `document_types` | []string | List of document types to synchronize | Both methods |

### OAuth Auto-filled Fields

| Field | Type | Description | Source |
|-------|------|-------------|--------|
| `access_token` | string | Access token for API calls | Automatically obtained via OAuth |
| `refresh_token` | string | Refresh token for token updates | Automatically obtained via OAuth |
| `token_expiry` | string | Token expiration time (RFC3339 format) | Automatically obtained via OAuth |
| `profile` | object | User information (ID, name, email, etc.) | Automatically obtained via OAuth |

### Sync Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `page_size` | int | 100 | Number of files per page |
| `interval` | string | "30s" | Synchronization interval |

## Supported Document Types

The Feishu connector supports the following cloud document types:

- **doc**: Feishu documents
- **sheet**: Feishu spreadsheets
- **slides**: Feishu presentations
- **mindnote**: Feishu mind notes
- **bitable**: Feishu multi-dimensional tables
- **file**: Regular files
- **docx**: Word documents
- **folder**: Folders (supports recursive search)
- **shortcut**: Shortcuts (directly use API returned URLs)

## Usage Instructions

### Method 1: OAuth Authentication (Recommended)

#### Step 1: Create Feishu App
1. Visit [Feishu Open Platform](https://open.feishu.cn/)
2. Create a new app, apply for `drive:read` permission
3. Record the app's `Client ID` and `Client Secret`

#### Step 2: Create Datasource
1. Create Feishu datasource in system management interface
2. Configure `client_id`, `client_secret`, and `document_types`
3. Save datasource configuration

#### Step 3: OAuth Authentication
1. Click "Connect" button
2. System redirects to Feishu authorization page
3. User completes authorization
4. System automatically updates datasource with OAuth token information

### Method 2: User Access Token

#### Step 1: Obtain User Access Token
1. Log in to Feishu Open Platform
2. Obtain user access token

#### Step 2: Create Datasource
1. Create Feishu datasource in system management interface
2. Configure `user_access_token` and `document_types`
3. Save datasource configuration

## Technical Implementation

### Architecture Design
- **BasePlugin Inheritance**: Inherits from `connectors.BasePlugin`
- **Modular Design**: OAuth processing logic separated into independent `api.go` file
- **Type Safety**: Uses Go's type system to ensure configuration and data type safety

### OAuth Route Registration
- **Route Endpoints**: 
  - `GET /connector/feishu/connect` - OAuth authorization request
  - `GET /connector/feishu/oauth_redirect` - OAuth callback processing
- **Authentication Requirements**: All OAuth endpoints require user login

### Special Type Processing

#### Recursive Folder Search
The connector automatically searches folder contents recursively, ensuring all documents in subfolders are indexed.

## Important Notes

1. **Authentication Method Selection**: You must choose either OAuth authentication or user access token authentication, they cannot be used simultaneously
2. **OAuth Recommended**: OAuth authentication is recommended for higher security and automatic token refresh support
3. **Token Management**: When using user access tokens, manual token expiration management is required
4. **Permission Requirements**: Feishu apps need `drive:read` permission to access cloud documents
5. **API Limits**: Pay attention to Feishu API call frequency limits

## Troubleshooting

### Common Issues

1. **Authentication Failure**
   - Check if `client_id` and `client_secret` are correct
   - Confirm if Feishu app has `drive:read` permission

2. **Token Expiration**
   - OAuth Authentication: System automatically refreshes tokens
   - User Access Token: Manual token updates required

3. **Sync Failure**
   - Check network connectivity
   - Confirm if token is valid
   - View system logs for detailed error information
