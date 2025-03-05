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
      requestParams: '请求参数',
      temperature: '随机性',
      temperatureDesc: '值越大，回复越随机',
      topP: '核采样',
      topPDesc: '与随机性类似，但不要和随机性一起更改',
      maxTokens: '单次回复限制',
      maxTokensDesc: '单次交互所用的最大 Token 数',
      presencePenalty: '话题新鲜度',
      presencePenaltyDesc: '值越大，越有可能扩展到新话题',
      frequencyPenalty: '频率惩罚度',
      frequencyPenaltyDesc: '值越大，越有可能降低重复字词',
      enhancedInference: '开启推理强度调整',
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
    },
    new:{
      title: "连接数据源",
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
        immediate_sync: "立即同步"
      }
    }
  }
};

export default page;
