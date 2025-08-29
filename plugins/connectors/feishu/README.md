# 飞书云文档连接器

飞书云文档连接器用于索引飞书中的云文档，包括文档、表格、思维笔记、多维表格和知识库等。

## 功能特性

- 🔍 **智能搜索**: 支持按关键词搜索云文档
- 📚 **多文档类型**: 支持 doc、sheet、slides、mindnote、bitable、file、docx、folder、shortcut 等类型
- 🔐 **双重认证**: 支持 OAuth 2.0 和用户访问令牌两种认证方式（二选一）
- ⚡ **高效同步**: 支持定时同步和手动同步
- 🔄 **递归搜索**: 自动递归搜索文件夹内容
- 🔄 **Token自动刷新**: OAuth认证支持access_token和refresh_token的自动刷新
- 🌐 **动态重定向**: 支持动态构建OAuth重定向URI，适配多环境部署

## 认证方式

飞书连接器支持两种认证方式，**必须选择其中一种**：

### 1. OAuth 2.0 认证（推荐）

使用OAuth流程自动获取用户访问令牌，支持token自动刷新和过期时间管理。

#### 配置要求
- `client_id`: 飞书应用的Client ID
- `client_secret`: 飞书应用的Client Secret
- `document_types`: 要同步的文档类型列表

#### 认证流程
1. 用户创建飞书数据源，配置`client_id`和`client_secret`
2. 点击"连接"按钮，系统重定向到飞书授权页面
3. 用户完成授权，系统自动获取`access_token`和`refresh_token`
4. 系统自动更新数据源配置，包含完整的OAuth信息和过期时间

#### 优势
- 安全性高，无需手动管理token
- 支持access_token和refresh_token的自动刷新
- 自动管理token过期时间
- 自动获取用户信息
- 符合OAuth 2.0标准
- 支持多环境部署（动态重定向URI）

### 2. 用户访问令牌认证（备选）

直接使用用户的访问令牌，适用于已有token的场景。

#### 配置要求
- `user_access_token`: 用户的访问令牌
- `document_types`: 要同步的文档类型列表

#### 使用场景
- 已有有效的用户访问令牌
- 不想使用OAuth流程
- 测试或开发环境

#### 注意事项
- 需要手动管理token的有效期
- token过期后需要手动更新
- 安全性相对较低

## 配置架构

### 连接器级别（固定配置）
```yaml
connector:
  feishu:
    enabled: true
    interval: "30s"
    page_size: 100
    o_auth_config:
      auth_url: "https://accounts.feishu.cn/open-apis/authen/v1/authorize"
      token_url: "https://open.feishu.cn/open-apis/authen/v2/oauth/token"
      redirect_uri: "/connector/feishu/oauth_redirect"  # 动态构建，支持多环境
```

### 数据源级别（用户配置）
```yaml
datasource:
  name: "飞书云文档"
  connector:
    id: "feishu"
    config:
      # 方式1: OAuth认证（推荐）
      client_id: "cli_xxxxxxxxxxxxxxxx"
      client_secret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
      document_types: ["doc", "sheet", "slides", "mindnote", "bitable"]
      
      # 方式2: 用户访问令牌（备选）
      # user_access_token: "u-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
      # document_types: ["doc", "sheet", "slides", "mindnote", "bitable"]
```

## 配置参数说明

### 必填参数

| 参数 | 类型 | 说明 | 认证方式 |
|------|------|------|----------|
| `client_id` | string | 飞书应用的Client ID | OAuth认证 |
| `client_secret` | string | 飞书应用的Client Secret | OAuth认证 |
| `user_access_token` | string | 用户访问令牌 | 令牌认证 |
| `document_types` | []string | 要同步的文档类型列表 | 两种方式都需要 |

### OAuth自动填充字段

| 参数 | 类型 | 说明 | 来源 |
|------|------|------|------|
| `access_token` | string | 访问令牌 | OAuth流程自动获取 |
| `refresh_token` | string | 刷新令牌 | OAuth流程自动获取 |
| `token_expiry` | string | 访问令牌过期时间 | OAuth流程自动获取 |
| `refresh_token_expiry` | string | 刷新令牌过期时间 | OAuth流程自动获取 |
| `profile` | object | 用户信息 | OAuth流程自动获取 |

### 同步配置

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page_size` | int | 100 | 每页获取的文件数量 |
| `interval` | string | "30s" | 同步间隔 |

## 支持的文档类型

飞书连接器支持以下云文档类型：

- **doc**: 飞书文档
- **sheet**: 飞书表格  
- **slides**: 飞书幻灯片
- **mindnote**: 飞书思维笔记
- **bitable**: 飞书多维表格
- **file**: 普通文件
- **docx**: Word文档
- **folder**: 文件夹（支持递归搜索）
- **shortcut**: 快捷方式（直接使用API返回的URL）

## 飞书应用权限配置

### 必需权限

飞书连接器需要以下权限才能正常工作：

| 权限 | 权限代码 | 说明 | 用途 |
|------|----------|------|------|
| **云文档访问** | `drive:drive` | 访问用户的云文档、表格、幻灯片等 | 读取和索引云文档内容 |
| **知识库检索** | `space:document:retrieve` | 检索知识库中的文档 | 访问知识库和空间文档 |
| **离线访问** | `offline_access` | 在用户不在线时访问资源 | 支持后台同步任务 |

### 权限申请步骤

1. **登录飞书开放平台**
   - 访问 [https://open.feishu.cn/](https://open.feishu.cn/)
   - 使用飞书账号登录

2. **创建应用**
   - 点击"创建应用"
   - 选择"企业自建应用"
   - 填写应用名称和描述

3. **申请权限**
   - 进入"权限管理"页面
   - 搜索并添加上述三个权限
   - 提交权限申请

4. **发布应用**
   - 完成权限申请后，发布应用到企业
   - 记录应用的 `Client ID` 和 `Client Secret`

### 权限说明

- **`drive:drive`**: 这是访问云文档的核心权限，允许应用读取用户的文档、表格、幻灯片等文件
- **`space:document:retrieve`**: 用于访问知识库和空间中的文档，扩展了文档访问范围
- **`offline_access`**: 允许应用在用户不在线时访问资源，这对于后台同步任务至关重要

## 使用方法

### 方法1: OAuth认证（推荐）

#### 步骤1: 创建飞书应用
1. 访问 [飞书开放平台](https://open.feishu.cn/)
2. 创建新应用，申请以下权限：
   - **`drive:drive`** - 云文档访问权限
   - **`space:document:retrieve`** - 知识库文档检索权限  
   - **`offline_access`** - 离线访问权限
3. 记录应用的 `Client ID` 和 `Client Secret`

#### 步骤2: 创建数据源
1. 在系统管理界面创建飞书数据源
2. 配置 `client_id`、`client_secret` 和 `document_types`
3. 保存数据源配置

#### 步骤3: OAuth认证
1. 点击"连接"按钮
2. 系统重定向到飞书授权页面
3. 用户完成授权
4. 系统自动更新数据源，包含OAuth token信息和过期时间

### 方法2: 用户访问令牌

#### 步骤1: 获取用户访问令牌
1. 登录飞书开放平台
2. 获取用户访问令牌

#### 步骤2: 创建数据源
1. 在系统管理界面创建飞书数据源
2. 配置 `user_access_token` 和 `document_types`
3. 保存数据源配置

## 技术实现

### 架构设计
- **BasePlugin继承**: 继承自`connectors.BasePlugin`
- **模块化设计**: OAuth处理逻辑分离到独立的`api.go`文件
- **类型安全**: 使用Go的类型系统确保配置和数据的类型安全

### OAuth路由注册
- **路由端点**: 
  - `GET /connector/feishu/connect` - OAuth授权请求
  - `GET /connector/feishu/oauth_redirect` - OAuth回调处理
- **认证要求**: 所有OAuth端点都需要用户登录
- **Scope配置**: 使用 `drive:drive space:document:retrieve offline_access` 权限范围

### Token生命周期管理
- **自动刷新**: 当access_token过期时，自动使用refresh_token刷新
- **过期检查**: 同时检查access_token和refresh_token的过期时间
- **智能处理**: 如果两个token都过期，停止同步并记录错误
- **数据持久化**: 自动保存刷新后的token信息到数据源配置

### 特殊类型处理

#### 递归文件夹搜索
连接器自动递归搜索文件夹内容，确保所有子文件夹中的文档都能被索引。

## 注意事项

1. **认证方式二选一**: 必须选择OAuth认证或用户访问令牌认证中的一种，不能同时使用
2. **OAuth推荐**: 建议使用OAuth认证，安全性更高，支持token自动刷新和过期时间管理
3. **Token管理**: 使用用户访问令牌时，需要手动管理token的有效期
4. **权限要求**: 飞书应用需要申请并获得以下权限：
   - `drive:drive` - 云文档访问权限
   - `space:document:retrieve` - 知识库检索权限  
   - `offline_access` - 离线访问权限
5. **API限制**: 注意飞书API的调用频率限制

## 故障排除

### 常见问题

1. **认证失败**
   - 检查`client_id`和`client_secret`是否正确
   - 确认飞书应用是否已申请并获得了以下权限：
     - `drive:drive` - 云文档访问权限
     - `space:document:retrieve` - 知识库检索权限
     - `offline_access` - 离线访问权限
   - 检查OAuth重定向URI配置
   - 确认应用是否已发布到企业

2. **Token过期**
   - OAuth认证：系统会自动刷新token，检查refresh_token是否也过期
   - 用户访问令牌：需要手动更新token

3. **同步失败**
   - 检查网络连接
   - 确认token是否有效
   - 查看系统日志获取详细错误信息
   - 检查两个token的过期时间

4. **OAuth重定向错误**
   - 确认应用配置中的重定向URI
   - 检查网络环境是否支持动态URI构建
   - 查看系统日志中的重定向URI构建过程

### 日志调试
连接器提供详细的日志记录，包括：
- OAuth流程的每个步骤
- Token刷新过程
- 过期时间检查
- 错误详情和堆栈信息

使用日志可以快速定位和解决问题。