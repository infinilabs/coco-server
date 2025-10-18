---
title: "PostgreSQL"
weight: 50
---
# PostgreSQL Connector

## Register PostgreSQL Connector

```shell
curl -XPUT "http://localhost:9000/connector/postgresql?replace=true" -d '{
  "name": "PostgreSQL Connector",
  "description": "Fetch data from PostgreSQL database.",
  "category": "database",
  "icon": "/assets/icons/connector/postgresql/icon.png",
  "tags": [
    "sql",
    "storage",
    "web"
  ],
  "url": "http://coco.rs/connectors/postgresql",
  "assets": {
    "icons": {
      "default": "/assets/icons/connector/postgresql/icon.png"
    }
  },
  "processor": {
    "enabled": true,
    "name": "postgresql"
  }
}'
```

> Use `postgresql` as a unique identifier, as it is a builtin connector.

> **Note**: Starting from version **0.4.0**, the PostgreSQL connector uses a **pipeline-based architecture** for better performance and flexibility. The `processor` configuration is required for the connector to work properly.

## Pipeline Architecture

Starting from version **0.4.0**, the PostgreSQL connector uses a **pipeline-based architecture** instead of the legacy scheduled task approach. This provides:

- **Better Performance**: Centralized dispatcher manages all connector sync operations
- **Per-Datasource Configuration**: Each datasource can have its own sync interval
- **Enrichment Pipeline Support**: Optional data enrichment pipelines per datasource
- **Resource Efficiency**: Optimized scheduling and resource management

### Pipeline Configuration (coco.yml)

The connector is managed by the centralized dispatcher pipeline:

```yaml
pipeline:
  - name: connector_dispatcher
    auto_start: true
    keep_running: true
    singleton: true
    retry_delay_in_ms: 10000
    processor:
      - connector_dispatcher:
          max_running_timeout_in_seconds: 1200
```

> **Important**: This pipeline configuration replaces the old connector-level config. The dispatcher automatically manages all enabled connectors.

### Connector Configuration

The PostgreSQL connector is configured via the management interface or API:

```json
{
  "id": "postgresql",
  "name": "PostgreSQL Connector",
  "builtin": true,
  "processor": {
    "enabled": true,
    "name": "postgresql"
  }
}
```

### Explanation of Connector Config Parameters

| **Field**           | **Type**  | **Description**                                                      |
|---------------------|-----------|----------------------------------------------------------------------|
| `processor.enabled` | `boolean` | Enables the pipeline processor (required).                           |
| `processor.name`    | `string`  | Processor name, must be "postgresql" (required).                     |

## Use the PostgreSQL Connector

The PostgreSQL Connector allows you to index data from your database by executing a custom SQL query.

### Configure PostgreSQL Datasource

To configure your PostgreSQL connection, you'll need to provide the connection details and the query to fetch the data.

`Connection URI`: The full PostgreSQL connection string, including user, password, host, port, and database name.

`SQL Query`: The SQL query that will be executed to fetch the data for indexing. You can use `JOIN`s and select specific columns.

`Last Modified Field`: (Optional) For incremental sync, specify a timestamp or datetime column (e.g., `updated_at`). The connector will only fetch rows where this field's value is newer than the last sync time.

`Enable Pagination`: (Optional) A boolean (`true` or `false`) to enable paginated fetching. This is highly recommended for large tables to avoid high memory usage.

`Page Size`: (Optional) The number of records to fetch per page when pagination is enabled. Defaults to `500`.

`Field Mapping`: (Optional) An advanced feature to map columns from your SQL query to specific document fields like `id`, `title`, `content`, and custom metadata.

### Datasource Configuration

Each datasource has its own sync configuration and PostgreSQL settings:

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '{
    "name": "My PostgreSQL Documents",
    "type": "connector",
    "enabled": true,
    "connector": {
        "id": "postgresql",
        "config": {
            "connection_uri": "postgres://username:password@localhost:5432/coco?sslmode=disable&timezone=Asia/Shanghai",
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
    },
    "sync": {
        "enabled": true,
        "interval": "30s"
    }
}'
```

## Supported Config Parameters for PostgreSQL Connector

Below are the configuration parameters supported by the PostgreSQL Connector:

### Datasource Config Parameters

| **Field**              | **Type**   | **Description**                                                                                         |
|------------------------|------------|---------------------------------------------------------------------------------------------------------|
| `connection_uri`       | `string`   | The full PostgreSQL connection URI (e.g., postgresql://user:pass@host:port/db?sslmode=disable) (required). |
| `sql`                  | `string`   | The SQL query to execute for fetching data (required).                                                 |
| `last_modified_field`  | `string`   | Optional. The name of a timestamp/datetime column used for incremental synchronization.                 |
| `pagination`           | `boolean`  | Optional. Set to true to enable pagination for large queries. Defaults to false.                        |
| `page_size`            | `integer`  | Optional. The number of records to fetch per page when pagination is enabled. Defaults to 500.          |
| `field_mapping`        | `object`   | Optional. Provides advanced control to map query columns to standard document fields.                   |
| `sync.enabled`         | `boolean`  | Enable/disable syncing for this datasource.                                                             |
| `sync.interval`        | `string`   | Sync interval for this datasource (e.g., "30s", "5m", "1h").                                            |
