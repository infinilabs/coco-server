# Box Cloud Storage Connector

Box cloud storage connector is used to index files and folders from Box, supporting both Free and Enterprise accounts.

## Features

- 🔍 **Smart Search**: Search files and folders by keywords
- 📁 **Hierarchical Structure**: Maintains original folder structure with path hierarchy
- 🔐 **Dual Account Support**: Supports both Box Free Account and Box Enterprise Account
- 👥 **Multi-User Support**: Enterprise accounts can index files from all users
- ⚡ **Efficient Sync**: Pipeline-based architecture with unified scheduler
- 🔄 **Recursive Folder Processing**: Automatically processes all subfolders
- 🔄 **Automatic Token Refresh**: Built-in token caching and auto-refresh mechanism
- 🏗️ **Unified Architecture**: Follows coco-server connector standards

## Account Types

### Box Free Account
- **Access Scope**: Current authenticated user's files only
- **Required Credentials**: 
  - `client_id`
  - `client_secret`

### Box Enterprise Account
- **Authentication**: OAuth 2.0 client credentials
- **Access Scope**: All users' files in the enterprise
- **Required Credentials**:
  - `client_id`
  - `client_secret`
  - `enterprise_id`

## Configuration

### Required Parameters

| Parameter | Type | Description | Account Type |
|-----------|------|-------------|--------------|
| `is_enterprise` | string | Account type: "box_free" or "box_enterprise" | Both |
| `client_id` | string | Box application Client ID | Both |
| `client_secret` | string | Box application Client Secret | Both |
| `enterprise_id` | string | Box Enterprise ID | Enterprise Account only |

### Sync Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `sync.enabled` | bool | true | Enable synchronization |
| `sync.interval` | string | "30s" | Sync interval per datasource |

**Note**: Sync interval is configured at datasource level, not connector level. Each datasource can have different sync intervals.

## Box Application Setup

### Creating a Box Application

1. **Visit Box Developer Console**
   - Go to [Box Developer Console](https://app.box.com/developers/console)
   - Sign in with your Box account

2. **Create New App**
   - Click "Create New App"
   - Choose "Custom App"
   - Select "OAuth 2.0 with JWT (Server Authentication)" for Enterprise or "Standard OAuth 2.0 (User Authentication)" for Free account
   - Enter app name and description

3. **Configure OAuth**
   - For Free Account: Enable "Authorization Code Grant" and "Refresh Token"
   - For Enterprise Account: Enable "Client Credentials Grant"
   - Add redirect URI (if using OAuth flow)

4. **Get Credentials**
   - Copy `Client ID` from app configuration
   - Copy `Client Secret` from app configuration
   - For Enterprise: Copy `Enterprise ID` from account settings

### Required Scopes

For proper functionality, the Box application needs:

- **Read files and folders**: Access to read file metadata and folder structure
- **User information**: Read user profile information (for Free account)
- **Enterprise content**: Access enterprise content (for Enterprise account)

## Usage

### Method 1: Box Free Account

#### Step 1: Obtain client_id and client_secret
You need to obtain a client_id and client_secret through OAuth flow first (this can be done outside the system or through a separate OAuth setup).

#### Step 2: Configure Datasource
```json
{
  "id": "my-box-free",
  "name": "My Box Files",
  "type": "connector",
  "enabled": true,
  "sync": {
    "enabled": true,
    "interval": "30s"
  },
  "connector": {
    "id": "box",
    "config": {
      "client_id": "your_client_id",
      "client_secret": "your_client_secret",
    }
  }
}
```

### Method 2: Box Enterprise Account

#### Step 1: Get Enterprise Credentials
1. Obtain Client ID and Client Secret from Box Developer Console
2. Get Enterprise ID from Box Admin Console

#### Step 2: Configure Datasource
```json
{
  "id": "my-box-enterprise",
  "name": "Company Box Files",
  "type": "connector",
  "enabled": true,
  "sync": {
    "enabled": true,
    "interval": "30s"
  },
  "connector": {
    "id": "box",
    "config": {
      "is_enterprise": "box_enterprise",
      "client_id": "your_client_id",
      "client_secret": "your_client_secret",
      "enterprise_id": "12345"
    }
  }
}
```

## Architecture

### Pipeline Architecture

Box connector adopts **pipeline-based architecture**, consistent with other connectors:

- **Processor Registration**: Registered as pipeline processor in `init()` function
- **Scheduler Management**: Sync interval managed by connector_dispatcher
- **Per-Datasource Configuration**: Each datasource has independent sync interval and config
- **No Independent Scheduler**: Completely uses pipeline framework for data fetching

### Core Implementation

```go
func init() {
    // Register pipeline processor
    pipeline.RegisterProcessorPlugin(NAME, New)
}

func (processor *Processor) Fetch(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
    // Validate configuration
    // Create Box client
    // Authenticate with Box
    // Process files recursively
    return nil
}
```

### File Hierarchy

Box connector preserves the original folder structure:

- **Root Path**: `/`
- **Enterprise Account**: Each user's files are organized under user name
  - `/John Doe/Documents/file.pdf`
  - `/Jane Smith/Reports/report.xlsx`
- **Free Account**: Files organized directly from root
  - `/Documents/file.pdf`
  - `/Photos/image.jpg`

## Technical Details

### Authentication Flow

#### Free Account
```
1. Cache access_token with expiry time
2. Auto-refresh when token expires
```

#### Enterprise Account
```
1. Use client_credentials grant
2. Request with enterprise_id and box_subject_type
3. Cache access_token with expiry time
4. Auto-refresh when token expires
```

### Multi-User Support (Enterprise)

For Enterprise accounts:
1. Fetch all users from `/2.0/users` endpoint
2. For each user, fetch files with `as-user` header
3. Organize files under user's name in hierarchy
4. Generate unique document IDs including user ID

### Token Management

- **Token Caching**: In-memory cache with thread-safe operations
- **Automatic Refresh**: Tokens refresh 5 minutes before expiry
- **401 Retry**: Automatic re-authentication on 401 errors
- **Refresh Token Rotation**: Supports refresh token rotation (Free account)

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   - Verify `client_id` and `client_secret` are correct
   - Check if Box application is approved and published
   - For Enterprise account: Verify `enterprise_id` is correct

2. **Token Expired**
   - System automatically refreshes tokens
   - Verify application credentials haven't changed

3. **No Files Found**
   - Check user permissions in Box
   - Verify application has required scopes
   - For Enterprise: Ensure users have files in their accounts

4. **Sync Failures**
   - Check network connectivity to `https://api.box.com`
   - Verify rate limits aren't exceeded
   - Review system logs for detailed error messages

### Debug Logging

The connector provides detailed logging:
- Authentication process
- Token refresh operations
- API requests and responses
- User enumeration (Enterprise)
- File processing progress

Use logs to quickly identify and resolve issues.

## Notes

1. **Account Type Selection**: Must choose either "box_free" or "box_enterprise"
2. **Credentials Required**: Different account types require different credentials
3. **Enterprise Multi-User**: Enterprise accounts automatically fetch files from all users
4. **Token Management**: Tokens are automatically managed, no manual refresh needed
5. **API Rate Limits**: Be aware of Box API rate limits
6. **Content Extraction**: File content extraction is handled by coco-server framework
