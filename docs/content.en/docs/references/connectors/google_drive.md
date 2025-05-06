---
title: "Google Drive"
weight: 10
---

# Google Drive Connector

## Obtain Google Drive credentials
To use the Google Drive Connector, follow these steps to obtain your token:
[Google Drive API Quickstart](https://developers.google.com/drive/api/quickstart/go).

1. Create your own Google OAuth App, set the **Data Access** as shown in the following screenshot:

{{% load-img "/img/google_drive_scopes.png" "Create a APP" %}}

2. Set the **Authorized Redirect URIs** as shown in the following screenshot:

{{% load-img "/img/google_drive_token.jpg" "Authorized Redirect URIs" %}}

3. The Google Drive connector uses `/connector/google_drive/oauth_redirect` as the callback URL to receive authorization responses.

4. Once the token is successfully obtained, download the `credentials.json` file.

{{% load-img "/img/download_google_drive_token.png" "credentials.json" %}}

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

> **Notes**:
> - Ensure `redirect_uris` and `javascript_origins` are properly configured for your deployment.

## Register Google Drive Connector

Starting from version **0.4.0**, **Coco Server** includes the **Google Drive Connector** as a built-in feature.

### Step 1: Access Connector Settings

After installation and initialization, navigate to the **Coco Server** admin panel:

As shown below:

{{% load-img "/img/connector/list.png" "connector list" %}}

### Step 2: Configure Google Drive OAuth

Click the **edit** button to enter the configuration page.
{{% load-img "/img/connector/google_drive_config.png" "connector config" %}}

Configure the following fields based on your Google OAuth credentials:

- `Client ID`
- `Client Secret`
- `Redirect URI`
- `Auth URI`
- `Token URI`

Then click the **Save** button to store the configuration.

> ⚠️ **Upgrade Note for Older Versions:**  
> If you're upgrading from an earlier version, make sure to delete the existing data directory first:
>
> ```bash
> rm -rf data
> ```
> Then restart **Coco Server** and reinitialize the system.

## Connect to Your Google Drive

To add a Google Drive data source, go to the **Coco Server** admin panel and click the left-side menu `Data Source`, as shown below:

{{% load-img "/img/connector/google_drive_add1.png" "add google drive datasource" %}}

Click the **Add** button to open the connector type selection page, as shown below:

{{% load-img "/img/connector/google_drive_add2.png" "add google drive datasource" %}}

Then, hover over the **Google Drive** connector card and click the **Add** button that appears:

{{% load-img "/img/connector/google_drive_add3.png" "add google drive datasource" %}}

Click the **Connect** button, and the page will redirect to Google for OAuth authorization.  
Once authorized successfully, the new Google Drive data source will appear in your data source list.
