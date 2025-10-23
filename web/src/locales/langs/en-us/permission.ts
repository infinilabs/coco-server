const permission = {
  // 📁 Document
  document: 'Document',
  'coco#document/create': 'Create Document',
  'coco#document/read': 'Read Document',
  'coco#document/update': 'Update Document',
  'coco#document/delete': 'Delete Document',
  'coco#document/search': 'Search Document',

  // 📎 Attachment
  attachment: 'Attachment',
  'coco#attachment/create': 'Create Attachment',
  'coco#attachment/read': 'Read Attachment',
  'coco#attachment/update': 'Update Attachment',
  'coco#attachment/delete': 'Delete Attachment',
  'coco#attachment/search': 'Search Attachment',

  // 🔗 Integration
  integration: 'Integration',
  'coco#integration/create': 'Create Integration',
  'coco#integration/read': 'Read Integration',
  'coco#integration/update': 'Update Integration',
  'coco#integration/delete': 'Delete Integration',
  'coco#integration/search': 'Search Integration',
  'coco#integration/view_suggest_topics': 'View Suggested Topics',
  'coco#integration/update_suggest_topics': 'Update Suggested Topics',

  // 🔌 Connector
  connector: 'Connector',
  'coco#connector/create': 'Create Connector',
  'coco#connector/read': 'Read Connector',
  'coco#connector/update': 'Update Connector',
  'coco#connector/delete': 'Delete Connector',
  'coco#connector/search': 'Search Connector',

  // 🧩 Model Provider
  model_provider: 'Model Provider',
  'coco#model_provider/create': 'Create Model Provider',
  'coco#model_provider/read': 'Read Model Provider',
  'coco#model_provider/update': 'Update Model Provider',
  'coco#model_provider/delete': 'Delete Model Provider',
  'coco#model_provider/search': 'Search Model Provider',

  // 💬 Session
  session: 'Session',
  'coco#session/create': 'Create Session',
  'coco#session/read': 'Read Session',
  'coco#session/update': 'Update Session',
  'coco#session/delete': 'Delete Session',
  'coco#session/search': 'Search Session',
  'coco#session/view_single_session_history': 'View Single Session History',
  'coco#session/view_all_session_history': 'View All Session History',

  // 🧠 Assistant
  assistant: 'AI Assistant',
  'coco#assistant/ask': 'Ask AI Assistant',
  'coco#assistant/quick_ai_access': 'Quick AI Access',

  // 🗃️ Datasource
  datasource: 'Datasource',
  'coco#datasource/create': 'Create Datasource',
  'coco#datasource/read': 'Read Datasource',
  'coco#datasource/update': 'Update Datasource',
  'coco#datasource/delete': 'Delete Datasource',
  'coco#datasource/search': 'Search Datasource',

  // 🧱 MCP Server
  mcp_server: 'MCP Server',
  'coco#mcp_server/create': 'Create MCP Server',
  'coco#mcp_server/read': 'Read MCP Server',
  'coco#mcp_server/update': 'Update MCP Server',
  'coco#mcp_server/delete': 'Delete MCP Server',
  'coco#mcp_server/search': 'Search MCP Server',

  // 🛒 Store Extensions
  'store:extensions': 'Store Extensions',
  'coco#store:extensions/create': 'Create Extension',
  'coco#store:extensions/read': 'Read Extension',
  'coco#store:extensions/update': 'Update Extension',
  'coco#store:extensions/delete': 'Delete Extension',
  'coco#store:extensions/search': 'Search Extensions',
  'coco#store:extensions/reindex': 'Reindex Extensions',

  // ⚙️ System
  system: 'System',
  'coco#system/read': 'Read System Config',
  'coco#system/update': 'Update System Config',

  // 🧭 Search
  search: 'Search',
  'coco#search/search': 'Execute Search',

  // 🔐 Generic Security
  'security:role': 'Security Role',
  'generic#security:role/create': 'Create Role',
  'generic#security:role/read': 'Read Role',
  'generic#security:role/update': 'Update Role',
  'generic#security:role/delete': 'Delete Role',
  'generic#security:role/search': 'Search Role',

  'security:permission': 'Security Permission',
  'generic#security:permission/read': 'Read Permission',

  // 🧭 Others
  cancel_session: 'Cancel Session'
};

export default permission;
