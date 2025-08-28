# 飞书云文档连接器

飞书云文档连接器用于索引飞书中的云文档，包括文档、表格、思维笔记、多维表格和知识库等。

## 功能特性

- 🔍 **智能搜索**: 支持按关键词搜索云文档
- 📚 **多文档类型**: 支持 doc、sheet、slides、mindnote、bitable、file、docx、folder、shortcut 等类型
- 🔐 **双重认证**: 支持 OAuth 2.0 和用户访问令牌两种认证方式（二选一）
- ⚡ **高效同步**: 支持定时同步和手动同步
- 🔄 **递归搜索**: 自动递归搜索文件夹内容

## 认证方式

飞书连接器支持两种认证方式，**必须选择其中一种**：

### 1. OAuth 2.0 认证（推荐）

使用OAuth流程自动获取用户访问令牌，支持token自动刷新。

#### 配置要求
- `client_id`: 飞书应用的Client ID
- `client_secret`: 飞书应用的Client Secret
- `document_types`: 要同步的文档类型列表

#### 认证流程
1. 用户创建飞书数据源，配置`client_id`和`client_secret`
2. 点击"连接"按钮，系统重定向到飞书授权页面
3. 用户完成授权，系统自动获取`access_token`和`refresh_token`
4. 系统自动更新数据源配置，包含完整的OAuth信息

#### 优势
- 安全性高，无需手动管理token
- 支持token自动刷新
- 自动获取用户信息
- 符合OAuth 2.0标准

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
    oauth:
      auth_url: "https://accounts.feishu.cn/open-apis/authen/v1/authorize"
      token_url: "https://open.feishu.cn/open-apis/authen/v2/oauth/token"
      redirect_uri: "/connector/feishu/oauth_redirect"
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
| `token_expiry` | string | 令牌过期时间 | OAuth流程自动获取 |
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

## 使用方法

### 方法1: OAuth认证（推荐）

#### 步骤1: 创建飞书应用
1. 访问 [飞书开放平台](https://open.feishu.cn/)
2. 创建新应用，申请 `drive:read` 权限
3. 记录应用的 `Client ID` 和 `Client Secret`

#### 步骤2: 创建数据源
1. 在系统管理界面创建飞书数据源
2. 配置 `client_id`、`client_secret` 和 `document_types`
3. 保存数据源配置

#### 步骤3: OAuth认证
1. 点击"连接"按钮
2. 系统重定向到飞书授权页面
3. 用户完成授权
4. 系统自动更新数据源，包含OAuth token信息

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

### 特殊类型处理

#### 递归文件夹搜索
连接器自动递归搜索文件夹内容，确保所有子文件夹中的文档都能被索引。

## 注意事项

1. **认证方式二选一**: 必须选择OAuth认证或用户访问令牌认证中的一种，不能同时使用
2. **OAuth推荐**: 建议使用OAuth认证，安全性更高，支持token自动刷新
3. **Token管理**: 使用用户访问令牌时，需要手动管理token的有效期
4. **权限要求**: 飞书应用需要`drive:read`权限才能访问云文档
5. **API限制**: 注意飞书API的调用频率限制

## 故障排除

### 常见问题

1. **认证失败**
   - 检查`client_id`和`client_secret`是否正确
   - 确认飞书应用是否有`drive:read`权限

2. **Token过期**
   - OAuth认证：系统会自动刷新token
   - 用户访问令牌：需要手动更新token

3. **同步失败**
   - 检查网络连接
   - 确认token是否有效
   - 查看系统日志获取详细错误信息