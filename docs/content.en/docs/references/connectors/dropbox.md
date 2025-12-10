---
title: "Dropbox"
weight: 60
---

# Dropbox Connector

## Register Dropbox Connector

```shell
curl -XPOST "http://localhost:9000/connector/" -d '{
    "name": "Dropbox Connector",
    "description": "Index files and folders from Dropbox",
    "icon": "/assets/icons/connector/dropbox/icon.png",
    "category": "cloud_storage",
    "path_hierarchy": false,
    "tags": [
        "dropbox",
        "cloud_storage",
        "file_sharing"
    ],
    "url": "http://coco.rs/connectors/dropbox",
    "assets": {
        "icons": {
            "default": "/assets/icons/connector/dropbox/icon.png",
            "docx": "/assets/icons/connector/dropbox/docx.png",
            "xlsx": "/assets/icons/connector/dropbox/xlsx.png",
            "pptx": "/assets/icons/connector/dropbox/pptx.png",
            "pdf": "/assets/icons/connector/dropbox/pdf.png",
            "photo": "/assets/icons/connector/dropbox/photo.png",
            "zip": "/assets/icons/connector/dropbox/zip.png",
            "document": "/assets/icons/connector/dropbox/document.png",
            "paper": "/assets/icons/connector/dropbox/paper.png",
            "gdoc": "/assets/icons/connector/dropbox/gdoc.png",
            "gexcel": "/assets/icons/connector/dropbox/gexcel.png",
            "gppt": "/assets/icons/connector/dropbox/gppt.png"
        }
    },
    "processor": {
        "enabled": true,
        "name": "dropbox"
    }
}'
```

## Use the Dropbox Connector

The Dropbox Connector allows you to index files and folders from Dropbox.

### Features

- **OAuth 2.0 Authentication**: Secure authentication using Dropbox OAuth 2.0 with automatic token refresh.
- **Hierarchical Indexing**: Preserves the folder structure from Dropbox.
- **Recursive Indexing**: Automatically indexes subfolders.
- **Metadata Extraction**: Extracts file metadata including size, modification times, and content hash.
- **File Type Icons**: Maps common file extensions to appropriate icons.
- **Auto-Refresh Token**: Automatically refreshes access tokens when they expire.

### Setup Dropbox Application

Before using this connector, you need to create a Dropbox App in the Dropbox Developer Console.

#### 1. Create App

1. Go to [Dropbox App Console](https://www.dropbox.com/developers/apps).
2. Click "Create app".
3. Choose "Scoped access".
4. Choose "Full Dropbox" or "App folder" depending on your needs.
5. Name your app.

#### 2. Configure Permissions

In the "Permissions" tab, enable the following permissions:
- `files.content.read`
- `files.metadata.read`
- `sharing.read`
- `account_info.read`
- `team_data.member` (if applicable)

#### 3. Configure Redirect URI

In the "Settings" tab, add the Redirect URI. The URI usually follows this pattern:

`YOUR_COCO_SERVER_URL/api/coco/connector/{connector_id}/dropbox/oauth_redirect`

Replace `YOUR_COCO_SERVER_URL` with your actual server URL (e.g., `http://localhost:8080`).

#### 4. Get Credentials

Copy the "App key" and "App secret" from the "Settings" tab.

### Connector Configuration

To configure the Dropbox connector, you need to provide the App Key and App Secret.

| Field | Description | Required |
|---|---|---|
| `client_key` | The App Key from Dropbox | Yes |
| `client_secret` | The App Secret from Dropbox | Yes |

### OAuth Flow

1. **Configure Connector**: Register the connector with your `client_key` (App Key) and `client_secret` (App Secret).
2. **Connect**: Use the UI to initiate the connection. This will redirect you to Dropbox to authorize the application.
3. **Authorization**: After authorization, you will be redirected back to the application.
4. **Token Management**: The connector will automatically exchange the authorization code for an access token and a refresh token. These are stored securely and used for subsequent requests. The refresh token allows the connector to maintain access without user intervention.

### Datasource Configuration Example

The datasource is typically created automatically after the OAuth flow completes. However, internal configuration looks like this:

```json
{
    "connector": {
        "id": "dropbox-connector-id",
        "config": {
            "client_key": "YOUR_APP_KEY",
            "client_secret": "YOUR_APP_SECRET",
            "refresh_token": "AUTOMATICALLY_OBTAINED",
            "profile": {
                "account_id": "dbid:...",
                "email": "user@example.com",
                "name": "User Name"
            }
        }
    }
}
```

> **Note**: You do not need to manually provide the `refresh_token` or `profile`. These are populated automatically.

### File Processing

The connector lists files starting from the root directory (or a configured path). It iterates through folders recursively.

- **Files**: Indexed with their metadata and content (if supported by pipeline).
- **Folders**: Indexed as folder documents to maintain hierarchy structure.
- **Icons**: Files are assigned icons based on their extension (e.g., Word docs, PDFs, Images).

### API Endpoints Used

- `https://api.dropboxapi.com/oauth2/token`: For token exchange and refresh.
- `https://api.dropboxapi.com/2/users/get_current_account`: To get user profile information.
- `https://api.dropboxapi.com/2/files/list_folder`: To list files and folders.
- `https://api.dropboxapi.com/2/files/list_folder/continue`: To paginate through file lists.

### Troubleshooting

- **Invalid Redirect URI**: Ensure the Redirect URI in Dropbox Console matches exactly what the application uses.
- **Permissions Error**: Ensure all required permissions are checked in the Dropbox App Console.
- **Token Expiry**: If the refresh token becomes invalid (e.g., user revokes access), you will need to reconnect the datasource via the UI.

