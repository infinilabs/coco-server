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
    userDetail: {
      content: `loader 会让网络请求跟懒加载的文件几乎一起发出请求 然后 一边解析懒加载的文件 一边去等待 网络请求
        待到网络请求完成页面 一起显示 配合react的fiber架构 可以做到 用户如果嫌弃等待时间较长 在等待期间用户可以去
        切换不同的页面 这是react 框架和react-router数据路由器的优势 而不用非得等到 页面的显现 而不是常规的
        请求懒加载的文件 - 解析 - 请求懒加载的文件 - 挂载之后去发出网络请求 - 然后渲染页面 - 渲染完成
        还要自己加loading效果`,
      explain: '这个页面仅仅是为了展示 react-router-dom 的 loader 的强大能力，数据是随机的对不上很正常'
    }
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
