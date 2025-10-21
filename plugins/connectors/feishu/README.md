# 飞书/Lark 云文档连接器

飞书/Lark 云文档连接器用于索引飞书和Lark中的云文档，包括文档、表格、思维笔记、多维表格和知识库等。

## 功能特性

- 🔍 **智能搜索**: 支持按关键词搜索云文档
- 📚 **多文档类型**: 支持 doc、sheet、slides、mindnote、bitable、file、docx、folder、shortcut 等类型
- 🔐 **双重认证**: 支持 OAuth 2.0 和用户访问令牌两种认证方式（二选一）
- ⚡ **高效同步**: 基于pipeline架构，由统一调度器管理同步
- 🔄 **递归搜索**: 自动递归搜索文件夹内容
- 🔄 **Token自动刷新**: OAuth认证支持access_token和refresh_token的自动刷新
- 🌐 **动态重定向**: 支持动态构建OAuth重定向URI，适配多环境部署
- 🏗️ **统一架构**: 飞书和Lark共享基础实现，代码复用率高达95%
- 📁 **目录访问**: 支持按飞书云文档原始目录结构的层次化浏览，自动创建文件夹目录
- 🚀 **Pipeline集成**: 完全基于pipeline架构，无独立调度任务，与其他连接器保持一致

## 支持的平台

### 飞书 (Feishu)
- **域名**: `open.feishu.cn` / `accounts.feishu.cn`
- **连接器ID**: `feishu`
- **适用地区**: 中国大陆

### Lark
- **域名**: `open.larksuite.com` / `accounts.larksuite.com`
- **连接器ID**: `lark`
- **适用地区**: 海外地区

## 认证方式

飞书/Lark连接器支持两种认证方式，**必须选择其中一种**：

### 1. OAuth 2.0 认证（推荐）

使用OAuth流程自动获取用户访问令牌，支持token自动刷新和过期时间管理。

#### 配置要求
- `client_id`: 飞书/Lark应用的Client ID
- `client_secret`: 飞书/Lark应用的Client Secret
- `document_types`: 要同步的文档类型列表

#### 认证流程
1. 用户创建飞书/Lark数据源，配置`client_id`和`client_secret`
2. 点击"连接"按钮，系统重定向到飞书/Lark授权页面
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

## 架构设计

### Pipeline架构

飞书/Lark连接器采用**pipeline-based架构**，与其他连接器保持一致：

- **处理器注册**: 在`init()`函数中注册为pipeline处理器
- **调度器管理**: 同步间隔和调度由connector_dispatcher统一管理
- **每数据源配置**: 每个数据源有独立的同步间隔和配置
- **Enrichment Pipeline支持**: 支持每个数据源可选的enrichment pipeline
- **OAuth路由注册**: OAuth路由在`init()`函数中直接注册，遵循google_drive模式
- **无独立调度任务**: 完全移除scheduled tasks，由pipeline框架处理数据获取

### 核心实现

```go
func init() {
    // 注册pipeline处理器
    pipeline.RegisterProcessorPlugin(ConnectorFeishu, NewFeishu)
    pipeline.RegisterProcessorPlugin(ConnectorLark, NewLark)

    // 注册OAuth路由
    api.HandleUIMethod(api.GET, "/connector/:id/feishu/connect", feishuConnect, api.RequireLogin())
    api.HandleUIMethod(api.GET, "/connector/:id/feishu/oauth_redirect", feishuOAuthRedirect, api.RequireLogin())

    api.HandleUIMethod(api.GET, "/connector/:id/lark/connect", larkConnect, api.RequireLogin())
    api.HandleUIMethod(api.GET, "/connector/:id/lark/oauth_redirect", larkOAuthRedirect, api.RequireLogin())
}

func (this *Plugin) Fetch(ctx *pipeline.Context, connector *common.Connector, datasource *common.DataSource) error {
    // 处理数据获取逻辑
    // 自动token刷新
    // 递归文件搜索
    // 文档收集
    return nil
}
```

## 配置架构

### 连接器级别（OAuth配置）

OAuth配置在连接器级别管理，提供更好的安全性和集中管理。

#### 飞书连接器配置
```json
{
  "id": "feishu",
  "name": "飞书云文档连接器",
  "builtin": true,
  "oauth_connect_implemented": true,
  "processor": {
    "enabled": true,
    "name": "feishu"
  },
  "config": {
    "client_id": "cli_xxxxxxxxxxxxxxxx",
    "client_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "auth_url": "https://accounts.feishu.cn/open-apis/authen/v1/authorize",
    "token_url": "https://open.feishu.cn/open-apis/authen/v2/oauth/token"
  }
}
```

#### Lark连接器配置
```json
{
  "id": "lark",
  "name": "Lark Document Connector",
  "builtin": true,
  "oauth_connect_implemented": true,
  "processor": {
    "enabled": true,
    "name": "lark"
  },
  "config": {
    "client_id": "cli_xxxxxxxxxxxxxxxx",
    "client_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "auth_url": "https://accounts.larksuite.com/open-apis/authen/v1/authorize",
    "token_url": "https://open.larksuite.com/open-apis/authen/v2/oauth/token"
  }
}
```

#### Pipeline配置 (coco.yml)
连接器由统一调度器管理：
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

### 数据源级别（自动生成）

使用OAuth认证时，数据源在OAuth流程中自动创建。系统自动生成：

#### 自动生成的飞书数据源
```json
{
  "id": "auto-generated-md5-hash",
  "name": "张三的飞书",
  "type": "connector",
  "enabled": true,
  "sync": {
    "enabled": true,
    "interval": "30s"
  },
  "connector": {
    "id": "feishu",
    "config": {
      "access_token": "u-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "refresh_token": "r-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "token_expiry": "2025-01-01T12:00:00Z",
      "refresh_token_expiry": "2025-01-31T12:00:00Z",
      "profile": {
        "user_id": "ou_xxxxxxxxxxxxxxxx",
        "name": "张三",
        "email": "zhangsan@example.com"
      }
    }
  }
}
```

#### 自动生成的Lark数据源
```json
{
  "id": "auto-generated-md5-hash",
  "name": "John's Lark",
  "type": "connector",
  "enabled": true,
  "sync": {
    "enabled": true,
    "interval": "30s"
  },
  "connector": {
    "id": "lark",
    "config": {
      "access_token": "u-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "refresh_token": "r-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "token_expiry": "2025-01-01T12:00:00Z",
      "refresh_token_expiry": "2025-01-31T12:00:00Z",
      "profile": {
        "user_id": "ou_xxxxxxxxxxxxxxxx",
        "name": "John Doe",
        "email": "john@example.com"
      }
    }
  }
}
```

## 配置参数说明

### 必填参数

| 参数 | 类型 | 说明 | 认证方式 |
|------|------|------|----------|
| `client_id` | string | 飞书/Lark应用的Client ID | OAuth认证 |
| `client_secret` | string | 飞书/Lark应用的Client Secret | OAuth认证 |
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
| `sync.enabled` | bool | true | 是否启用同步 |
| `sync.interval` | string | "30s" | 每个数据源的同步间隔 |

**注意**: 同步间隔现在在数据源级别配置，而不是连接器级别。每个数据源可以有不同的同步间隔。

## 支持的文档类型

飞书/Lark连接器支持以下云文档类型：

- **doc**: 飞书/Lark文档
- **sheet**: 飞书/Lark表格  
- **slides**: 飞书/Lark幻灯片
- **mindnote**: 飞书/Lark思维笔记
- **bitable**: 飞书/Lark多维表格
- **file**: 普通文件
- **docx**: Word文档
- **folder**: 文件夹（支持递归搜索）
- **shortcut**: 快捷方式（直接使用API返回的URL）

### 目录访问特性

- **自动创建目录**: 为每个文件夹自动创建目录文档，支持层次化浏览
- **保持原始结构**: 完全按照飞书云文档中的文件夹层次结构
- **递归处理**: 自动遍历所有子文件夹并创建对应的目录
- **混合文档类型**: 同一文件夹中可以包含不同类型的文档
- **元数据支持**: 每个目录包含创建时间、修改时间等元数据

## 飞书/Lark应用权限配置

### 必需权限

飞书/Lark连接器需要以下权限才能正常工作：

| 权限 | 权限代码 | 说明 | 用途 |
|------|----------|------|------|
| **云文档访问** | `drive:drive` | 访问用户的云文档、表格、幻灯片等 | 读取和索引云文档内容 |
| **知识库检索** | `space:document:retrieve` | 检索知识库中的文档 | 访问知识库和空间文档 |
| **离线访问** | `offline_access` | 在用户不在线时访问资源 | 支持后台同步任务 |

### 权限申请步骤

#### 飞书应用
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

#### Lark应用
1. **登录Lark开放平台**
   - 访问 [https://open.larksuite.com/](https://open.larksuite.com/)
   - 使用Lark账号登录

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

#### 步骤1: 创建飞书/Lark应用
1. 访问对应的开放平台：
   - 飞书：[飞书开放平台](https://open.feishu.cn/)
   - Lark：[Lark开放平台](https://open.larksuite.com/)
2. 创建新应用，申请以下权限：
   - **`drive:drive`** - 云文档访问权限
   - **`space:document:retrieve`** - 知识库文档检索权限  
   - **`offline_access`** - 离线访问权限
3. 记录应用的 `Client ID` 和 `Client Secret`

#### 步骤2: 配置连接器
1. 进入系统管理界面的连接器管理
2. 编辑飞书或Lark连接器配置
3. 配置以下字段：
   - `client_id`: 应用的Client ID
   - `client_secret`: 应用的Client Secret
   - `document_types`: 要同步的文档类型列表
   - `auth_url`、`token_url`、`redirect_url`: OAuth端点（预配置）
4. 保存连接器配置

#### 步骤3: 创建数据源（OAuth流程）
1. 进入数据源管理，点击"添加数据源"
2. 选择飞书或Lark连接器
3. 点击"连接"按钮（无需手动配置）
4. 系统重定向到飞书/Lark授权页面
5. 用户完成授权
6. 系统自动创建数据源，包含OAuth令牌和用户配置文件信息

### 方法2: 用户访问令牌

#### 步骤1: 获取用户访问令牌
1. 登录对应的开放平台
2. 获取用户访问令牌

#### 步骤2: 创建数据源
1. 在系统管理界面创建对应的数据源
2. 配置 `user_access_token` 和 `document_types`
3. 保存数据源配置

## 技术实现

### Pipeline架构集成

#### 重构后的架构 (2025-10版本)
- **完全Pipeline化**: 移除所有scheduled tasks，改用pipeline架构
- **Google Drive模式**: OAuth路由在`init()`中注册，与google_drive保持一致
- **统一调度**: 所有数据源由connector_dispatcher统一管理
- **插件类型抽象**: 使用`PluginType`枚举区分飞书和Lark
- **动态API配置**: 根据插件类型动态选择API端点
- **代码复用最大化**: 95%的代码被共享，只有配置和路由不同
- **OAuth配置动态加载**: OAuth凭据从connector数据库动态加载
- **自动数据源创建**: 数据源在OAuth流程中自动创建
- **ConnectorProcessorBase**: 使用统一的processor基类

#### 核心组件
```go
// 插件类型定义
type PluginType string
const (
    PluginTypeFeishu PluginType = "feishu"
    PluginTypeLark   PluginType = "lark"
)

// 统一OAuth配置结构
type OAuthConfig struct {
    // OAuth端点
    AuthURL     string
    TokenURL    string
    RedirectURL string
    
    // OAuth凭据
    ClientID         string
    ClientSecret     string
    DocumentTypes    []string
    UserAccessToken  string
}

// API配置结构
type APIConfig struct {
    BaseURL     string
    AuthURL     string
    TokenURL    string
    UserInfoURL string
    DriveURL    string
}

// 基础Plugin结构
type Plugin struct {
    // ... 原有字段
    PluginType  PluginType
    apiConfig   *APIConfig
    OAuthConfig *OAuthConfig  // 统一OAuth配置
}
```

#### 处理器实现
- **NewFeishu()**: 创建飞书处理器，设置`PluginTypeFeishu`
- **NewLark()**: 创建Lark处理器，设置`PluginTypeLark`
- **统一API处理**: 所有API调用使用动态配置的端点
- **Fetch()方法**: 实现数据获取逻辑，包括token刷新和文件递归搜索

### OAuth路由注册

#### 飞书路由
- **路由端点**:
  - `GET /connector/:id/feishu/connect` - OAuth授权请求
  - `GET /connector/:id/feishu/oauth_redirect` - OAuth回调处理

#### Lark路由
- **路由端点**:
  - `GET /connector/:id/lark/connect` - OAuth授权请求
  - `GET /connector/:id/lark/oauth_redirect` - OAuth回调处理

- **认证要求**: 所有OAuth端点都需要用户登录
- **Scope配置**: 使用 `drive:drive space:document:retrieve offline_access` 权限范围
- **动态配置加载**: OAuth配置从connector数据库动态加载，支持多connector实例

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
3. **连接器级别配置**: OAuth凭据现在在连接器级别配置，不在数据源级别
4. **自动数据源创建**: 使用OAuth时，数据源在授权流程中自动创建
5. **Token管理**: 使用用户访问令牌时，需要手动管理token的有效期
6. **权限要求**: 飞书/Lark应用需要申请并获得以下权限：
   - `drive:drive` - 云文档访问权限
   - `space:document:retrieve` - 知识库检索权限  
   - `offline_access` - 离线访问权限
7. **API限制**: 注意飞书/Lark API的调用频率限制
8. **平台选择**: 根据用户所在地区选择合适的平台（飞书适用于中国大陆，Lark适用于海外地区）

## 故障排除

### 常见问题

1. **认证失败**
   - 检查`client_id`和`client_secret`是否正确
   - 确认飞书/Lark应用是否已申请并获得了以下权限：
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

5. **平台选择错误**
   - 确认用户所在地区
   - 检查应用的域名配置是否正确
   - 验证API端点是否可访问

### 日志调试
连接器提供详细的日志记录，包括：
- OAuth流程的每个步骤
- Token刷新过程
- 过期时间检查
- 错误详情和堆栈信息
- 插件类型标识（`[feishu connector]` 或 `[lark connector]`）

使用日志可以快速定位和解决问题。

## 扩展性

重构后的架构支持轻松添加新的插件类型：

1. **定义新的插件类型**
   ```go
   const PluginTypeLarkInternational PluginType = "lark_international"
   ```

2. **添加API配置**
   ```go
   case PluginTypeLarkInternational:
       return &APIConfig{
           BaseURL: "https://open.larksuite.com",
           // ... 其他配置
       }
   ```

3. **创建新插件**
   ```go
   type LarkInternationalPlugin struct {
       Plugin
   }
   
   func (this *LarkInternationalPlugin) Setup() {
       this.SetPluginType(PluginTypeLarkInternational)
       // 其余配置自动处理
   }
   ```

这种设计为未来的功能扩展和维护奠定了良好的基础。