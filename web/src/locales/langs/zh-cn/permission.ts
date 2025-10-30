const permission = {
  // 📁 文档
  document: '文档',
  'coco#document/create': '创建文档',
  'coco#document/read': '读取文档',
  'coco#document/update': '更新文档',
  'coco#document/delete': '删除文档',
  'coco#document/search': '搜索文档',

  // 📎 附件
  attachment: '附件',
  'coco#attachment/create': '创建附件',
  'coco#attachment/read': '读取附件',
  'coco#attachment/update': '更新附件',
  'coco#attachment/delete': '删除附件',
  'coco#attachment/search': '搜索附件',

  // 🔗 集成
  integration: '集成',
  'coco#integration/create': '创建集成',
  'coco#integration/read': '读取集成',
  'coco#integration/update': '更新集成',
  'coco#integration/delete': '删除集成',
  'coco#integration/search': '搜索集成',
  'coco#integration/view_suggest_topics': '查看推荐主题',
  'coco#integration/update_suggest_topics': '更新推荐主题',

  // 🔌 连接器
  connector: '连接器',
  'coco#connector/create': '创建连接器',
  'coco#connector/read': '读取连接器',
  'coco#connector/update': '更新连接器',
  'coco#connector/delete': '删除连接器',
  'coco#connector/search': '搜索连接器',

  // 🧩 模型提供商
  model_provider: '模型提供商',
  'coco#model_provider/create': '创建模型提供商',
  'coco#model_provider/read': '读取模型提供商',
  'coco#model_provider/update': '更新模型提供商',
  'coco#model_provider/delete': '删除模型提供商',
  'coco#model_provider/search': '搜索模型提供商',

  // 💬 会话
  session: '会话',
  'coco#session/create': '创建会话',
  'coco#session/read': '读取会话',
  'coco#session/update': '更新会话',
  'coco#session/delete': '删除会话',
  'coco#session/search': '搜索会话',
  'coco#session/view_single_session_history': '查看单个会话历史',
  'coco#session/view_all_session_history': '查看所有会话历史',

  // 🧠 AI 助手
  assistant: 'AI 助手',
  'coco#assistant/create': '创建 AI 助手',
  'coco#assistant/read': '读取 AI 助手',
  'coco#assistant/update': '更新 AI 助手',
  'coco#assistant/delete': '删除 AI 助手',
  'coco#assistant/search': '搜索 AI 助手',
  'coco#assistant/ask': '提问 AI 助手',
  'coco#assistant/quick_ai_access': '快速访问',

  // 🗃️ 数据源
  datasource: '数据源',
  'coco#datasource/create': '创建数据源',
  'coco#datasource/read': '读取数据源',
  'coco#datasource/update': '更新数据源',
  'coco#datasource/delete': '删除数据源',
  'coco#datasource/search': '搜索数据源',

  // 🧱 MCP 服务
  mcp_server: 'MCP 服务',
  'coco#mcp_server/create': '创建 MCP 服务',
  'coco#mcp_server/read': '读取 MCP 服务',
  'coco#mcp_server/update': '更新 MCP 服务',
  'coco#mcp_server/delete': '删除 MCP 服务',
  'coco#mcp_server/search': '搜索 MCP 服务',

  // 🛒 扩展商店
  'store:extensions': '扩展商店',
  'coco#store:extensions/create': '创建扩展',
  'coco#store:extensions/read': '读取扩展',
  'coco#store:extensions/update': '更新扩展',
  'coco#store:extensions/delete': '删除扩展',
  'coco#store:extensions/search': '搜索扩展',
  'coco#store:extensions/reindex': '重新索引扩展',

  // ⚙️ 系统
  system: '系统',
  'coco#system/read': '读取系统配置',
  'coco#system/update': '更新系统配置',

  // 🧭 搜索
  search: '搜索',
  'coco#search/search': '执行搜索',

  // 🔐 通用安全
  'security:role': '角色',
  'generic#security:role/create': '创建角色',
  'generic#security:role/read': '读取角色',
  'generic#security:role/update': '更新角色',
  'generic#security:role/delete': '删除角色',
  'generic#security:role/search': '搜索角色',

  'security:permission': '权限',
  'generic#security:permission/read': '读取权限',

  'entity:card': '实体卡片',
  'generic#entity:card/read': '读取实体卡片',

  'entity:label': '实体标签',
  'generic#entity:label/read': '读取实体标签',

  'sharing': '资源分享',
  'generic#sharing/read': '读取资源分享',
  'generic#sharing/create': '创建资源分享',
  'generic#sharing/update': '更新资源分享',
  'generic#sharing/delete': '删除资源分享',
  'generic#sharing/search': '搜索资源分享',

  'security:authorization': '授权',
  'generic#security:authorization/read': '读取授权',
  'generic#security:authorization/create': '创建授权',
  'generic#security:authorization/update': '更新授权',
  'generic#security:authorization/delete': '删除授权',
  'generic#security:authorization/search': '搜索授权',

  'security:principal': '资源',
  'generic#security:principal/update': '更新资源',
  'generic#security:principal/search': '搜索资源',
  
  'security:user': '用户',
  'generic#security:user/create': '创建用户',
  'generic#security:user/read': '读取用户',
  'generic#security:user/update': '更新用户',
  'generic#security:user/delete': '删除用户',
  'generic#security:user/search': '搜索用户',

  'security:auth:api-token': 'API Token',
  'generic#security:auth:api-token/create': '创建 API Token',
  'generic#security:auth:api-token/update': '更新 API Token',
  'generic#security:auth:api-token/delete': '删除 API Token',
  'generic#security:auth:api-token/search': '搜索 API Token',

  // 🧭 其他
  cancel_session: '取消会话'
};

export default permission;
