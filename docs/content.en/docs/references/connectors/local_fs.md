---
title: "Notion"
weight: 30
---

# Local FS Connector

## Register Local FS Connector

```shell
curl -XPUT "http://localhost:9000/connector/local_fs?replace=true" -d '{
  "name" : "Local Filesystem Connector",
  "description" : "Scan and fetch metadata from local files.",
  "category" : "local_storage",
  "icon" : "/assets/icons/connector/local_fs/icon.png",
  "tags" : [
    "storage",
    "filesystem"
  ],
  "url" : "http://coco.rs/connectors/local_fs",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/local_fs/icon.png"
    }
  }
}'
```


> Use `local_fs` as a unique identifier, as it is a builtin connector.


## Update coco-server's config

Below is an example configuration for enabling the Notion Connector in coco-server:

```shell
connector:
  local_fs:
    enabled: true
    queue:
      name: indexing_documents
    interval: 10s
```

### Explanation of Config Parameters

| **Field**      | **Type**  | **Description**                                                                         |
|-----------------|-----------|-----------------------------------------------------------------------------------------|
| `enabled`      | `boolean` | Enables or disables the Local FS connector. Set to `true` to activate it.           |
| `interval`     | `string`  | Specifies the time interval (e.g., `60s`) at which the connector will check for updates. |
| `queue.name`   | `string`  | Defines the name of the queue where indexing tasks will be added.                       |

## Use the Local FS Connector

The Local FS Connector allows you to index data from your local filesystem into your system. Follow these steps to set it up:

### Configure file folders & extensions

You need to configure the folder path, and the connector will scan the metadata of all files under the folder, including subfolders.
You can add file extension configuration, and the connector will only scan files with the specified extension you specify.

### Example Request

Here is an example request to configure the Notion Connector:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
    "name":"My Local Documents",
    "type":"connector",
    "connector":{
        "id":"local_fs",
         "config":{
            "paths": [ "/path/to/my/documents", "/path/to/another/folder" ],
            "extensions": [ "pdf", "docx", "txt" ]
        }
    }
}''
```

## Supported Config Parameters for Local FS Connector

Below are the configuration parameters supported by the Local FS Connector:

| **Field**               | **Type**    | **Description**                                                                                                     |
|--------------------------|-------------|---------------------------------------------------------------------------------------------------------------------|
| `paths`   | `[]string`  | An array of absolute paths to the folders you want to scan.                                                         |
| `extensions`  | `[]string`  | Optional. An array of file extensions to include (e.g., pdf, docx). If omitted or empty, all files will be indexed. |

