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

  // 🧩 模型提供方
  model_provider: '模型提供方',
  'coco#model_provider/create': '创建模型提供方',
  'coco#model_provider/read': '读取模型提供方',
  'coco#model_provider/update': '更新模型提供方',
  'coco#model_provider/delete': '删除模型提供方',
  'coco#model_provider/search': '搜索模型提供方',

  // 💬 会话
  session: '会话',
  'coco#session/create': '创建会话',
  'coco#session/read': '读取会话',
  'coco#session/update': '更新会话',
  'coco#session/delete': '删除会话',
  'coco#session/search': '搜索会话',
  'coco#session/view_single_session_history': '查看单个会话历史',
  'coco#session/view_all_session_history': '查看所有会话历史',

  // 🧠 智能助手
  assistant: '智能助手',
  'coco#assistant/ask': '提问智能助手',
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
  'security:role': '安全角色',
  'generic#security:role/create': '创建角色',
  'generic#security:role/read': '读取角色',
  'generic#security:role/update': '更新角色',
  'generic#security:role/delete': '删除角色',
  'generic#security:role/search': '搜索角色',

  'security:permission': '安全权限',
  'generic#security:permission/read': '读取权限',

  // 🧭 其他
  cancel_session: '取消会话'
};

export default permission;
