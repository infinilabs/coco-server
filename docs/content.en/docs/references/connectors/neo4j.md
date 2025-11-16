---
title: "Neo4j"
weight: 60
---
# Neo4j Connector

## Register Neo4j Connector

```shell
curl -XPOST "http://localhost:9000/connector/" -d '
{
  "name" : "Neo4j Connector",
  "description" : "Fetch data from Neo4j graph database using Cypher.",
  "category" : "database",
  "icon" : "/assets/icons/connector/neo4j/icon.png",
  "tags" : [
    "graph",
    "database",
    "neo4j"
  ],
  "url" : "http://coco.rs/connectors/neo4j",
  "assets" : {
    "icons" : {
      "default" : "/assets/icons/connector/neo4j/icon.png"
    }
  },
  "processor": {
    "enabled": true,
    "name": "neo4j"
  }
}'
```

## Use the Neo4j Connector

The Neo4j connector runs a Cypher query, transforms the resulting rows into documents, and streams them into coco-server.

### Configure Neo4j Datasource

`Connection URI`: Bolt/Neo4j URI. Supported schemes include `neo4j://`, `neo4j+s://`, `bolt://`, etc.

`Database`: Optional database name when running in multi-database installations.

`Cypher Query`: The statement that returns the records you want to index. The connector treats each row as a document candidate.

`Query Parameters`: Optional key/value pairs passed to the Cypher query.

`Enable Pagination`: Enables driver-level pagination (recommended for large result sets).

`Page Size`: Number of records fetched per page when pagination is enabled. Defaults to `500`.

`Field Mapping`: Optional mapping that aligns result fields with document properties (`id`, `title`, `content`, metadata, etc.).

#### Incremental Sync Fields

Neo4j incremental sync relies on two expressions:

- `Watermark Property`: a Cypher expression returned by your query that monotonically increases (e.g., `datetime.coalesce(n.updated_at, n.created_at)`).
- `Tie-breaker Property`: a secondary expression that uniquely orders records sharing the same watermark value (e.g., `elementId(n)`).

During the first run you can optionally set `resume_from` to seed the starting watermark. Afterwards coco-server persists the last `(property, tie_breaker)` tuple and resumes from there.

### Example Request
#### Case 1
When a query returns a full node object without explicitly projecting its properties (e.g., using RETURN n), the Neo4j connector will automatically expand/serialize those properties. To correctly map the resulting fields, you must reference them using the node alias prefix (e.g., n.propertyName). 
- In this case, use the `datetime` type field `updated` as the increment property.
- `elementId` as the tie-breaker.

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
  "name": "Neo4j Docs",
  "type": "connector",
  "connector": {
    "id": "neo4j",
    "config": {
      "connection_uri": "neo4j+s://graph.example.com:7687",
      "username": "coco_sync",
      "password": "changeme",
      "database": "neo4j",
      "cypher": "MATCH (n:Document) RETURN n",
      "parameters": {
        "limit": 1000
      },
      "pagination": true,
      "page_size": 500,
      "incremental": {
        "enabled": true,
        "mode": "property_watermark",
        "property": "n.updated",
        "property_type": "datetime",
        "tie_breaker": "elementId(n)"
      },
      "field_mapping": {
        "enabled": true,
        "mapping": {
          "id": "id",
          "title": "title",
          "content": "content",
          "updated": "updated"
        }
      }
    }
  }
}'
```

The final Cypher query string will look like this:
```cypher
MATCH (n:Document)
WITH 
    n,
    n.updated AS coco_property, // The primary incremental field (timestamp)
    elementId(n) AS coco_tie    // The secondary incremental field (unique string ID)

// Filtering Logic: Retrieve all documents newer than the cursor time OR 
// documents at the exact cursor time that have a greater elementId.
WHERE coco_property > datetime("2025-10-09T12:59:06.836Z") 
   OR (
       coco_property = datetime("2025-10-09T12:59:06.836Z") 
       AND coco_tie > "4:1093e8af-4022-4813-9534-8c7121ac10d4:33"
   )

// Sorting: Order by time first, then by the unique ID to break ties. 
// This dual-sort structure is essential for robust cursor pagination.
ORDER BY 
    coco_property ASC, 
    coco_tie ASC 

// Limit the results to the next batch.
LIMIT 500 

// Return the full node object and the projected fields.
RETURN n, coco_property, coco_tie
```


#### Case 2
The `global_version` field is designated as the global incremental property for the :Document node. This property must be monotonically incremented by the system across all data manipulation operations—specifically CREATE, SET/UPDATE, and DELETE—to ensure a globally traceable change history for the entity.
- In this case, use the `global_version` as the increment property.
- `elementId` as the tie-breaker.

```shell
curl -H 'Content-Type: application/json' -XPOST "http://localhost:9000/datasource/" -d '
{
  "name": "Neo4j Docs",
  "type": "connector",
  "connector": {
    "id": "neo4j",
    "config": {
      "connection_uri": "neo4j+s://graph.example.com:7687",
      "username": "coco_sync",
      "password": "changeme",
      "database": "neo4j",
      "cypher": "MATCH (n:Document) RETURN n.uuid AS id, n.title AS title, n.content AS content, n.updated AS updated, elementId(n) AS eid, n.global_version as version",
      "parameters": {
        "limit": 1000
      },
      "pagination": true,
      "page_size": 500,
      "incremental": {
        "enabled": true,
        "mode": "property_watermark",
        "property": "version",
        "property_type": "int",
        "tie_breaker": "eid",
        "resume_from": "1"
      },
      "field_mapping": {
        "enabled": true,
        "mapping": {
          "id": "id",
          "title": "title",
          "content": "content",
          "updated": "updated"
        }
      }
    }
  }
}'
```

## Supported Config Parameters for Neo4j Connector

| **Field**               | **Type**    | **Description**                                                                                      |
|-------------------------|-------------|------------------------------------------------------------------------------------------------------|
| `connection_uri`        | `string`    | Neo4j Bolt URI, including scheme and port (e.g., `neo4j+s://graph.example.com:7687`).                |
| `username` / `password` | `string`    | Credentials for basic authentication. Use `auth_token` instead for bearer auth.                      |
| `auth_token`            | `string`    | Optional bearer token. Overrides username/password when provided.                                    |
| `database`              | `string`    | Optional database name. Defaults to the Neo4j default DB.                                            |
| `cypher`                | `string`    | Cypher statement executed for each sync.                                                             |
| `parameters`            | `object`    | Optional key/value parameters supplied to the Cypher query.                                          |
| `pagination`            | `boolean`   | Enables driver pagination. Recommended for large result sets.                                        |
| `page_size`             | `integer`   | Number of records fetched per page when pagination is enabled. Defaults to `500`.                    |
| `incremental`           | `object`    | Optional incremental configuration (see table below).                                                |
| `field_mapping`         | `object`    | Optional mapping from result fields to document schema.                                              |

### Incremental Config Fields

| **Field**         | **Type**   | **Description**                                                                                           | 
|-------------------|------------|-----------------------------------------------------------------------------------------------------------|
| `enabled`         | `boolean`  | Turn incremental sync on/off.                                                                             |
| `mode`            | `string`   | Only `property_watermark` is currently supported.                                                         |
| `property`        | `string`   | Cypher expression returned by your query that acts as the watermark (e.g., `n.updated_at`).               |
| `property_type`   | `string`   | Hint for how to interpret the watermark (`datetime`, `int`, `float`, or `string`).                        |
| `tie_breaker`     | `string`   | Cypher expression used to order rows sharing the same watermark (e.g., `elementId(n)`).                   |
| `resume_from`     | `string`   | Optional starting watermark for the first run (RFC3339 timestamp for datetime properties).                |

With these settings, the Neo4j connector reliably pages through large result sets, resuming from the precise `(watermark, tie-breaker)` position on each sync.
