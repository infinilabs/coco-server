---
title: "MongoDB"
weight: 65
---
# MongoDB Connector

## Register MongoDB Connector

```shell
curl -XPUT "http://localhost:9000/connector/mongodb?replace=true" -d '
{
  "name" : "MongoDB Connector",
  "description" : "Fetch data from MongoDB collections with incremental sync support.",
  "category" : "database",
  "icon" : "/assets/icons/connector/mongodb/icon.png",
  "tags" : [
    "nosql",
    "database",
    "mongodb"
  ],
  "url" : "http://coco.rs/connectors/mongodb",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/mongodb/icon.png"
    }
  },
  "processor": {
    "enabled": true,
    "name": "mongodb"
  }
}'
```

> Use `mongodb` as the unique identifier because it is a built-in connector.

## Use the MongoDB Connector

The MongoDB connector queries a collection using optional BSON filters, transforms the resulting documents, and streams them into coco-server.

### Configure MongoDB Datasource

`Connection URI`: MongoDB connection string. Supports `mongodb://` and `mongodb+srv://` schemes.

`Database`: The MongoDB database name to connect to.

`Collection`: The collection name to query.

`Query Filter`: Optional BSON query filter in JSON format (e.g., `{"status": "published"}`).

`Sort Specification`: Optional sort specification in JSON format (e.g., `{"updated_at": 1, "_id": 1}`).

`Enable Pagination`: Enables cursor-based pagination (recommended for large collections).

`Page Size`: Number of documents fetched per page when pagination is enabled. Range: 1-10,000. Defaults to `500`.

`Field Mapping`: Optional mapping that aligns document fields with the document schema (`id`, `title`, `content`, metadata, etc.).

#### Incremental Sync Fields

MongoDB incremental sync relies on two fields:

- `Tracking Property`: a field in your documents that monotonically increases (e.g., `updated_at`, `_id`).
- `Tie-breaker Field`: a secondary field that uniquely orders documents sharing the same tracking property value (typically `_id`).

Example snippet inside the datasource config:

```yaml
incremental:
  enabled: true
  property: updated_at
  property_type: datetime
  tie_breaker: _id
  resume_from: 2025-01-01T00:00:00Z
```

During the first run you can optionally set `resume_from` to seed the starting value. Afterwards coco-server persists the last `(property, tie_breaker)` tuple and resumes from there.

### Example Request
#### Case 1: Timestamp-based Incremental Sync
Use a timestamp field like `updated_at` for tracking changes, with `_id` as the tie-breaker.

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
  "name": "MongoDB Articles",
  "type": "connector",
  "connector": {
    "id": "mongodb",
    "config": {
      "connection_uri": "mongodb://username:password@localhost:27017",
      "database": "myapp",
      "collection": "articles",
      "query": "{\"status\": \"published\"}",
      "sort": "{\"updated_at\": 1, \"_id\": 1}",
      "pagination": true,
      "page_size": 500,
      "incremental": {
        "enabled": true,
        "property": "updated_at",
        "property_type": "datetime",
        "tie_breaker": "_id",
        "resume_from": "2025-01-01T00:00:00Z"
      },
      "field_mapping": {
        "enabled": true,
        "mapping": {
          "id": "_id",
          "title": "title",
          "content": "body",
          "updated": "updated_at"
        }
      }
    }
  }
}'
```

The MongoDB query will be constructed as:

```javascript
// Filter: Combine user query with incremental logic
{
  "$and": [
    {"status": "published"},  // User's base query
    {
      "$or": [
        {"updated_at": {"$gt": ISODate("2025-01-01T00:00:00Z")}},
        {
          "$and": [
            {"updated_at": {"$eq": ISODate("2025-01-01T00:00:00Z")}},
            {"_id": {"$gt": ObjectId("507f1f77bcf86cd799439011")}}
          ]
        }
      ]
    }
  ]
}

// Sort: Ensure stable ordering
{"updated_at": 1, "_id": 1}

// Limit: Page size
.limit(500)
```

#### Case 2: ObjectId-based Incremental Sync
Use MongoDB's `_id` field directly for tracking. Since `_id` is unique, it serves as both the tracking property and tie-breaker.

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
  "name": "MongoDB Logs",
  "type": "connector",
  "connector": {
    "id": "mongodb",
    "config": {
      "connection_uri": "mongodb+srv://username:password@cluster.mongodb.net/logs",
      "database": "logs",
      "collection": "events",
      "pagination": true,
      "page_size": 1000,
      "incremental": {
        "enabled": true,
        "property": "_id",
        "property_type": "string",
        "tie_breaker": "_id",
        "resume_from": "507f1f77bcf86cd799439011"
      },
      "field_mapping": {
        "enabled": true,
        "mapping": {
          "id": "_id",
          "content": "message",
          "updated": "timestamp"
        }
      }
    }
  }
}'
```

#### Case 3: Integer-based Version Field
Use a monotonically increasing integer version field for tracking changes.

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
  "name": "MongoDB Products",
  "type": "connector",
  "connector": {
    "id": "mongodb",
    "config": {
      "connection_uri": "mongodb://localhost:27017",
      "database": "inventory",
      "collection": "products",
      "query": "{\"active\": true}",
      "pagination": true,
      "page_size": 500,
      "incremental": {
        "enabled": true,
        "property": "version",
        "property_type": "int",
        "tie_breaker": "_id",
        "resume_from": "1"
      },
      "field_mapping": {
        "enabled": true,
        "mapping": {
          "id": "_id",
          "title": "name",
          "content": "description"
        }
      }
    }
  }
}'
```

## Supported Config Parameters for MongoDB Connector

| **Field**          | **Type**    | **Description**                                                                                       |
|--------------------|-------------|-------------------------------------------------------------------------------------------------------|
| `connection_uri`   | `string`    | MongoDB connection string (e.g., `mongodb://localhost:27017` or `mongodb+srv://cluster.mongodb.net`). |
| `database`         | `string`    | Database name to connect to.                                                                          |
| `collection`       | `string`    | Collection name to query.                                                                             |
| `query`            | `string`    | Optional BSON query filter in JSON format (e.g., `{"status": "active"}`).                             |
| `sort`             | `string`    | Optional sort specification in JSON format (e.g., `{"updated_at": 1, "_id": 1}`).                     |
| `pagination`       | `boolean`   | Enables cursor-based pagination. Recommended for large collections.                                   |
| `page_size`        | `integer`   | Number of documents fetched per page. Range: 1-10,000. Defaults to `500`.                             |
| `incremental`      | `object`    | Optional incremental configuration (see table below).                                                 |
| `field_mapping`    | `object`    | Optional mapping from MongoDB fields to document schema.                                              |

### Incremental Config Fields

| **Field**         | **Type**   | **Description**                                                                                           |
|-------------------|------------|-----------------------------------------------------------------------------------------------------------|
| `enabled`         | `boolean`  | Turn incremental sync on/off.                                                                             |
| `property`        | `string`   | Field name to track changes (e.g., `updated_at`, `_id`, `version`).                                       |
| `property_type`   | `string`   | Data type of the tracking property: `datetime`, `int`, or `string`.                                       |
| `tie_breaker`     | `string`   | Secondary field for ordering documents with the same property value (typically `_id`).                    |
| `resume_from`     | `string`   | Optional starting value for the first run (RFC3339 timestamp for datetime, hex string for ObjectId).      |

### BSON Type Handling

The MongoDB connector automatically handles various BSON types:

| **BSON Type**    | **Conversion**                                   |
|------------------|--------------------------------------------------|
| `ObjectID`       | Hex string (preserved in cursor with type info)  |
| `DateTime`       | time.Time (Go) / RFC3339Nano string              |
| `Binary`         | Byte array                                       |
| `Decimal128`     | String representation                            |
| `Int32/Int64`    | Integer                                          |
| `Double`         | Float                                            |
| `Embedded Doc`   | Nested map/object                                |
| `Array`          | Slice/array                                      |

With these settings, the MongoDB connector reliably pages through large collections, resuming from the precise `(property, tie_breaker)` position on each sync, while preserving BSON type information for accurate query construction.
