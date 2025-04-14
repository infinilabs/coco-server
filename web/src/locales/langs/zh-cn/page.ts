const page: App.I18n.Schema['translation']['page'] = {
  home: {
    server: {
      title: '{{user}} 的 Coco Server',
      address: '服务器地址',
      addressDesc: '在 Coco AI 的连接设置中，将服务器地址添加到服务列表，你就可以在 Coco AI 中访问该服务了。',
      downloadCocoAI: '下载 Coco AI'
    },
    settings: {
      llm: '大模型',
      llmDesc: '连接大模型以启用人工智能聊天、智能搜索和工作助手功能。',
      dataSource: '数据源',
      dataSourceDesc: '将数据源添加到服务列表，以进行统一搜索和分析。',
      aiAssistant: 'AI 助手',
      aiAssistantDesc: '设置个性化的人工智能助手，以高效处理任务并提供智能建议。',
    },
  },
  settings: {
    llm: {
      type: '类型',
      endpoint: 'Endpoint',
      defaultModel: '默认模型',
      keepalive: '保持连接',
      requestParams: '请求参数',
      temperature: '随机性',
      temperature_desc: '值越大，回复越随机',
      top_p: '核采样',
      top_p_desc: '与随机性类似，但不要和随机性一起更改',
      max_tokens: '单次回复限制',
      max_tokens_desc: '单次交互所用的最大 Token 数',
      presence_penalty: '话题新鲜度',
      presence_penalty_desc: '值越大，越有可能扩展到新话题',
      frequency_penalty: '频率惩罚度',
      frequency_penalty_desc: '值越大，越有可能降低重复字词',
      enhanced_inference: '开启推理强度调整',
      intent_analysis_model: "意图识别模型",
      picking_doc_model: "文档预选模型",
      answering_model: "应答模型",
    }
  },
  login: {
    title: '欢迎',
    desc: '输入您的凭证信息以访问您的账户。',
    password: '密码',
    common: {
      back: '返回',
      codeLogin: '验证码登录',
      codePlaceholder: '请输入验证码',
      confirm: '确定',
      confirmPasswordPlaceholder: '请再次输入密码',
      loginOrRegister: '登录 / 注册',
      loginSuccess: '登录成功',
      passwordPlaceholder: '请输入密码',
      phonePlaceholder: '请输入手机号',
      userNamePlaceholder: '请输入用户名',
      validateSuccess: '验证成功',
      welcomeBack: '欢迎回来，{{userName}} ！'
    },
    cocoAI: {
      title: '打开 Coco AI',
      autoDesc: '为了继续操作，如果 5 秒内未自动重定向，请点击以下链接：',
      launchCocoAI: '启动 Coco AI',
      copyDesc: '如果重定向不起作用，您可以复制以下 URL 并将其粘贴到 Coco AI 的连接设置窗口中：',
      enterCocoServer: '进入 Coco Server',
      enterCocoServerDesc: '或者，您也可以：'
    }
  },
  guide: {
    user: {
      title: '创建一个账户',
      desc: '设置一个新的账户以管理访问权限。',
      name: '姓名',
      email: '邮箱',
      password: '密码'
    },
    llm: {
      title: '创建一个账户',
      desc: '集成大模型后，您将解锁人工智能聊天功能，还能获得智能搜索服务和高效的工作助手。',
    },
    setupLater: '稍后设置',
  },
  datasource: {
    columns: {
      name: "名称",
      type: "类型",
      sync_policy: "同步策略",
      latest_sync_time: "最近同步",
      sync_status: "同步状态",
      enabled: "启用状态",
      searchable: "可搜索",
    },
    new:{
      title: "连接 {{connector}}",
      labels: {
        name: "数据源名称",
        type: "数据源类型",
        indexing_scope: "索引范围",
        data_sync: "数据同步",
        manual_sync: "手动同步",
        manual_sync_desc: '仅在用户点击 "同步" 按钮时同步',
        scheduled_sync: "定时同步",
        scheduled_sync_desc: "每隔固定时间同步一次",
        realtime_sync: "实时同步",
        realtime_sync_desc: "文件修改立即同步",
        immediate_sync: "立即同步",
        client_id: "客户端 ID",
        client_secret: "客户端密钥",
        redirect_uri: "重定向 URI",
        sync_enabled: "是否启用同步",
        enabled: "是否启用",
        site_urls: "站点地址",
        connect: "连接",
        insert_doc: "插入文档",
      }
    },
    edit:{
      title: "编辑数据源",
    },
    delete: {
      confirm: '确定删除这个数据源？'
    },
    every: '每',
    seconds: '秒',
    minutes: '分钟',
    hours: '小时',
    connect: '连接',
    site_urls: '站点地址',
    site_urls_add: '新增站点地址'
  },
  apitoken: {
    columns: {
      name: "名称",
      expire_in: "过期时间",
    },
    delete: {
      confirm: '确定删除这个 API Token？'
    },
    create: {
      store_desc: '请将此 Token 保存在安全且易于访问的地方。出于安全原因，你将无法通过 APl Token 管理界面再次查看它。如果你丢失了这个 Token，将需要重新创建。',
      limit: "Token 数量超过限制，最多可以创建 5 个 Token。",
    }
  },
  connector: {
    columns: {
      name: "名称",
      category: "类型",
      description: "描述",
      tags: "标签",
    },
    delete: {
      confirm: '确定删除连接器 "{{name}}"？'
    },
    edit: {
      title: "编辑连接器",
    },
    new: {
      title: "新建连接器",
      labels: {
        name: "名称",
        category: "类型",
        description: "描述",
        tags: "标签",
        assets_icons: "图标资源",
        icon: "连接器图标",
        client_id: "客户端 ID",
        client_secret: "客户端密钥",
        redirect_url: "重定向地址",
        auth_url: "认证地址",
        token_url: "Token 地址",
        asset_icon: "图标",
        asset_type: "类型",
      }
    },
  },
  modelprovider: {
    labels: {
      name: "名称",
      base_url: "Base URL",
      description: "描述",
      api_type: "API 类型",
      models: "模型",
      enabled: "是否禁用",
      icon: "图标",
      api_key: "API 密钥",
      api_key_source: "从 {{model_provider}} 获取 API 密钥",
    },
    delete: {
      confirm: '您确定要删除这个模型提供商吗?'
    },
  },
  integration: {
    columns: {
      name: "名称",
      type: "类型",
      description: "描述",
      datasource: "数据源",
      enabled: "启用状态",
    },
    form: {
      title: {
        new: "新增嵌入组件",
        edit: "编辑嵌入组件"
      },
      labels: {
        name: "名称",
        type: "类型",
        type_embedded: "内嵌的",
        type_floating: "浮动的",
        type_all: "内嵌和浮动",
        type_embedded_placeholder: "内嵌组件提示文本",
        hotkey: "快捷键",
        hotkey_placeholder: "设置呼出 CocoAI 的快捷键",
        datasource: "数据源",
        enable_module: "启用模块",
        module_search: "搜索",
        module_search_placeholder: "搜索输入框提示文本",
        module_chat: "AI 聊天",
        module_chat_placeholder: "AI 聊天输入框提示文本",
        feature_Control: "功能控制",
        feature_search: "显示数据源搜索",
        feature_search_active: "开启数据源搜索",
        feature_think: "显示深度思考",
        feature_think_active: "开启深度思考",
        feature_chat_history: "聊天历史",
        access_control: "访问控制",
        enable_auth: "启用认证",
        appearance: "外观设置",
        theme: "主题",
        theme_auto: "自动",
        theme_light: "浅色",
        theme_dark: "深色",
        cors: "跨域设置",
        allow_origin: "Allow-Origin",
        allow_origin_placeholder: "请输入允许的 Origin，以 http:// 或 https:// 开头多个 Origin 间用英文逗号分隔，允许所有 Origin 访问则填 *",
        description: "描述",
      }
    },
    delete: {
      confirm: `确定删除嵌入组件 "{{name}}"？`
    },
    update: {
      enable_confirm: `确定启用嵌入组件 "{{name}}"？`,
      disable_confirm: `确定禁用嵌入组件 "{{name}}"？`
    },
    code: {
      title: "嵌入代码",
      desc: "将这段代码插入到你的网站<body>和</body>之间，即可开始搜索和聊天。",
      preview: "预览",
      exit: "退出预览"
    }
  },
  assistant: {
    labels: {
      name: "名称",
      type: "类型",
      default_model: "默认模型",
      enabled: "启用状态",
      icon: "图标",
      description: "描述",
      intent_analysis_model: "意图识别",
      picking_doc_model: "文档预选",
      deep_think_model: "深度思考模型",
      answering_model: "应答模型",
      keepalive: "保持连接",
      role_prompt: "角色提示",
      temperature: "随机性（temperature） ",
      temperature_desc: "控制生成文本的随机性，值越高，内容越丰富但不稳定；值越低，内容更可预测",
      top_p: "词汇多样性 (top_p)",
      top_p_desc: "限制选择的词汇范围，值越低，结果更可预测；值越高，可能性更多样，不推荐和随机性一起更改",
      max_tokens: "单次回复限制(max tokens)",
      max_tokens_desc: "单次交互所用的最大 Token 数",
      presence_penalty: "表述发散度（Presence Penalty）",
      presence_penalty_desc: "控制 AI 是否倾向于使用新主题，值越高，AI 更倾向于引入新内容",
      frequency_penalty: "减少重复（Frequency Penalty）",
      frequency_penalty_desc: "控制 AI 对同一词汇的重复使用，值越高，表达越丰富",
      chat_settings: "聊天设置",
      greeting_settings: "问候设置",
      suggested_chat: "推荐对话",
      input_preprocessing: "用户输入预处理",
      input_preprocessing_desc: "用户最新的一条消息会填充到此模版",
      input_preprocessing_placeholder: "预处理模版 {{text}} 将替换为实时输入信息",
      history_message_number: "附带历史消息数",
      history_message_number_desc: "每次请求携带的历史消息数",
      history_message_compression_threshold: "历史消息长度压缩阈值",
      history_message_compression_threshold_desc: "当未压缩的历史消息超过该值时，将进行压缩",
      history_summary: "历史消息摘要",
      history_summary_desc: "自动压缩聊天记录并作为上下文发送",
      model_settings: "模型设置",
      datasource: "数据源",
      mcp_servers: "MCP 服务器",
      show_in_chat: "在聊天界面中显示",
    },
    mode:{
      simple: "简单模式",
      deep_think: "深度思考",
      workflow: "外部工作流",
    },
    delete: {
      confirm: 'Are you sure you want to delete this ai assistant "{{name}}"?'
    },
  },
  mcpserver: {
    labels: {
      name: "名称",
      type: "类型",
      enabled: "启用",
      description: "描述",
      config: {
        command: "命令",
        args: "参数",
        env: "环境变量",
      }
    },
    delete: {
      confirm: 'Are you sure you want to delete this ai MCP server "{{name}}"?'
    },
  },
};

export default page;
