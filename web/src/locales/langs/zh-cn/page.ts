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
    password: '密码'
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
    setupLater: '稍后设置'
  }
};

export default page;
