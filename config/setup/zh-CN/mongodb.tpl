# MongoDB 连接器配置

## 概述

MongoDB 连接器是一个强大的数据连接器，支持从MongoDB数据库高效地同步数据。它提供了灵活的配置选项，支持增量同步、字段映射、分页处理等高级功能。

## 配置结构

### 基础配置

```json
{
  "connection_uri": "mongodb://username:password@localhost:27017/database",
  "database": "database_name",
  "auth_database": "admin",
  "cluster_type": "standalone",
  "collections": [
    {
      "name": "collection_name",
      "filter": {"status": "active"},
      "title_field": "title",
      "content_field": "content"
    }
  ],
  "pagination": true,
  "page_size": 500,
  "last_modified_field": "updated_at",
  "field_mapping": {
    "enabled": true,
    "mapping": {
      "id": "custom_id",
      "title": "custom_title"
    }
  }
}
```

## 配置参数详解

### 1. 连接配置

#### `connection_uri` (必需)
- **类型**: 字符串
- **描述**: MongoDB连接字符串
- **格式**: `mongodb://[username:password@]host[:port]/database[?options]`
- **示例**: 
  - `mongodb://localhost:27017/test`
  - `mongodb://user:pass@localhost:27017/test`
  - `mongodb://localhost:27017,localhost:27018/test?replicaSet=rs0`

#### `database` (必需)
- **类型**: 字符串
- **描述**: 要连接的MongoDB数据库名称
- **示例**: `"test"`, `"production"`, `"analytics"`

#### `auth_database` (可选)
- **类型**: 字符串
- **描述**: 认证数据库名称，用户凭据存储的数据库
- **默认值**: `"admin"`
- **说明**: 当用户存在于admin数据库而不是目标数据库中时，需要设置此字段
- **示例**: `"admin"`, `"auth"`

#### `cluster_type` (可选)
- **类型**: 字符串
- **描述**: MongoDB集群类型，影响连接优化和读写策略
- **默认值**: `"standalone"`
- **可选值**: 
  - `"standalone"`: 单机MongoDB实例
  - `"replica_set"`: 复制集集群
  - `"sharded"`: 分片集群
- **说明**: 根据集群类型自动优化连接参数、读写偏好和写入关注点

### 2. 集合配置

#### `collections` (必需)
- **类型**: 数组
- **描述**: 要同步的集合列表

##### `name` (必需)
- **类型**: 字符串
- **描述**: 集合名称

##### `filter` (可选)
- **类型**: 对象
- **描述**: MongoDB查询过滤器，用于限制同步的文档

##### `title_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档标题的字段名

##### `content_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档内容的字段名

##### `category_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档分类的字段名

##### `tags_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档标签的字段名

##### `url_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档URL的字段名

##### `timestamp_field` (可选)
- **类型**: 字符串
- **描述**: 用作时间戳的字段名，用于增量同步

### 3. 分页配置

#### `pagination` (可选)
- **类型**: 布尔值
- **描述**: 是否启用分页处理
- **默认值**: `false`

#### `page_size` (可选)
- **类型**: 整数
- **描述**: 每页处理的文档数量
- **默认值**: `500`
- **范围**: 1-10000

### 4. 增量同步配置

#### `last_modified_field` (可选)
- **类型**: 字符串
- **描述**: 用于增量同步的时间戳字段名

#### `sync_strategy` (可选)
- **类型**: 字符串
- **描述**: 同步策略
- **可选值**: `"full"`, `"incremental"`
- **默认值**: `"full"`

### 5. 字段映射配置

#### `field_mapping` (可选)
- **类型**: 对象
- **描述**: 全局字段映射配置

##### `enabled` (必需)
- **类型**: 布尔值
- **描述**: 是否启用字段映射
- **默认值**: `false`

##### `mapping` (必需)
- **类型**: 对象
- **描述**: 字段映射规则

### 6. 性能优化配置

#### `batch_size` (可选)
- **类型**: 整数
- **描述**: 批处理大小
- **默认值**: `1000`

#### `max_pool_size` (可选)
- **类型**: 整数
- **描述**: 连接池最大连接数
- **默认值**: `10`

#### `timeout` (可选)
- **类型**: 字符串
- **描述**: 连接超时时间
- **默认值**: `"30s"`

#### `enable_projection` (可选)
- **类型**: 布尔值
- **描述**: 是否启用投影下推优化
- **默认值**: `true`

#### `enable_index_hint` (可选)
- **类型**: 布尔值
- **描述**: 是否启用索引提示
- **默认值**: `true`

## 配置示例

### 示例1: 基础用户同步（带认证）

```json
{
  "connection_uri": "mongodb://user:pass@localhost:27017/userdb",
  "database": "userdb",
  "auth_database": "admin",
  "cluster_type": "replica_set",
  "collections": [
    {
      "name": "users",
      "filter": {"status": "active"},
      "title_field": "username",
      "content_field": "profile",
      "timestamp_field": "last_updated"
    }
  ],
  "pagination": true,
  "page_size": 1000,
  "sync_strategy": "incremental",
  "last_modified_field": "last_updated"
}
```

### 示例2: 产品目录同步

```json
{
  "connection_uri": "mongodb://user:pass@localhost:27017/catalog",
  "database": "catalog",
  "auth_database": "admin",
  "cluster_type": "sharded",
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

### 示例3: 高性能配置

```json
{
  "connection_uri": "mongodb://localhost:27017/analytics",
  "database": "analytics",
  "auth_database": "admin",
  "cluster_type": "standalone",
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

## 集群类型说明

### 集群类型对性能的影响

MongoDB 连接器根据不同的集群类型自动优化连接参数和读写策略：

#### 1. **Standalone (单机实例)**
- **读写偏好**: `PrimaryPreferred` - 优先从主节点读取，主节点不可用时从其他节点读取
- **写入关注点**: 默认 - 写入到主节点即可
- **适用场景**: 开发环境、小型应用、单机部署

#### 2. **Replica Set (复制集)**
- **读写偏好**: `SecondaryPreferred` - 优先从从节点读取，分散主节点负载
- **写入关注点**: `{W: "majority", J: true}` - 写入到多数节点并等待日志持久化
- **重试写入**: 启用 - 网络故障时自动重试
- **适用场景**: 生产环境、高可用性要求、读写分离

#### 3. **Sharded Cluster (分片集群)**
- **读写偏好**: `Nearest` - 从最近的节点读取，减少网络延迟
- **写入关注点**: `{W: "majority", J: true}` - 写入到多数分片并等待日志持久化
- **重试写入**: 启用 - 分片间网络故障时自动重试
- **适用场景**: 大数据量、高并发、地理分布式部署

### 自动优化特性

- **连接池管理**: 根据集群类型自动调整连接池大小和超时设置
- **读写分离**: 复制集和分片集群自动启用读写分离优化
- **故障恢复**: 自动检测节点故障并切换到可用节点
- **性能监控**: 根据集群类型提供相应的性能指标

## 认证数据库说明

### 为什么需要认证数据库？

在MongoDB中，用户认证信息通常存储在`admin`数据库中，而不是在业务数据库中。当连接到需要认证的MongoDB实例时，需要指定正确的认证数据库。

### 认证数据库配置方式

1. **通过连接字符串**：
   ```
   mongodb://username:password@localhost:27017/database?authSource=admin
   ```

2. **通过配置字段**（推荐）：
   ```json
   {
     "connection_uri": "mongodb://username:password@localhost:27017/database",
     "auth_database": "admin"
   }
   ```

### 常见认证场景

- **用户存在于admin数据库**：设置 `"auth_database": "admin"`
- **用户存在于目标数据库**：设置 `"auth_database": "database_name"` 或留空
- **无认证**：连接字符串中不包含用户名密码，`auth_database` 字段无效

## 最佳实践

### 1. 连接配置
- 使用环境变量存储敏感信息（用户名、密码）
- 为生产环境配置适当的连接池大小
- 设置合理的超时时间
- 正确配置认证数据库
- 根据实际部署选择正确的集群类型

### 2. 集合配置
- 使用过滤器减少不必要的数据传输
- 为时间戳字段创建索引以提高增量同步性能
- 合理设置字段映射，避免获取无用数据

### 3. 性能优化
- 根据数据量调整页面大小和批处理大小
- 启用投影下推减少网络传输
- 使用索引提示优化查询性能

### 4. 增量同步
- 确保时间戳字段有适当的索引
- 定期清理旧的同步状态文件
- 监控同步性能，调整配置参数

## 故障排除

### 常见问题

#### 1. 连接失败
- 检查连接字符串格式
- 验证网络连接和防火墙设置
- 确认MongoDB服务正在运行
- 检查认证数据库配置是否正确
- 确认集群类型配置与实际部署一致

#### 2. 认证失败
- 确认用户名和密码正确
- 检查用户是否存在于指定的认证数据库中
- 验证用户是否有访问目标数据库的权限
- 检查MongoDB的认证机制（SCRAM-SHA-1, SCRAM-SHA-256等）

#### 3. 同步性能差
- 检查是否有适当的索引
- 调整页面大小和批处理大小
- 启用投影下推优化

#### 4. 增量同步不工作
- 确认`last_modified_field`设置正确
- 检查时间戳字段的数据类型
- 验证增量同步策略配置

#### 5. 集群性能问题
- 检查集群类型配置是否正确
- 验证读写偏好设置是否适合业务需求
- 确认连接池大小适合集群规模
- 检查网络延迟和带宽限制

## 总结

MongoDB 连接器现在完全支持认证数据库和集群类型配置，提供了灵活且强大的配置选项，可以满足各种数据同步需求。通过合理配置，特别是正确设置认证数据库和集群类型，可以实现高效、可靠的数据同步，同时保持良好的性能表现。

### 新增功能亮点

1. **集群类型感知**: 自动识别并优化不同集群类型的连接参数
2. **智能读写分离**: 根据集群类型自动选择最优的读写策略
3. **故障恢复增强**: 复制集和分片集群的自动故障检测和恢复
4. **性能自动调优**: 根据集群类型自动调整连接池和超时设置

这些改进使得MongoDB 连接器能够更好地适应不同的生产环境，提供更稳定、更高效的数据同步服务。
