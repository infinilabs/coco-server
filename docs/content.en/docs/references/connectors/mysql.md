---
title: "MySQL"
weight: 50
---
# MySQL Connector

## Register MySQL Connector

```shell
curl -XPUT "http://localhost:9000/connector/mysql?replace=true" -d '
{
  "name" : "MySQL Connector",
  "description" : "Fetch data from MySQL database.",
  "category" : "database",
  "icon" : "/assets/icons/connector/mysql/icon.png",
  "tags" : [
    "sql",
    "storage",
    "web"
  ],
  "url" : "http://coco.rs/connectors/mysql",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/mysql/icon.png"
    }
  }
}'
```

> Use `mysql` as a unique identifier, as it is a builtin connector.

## Update coco-server's config

Below is an example configuration for enabling the MySQL Connector in coco-server:

```shell
connector:
  mysql:
    enabled: true
    queue:
      name: indexing_documents
    interval: 30s
```

### Explanation of Config Parameters

| **Field**    | **Type**  | **Description**                                                                          |
|--------------|-----------|------------------------------------------------------------------------------------------|
| `enabled`    | `boolean` | Enables or disables the MySQL connector. Set to`true` to activate it.                    |
| `interval`   | `string`  | Specifies the time interval (e.g., `30s`) at which the connector will check for updates. |
| `queue.name` | `string`  | Defines the name of the queue where indexing tasks will be added.                        |

## Use the MySQL Connector

The MySQL Connector allows you to index data from your database by executing a custom SQL query.

### Configure MySQL Datasource

To configure your MySQL connection, you'll need to provide the connection details and the query to fetch the data.

`Connection URI`: The full MySQL connection string, including user, password, host, port, and database name.

`SQL Query`: The SQL query that will be executed to fetch the data for indexing. You can use `JOIN`s and select specific columns.

`Last Modified Field`: (Optional) For incremental sync, specify a timestamp or datetime column (e.g., `updated_at`). The connector will only fetch rows where this field's value is newer than the last sync time.

`Enable Pagination`: (Optional) A boolean (`true` or `false`) to enable paginated fetching. This is highly recommended for large tables to avoid high memory usage.

`Page Size`: (Optional) The number of records to fetch per page when pagination is enabled. Defaults to `500`.

`Field Mapping`: (Optional) An advanced feature to map columns from your SQL query to specific document fields like `id`, `title`, `content`, and custom metadata.

### Example Request

Here is an example request to configure the MySQL Connector:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
    "name":"My MySQL Documents",
    "type":"connector",
    "connector":{
        "id": "mysql",
        "config": { 
            "connection_uri": "mysql://user:password@tcp(localhost:3306)/database",
            "sql": "SELECT * from DOC",
            "pagination": true,
            "page_size": 500,
            "last_modified_field": "updated",
            "field_mapping": { 
                "enabled": true,
                "mapping": { 
                    "hashed": true,
                    "id": "id",
                    "title": "title",
                    "url": "url",
                    "metadata": [ 
                      { 
                        "name": "version",
                        "value": "v"
                      }
                    ]
                }
            }
        }
    }
}'
```

## Supported Config Parameters for MySQL Connector 
Below are the configuration parameters supported by the MySQL Connector: 

| **Field**              | **Type**   | **Description**                                                                                   | 
|------------------------|------------|---------------------------------------------------------------------------------------------------| 
| `connection_uri`       | `string`   | The full MySQL connection URI (e.g., mysql://user:password@tcp(localhost:3306)/database.          |
| `sql`                  | `string`   | The SQL query to execute for fetching data.                                                       |
| `last_modified_field`  | `string`   | Optional. The name of a timestamp/datetime column used for incremental synchronization.           |
| `pagination`           | `boolean`  | Optional. Set to true to enable pagination for large queries. Defaults to false.                  |
| `page_size`            | `integer`  | Optional. The number of records to fetch per page when pagination is enabled. Defaults to 500.    |
| `field_mapping`        | `object`   | Optional. Provides advanced control to map query columns to standard document fields.             |
