---
title: "Network Drive"
weight: 30
---
# Network Drive Connector

## Register Network Drive Connector

```shell
curl -XPUT "http://localhost:9000/connector/network_drive?replace=true" -d '{
  "name" : "Network drive Connector",
  "description" : "Scan and extract metadata from network shared files.",
  "category" : "cloud_storage",
  "icon" : "/assets/icons/connector/network_drive/icon.png",
  "tags" : [
    "filesystem",
    "storage",
    "web"
  ],
  "url" : "http://coco.rs/connectors/network_drive",
  "assets" : {
  "icons" : {
    "default" : "/assets/icons/connector/network_drive/icon.png"
    }
  }
}'
```

> Use `Network Drive` as a unique identifier, as it is a builtin connector.

## Update coco-server's config

Below is an example configuration for enabling the Network Drive Connector in coco-server:

```shell
connector:
  network_drive:
    enabled: true
    queue:
      name: indexing_documents
    interval: 60s
```

### Explanation of Config Parameters


| **Field**    | **Type**  | **Description**                                                                        |
| ------------ | --------- |----------------------------------------------------------------------------------------|
| `enabled`    | `boolean` | Enables or disables the network drive connector. Set to`true` to activate it.          |
| `interval`   | `string`  | Specifies the time interval (e.g.,`60s`) at which the connector will check for updates. |
| `queue.name` | `string`  | Defines the name of the queue where indexing tasks will be added.                      |

## Use the Network Drive Connector

The Network Drive Connector allows you to index data from your SMB/CIFS shares into your system. Follow these steps to set it up:

### Configure Network Drive Client

To configure your Network Drive connection, you'll need to provide the server details and credentials. This setup allows your application to securely authenticate with and access files within a specified network share.

First, you'll need your server information and credentials:

`endpoint`: This is the IP address and port of your SMB server (e.g., `192.168.1.100:445`).

`share`: This is the name of the shared folder you want to access.

`username`: The username for authenticating to the share.

`password`: The password for the specified user.

`domain`: (Optional) The domain for NTLM authentication, often `WORKGROUP`.

Next, you can specify which folders and files to index:

`paths`: An array of strings representing the subdirectories to scan within the share. If you want to scan the entire share, you can use `["."]` or `[""]`.

`extensions`: An array of strings defining specific file extensions to include (e.g., `["pdf", "docx"]`). If this list is empty or omitted, all file types in the specified paths will be considered.


### Example Request

Here is an example request to configure the Network Drive Connector:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
    "name":"My Shared Documents",
    "type":"connector",
    "connector":{
        "id":"network_drive",
         "config":{
            "endpoint": "your-smb-server:445",
            "share": "documents",
            "username": "your-username",
            "password": "your-password",
            "domain": "WORKGROUP",
            "paths": ["."],
            "extensions": [ "pdf", "docx", "txt" ]
        }
    }
}'
```

## Supported Config Parameters for Network Drive Connector

Below are the configuration parameters supported by the Network Drive Connector:


| **Field**      | **Type**   | **Description**                                                                                                         |
|----------------| ---------- |-------------------------------------------------------------------------------------------------------------------------|
| `endpoint`     | `string`   | The IP address and port of the SMB server (e.g., `192.168.1.100:445`).                                                  |
| `share`        | `string`   | The name of the network share to index.                                                                                 |
| `username`     | `string`   | The username for authentication.                                                                                        |
| `password`     | `string`   | The password for authentication.                                                                                        |
| `domain`       | `string`   | Optional. The NTLM authentication domain (e.g., `WORKGROUP`).                                                           |
| `paths`        | `[]string` | An array of subdirectories to scan within the share. Use `["."]` to scan the root.                                      |
| `extensions`   | `[]string` | Optional. An array of file extensions to include (e.g., `pdf`, `docx`). If omitted or empty, all files will be indexed. |