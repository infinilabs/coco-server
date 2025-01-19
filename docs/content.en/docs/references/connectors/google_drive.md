---
title: "Google Drive"
weight: 10
---

# Google Drive Connector

## Register Google Drive Connector

```shell
curl -XPUT http://localhost:9000/connector/google_drive?replace=true -d '{
    "name": "Google Drive Connector", 
    "description": "Fetch the files metadata from Google Drive.", 
    "icon": "/assets/connector/google_drive/icon.png", 
    "category": "cloud_storage", 
    "tags": [
        "google", 
        "storage"
    ], 
    "url": "http://coco.rs/connectors/google_drive", 
    "assets": {
        "icons": {
            "default": "/assets/connector/google_drive/icon.png", 
            "audio": "/assets/connector/google_drive/audio.png", 
            "form": "/assets/connector/google_drive/form.png", 
            "document": "/assets/connector/google_drive/document.png", 
            "drawing": "/assets/connector/google_drive/drawing.png", 
            "folder": "/assets/connector/google_drive/folder.png", 
             "fusiontable": "/assets/connector/google_drive/fusiontable.png", 
             "jam": "/assets/connector/google_drive/jam.png", 
             "map": "/assets/connector/google_drive/map.png", 
             "ms_excel": "/assets/connector/google_drive/ms_excel.png", 
             "ms_powerpoint": "/assets/connector/google_drive/ms_powerpoint.png", 
             "ms_word": "/assets/connector/google_drive/ms_word.png", 
             "pdf": "/assets/connector/google_drive/pdf.png", 
             "photo": "/assets/connector/google_drive/photo.png", 
            "presentation": "/assets/connector/google_drive/presentation.png", 
            "script": "/assets/connector/google_drive/script.png", 
            "site": "/assets/connector/google_drive/site.png", 
            "spreadsheet": "/assets/connector/google_drive/spreadsheet.png",
            "video": "/assets/connector/google_drive/video.png",
            "zip": "/assets/connector/google_drive/zip.png"
        }
    }
}'
```

> Use `google_drive` as a unique identifier, as it is a builtin connector.

# Using the Google Drive Connector

To use the Google Drive Connector, follow these steps to obtain your token:
[Google Drive API Quickstart](https://developers.google.com/drive/api/quickstart/go).

## Obtain Google Drive credentials

1. Set the **Authorized Redirect URIs** as shown in the following screenshot:

   ![Authorized Redirect URIs](/img/google_drive_token.jpg)

2. The Google Drive connector uses `/connector/google_drive/oauth_redirect` as the callback URL to receive authorization responses.

3. Once the token is successfully obtained, download the `credentials.json` file.

   ![credentials.json](/img/download_google_drive_token.png)


### Important Notes:
- If you deploy the **coco-server** in your production environment, ensure you:
  - Update the domain name accordingly.
  - Adjust the callback URL or configure a custom prefix if you have an **Nginx** instance in front of the server.

## Update coco-server's config

Below is an example configuration for enabling the Google Drive Connector in coco-server:

```shell
connector:
  google_drive:
    enabled: true
    queue:
      name: indexing_documents
#    credential_file: credentials.json
    credential:
      "client_id": "YOUR_XXX.apps.googleusercontent.com"
      "project_id": "infini-YOUR_XXX"
      "auth_uri": "https://accounts.google.com/o/oauth2/auth"
      "token_uri": "https://oauth2.googleapis.com/token"
      "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs"
      "client_secret": "YOUR_XXX-YOUR_XXX"
      "redirect_uris":  "http://localhost:9000/connector/google_drive/oauth_redirect"
      "javascript_origins": [ "http://localhost:9000" ]
    interval: 10s
    skip_invalid_token: true
```

Below is the config parameters supported by this connector.

| **Field**                     | **Type**     | **Description**                                                                                  |
|-------------------------------|--------------|--------------------------------------------------------------------------------------------------|
| `enabled`                     | `bool`       | Set to `true` to enable the Google Drive Connector.                                             |
| `queue.name`                  | `string`     | Specifies the queue name for indexing documents.                                                |
| `interval`                    | `duration`   | Interval for polling the Google Drive API.                                                     |
| `skip_invalid_token`          | `bool`       | Skip errors caused by invalid tokens if set to `true`.                                         |
| `credential`                  | `object`     | Inline Google Drive API credentials.                                                            |
| `credential_file`             | `string`     | Path to the `credentials.json` file (optional if `credential` is used).                         |
| `client_id`                   | `string`     | Google Drive client ID obtained from Google API Console.                                        |
| `project_id`                  | `string`     | Project ID for the Google Drive API.                                                            |
| `auth_uri`                    | `string`     | URI for Google authentication.                                                                  |
| `token_uri`                   | `string`     | URI to exchange tokens.                                                                         |
| `auth_provider_x509_cert_url` | `string`     | URI for Google's certificate provider.                                                          |
| `client_secret`               | `string`     | Client secret for Google Drive API.                                                             |
| `redirect_uris`               | `string`     | Callback URI for authorization responses.                                                       |
| `javascript_origins`          | `[]string`   | List of allowed JavaScript origins for the application.                                         |

> **Notes**:
> - Use either `credential_file` or `credential` for providing credentials.
> - Ensure `redirect_uris` and `javascript_origins` are properly configured for your deployment.

## Connect to Your Google Drive

To connect your Google Drive, follow these steps:

1. Visit the URL: `http://localhost:9000/connector/google_drive/connect`.
2. You will be redirected to Google's authentication page.
3. After successfully authenticating, the connector will begin indexing your Google Drive files.

> **Note**:
> Ensure that `coco-server` is running and configured correctly before accessing the connection URL.