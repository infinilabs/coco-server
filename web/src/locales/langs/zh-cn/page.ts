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
        site_urls: "站点地址",
        connect: "连接",
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
      }
    },
  }
};

export default page;
