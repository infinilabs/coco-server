# MongoDB Connector

## Register MongoDB Connector

```shell
curl -XPUT "http://localhost:9000/connector/mongodb?replace=true" -d '{
  "name" : "MongoDB Connector",
  "description" : "Scan and fetch documents from MongoDB collections.",
  "enabled" : true
}'
```

## Create MongoDB Data Source

```shell
curl -XPOST "http://localhost:9000/datasource" -d '{
  "name": "My MongoDB Database",
  "type": "connector",
  "enabled": true,
  "sync_enabled": true,
  "connector": {
    "id": "mongodb",
    "config": {
      "host": "localhost",
      "port": 27017,
      "database": "mydb",
      "username": "user",
      "password": "password",
      "auth_database": "admin",
      "batch_size": 1000,
      "max_pool_size": 10,
      "timeout": "30s",
      "sync_strategy": "full",
      "collections": [
        {
          "name": "articles",
          "title_field": "title",
          "content_field": "content",
          "category_field": "category",
          "tags_field": "tags",
          "url_field": "url",
          "timestamp_field": "updated_at",
          "filter": {
            "status": "published"
          },
          "fields": ["title", "content", "category", "tags", "url", "updated_at"]
        }
      ]
    }
  }
}'
```

## Configuration Options

### Connection Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `connection_uri` | string | No | MongoDB connection string (alternative to individual fields) |
| `host` | string | Yes* | MongoDB host address |
| `port` | int | No | MongoDB port (default: 27017) |
| `username` | string | No | Authentication username |
| `password` | string | No | Authentication password |
| `database` | string | Yes* | Target database name |
| `auth_database` | string | No | Authentication database (default: admin) |

*Required if `connection_uri` is not provided

### Replica Set and Sharding

| Field | Type | Description |
|-------|------|-------------|
| `replica_set` | string | Replica set name for replica set deployments |
| `read_preference` | string | Read preference: primary, secondary, nearest, primaryPreferred, secondaryPreferred |

### TLS/SSL Configuration

| Field | Type | Description |
|-------|------|-------------|
| `enable_tls` | bool | Enable TLS/SSL connection |
| `tls_ca_file` | string | Path to CA certificate file |
| `tls_cert_file` | string | Path to client certificate file |
| `tls_key_file` | string | Path to client private key file |
| `tls_insecure` | bool | Skip certificate verification |

### Performance Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `batch_size` | int | 1000 | Number of documents to process in each batch |
| `timeout` | string | "30s" | Connection timeout duration |
| `max_pool_size` | int | 10 | Maximum connection pool size |

### Sync Strategy

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `sync_strategy` | string | "full" | Sync strategy: "full" or "incremental" |
| `timestamp_field` | string | - | Field to use for incremental sync |

### Collection Configuration

Each collection in the `collections` array supports:

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Collection name (required) |
| `filter` | object | MongoDB query filter |
| `fields` | array | List of fields to include (projection) |
| `title_field` | string | Field to map to document title |
| `content_field` | string | Field to map to document content |
| `category_field` | string | Field to map to document category |
| `tags_field` | string | Field to map to document tags |
| `url_field` | string | Field to map to document URL |
| `timestamp_field` | string | Field to use for timestamps |

## Examples

### Single Instance Connection

```json
{
  "host": "localhost",
  "port": 27017,
  "database": "myapp",
  "username": "reader",
  "password": "secret",
  "collections": [
    {
      "name": "posts",
      "title_field": "title",
      "content_field": "body"
    }
  ]
}
```

### Replica Set Connection

```json
{
  "connection_uri": "mongodb://user:pass@host1:27017,host2:27017,host3:27017/mydb?replicaSet=rs0",
  "read_preference": "secondaryPreferred",
  "collections": [
    {
      "name": "articles",
      "title_field": "headline",
      "content_field": "text",
      "timestamp_field": "publishedAt",
      "filter": {
        "status": "published",
        "publishedAt": {"$gte": "2024-01-01"}
      }
    }
  ]
}
```

### Sharded Cluster Connection

```json
{
  "connection_uri": "mongodb://mongos1:27017,mongos2:27017/mydb",
  "batch_size": 500,
  "max_pool_size": 20,
  "collections": [
    {
      "name": "logs",
      "content_field": "message",
      "timestamp_field": "timestamp",
      "fields": ["message", "level", "timestamp", "source"]
    }
  ]
}
```

### Incremental Sync Configuration

```json
{
  "host": "localhost",
  "database": "cms",
  "sync_strategy": "incremental",
  "collections": [
    {
      "name": "articles",
      "title_field": "title",
      "content_field": "content",
      "timestamp_field": "updated_at",
      "filter": {
        "status": "published"
      }
    }
  ]
}
```