---
title: "Milvus"
weight: 65
---
# Milvus Connector

## Register Milvus Connector

```shell
curl -XPOST "http://localhost:9000/connector/" -d '
{
  "name" : "Milvus Vector Database Connector",
  "description" : "Fetch vector and scalar data from Milvus collections with incremental sync and pagination.",
  "category" : "vector_database",
  "icon" : "/assets/icons/connector/milvus/icon.png",
  "tags" : [
    "vector",
    "database",
    "milvus",
    "vector_store"
  ],
  "url" : "http://coco.rs/connectors/milvus",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/milvus/icon.png"
    }
  },
  "processor": {
    "enabled": true,
    "name": "milvus"
  }
}'
```

## Use the Milvus Connector

The Milvus connector queries a Milvus collection, retrieves vectors and scalar fields, and streams them as documents into coco-server.

### Configure Milvus Datasource

`Address`: Milvus service address (e.g., `localhost:19530`)

`Username`: Optional username for Milvus authentication

`Password`: Optional password for Milvus authentication

`Database Name`: Optional database name (Milvus 2.2.0+). Leave empty to use the default database.

`Collection`: Name of the Milvus collection to query

`Output Fields`: Comma-separated list of fields to retrieve. Leave empty to retrieve all scalar fields.

`Filter`: Optional scalar filtering expression (e.g., `age > 10 and name like "abc%"`)

`Page Size`: Number of documents to fetch per page (1-10000). Defaults to `1000`.

`Field Mapping`: Optional mapping that aligns result fields with document properties (`id`, `title`, `content`, metadata, etc.)

#### Incremental Sync Fields

Milvus incremental sync relies on a watermark field in your collection:

- `Watermark Property`: A field name that monotonically increases (e.g., `updated_at` for datetime or `version` for integers)
- `Property Type`: The data type of the watermark field (`datetime`, `int`, `float`, or `string`)
- `Tie-breaker`: A secondary field that uniquely orders documents sharing the same watermark value (e.g., `id`)

During the first run you can optionally set `resume_from` to seed the starting watermark. Afterwards coco-server persists the last `(property, tie_breaker)` tuple and resumes from there.

### Example Request

#### Case 1: Datetime-based Incremental Sync

Use a `datetime` type field `updated_at` as the increment property with `id` as the tie-breaker.

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
  "name": "Milvus Product Vectors",
  "type": "connector",
  "connector": {
    "id": "milvus",
    "config": {
      "address": "localhost:19530",
      "username": "root",
      "password": "Milvus",
      "db_name": "default",
      "collection": "product_embeddings",
      "output_fields": ["id", "title", "description", "updated_at", "embedding"],
      "filter": "status == \"published\"",
      "page_size": 1000,
      "incremental": {
        "enabled": true,
        "mode": "property_watermark",
        "property": "updated_at",
        "property_type": "datetime",
        "tie_breaker": "id"
      },
      "field_mapping": {
        "enabled": true,
        "mapping": {
          "id": "id",
          "title": "title",
          "content": "description"
        }
      }
    }
  }
}'
```

The connector will generate filter expressions like:
```
(updated_at > 1704067320000) or ((updated_at == 1704067320000) and (id > 2))
```

This ensures:
- All documents with `updated_at` greater than the cursor are fetched
- Documents with the exact cursor timestamp but higher `id` are also fetched (tie-breaking)
- Proper ordering for incremental pagination

#### Case 2: Integer Version-based Incremental Sync

Use a monotonically increasing `version` field as the increment property.

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
  "name": "Milvus Document Vectors",
  "type": "connector",
  "connector": {
    "id": "milvus",
    "config": {
      "address": "milvus.example.com:19530",
      "username": "coco_sync",
      "password": "changeme",
      "db_name": "docs",
      "collection": "document_embeddings",
      "output_fields": ["id", "title", "content", "version", "embedding"],
      "page_size": 500,
      "incremental": {
        "enabled": true,
        "mode": "property_watermark",
        "property": "version",
        "property_type": "int",
        "tie_breaker": "id",
        "resume_from": "1"
      },
      "field_mapping": {
        "enabled": true,
        "mapping": {
          "id": "id",
          "title": "title",
          "content": "content"
        }
      }
    }
  }
}'
```

The connector will generate filter expressions like:
```
(version > 1000) or ((version == 1000) and (id > "doc-123"))
```

## Supported Config Parameters for Milvus Connector

| **Field**               | **Type**    | **Description**                                                                                       |
|-------------------------|-------------|-------------------------------------------------------------------------------------------------------|
| `address`               | `string`    | Milvus service address (e.g., `localhost:19530`).                                                     |
| `username`              | `string`    | Optional username for Milvus authentication.                                                          |
| `password`              | `string`    | Optional password for Milvus authentication.                                                          |
| `db_name`               | `string`    | Optional database name (Milvus 2.2.0+). Defaults to the default database if empty.                    |
| `collection`            | `string`    | Name of the Milvus collection to query.                                                               |
| `output_fields`         | `[]string`  | List of fields to retrieve. Leave empty to retrieve all scalar fields.                                |
| `filter`                | `string`    | Optional scalar filtering expression (e.g., `age > 10 and name like "abc%"`).                         |
| `page_size`             | `integer`   | Number of documents to fetch per page (1-10000). Defaults to `1000`.                                  |
| `incremental`           | `object`    | Optional incremental configuration (see table below).                                                 |
| `field_mapping`         | `object`    | Optional mapping from result fields to document schema.                                               |

### Incremental Config Fields

| **Field**         | **Type**   | **Description**                                                                                              |
|-------------------|------------|--------------------------------------------------------------------------------------------------------------|
| `enabled`         | `boolean`  | Turn incremental sync on/off.                                                                                |
| `mode`            | `string`   | Only `property_watermark` is currently supported.                                                            |
| `property`        | `string`   | Field name in your collection that acts as the watermark (e.g., `updated_at` or `version`).                  |
| `property_type`   | `string`   | Data type of the watermark field: `datetime`, `int`, `float`, or `string`.                                   |
| `tie_breaker`     | `string`   | Field used to order documents sharing the same watermark value (e.g., `id`).                                 |
| `resume_from`     | `string`   | Optional starting watermark for the first run (RFC3339 timestamp for datetime, numeric value for int/float). |

## Filter Expression Syntax

Milvus uses a specific query expression syntax for filtering:

- **Equality**: Use `==` (not `=`). Example: `status == "published"`
- **Comparison**: Use `>`, `<`, `>=`, `<=`. Example: `age > 18`
- **String matching**: Use `like` with `%` wildcard. Example: `name like "John%"`
- **Logical operators**: Use `and`, `or`, `not`
- **Parentheses**: Use for grouping and precedence

Example complex filter:
```
(age > 18 and status == "active") or (premium == true)
```

## How Incremental Sync Works

The Milvus connector builds query filter expressions to support incremental scanning:

1. **Initial Scan**: On first run, if `resume_from` is set, the connector starts from that watermark value. Otherwise, it starts from the beginning.

2. **Cursor Tracking**: After each successful sync, the connector saves the highest `(property, tie_breaker)` values seen.

3. **Resume Logic**: On subsequent runs, the connector generates a filter expression:
   ```
   (property > last_value) or ((property == last_value) and (tie_breaker > last_tie))
   ```

4. **Ordering**: Results are ordered by `property ASC, tie_breaker ASC` to ensure consistent pagination.

5. **State Persistence**: The cursor state is automatically saved to Elasticsearch and restored on the next sync.

This dual-field approach ensures:
- No documents are missed even if many share the same watermark timestamp
- Stable ordering for reliable pagination
- Efficient incremental updates as your Milvus collection grows

With these settings, the Milvus connector reliably pages through large collections, resuming from the precise `(watermark, tie-breaker)` position on each sync.
