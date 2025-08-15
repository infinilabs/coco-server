# MongoDB Connector 配置指南

## 概述

MongoDB Connector 是一个强大的数据连接器，支持从MongoDB数据库高效地同步数据。它提供了灵活的配置选项，支持增量同步、字段映射、分页处理等高级功能。

## 配置结构

### 基础配置

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

## 配置参数详解

### 1. 连接配置

#### `connection_uri` (必需)
- **类型**: 字符串
- **描述**: MongoDB连接字符串
- **格式**: `mongodb://[username:password@]host[:port]/database[?options]`
- **示例**: 
  - `mongodb://localhost:27017/test`
  - `mongodb://user:pass@localhost:27017/test?authSource=admin`
  - `mongodb://localhost:27017,localhost:27018/test?replicaSet=rs0`

#### `database` (必需)
- **类型**: 字符串
- **描述**: 要连接的MongoDB数据库名称
- **示例**: `"test"`, `"production"`, `"analytics"`

### 2. 集合配置

#### `collections` (必需)
- **类型**: 数组
- **描述**: 要同步的集合列表
- **每个集合包含以下字段**:

##### `name` (必需)
- **类型**: 字符串
- **描述**: 集合名称
- **示例**: `"users"`, `"products"`, `"orders"`

##### `filter` (可选)
- **类型**: 对象
- **描述**: MongoDB查询过滤器，用于限制同步的文档
- **示例**: 
  ```json
  {"status": "active"}
  {"age": {"$gte": 18}}
  {"category": {"$in": ["tech", "business"]}}
  ```

##### `title_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档标题的字段名
- **示例**: `"name"`, `"title"`, `"subject"`

##### `content_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档内容的字段名
- **示例**: `"bio"`, `"description"`, `"body"`

##### `category_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档分类的字段名
- **示例**: `"category"`, `"type"`, `"department"`

##### `tags_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档标签的字段名
- **示例**: `"tags"`, `"keywords"`, `"labels"`

##### `url_field` (可选)
- **类型**: 字符串
- **描述**: 用作文档URL的字段名
- **示例**: `"url"`, `"link"`, `"website"`

##### `timestamp_field` (可选)
- **类型**: 字符串
- **描述**: 用作时间戳的字段名，用于增量同步
- **示例**: `"updated_at"`, `"modified"`, `"timestamp"`

### 3. 分页配置

#### `pagination` (可选)
- **类型**: 布尔值
- **描述**: 是否启用分页处理
- **默认值**: `false`
- **说明**: 启用分页可以提高大数据集的处理性能

#### `page_size` (可选)
- **类型**: 整数
- **描述**: 每页处理的文档数量
- **默认值**: `500`
- **范围**: 1-10000
- **说明**: 较小的页面大小可以减少内存使用，较大的页面大小可以提高处理效率

### 4. 增量同步配置

#### `last_modified_field` (可选)
- **类型**: 字符串
- **描述**: 用于增量同步的时间戳字段名
- **示例**: `"updated_at"`, `"modified"`, `"last_updated"`
- **说明**: 设置此字段后，系统将只同步该字段值大于上次同步时间的文档

#### `sync_strategy` (可选)
- **类型**: 字符串
- **描述**: 同步策略
- **可选值**: `"full"`, `"incremental"`
- **默认值**: `"full"`
- **说明**: 
  - `"full"`: 全量同步，每次同步所有文档
  - `"incremental"`: 增量同步，只同步新增或更新的文档

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
- **格式**: `{"目标字段": "源字段"}`
- **示例**:
  ```json
  {
    "id": "user_id",
    "title": "user_name",
    "content": "user_bio",
    "category": "user_role"
  }
  ```

### 6. 性能优化配置

#### `batch_size` (可选)
- **类型**: 整数
- **描述**: 批处理大小
- **默认值**: `1000`
- **范围**: 100-10000
- **说明**: 控制每次从MongoDB读取的文档数量

#### `max_pool_size` (可选)
- **类型**: 整数
- **描述**: 连接池最大连接数
- **默认值**: `10`
- **范围**: 1-100
- **说明**: 控制与MongoDB的并发连接数

#### `timeout` (可选)
- **类型**: 字符串
- **描述**: 连接超时时间
- **默认值**: `"30s"`
- **格式**: Go时间格式（如 `"5s"`, `"1m"`, `"2h"`）

#### `enable_projection` (可选)
- **类型**: 布尔值
- **描述**: 是否启用投影下推优化
- **默认值**: `true`
- **说明**: 启用后只获取必要的字段，提高性能

#### `enable_index_hint` (可选)
- **类型**: 布尔值
- **描述**: 是否启用索引提示
- **默认值**: `true`
- **说明**: 启用后建议MongoDB使用特定索引

## 配置示例

### 示例1: 基础用户同步

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

### 示例2: 产品目录同步

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

### 示例3: 高性能配置

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

## 最佳实践

### 1. 连接配置
- 使用环境变量存储敏感信息（用户名、密码）
- 为生产环境配置适当的连接池大小
- 设置合理的超时时间

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

#### 2. 同步性能差
- 检查是否有适当的索引
- 调整页面大小和批处理大小
- 启用投影下推优化

#### 3. 增量同步不工作
- 确认`last_modified_field`设置正确
- 检查时间戳字段的数据类型
- 验证增量同步策略配置

#### 4. 内存使用过高
- 减少页面大小和批处理大小
- 启用分页处理
- 检查字段映射配置

## 监控和日志

### 日志级别
- `DEBUG`: 详细的调试信息
- `INFO`: 一般操作信息
- `WARN`: 警告信息
- `ERROR`: 错误信息

### 关键指标
- 同步文档数量
- 处理时间
- 内存使用情况
- 错误率

### 监控建议
- 定期检查同步状态
- 监控系统资源使用
- 设置告警阈值
- 记录性能指标
