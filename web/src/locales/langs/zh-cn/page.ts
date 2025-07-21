const page: App.I18n.Schema['translation']['page'] = {
  apitoken: {
    columns: {
      expire_in: '过期时间',
      name: '名称'
    },
    create: {
      limit: 'Token 数量超过限制，最多可以创建 5 个 Token。',
      store_desc:
        '请将此 Token 保存在安全且易于访问的地方。出于安全原因，你将无法通过 APl Token 管理界面再次查看它。如果你丢失了这个 Token，将需要重新创建。'
    },
    delete: {
      confirm: '确定删除这个 API Token？'
    }
  },
  connector: {
    columns: {
      category: '类型',
      description: '描述',
      name: '名称',
      tags: '标签'
    },
    delete: {
      confirm: '确定删除连接器 "{{name}}"？'
    },
    edit: {
      title: '编辑连接器'
    },
    new: {
      labels: {
        asset_icon: '图标',
        asset_type: '类型',
        assets_icons: '图标资源',
        auth_url: '认证地址',
        category: '类型',
        client_id: '客户端 ID',
        client_secret: '客户端密钥',
        description: '描述',
        icon: '连接器图标',
        name: '名称',
        redirect_url: '重定向地址',
        tags: '标签',
        token_url: 'Token 地址'
      },
      title: '新建连接器'
    }
  },
  datasource: {
    columns: {
      enabled: '启用状态',
      latest_sync_time: '最近同步',
      name: '名称',
      searchable: '可搜索',
      sync_policy: '同步策略',
      sync_status: '同步状态',
      type: '类型'
    },
    connect: '连接',
    missing_config_tip: "Google 授权相关参数没有设置，需设置后才能连接，点击确认跳转到设置页面。",
    delete: {
      confirm: '确定删除这个数据源？'
    },
    edit: {
      title: '编辑数据源'
    },
    every: '每',
    hours: '小时',
    minutes: '分钟',
    new: {
      labels: {
        client_id: '客户端 ID',
        client_secret: '客户端密钥',
        connect: '连接',
        data_sync: '数据同步',
        enabled: '启用状态',
        immediate_sync: '立即同步',
        indexing_scope: '索引范围',
        insert_doc: '插入文档',
        manual_sync: '手动同步',
        manual_sync_desc: '仅在用户点击 "同步" 按钮时同步',
        name: '数据源名称',
        realtime_sync: '实时同步',
        realtime_sync_desc: '文件修改立即同步',
        redirect_uri: '重定向 URI',
        scheduled_sync: '定时同步',
        scheduled_sync_desc: '每隔固定时间同步一次',
        site_urls: '站点地址',
        sync_enabled: '是否启用同步',
        type: '数据源类型',
        folder_paths: '文件夹路径',
        file_extensions: '文件扩展名（可选）'
      },
      title: '连接 {{connector}}',
      tooltip: {
        file_extensions: '逗号分隔列表。例如 pdf、docx、txt',
        folder_paths: '您要扫描的文件夹的绝对路径。'
      }
    },
    seconds: '秒',
    site_urls: '站点地址',
    site_urls_add: '新增站点地址',
    file_paths_add: '添加文件路径'
  },
  guide: {
    llm: {
      desc: '集成大模型后，您将解锁人工智能聊天功能，还能获得智能搜索服务和高效的工作助手。',
      title: '创建一个账户'
    },
    setupLater: '稍后设置',
    user: {
      desc: '设置一个新的账户以管理访问权限。',
      email: '邮箱',
      name: '姓名',
      password: '密码',
      title: '创建一个账户',
      language: '语言',
    }
  },
  home: {
    server: {
      address: '服务器地址',
      addressDesc: '在 Coco AI 的连接设置中，将服务器地址添加到服务列表，你就可以在 Coco AI 中访问该服务了。',
      downloadCocoAI: '下载 Coco AI',
      title: '{{user}} 的 Coco Server'
    },
    settings: {
      aiAssistant: 'AI 助手',
      aiAssistantDesc: '设置个性化的人工智能助手，以高效处理任务并提供智能建议。',
      dataSource: '数据源',
      dataSourceDesc: '将数据源添加到服务列表，以进行统一搜索和分析。',
      llm: '大模型',
      llmDesc: '连接大模型以启用人工智能聊天、智能搜索和工作助手功能。'
    }
  },
  integration: {
    code: {
      desc: '将这段代码插入到你的网站<body>和</body>之间，即可开始搜索和聊天。',
      exit: '退出预览',
      preview: '预览',
      title: '添加组件到你的网站',
      enabled_tips: '请将组件启用状态设置为开启并更新保存，才能预览!'
    },
    columns: {
      datasource: '数据源',
      description: '描述',
      enabled: '启用状态',
      name: '名称',
      type: '类型',
      operation: {
        topics: '推荐话题'
      },
      token_expire_in: 'Token 过期时间',
    },
    delete: {
      confirm: `确定删除嵌入组件 "{{name}}"？`
    },
    form: {
      labels: {
        access_control: '访问控制',
        allow_origin: 'Allow-Origin',
        allow_origin_placeholder:
          '请输入允许的 Origin，以 http:// 或 https:// 开头多个 Origin 间用英文逗号分隔，允许所有 Origin 访问则填 *',
        appearance: '外观设置',
        cors: '跨域设置',
        datasource: '数据源',
        description: '描述',
        enable_auth: '启用认证',
        enable_module: '启用模块',
        feature_chat_history: '聊天历史',
        feature_Control: '功能控制',
        feature_search: '显示数据源搜索',
        feature_search_active: '开启数据源搜索',
        feature_think: '显示深度思考',
        feature_think_active: '开启深度思考',
        hotkey: '快捷键',
        hotkey_placeholder: '设置呼出 CocoAI 的快捷键',
        module_ai_widgets: 'AI 组件',
        module_ai_widgets_title: '组件',
        module_ai_overview: 'AI Overview',
        module_ai_overview_title: '标题',
        module_ai_overview_height: '高度',
        module_ai_overview_output: '输出',
        module_chat: 'AI 聊天',
        module_chat_ai_assistant: 'AI 助手',
        module_chat_placeholder: 'AI 聊天输入框提示文本',
        module_search: '搜索',
        module_search_placeholder: '搜索输入框提示文本',
        module_search_welcome: '欢迎语',
        logo: '图标',
        logo_mobile: '移动端图标',
        name: '名称',
        enabled: '启用状态',
        theme: '主题',
        theme_auto: '自动',
        theme_dark: '深色',
        theme_light: '浅色',
        type: '类型',
        type_searchbox: 'SearchBox',
        type_fullscreen: 'Fullscreen',
        mode: '模式',
        mode_all: '内嵌和浮动',
        mode_embedded: '内嵌的',
        mode_embedded_placeholder: '内嵌组件提示文本',
        mode_embedded_icon: '内嵌组件图标',
        mode_floating: '浮动的',
        mode_floating_placeholder: '浮动组件提示文本',
        mode_floating_icon: '浮动组件图标',
      },
      title: {
        edit: '编辑嵌入组件',
        new: '新增嵌入组件'
      }
    },
    update: {
      disable_confirm: `确定禁用嵌入组件 "{{name}}"？`,
      enable_confirm: `确定启用嵌入组件 "{{name}}"？`
    },
    topics: {
      title: '更新推荐话题',
      label: '话题',
      new: '新增话题',
      delete: '删除'
    }
  },
  login: {
    cloud: '使用 INFINI Cloud 登录',
    cocoAI: {
      autoDesc: '为了继续操作，如果 5 秒内未自动重定向，请点击以下链接：',
      copyDesc: '如果重定向不起作用，您可以复制以下 URL 并将其粘贴到 Coco AI 的连接设置窗口中：',
      enterCocoServer: '进入 Coco Server',
      enterCocoServerDesc: '或者，您也可以：',
      launchCocoAI: '启动 Coco AI',
      title: '打开 Coco AI'
    },
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
    desc: '输入您的凭证信息以访问您的账户。',
    password: '密码',
    title: '欢迎'
  },
  settings: {
    llm: {
      title: '创建一个账户',
      desc: '集成大模型后，您将解锁人工智能聊天功能，还能获得智能搜索服务和高效的工作助手。',
      answering_model: '应答模型',
      defaultModel: '默认模型',
      reasoning: '推理模式',
      endpoint: 'Endpoint',
      enhanced_inference: '开启推理强度调整',
      frequency_penalty: '频率惩罚度',
      frequency_penalty_desc: '值越大，越有可能降低重复字词',
      intent_analysis_model: '意图识别模型',
      keepalive: '保持连接',
      max_tokens: '单次回复限制',
      max_tokens_desc: '单次交互所用的最大 Token 数',
      picking_doc_model: '文档预选模型',
      presence_penalty: '话题新鲜度',
      presence_penalty_desc: '值越大，越有可能扩展到新话题',
      requestParams: '请求参数',
      temperature: '随机性',
      temperature_desc: '值越大，回复越随机',
      top_p: '核采样',
      top_p_desc: '与随机性类似，但不要和随机性一起更改',
      type: '类型',
    },
    connector: {
      title: "连接器"
    },
    app_settings: {
      title: "应用设置",
      chat_settings: {
        title: "聊天设置",
        labels: {
          start_page: '起始页',
          start_page_placeholder: '你可以启用和配置聊天模式起始页，自定义公司徽标、介绍文本和常用人工智能助手，以帮助用户快速选择并开始聊天。',
          logo: '图标',
          logo_placeholder: '上传公司图标以显示在起始页上。',
          logo_size_placeholder: '图像尺寸限制：最大高度为 30 像素，最大宽度为 300 像素。',
          logo_light: '浅色主题（常规版本图标）',
          logo_dark: '深色主题（白色或浅色版本图标）',
          introduction: '介绍文本',
          introduction_placeholder: '输入显示在起始页面上的欢迎文本或人工智能工具介绍（60 字符以内）',
          assistant: 'AI 助手展示',
        }
      }
    },
    setupLater: '稍后设置'
  },
  modelprovider: {
    labels: {
      name: "名称",
      base_url: "Base URL",
      description: "描述",
      api_type: "API 类型",
      models: "模型",
      enabled: "启用状态",
      icon: "图标",
      api_key: "API 密钥",
      api_key_source: "从 {{model_provider}} 获取 API 密钥",
      api_key_source_normal: "点击这里获取 API 密钥",
      builtin: "内置",
    },
    delete: {
      confirm: '您确定要删除这个模型提供商吗?'
    },
  },
  assistant: {
    labels: {
      name: "名称",
      type: "类型",
      category: "分类",
      tags: "标签",
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
      reasoning: "推理模式（Reasoning Mode）",
      reasoning_desc: "该模型是否支持  Reasoning 推理模式",
      chat_settings: "聊天设置",
      greeting_settings: "问候设置",
      suggested_chat: "推荐对话",
      input_preprocessing: "用户输入预处理",
      input_preprocessing_desc: "用户最新的一条消息会填充到此模版",
      input_preprocessing_placeholder: "预处理模版 {{text}} 将替换为实时输入信息",
      input_placeholder: "输入提示",
      history_message_number: "附带历史消息数",
      history_message_number_desc: "每次请求携带的历史消息数",
      history_message_compression_threshold: "历史消息长度压缩阈值",
      history_message_compression_threshold_desc: "当未压缩的历史消息超过该值时，将进行压缩",
      history_summary: "历史消息摘要",
      history_summary_desc: "自动压缩聊天记录并作为上下文发送",
      model_settings: "模型设置",
      datasource: "数据源",
      mcp_servers: "MCP 服务器",
      upload: "上传设置",
      show_in_chat: "在聊天界面中显示",
      enabled_by_default: "默认启用",
      pick_datasource: "是否挑选数据源",
      pick_tools: "是否挑选工具",
      max_iterations: "最大迭代次数",
      caller_model: "调用模型",
      filter: "数据过滤",
      tools: "调用大模型工具",
      builtin_tools: "内置工具",
      prompt_settings: "提示词设置",
      prompt_settings_template: "模板",
      allowed_file_extensions: "允许文件扩展名（示例：pdf,doc ）",
      max_file_size_in_bytes: "最大文件大小",
      max_file_count: "最大文件数量"
    },
    mode: {
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
      enabled: "启用状态",
      description: "描述",
      icon: "图标",
      category: "分类",
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
