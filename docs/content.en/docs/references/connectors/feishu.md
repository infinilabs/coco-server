---
title: "Feishu/Lark"
weight: 35
---

# Feishu/Lark Connector

The Feishu/Lark connector indexes cloud documents from Feishu and Lark, including documents, spreadsheets, mind notes, multi-dimensional tables, and knowledge bases.

## Features

- 🔍 **Smart Search**: Keyword-based search for cloud documents
- 📚 **Multiple Document Types**: Support for doc, sheet, slides, mindnote, bitable, file, docx, folder, shortcut
- 🔐 **Dual Authentication**: OAuth 2.0 and user access token authentication (choose one)
- ⚡ **Efficient Sync**: Scheduled and manual synchronization
- 🔄 **Recursive Search**: Automatically search folder contents recursively
- 🔄 **Token Auto-refresh**: OAuth authentication supports automatic refresh of access_token and refresh_token
- 🌐 **Dynamic Redirect**: Supports dynamic OAuth redirect URI construction for multi-environment deployment
- 🏗️ **Unified Architecture**: Feishu and Lark share base implementation with 95% code reuse

## Supported Platforms

### Feishu
- **Domain**: `open.feishu.cn` / `accounts.feishu.cn`
- **Connector ID**: `feishu`
- **Region**: Mainland China

### Lark
- **Domain**: `open.larksuite.com` / `accounts.larksuite.com`
- **Connector ID**: `lark`
- **Region**: Overseas regions

## Authentication Methods

The Feishu/Lark connector supports two authentication methods. **You must choose one**:

### 1. OAuth 2.0 Authentication (Recommended)

Uses OAuth flow to automatically obtain user access tokens with automatic refresh support and expiration time management.

#### Requirements
- `client_id`: Feishu/Lark app Client ID
- `client_secret`: Feishu/Lark app Client Secret
- `document_types`: List of document types to synchronize

#### Authentication Flow
1. User creates Feishu/Lark datasource with `client_id` and `client_secret`
2. Clicks "Connect" button, system redirects to Feishu/Lark authorization page
3. User completes authorization, system automatically obtains `access_token` and `refresh_token`
4. System automatically updates datasource configuration with complete OAuth information and expiration times

#### Advantages
- High security, no manual token management required
- Automatic access_token and refresh_token refresh support
- Automatic token expiration time management
- Automatic user information retrieval
- Compliant with OAuth 2.0 standards
- Supports multi-environment deployment (dynamic redirect URI)

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

## Feishu/Lark App Permission Configuration

### Required Permissions

The Feishu/Lark connector requires the following permissions to function properly:

| Permission | Permission Code | Description | Purpose |
|------------|-----------------|-------------|---------|
| **Cloud Document Access** | `drive:drive` | Access user's cloud documents, spreadsheets, slides, etc. | Read and index cloud document content |
| **Knowledge Base Retrieval** | `space:document:retrieve` | Retrieve documents from knowledge bases | Access knowledge bases and space documents |
| **Offline Access** | `offline_access` | Access resources when user is offline | Support background sync tasks |

### Permission Application Steps

#### Feishu App
1. **Login to Feishu Open Platform**
   - Visit [https://open.feishu.cn/](https://open.feishu.cn/)
   - Login with Feishu account

2. **Create Application**
   - Click "Create Application"
   - Select "Enterprise Self-built Application"
   - Fill in application name and description

3. **Apply for Permissions**
   - Go to "Permission Management" page
   - Search and add the three permissions above
   - Submit permission application

4. **Publish Application**
   - After completing permission application, publish application to enterprise
   - Record the app's `Client ID` and `Client Secret`

#### Lark App
1. **Login to Lark Open Platform**
   - Visit [https://open.larksuite.com/](https://open.larksuite.com/)
   - Login with Lark account

2. **Create Application**
   - Click "Create Application"
   - Select "Enterprise Self-built Application"
   - Fill in application name and description

3. **Apply for Permissions**
   - Go to "Permission Management" page
   - Search and add the three permissions above
   - Submit permission application

4. **Publish Application**
   - After completing permission application, publish application to enterprise
   - Record the app's `Client ID` and `Client Secret`

### Permission Description

- **`drive:drive`**: This is the core permission for accessing cloud documents, allowing the app to read user's documents, spreadsheets, slides, and other files
- **`space:document:retrieve`**: Used to access documents in knowledge bases and spaces, expanding document access scope
- **`offline_access`**: Allows the app to access resources when user is offline, which is crucial for background sync tasks

## Configuration Architecture

### Connector Level (Fixed Configuration)

#### Feishu Configuration
```yaml
connector:
  feishu:
    enabled: true
    interval: "30s"
    page_size: 100
    o_auth_config:
      auth_url: "https://accounts.feishu.cn/open-apis/authen/v1/authorize"
      token_url: "https://open.feishu.cn/open-apis/authen/v2/oauth/token"
      redirect_uri: "/connector/feishu/oauth_redirect"  # Dynamically built, supports multi-environment
```

#### Lark Configuration
```yaml
connector:
  lark:
    enabled: true
    interval: "30s"
    page_size: 100
    o_auth_config:
      auth_url: "https://accounts.larksuite.com/open-apis/authen/v1/authorize"
      token_url: "https://open.larksuite.com/open-apis/authen/v2/oauth/token"
      redirect_uri: "/connector/lark/oauth_redirect"  # Dynamically built, supports multi-environment
```

### Datasource Level (User Configuration)

#### Feishu Datasource
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

#### Lark Datasource
```yaml
datasource:
  name: "Lark Cloud Documents"
  connector:
    id: "lark"
    config:
      # Method 1: OAuth Authentication (Recommended)
      client_id: "cli_xxxxxxxxxxxxxxxx"
      client_secret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
      document_types: ["doc", "sheet", "slides", "mindnote", "bitable"]
      
      # Method 2: User Access Token (Alternative)
      # user_access_token: "u-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
      # document_types: ["doc", "sheet", "slides", "mindnote", "bitable"]
```

## Register Connectors

### Register Feishu Connector

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

### Register Lark Connector

```shell
curl -XPUT "http://localhost:9000/connector/lark?replace=true" -d '{
  "name": "Lark Connector",
  "description": "Index Lark cloud documents with OAuth 2.0 support.",
  "icon": "/assets/connector/lark/icon.png",
  "category": "cloud_storage",
  "tags": ["lark", "docs", "cloud"],
  "url": "http://coco.rs/connectors/lark",
  "assets": {"icons": {"default": "/assets/connector/lark/icon.png"}},
  "config": {
    "auth_url": "https://accounts.larksuite.com/open-apis/authen/v1/authorize",
    "token_url": "https://open.larksuite.com/open-apis/authen/v2/oauth/token",
    "redirect_uri": "/connector/lark/oauth_redirect"
  }
}'
```

> Use `feishu` or `lark` as the unique connector ID.

## Update coco-server config

### Feishu Configuration
```yaml
connector:
  feishu:
    enabled: true
    queue:
      name: indexing_documents
    interval: "30s"
    page_size: 100
```

### Lark Configuration
```yaml
connector:
  lark:
    enabled: true
    queue:
      name: indexing_documents
    interval: "30s"
    page_size: 100
```

## Create a Datasource

### Method 1: OAuth Authentication (Recommended)

#### Feishu Datasource
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

#### Lark Datasource
```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
  "name": "Lark Cloud Documents",
  "type": "connector",
  "connector": {
    "id": "lark",
    "config": {
      "client_id": "cli_xxxxxxxxxxxxxxxx",
      "client_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "document_types": ["doc", "sheet", "slides", "mindnote", "bitable", "file", "docx", "folder", "shortcut"]
    }
  }
}'
```

### Method 2: User Access Token Authentication

#### Feishu Datasource
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

#### Lark Datasource
```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
  "name": "Lark Cloud Documents",
  "type": "connector",
  "connector": {
    "id": "lark",
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
| `client_id` | string | Feishu/Lark app Client ID | OAuth Authentication |
| `client_secret` | string | Feishu/Lark app Client Secret | OAuth Authentication |
| `user_access_token` | string | User access token | Token Authentication |
| `document_types` | []string | List of document types to synchronize | Both methods |

### OAuth Auto-filled Fields

| Field | Type | Description | Source |
|-------|------|-------------|--------|
| `access_token` | string | Access token for API calls | Automatically obtained via OAuth |
| `refresh_token` | string | Refresh token for token updates | Automatically obtained via OAuth |
| `token_expiry` | string | Access token expiration time (RFC3339 format) | Automatically obtained via OAuth |
| `refresh_token_expiry` | string | Refresh token expiration time (RFC3339 format) | Automatically obtained via OAuth |
| `profile` | object | User information (ID, name, email, etc.) | Automatically obtained via OAuth |

### Sync Configuration

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `page_size` | int | 100 | Number of files per page |
| `interval` | string | "30s" | Synchronization interval |

## Supported Document Types

The Feishu/Lark connector supports the following cloud document types:

- **doc**: Feishu/Lark documents
- **sheet**: Feishu/Lark spreadsheets
- **slides**: Feishu/Lark presentations
- **mindnote**: Feishu/Lark mind notes
- **bitable**: Feishu/Lark multi-dimensional tables
- **file**: Regular files
- **docx**: Word documents
- **folder**: Folders (supports recursive search)
- **shortcut**: Shortcuts (directly use API returned URLs)

## Usage Instructions

### Method 1: OAuth Authentication (Recommended)

#### Step 1: Create Feishu/Lark App
1. Visit the corresponding open platform:
   - Feishu: [Feishu Open Platform](https://open.feishu.cn/)
   - Lark: [Lark Open Platform](https://open.larksuite.com/)
2. Create a new app, apply for the following permissions:
   - **`drive:drive`** - Cloud document access permission
   - **`space:document:retrieve`** - Knowledge base document retrieval permission
   - **`offline_access`** - Offline access permission
3. Record the app's `Client ID` and `Client Secret`

#### Step 2: Create Datasource
1. Create corresponding datasource (Feishu or Lark) in system management interface
2. Configure `client_id`, `client_secret`, and `document_types`
3. Save datasource configuration

#### Step 3: OAuth Authentication
1. Click "Connect" button
2. System redirects to corresponding authorization page
3. User completes authorization
4. System automatically updates datasource with OAuth token information and expiration times

### Method 2: User Access Token

#### Step 1: Obtain User Access Token
1. Log in to corresponding open platform
2. Obtain user access token

#### Step 2: Create Datasource
1. Create corresponding datasource in system management interface
2. Configure `user_access_token` and `document_types`
3. Save datasource configuration

## Technical Implementation

### Architecture Design

#### Refactored Architecture
- **Plugin Type Abstraction**: Uses `PluginType` enum to distinguish between Feishu and Lark
- **Dynamic API Configuration**: Dynamically selects API endpoints based on plugin type
- **Enhanced Base Plugin**: Adds plugin type management and API configuration functionality to base plugin
- **Maximum Code Reuse**: 95% of code is shared, only configuration and routes differ

#### Core Components
```go
// Plugin type definition
type PluginType string
const (
    PluginTypeFeishu PluginType = "feishu"
    PluginTypeLark   PluginType = "lark"
)

// API configuration structure
type APIConfig struct {
    BaseURL     string
    AuthURL     string
    TokenURL    string
    UserInfoURL string
    DriveURL    string
}

// Base plugin structure
type Plugin struct {
    // ... existing fields
    PluginType PluginType
    apiConfig  *APIConfig
}
```

#### Plugin Implementation
- **FeishuPlugin**: Inherits base plugin, sets `PluginTypeFeishu`
- **LarkPlugin**: Inherits base plugin, sets `PluginTypeLark`
- **Unified API Processing**: All API calls use dynamically configured endpoints

### OAuth Route Registration

#### Feishu Routes
- **Route Endpoints**: 
  - `GET /connector/feishu/connect` - OAuth authorization request
  - `GET /connector/feishu/oauth_redirect` - OAuth callback processing

#### Lark Routes
- **Route Endpoints**: 
  - `GET /connector/lark/connect` - OAuth authorization request
  - `GET /connector/lark/oauth_redirect` - OAuth callback processing

- **Authentication Requirements**: All OAuth endpoints require user login
- **Scope Configuration**: Uses `drive:drive space:document:retrieve offline_access` permission scope

### Token Lifecycle Management
- **Auto-refresh**: Automatically refreshes access_token when expired using refresh_token
- **Expiration Checking**: Checks expiration times for both access_token and refresh_token
- **Smart Handling**: Stops synchronization and logs errors if both tokens are expired
- **Data Persistence**: Automatically saves refreshed token information to datasource configuration

### Special Type Processing

#### Recursive Folder Search
The connector automatically searches folder contents recursively, ensuring all documents in subfolders are indexed.

## Important Notes

1. **Authentication Method Selection**: You must choose either OAuth authentication or user access token authentication, they cannot be used simultaneously
2. **OAuth Recommended**: OAuth authentication is recommended for higher security, automatic token refresh, and expiration time management
3. **Token Management**: When using user access tokens, manual token expiration management is required
4. **Permission Requirements**: Feishu/Lark apps need to apply for and obtain the following permissions:
   - `drive:drive` - Cloud document access permission
   - `space:document:retrieve` - Knowledge base retrieval permission
   - `offline_access` - Offline access permission
5. **API Limits**: Pay attention to Feishu/Lark API call frequency limits
6. **Platform Selection**: Choose the appropriate platform based on user region (Feishu for Mainland China, Lark for overseas regions)

## Troubleshooting

### Common Issues

1. **Authentication Failure**
   - Check if `client_id` and `client_secret` are correct
   - Confirm if Feishu/Lark app has applied for and obtained the following permissions:
     - `drive:drive` - Cloud document access permission
     - `space:document:retrieve` - Knowledge base retrieval permission
     - `offline_access` - Offline access permission
   - Check OAuth redirect URI configuration
   - Confirm if application has been published to enterprise

2. **Token Expiration**
   - OAuth Authentication: System automatically refreshes tokens, check if refresh_token is also expired
   - User Access Token: Manual token updates required

3. **Sync Failure**
   - Check network connectivity
   - Confirm if token is valid
   - View system logs for detailed error information
   - Check expiration times for both tokens

4. **OAuth Redirect Errors**
   - Confirm redirect URI in application configuration
   - Check if network environment supports dynamic URI construction
   - View redirect URI construction process in system logs

5. **Platform Selection Error**
   - Confirm user region
   - Check if application domain configuration is correct
   - Verify if API endpoints are accessible

### Log Debugging
The connector provides detailed logging, including:
- Each step of the OAuth flow
- Token refresh process
- Expiration time checking
- Error details and stack information
- Plugin type identification (`[feishu connector]` or `[lark connector]`)

Use logs to quickly locate and resolve issues.

## Extensibility

The refactored architecture supports easy addition of new plugin types:

1. **Define New Plugin Type**
   ```go
   const PluginTypeLarkInternational PluginType = "lark_international"
   ```

2. **Add API Configuration**
   ```go
   case PluginTypeLarkInternational:
       return &APIConfig{
           BaseURL: "https://open.larksuite.com",
           // ... other configurations
       }
   ```

3. **Create New Plugin**
   ```go
   type LarkInternationalPlugin struct {
       Plugin
   }
   
   func (this *LarkInternationalPlugin) Setup() {
       this.SetPluginType(PluginTypeLarkInternational)
       // other configurations handled automatically
   }
   ```

This design provides a solid foundation for future feature extensions and maintenance.
