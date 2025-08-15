# MongoDB Connector Configuration Guide

## Overview

MongoDB Connector is a powerful data connector that supports efficient data synchronization from MongoDB databases. It provides flexible configuration options, supporting incremental synchronization, field mapping, pagination processing, and other advanced features.

## Configuration Structure

### Basic Configuration

```json
{
  "connection_uri": "mongodb://localhost:27017/test",
  "database": "test",
  "collections": [
    {
      "name": "users",
      "filter": {"status": "active"},
      "title_field": "name",
      "content_field": "bio"
    }
  ],
  "pagination": true,
  "page_size": 500,
  "last_modified_field": "updated_at",
  "field_mapping": {
    "enabled": true,
    "mapping": {
      "id": "user_id",
      "title": "user_name"
    }
  }
}
```

## Configuration Parameters

### 1. Connection Configuration

#### `connection_uri` (Required)
- **Type**: String
- **Description**: MongoDB connection string
- **Format**: `mongodb://[username:password@]host[:port]/database[?options]`
- **Examples**: 
  - `mongodb://localhost:27017/test`
  - `mongodb://user:pass@localhost:27017/test?authSource=admin`
  - `mongodb://localhost:27017,localhost:27018/test?replicaSet=rs0`

#### `database` (Required)
- **Type**: String
- **Description**: Name of the MongoDB database to connect to
- **Examples**: `"test"`, `"production"`, `"analytics"`

### 2. Collections Configuration

#### `collections` (Required)
- **Type**: Array
- **Description**: List of collections to synchronize
- **Each collection contains the following fields**:

##### `name` (Required)
- **Type**: String
- **Description**: Collection name
- **Examples**: `"users"`, `"products"`, `"orders"`

##### `filter` (Optional)
- **Type**: Object
- **Description**: MongoDB query filter to limit synchronized documents
- **Examples**: 
  ```json
  {"status": "active"}
  {"age": {"$gte": 18}}
  {"category": {"$in": ["tech", "business"]}}
  ```

##### `title_field` (Optional)
- **Type**: String
- **Description**: Field name to use as document title
- **Examples**: `"name"`, `"title"`, `"subject"`

##### `content_field` (Optional)
- **Type**: String
- **Description**: Field name to use as document content
- **Examples**: `"bio"`, `"description"`, `"body"`

##### `category_field` (Optional)
- **Type**: String
- **Description**: Field name to use as document category
- **Examples**: `"category"`, `"type"`, `"department"`

##### `tags_field` (Optional)
- **Type**: String
- **Description**: Field name to use as document tags
- **Examples**: `"tags"`, `"keywords"`, `"labels"`

##### `url_field` (Optional)
- **Type**: String
- **Description**: Field name to use as document URL
- **Examples**: `"url"`, `"link"`, `"website"`

##### `timestamp_field` (Optional)
- **Type**: String
- **Description**: Field name to use as timestamp for incremental synchronization
- **Examples**: `"updated_at"`, `"modified"`, `"timestamp"`

### 3. Pagination Configuration

#### `pagination` (Optional)
- **Type**: Boolean
- **Description**: Whether to enable pagination processing
- **Default**: `false`
- **Note**: Enabling pagination can improve performance for large datasets

#### `page_size` (Optional)
- **Type**: Integer
- **Description**: Number of documents to process per page
- **Default**: `500`
- **Range**: 1-10000
- **Note**: Smaller page sizes reduce memory usage, larger page sizes improve processing efficiency

### 4. Incremental Synchronization Configuration

#### `last_modified_field` (Optional)
- **Type**: String
- **Description**: Timestamp field name for incremental synchronization
- **Examples**: `"updated_at"`, `"modified"`, `"last_updated"`
- **Note**: When set, the system will only synchronize documents where this field value is greater than the last synchronization time

#### `sync_strategy` (Optional)
- **Type**: String
- **Description**: Synchronization strategy
- **Values**: `"full"`, `"incremental"`
- **Default**: `"full"`
- **Note**: 
  - `"full"`: Full synchronization, synchronize all documents each time
  - `"incremental"`: Incremental synchronization, only synchronize new or updated documents

### 5. Field Mapping Configuration

#### `field_mapping` (Optional)
- **Type**: Object
- **Description**: Global field mapping configuration

##### `enabled` (Required)
- **Type**: Boolean
- **Description**: Whether to enable field mapping
- **Default**: `false`

##### `mapping` (Required)
- **Type**: Object
- **Description**: Field mapping rules
- **Format**: `{"target_field": "source_field"}`
- **Examples**:
  ```json
  {
    "id": "user_id",
    "title": "user_name",
    "content": "user_bio",
    "category": "user_role"
  }
  ```

### 6. Performance Optimization Configuration

#### `batch_size` (Optional)
- **Type**: Integer
- **Description**: Batch processing size
- **Default**: `1000`
- **Range**: 100-10000
- **Note**: Controls the number of documents read from MongoDB in each batch

#### `max_pool_size` (Optional)
- **Type**: Integer
- **Description**: Maximum number of connections in the connection pool
- **Default**: `10`
- **Range**: 1-100
- **Note**: Controls the number of concurrent connections to MongoDB

#### `timeout` (Optional)
- **Type**: String
- **Description**: Connection timeout
- **Default**: `"30s"`
- **Format**: Go time format (e.g., `"5s"`, `"1m"`, `"2h"`)

#### `enable_projection` (Optional)
- **Type**: Boolean
- **Description**: Whether to enable projection pushdown optimization
- **Default**: `true`
- **Note**: When enabled, only necessary fields are retrieved, improving performance

#### `enable_index_hint` (Optional)
- **Type**: Boolean
- **Description**: Whether to enable index hints
- **Default**: `true`
- **Note**: When enabled, suggests MongoDB to use specific indexes

## Configuration Examples

### Example 1: Basic User Synchronization

```json
{
  "connection_uri": "mongodb://localhost:27017/userdb",
  "database": "userdb",
  "collections": [
    {
      "name": "users",
      "filter": {"status": "active"},
      "title_field": "username",
      "content_field": "profile",
      "category_field": "role",
      "tags_field": "skills",
      "timestamp_field": "last_updated"
    }
  ],
  "pagination": true,
  "page_size": 1000,
  "sync_strategy": "incremental",
  "last_modified_field": "last_updated"
}
```

### Example 2: Product Catalog Synchronization

```json
{
  "connection_uri": "mongodb://user:pass@localhost:27017/catalog",
  "database": "catalog",
  "collections": [
    {
      "name": "products",
      "filter": {"active": true, "stock": {"$gt": 0}},
      "title_field": "name",
      "content_field": "description",
      "category_field": "category",
      "tags_field": "tags",
      "url_field": "product_url",
      "timestamp_field": "updated_at"
    }
  ],
  "pagination": true,
  "page_size": 500,
  "sync_strategy": "incremental",
  "last_modified_field": "updated_at",
  "field_mapping": {
    "enabled": true,
    "mapping": {
      "id": "product_id",
      "title": "product_name",
      "content": "product_description"
    }
  }
}
```

### Example 3: High-Performance Configuration

```json
{
  "connection_uri": "mongodb://localhost:27017/analytics",
  "database": "analytics",
  "collections": [
    {
      "name": "events",
      "filter": {"type": "user_action"},
      "title_field": "event_name",
      "content_field": "event_data",
      "timestamp_field": "created_at"
    }
  ],
  "pagination": true,
  "page_size": 2000,
  "batch_size": 5000,
  "max_pool_size": 20,
  "timeout": "10s",
  "enable_projection": true,
  "enable_index_hint": true
}
```

## Best Practices

### 1. Connection Configuration
- Use environment variables for sensitive information (username, password)
- Configure appropriate connection pool size for production environments
- Set reasonable timeout values

### 2. Collections Configuration
- Use filters to reduce unnecessary data transmission
- Create indexes for timestamp fields to improve incremental synchronization performance
- Set field mappings reasonably to avoid retrieving useless data

### 3. Performance Optimization
- Adjust page size and batch size based on data volume
- Enable projection pushdown to reduce network transmission
- Use index hints to optimize query performance

### 4. Incremental Synchronization
- Ensure timestamp fields have appropriate indexes
- Regularly clean up old synchronization state files
- Monitor synchronization performance and adjust configuration parameters

## Troubleshooting

### Common Issues

#### 1. Connection Failure
- Check connection string format
- Verify network connectivity and firewall settings
- Confirm MongoDB service is running

#### 2. Poor Synchronization Performance
- Check if appropriate indexes exist
- Adjust page size and batch size
- Enable projection pushdown optimization

#### 3. Incremental Synchronization Not Working
- Confirm `last_modified_field` is set correctly
- Check timestamp field data type
- Verify incremental synchronization strategy configuration

#### 4. High Memory Usage
- Reduce page size and batch size
- Enable pagination processing
- Check field mapping configuration

## Monitoring and Logging

### Log Levels
- `DEBUG`: Detailed debug information
- `INFO`: General operation information
- `WARN`: Warning information
- `ERROR`: Error information

### Key Metrics
- Number of synchronized documents
- Processing time
- Memory usage
- Error rate

### Monitoring Recommendations
- Regularly check synchronization status
- Monitor system resource usage
- Set alert thresholds
- Record performance metrics

## Summary

MongoDB Connector provides flexible and powerful configuration options that can meet various data synchronization needs. Through reasonable configuration, efficient and reliable data synchronization can be achieved while maintaining good performance. It is recommended to adjust configuration parameters based on actual usage scenarios and regularly monitor and optimize configurations.
