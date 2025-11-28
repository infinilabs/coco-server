---
title: "Box"
weight: 65
---

# Box Cloud Storage Connector

## Register Box Connector

```shell
curl -XPUT "http://localhost:9000/connector/box?replace=true" -d '{
    "name": "Box Cloud Storage Connector",
    "description": "Index files and folders from Box, supporting both Free and Enterprise accounts with multi-user access.",
    "icon": "/assets/icons/connector/box/icon.png",
    "category": "cloud_storage",
    "path_hierarchy": true,
    "tags": [
        "box",
        "cloud_storage",
        "file_sharing"
    ],
    "url": "http://coco.rs/connectors/box",
    "assets": {
        "icons": {
            "default": "/assets/icons/connector/box/icon.png",
            "file": "/assets/icons/connector/box/file.png",
            "folder": "/assets/icons/connector/box/folder.png",
            "document": "/assets/icons/connector/box/document.png",
            "spreadsheet": "/assets/icons/connector/box/spreadsheet.png",
            "presentation": "/assets/icons/connector/box/presentation.png",
            "pdf": "/assets/icons/connector/box/pdf.png"
        }
    },
    "processor": {
        "enabled": true,
        "name": "box"
    }
}'
```

> Use `box` as a unique identifier, as it is a builtin connector.

## Use the Box Connector

The Box Connector allows you to index files and folders from Box cloud storage with support for both Free and Enterprise accounts.

### Features

- **Dual Account Support**: Works with both Box Free Account and Box Enterprise Account
- **Multi-User Access**: Enterprise accounts can index files from all users
- **Hierarchical Structure**: Maintains original folder structure with path hierarchy
- **Automatic Token Management**: Built-in token caching and auto-refresh
- **Recursive Folder Processing**: Automatically processes all subfolders
- **Enterprise User Categorization**: Files from different users are properly categorized
- **Metadata Extraction**: Extracts comprehensive file and folder metadata
- **Pipeline Integration**: Full pipeline-based architecture for efficient syncing

### Account Types

#### Box Free Account

**Authentication**: OAuth 2.0 Refresh Token Flow
- **Access Scope**: Current authenticated user's files only
- **Token Management**: Automatic refresh token rotation
- **Use Case**: Personal file indexing

#### Box Enterprise Account

**Authentication**: OAuth 2.0 Client Credentials Flow
- **Access Scope**: All users' files in the enterprise
- **Multi-User Support**: Automatically fetches files from all enterprise users
- **Use Case**: Organization-wide file indexing

### Setup Box Application

Before using this connector, you need to create a Box application and configure OAuth2.

#### 1. Create a Box Application

1. **Visit Box Developer Console**
   - Go to [Box Developer Console](https://app.box.com/developers/console)
   - Sign in with your Box account

2. **Create New App**
   - Click "Create New App"
   - Choose "Custom App"
   - Select authentication method:
     - For **Free Account**: Choose "Standard OAuth 2.0 (User Authentication)"
     - For **Enterprise Account**: Choose "OAuth 2.0 with JWT (Server Authentication)" or "Server Authentication (with Client Credentials Grant)"

3. **Configure Application**
   - Enter application name
   - Enter application description
   - Configure redirect URI (if using OAuth flow for token generation)

4. **Get Credentials**
   - Copy `Client ID` from Configuration page
   - Copy `Client Secret` from Configuration page
   - For Enterprise: Copy `Enterprise ID` from Admin Console

#### 2. Required Scopes

For proper functionality, the Box application needs:

**For Free Account:**
- Read files and folders
- User information

**For Enterprise Account:**
- Manage users
- Manage enterprise content
- Read all files and folders

#### 3. Application Approval

- For Enterprise accounts, the application must be approved by Box administrator
- Ensure the application is published and authorized

### Access Connector Settings

1. Navigate to the **Data Sources** section in your Coco dashboard
2. Create a new data source or edit an existing Box data source
3. Configure the required credentials based on your account type

> **⚠️ Important**: Before you can use the Box connector, you must configure the following required parameters based on your account type:
>
> **For Box Free Account:**
> - `is_enterprise`: Set to "box_free"
> - `client_id`: OAuth2 client ID from your Box application
> - `client_secret`: OAuth2 client secret from your Box application
> - `refresh_token`: OAuth refresh token (obtained through OAuth flow)
>
> **For Box Enterprise Account:**
> - `is_enterprise`: Set to "box_enterprise"
> - `client_id`: OAuth2 client ID from your Box application
> - `client_secret`: OAuth2 client secret from your Box application
> - `enterprise_id`: Your Box Enterprise ID

### Datasource Configuration

#### Box Free Account Example

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "My Box Files",
    "type": "connector",
    "enabled": true,
    "connector": {
        "id": "box",
        "config": {
            "is_enterprise": "box_free",
            "client_id": "your_client_id",
            "client_secret": "your_client_secret",
            "refresh_token": "your_refresh_token",
            "concurrent_downloads": 15
        }
    },
    "sync": {
        "enabled": true,
        "interval": "30s"
    }
}'
```

#### Box Enterprise Account Example

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "Company Box Files",
    "type": "connector",
    "enabled": true,
    "connector": {
        "id": "box",
        "config": {
            "is_enterprise": "box_enterprise",
            "client_id": "your_client_id",
            "client_secret": "your_client_secret",
            "enterprise_id": "12345",
            "concurrent_downloads": 15
        }
    },
    "sync": {
        "enabled": true,
        "interval": "5m"
    }
}'
```

### Datasource Config Parameters

| **Field**               | **Type**  | **Description**                                                                   | **Required** | **Account Type** |
|-------------------------|-----------|-----------------------------------------------------------------------------------|--------------|------------------|
| `is_enterprise`         | `string`  | Account type: "box_free" or "box_enterprise"                                      | Yes          | Both             |
| `client_id`             | `string`  | Box application Client ID                                                         | Yes          | Both             |
| `client_secret`         | `string`  | Box application Client Secret                                                     | Yes          | Both             |
| `refresh_token`         | `string`  | OAuth refresh token (for Free account)                                            | Yes          | Free only        |
| `enterprise_id`         | `string`  | Box Enterprise ID (for Enterprise account)                                        | Yes          | Enterprise only  |
| `concurrent_downloads`  | `int`     | Maximum concurrent downloads (default: 15)                                        | No           | Both             |
| `sync.enabled`          | `boolean` | Enable/disable syncing for this datasource                                        | No           | Both             |
| `sync.interval`         | `string`  | Sync interval (e.g., "30s", "5m", "1h")                                          | No           | Both             |

## File Hierarchy

### Box Free Account

Files are organized directly from root:

```
/
├── Documents/
│   ├── report.pdf
│   └── 2024/
│       └── annual-report.pdf
├── Photos/
│   └── image.jpg
└── Shared/
    └── presentation.pptx
```

### Box Enterprise Account

Files are organized by user name to separate content from different users:

```
/
├── John Doe/
│   ├── Documents/
│   │   └── report.pdf
│   └── Photos/
│       └── image.jpg
├── Jane Smith/
│   ├── Documents/
│   │   └── report.pdf
│   └── Reports/
│       └── sales.xlsx
└── Bob Johnson/
    └── Presentations/
        └── deck.pptx
```

**Key Points:**
- Each user's files are under their name category
- Document IDs include user ID to avoid conflicts
- Metadata includes `user_id` field for filtering

## Advanced Features

### Automatic Token Management

The connector implements intelligent token management:

- **Token Caching**: In-memory cache with thread-safe operations
- **Expiry Buffer**: Refreshes tokens 5 minutes before expiry
- **Automatic Refresh**: Transparent token refresh on expiration
- **401 Retry**: Automatic re-authentication on unauthorized errors
- **Refresh Token Rotation**: Supports refresh token rotation (Free account)

### Multi-User Support (Enterprise)

For Enterprise accounts, the connector:

1. **Fetches All Users**: Automatically retrieves all users in the enterprise
2. **Per-User Processing**: Processes files for each user independently
3. **As-User Header**: Uses `as-user` header to access files as specific users
4. **User Categorization**: Organizes files under user names in hierarchy
5. **Unique Document IDs**: Generates unique IDs including user ID to avoid conflicts

### Metadata Extraction

The connector extracts comprehensive metadata:

**File Metadata:**
- File ID, Name, Type, Size
- Creation and modification timestamps
- Description and status
- Creator, modifier, and owner information
- Parent folder information
- ETag and sequence ID
- URLs (direct, download, thumbnail)
- Shared link information

**Folder Metadata:**
- Folder ID, Name, Type
- Creation and modification timestamps
- Size and hierarchy information
- Platform identifier

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   - **Free Account**: Verify `client_id`, `client_secret`, and `refresh_token` are correct
   - **Enterprise Account**: Verify `client_id`, `client_secret`, and `enterprise_id` are correct
   - Check if Box application is approved and published
   - Ensure application has required scopes

2. **Token Expired**
   - System automatically refreshes tokens
   - **Free Account**: Check if refresh_token is still valid
   - **Enterprise Account**: Verify application credentials haven't changed
   - Review token expiry settings

3. **No Files Found**
   - Check user permissions in Box
   - Verify application has file access permissions
   - **Enterprise**: Ensure users have files in their accounts
   - Check folder access permissions

4. **Multi-User Issues (Enterprise)**
   - Verify application has "Manage Users" permission
   - Check if users are active in the enterprise
   - Ensure `as-user` header is supported by your application type

5. **Sync Failures**
   - Check network connectivity to `https://api.box.com`
   - Verify API rate limits aren't exceeded
   - Review system logs for detailed error messages
   - Check datasource sync interval settings

### Debug Logging

The connector provides detailed logging:
- `[box connector]`: Main connector operations
- `[box client]`: API client operations
- Authentication process and token refresh
- User enumeration (Enterprise)
- File and folder processing
- API requests and errors

Use logs to quickly identify and resolve issues.

## Notes

1. **Account Type Selection**: Must specify either "box_free" or "box_enterprise"
2. **Different Credentials**: Free and Enterprise accounts require different configuration
3. **Refresh Token Requirement**: Free accounts must have a valid refresh_token
4. **Enterprise ID Requirement**: Enterprise accounts must have a valid enterprise_id
5. **Multi-User Automatic**: Enterprise accounts automatically fetch files from all users
6. **Token Auto-Refresh**: All tokens are automatically managed and refreshed
7. **Content Extraction**: File content extraction is handled by coco-server framework
8. **API Rate Limits**: Be aware of Box API rate limits (typically 1000 requests/minute)
9. **File Size Limits**: Large files may be excluded based on framework configuration
10. **Hierarchical Path**: Connector preserves original folder structure with `/` as root

## API Endpoints Used

The connector uses the following Box API endpoints:

| Endpoint | Purpose | Account Type |
|----------|---------|--------------|
| `/oauth2/token` | Authentication and token refresh | Both |
| `/2.0/users/me` | Ping test and user info | Both |
| `/2.0/users` | Fetch enterprise users | Enterprise only |
| `/2.0/folders/{id}/items` | List folder contents | Both |

All API calls include automatic retry on 401 errors and support for the `as-user` header in Enterprise accounts.

