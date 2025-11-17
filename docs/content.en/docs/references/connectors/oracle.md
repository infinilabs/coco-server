---
title: "Oracle"
weight: 52
---
# Oracle Connector

## Register Oracle Connector

```shell
curl -XPOST "http://localhost:9000/connector/" -d '
{
  "name" : "Oracle Connector",
  "description" : "Fetch data from Oracle database.",
  "category" : "database",
  "icon" : "/assets/icons/connector/oracle/icon.png",
  "tags" : [
    "sql",
    "storage",
    "web"
  ],
  "url" : "http://coco.rs/connectors/oracle",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/oracle/icon.png"
    }
  },
          "processor":{
             "enabled":true,
             "name":"oracle"
          }
}'
```

## Use the Oracle Connector

The Oracle Connector allows you to index data from your database by executing a custom SQL query.

### Configure Oracle Datasource

To configure your Oracle connection, you'll need to provide the connection details and the query to fetch the data.

`Connection URI`: The full Oracle connection string.

`SQL Query`: The SQL query that will be executed to fetch the data for indexing. You can use `JOIN`s and select specific columns.

`Last Modified Field`: (Optional) For incremental sync, specify a timestamp or datetime column (e.g., `updated_at`). The connector will only fetch rows where this field's value is newer than the last sync time.

`Enable Pagination`: (Optional) A boolean (`true` or `false`) to enable paginated fetching. This is highly recommended for large tables to avoid high memory usage. Your SQL query must include an `ORDER BY` clause for pagination to work correctly.

`Page Size`: (Optional) The number of records to fetch per page when pagination is enabled. Defaults to `500`.

`Field Mapping`: (Optional) An advanced feature to map columns from your SQL query to specific document fields like `id`, `title`, `content`, and custom metadata.

### Example Request

Here is an example request to configure the Oracle Connector:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
    "name":"My Oracle Documents",
    "type":"connector",
    "connector":{
        "id": "oracle",
        "config": {
            "connection_uri": "oracle://user:password@host:1521/service_name",
            "sql": "SELECT * from DOC ORDER BY id",
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

## Supported Config Parameters for Oracle Connector
Below are the configuration parameters supported by the Oracle Connector:

| **Field**              | **Type**   | **Description**                                                                                                |
|------------------------|------------|----------------------------------------------------------------------------------------------------------------|
| `connection_uri`       | `string`   | The full Oracle connection URI (e.g., oracle://user:password@host:1521/service_name).                          |
| `sql`                  | `string`   | The SQL query to execute for fetching data. Recommend including an `ORDER BY` clause if pagination is enabled. |
| `last_modified_field`  | `string`   | Optional. The name of a timestamp/datetime column used for incremental synchronization.                        |
| `pagination`           | `boolean`  | Optional. Set to true to enable pagination for large queries. Defaults to false.                               |
| `page_size`            | `integer`  | Optional. The number of records to fetch per page when pagination is enabled. Defaults to 500.                 |
| `field_mapping`        | `object`   | Optional. Provides advanced control to map query columns to standard document fields.                          |
