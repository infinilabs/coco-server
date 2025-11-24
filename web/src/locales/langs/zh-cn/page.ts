const page: App.I18n.Schema['translation']['page'] = {
  apitoken: {
    columns: {
      expire_in: '过期时间',
      name: '名称',
      permissions: '权限'
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
  assistant: {
    delete: {
      confirm: 'Are you sure you want to delete this ai assistant "{{name}}"?'
    },
    labels: {
      allowed_file_extensions: '允许文件扩展名（示例：pdf,doc ）',
      answering_model: '应答模型',
      builtin_tools: '内置工具',
      caller_model: '调用模型',
      category: '分类',
      chat_settings: '聊天设置',
      datasource: '数据源',
      deep_think_model: '深度思考模型',
      default_model: '默认模型',
      description: '描述',
      enabled: '启用状态',
      enabled_by_default: '默认启用',
      filter: '数据过滤',
      frequency_penalty: '减少重复（Frequency Penalty）',
      frequency_penalty_desc: '控制 AI 对同一词汇的重复使用，值越高，表达越丰富',
      greeting_settings: '问候设置',
      history_message_compression_threshold: '历史消息长度压缩阈值',
      history_message_compression_threshold_desc: '当未压缩的历史消息超过该值时，将进行压缩',
      history_message_number: '附带历史消息数',
      history_message_number_desc: '每次请求携带的历史消息数',
      history_summary: '历史消息摘要',
      history_summary_desc: '自动压缩聊天记录并作为上下文发送',
      icon: '图标',
      input_placeholder: '输入提示',
      input_preprocessing: '用户输入预处理',
      input_preprocessing_desc: '用户最新的一条消息会填充到此模版',
      input_preprocessing_placeholder: '预处理模版 {{text}} 将替换为实时输入信息',
      intent_analysis_model: '意图识别',
      keepalive: '保持连接',
      max_file_count: '最大文件数量',
      max_file_size_in_bytes: '最大文件大小',
      max_iterations: '最大迭代次数',
      max_tokens: '单次回复限制(max tokens)',
      max_tokens_desc: '单次交互所用的最大 Token 数',
      mcp_servers: 'MCP 服务器',
      model_settings: '模型设置',
      name: '名称',
      pick_datasource: '是否挑选数据源',
      pick_tools: '是否挑选工具',
      picking_doc_model: '文档预选',
      presence_penalty: '表述发散度（Presence Penalty）',
      presence_penalty_desc: '控制 AI 是否倾向于使用新主题，值越高，AI 更倾向于引入新内容',
      prompt_settings: '提示词设置',
      prompt_settings_template: '模板',
      reasoning: '推理模式（Reasoning Mode）',
      reasoning_desc: '该模型是否支持  Reasoning 推理模式',
      role_prompt: '角色提示',
      show_in_chat: '在聊天界面中显示',
      suggested_chat: '推荐对话',
      tags: '标签',
      temperature: '随机性（temperature） ',
      temperature_desc: '控制生成文本的随机性，值越高，内容越丰富但不稳定；值越低，内容更可预测',
      tools: '调用大模型工具',
      top_p: '词汇多样性 (top_p)',
      top_p_desc: '限制选择的词汇范围，值越低，结果更可预测；值越高，可能性更多样，不推荐和随机性一起更改',
      type: '类型',
      upload: '上传设置'
    },
    mode: {
      deep_think: '深度思考',
      simple: '简单模式',
      workflow: '外部工作流'
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
        config: '配置信息',
        description: '描述',
        icon: '连接器图标',
        name: '名称',
        path_hierarchy: '支持目录访问',
        redirect_url: '重定向地址',
        tags: '标签',
        token_url: 'Token 地址',
        processor: '处理器'
      },
      title: '新建连接器',
      tooltip: {
        category: '请选择或输入分类。',
        config: '以 JSON 格式填写配置信息。',
        path_hierarchy: '是否支持通过目录层级方式访问文档。'
      },
      placeholder: {
        category: '选择或输入分类'
      }
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
    delete: {
      confirm: '确定删除这个数据源？'
    },
    edit: {
      title: '编辑数据源'
    },
    every: '每',
    commons: {
      error: {
        datasource_name_required: '请输入数据源名称',
        extensions_format: '文件扩展名无效。请使用 pdf 或 .pdf 等格式，且仅包含字母和数字'
      }
    },
    confluence: {
      error: {
        endpoint_invalid: 'URL 地址无效',
        endpoint_prefix: 'URL 地址应包含 http:// 或 https:// 前缀',
        endpoint_required: '请输入 Confluence Endpoint',
        endpoint_slash: 'Endpoint 末尾不应包含斜杠 /',
        space_required: '请输入 Confluence 空间键名'
      },
      labels: {
        enable_attachments: '索引附件',
        enable_blogposts: '索引 blogposts',
        endpoint: 'Confluence 服务器地址',
        space: 'Wiki 空间键名',
        token: '访问令牌(可选)',
        username: '用户名 (可选)'
      },
      tooltip: {
        enable_attachments: '是否索引附件',
        enable_blogposts: '是否索引 blogposts ',
        endpoint: '您的 Confluence 服务器地址，例如：http://localhost:8090 或 https://wiki.example.com/confluence',
        space: '您想要索引的 Confluence 空间键名（例如“DS”或"KB"）',
        token: '您的 Confluence 个人访问令牌 (PAT)',
        username: '用于身份验证的用户名。如果使用匿名访问或个人访问令牌，可以留空'
      }
    },
    file_paths_add: '添加文件路径',
    git_commons: {
      error: {
        owner_required: '请输入代码库拥有者',
        repo_required: '请输入仓库名称',
        token_required: '请输入个人访问令牌'
      },
      labels: {
        index_issues: '索引 issues',
        owner: '代码库拥有者',
        repos: '代码库名称',
        token: '个人访问令牌'
      },
      tooltip: {
        index_issues: '是否索引 issues',
        owner: '代码库所属的用户名或组织名称',
        repos: '要索引的代码仓库。默认为空，表示索引所有的代码库'
      }
    },
    gitea: {
      labels: {
        base_url: 'Base URL (可选)',
        index_pull_requests: '索引 pull requests'
      },
      tooltip: {
        base_url: '自托管 Gitea Base URL。如果是 Gitea.com，请置空。',
        index_pull_requests: '是否索引 pull requests',
        token: '需要具有“repo”范围的 Gitea 个人访问令牌 (PAT)。'
      }
    },
    github: {
      labels: {
        index_pull_requests: '索引 pull requests'
      },
      tooltip: {
        index_pull_requests: '是否索引 pull requests',
        token: '需要具有“repo”范围的 GitHub 个人访问令牌 (PAT)。'
      }
    },
    gitlab: {
      labels: {
        base_url: 'Base URL (可选)',
        index_merge_requests: '索引 merge requests',
        index_snippets: '索引 snippets',
        index_wikis: '索引 wikis'
      },
      tooltip: {
        base_url: '自托管 GitLab Base URL。如果是 GitLab.com，请置空。',
        index_merge_requests: '是否索引 merge requests',
        index_snippets: '是否索引 snippets',
        index_wikis: '是否索引 wikis',
        token: '需要具有“repo”范围的 GitLub 个人访问令牌 (PAT)。'
      }
    },
    hours: '小时',
    minutes: '分钟',
    missing_config_tip: 'Google 授权相关参数没有设置，需设置后才能连接，点击确认跳转到设置页面。',
    mongodb: {
      error: {
        collection_required: '请输入集合名称！',
        connection_uri_invalid: '无效的 MongoDB URI。必须以 mongodb:// 或 mongodb+srv:// 开头',
        connection_uri_required: '请输入连接地址！',
        database_required: '请输入数据库名称！',
        page_size_required: '请输入页面大小！',
        property_required: '请输入跟踪属性名称！',
        query_invalid: '查询过滤器的 JSON 格式无效',
        sort_invalid: '排序规范的 JSON 格式无效',
        tie_breaker_required: '请输入并列项字段！'
      },
      labels: {
        collection: '集合',
        connection_uri: '连接地址',
        data_processing: '数据加工',
        database: '数据库',
        field_mapping: '字段映射',
        incremental_sync: '增量同步',
        incremental_sync_enable: '启用增量同步',
        page_size: '页面大小',
        pagination: '启用分页',
        property: '跟踪属性',
        property_type: '属性类型',
        property_type_datetime: '日期时间',
        property_type_int: '整数',
        property_type_string: '字符串',
        query: '查询过滤器',
        resume_from: '起始值',
        sort: '排序规范',
        tie_breaker: '并列项字段'
      },
      tooltip: {
        connection_uri: 'MongoDB 连接字符串，例如：mongodb://localhost:27017 或 mongodb+srv://cluster.mongodb.net',
        incremental_sync: '启用后，从上次进度继续扫描，仅同步增量变更。',
        page_size: '每页获取的文档数（1-10000）',
        pagination: '为大型集合启用分页',
        property: '用于跟踪变更的字段名，例如 "updated_at" 或 "_id"',
        property_type: '跟踪属性的数据类型，用于正确比较',
        query: '可选的 BSON 查询过滤器（JSON 格式），例如：{"status": "published"}',
        resume_from: '可选的首次同步手动起始点',
        sort: '可选的排序规范（JSON 格式），例如：{"updated_at": 1, "_id": 1}',
        tie_breaker: '当多个文档具有相同属性值时，用于打破平局的次要字段（通常是 "_id"）'
      }
    },
    neo4j: {
      error: {
        connection_uri_invalid: '无效的 Neo4j URI 格式。请使用 neo4j://host:port',
        connection_uri_required: '请输入连接地址！',
        cypher_required: '请输入 Cypher 查询！',
        incremental_property_required: '请输入增量同步属性！',
        incremental_resume_invalid: '请使用 RFC3339 时间格式，例如：2025-01-01T00:00:00Z。',
        incremental_resume_invalid_float: '请输入有效的数字！',
        incremental_resume_invalid_int: '请输入有效的整数值！',
        incremental_tie_breaker_required: '请输入并列项属性！',
        parameter_key_required: '请输入参数键！',
        password_required: '请输入密码！',
        path_property_required: '请输入路径属性！',
        property_required: '请输入跟踪属性名称！',
        tie_breaker_required: '请输入并列项字段！',
        username_required: '请输入用户名！'
      },
      labels: {
        add_parameter: '新增参数',
        advanced_settings: '高级设置',
        auth_token: '认证令牌',
        auth_token_placeholder: '可选的 Neo4j 认证令牌',
        connection_uri: '连接地址',
        content_field: '内容字段',
        cypher: 'Cypher 查询',
        database: '数据库',
        database_placeholder: '可选的数据库名称',
        enable_field_mapping: '启用字段映射',
        enable_pagination: '启用分页',
        field_mapping: '字段映射',
        hashable_id: '可哈希 ID',
        id_field: 'ID 字段',
        incremental_property: '水位属性',
        incremental_property_type: '水位属性类型',
        incremental_resume_from: '起始水位值',
        incremental_sync: '增量同步',
        incremental_tie_breaker: '并列项属性',
        label_property: '标签属性',
        last_modified_field: '最后修改字段',
        page_size: '页面大小',
        parameter_key_placeholder: '参数键',
        parameter_value_placeholder: '参数值',
        parameters: '查询参数',
        parent_relationship: '父级关系',
        password: '密码',
        password_placeholder: '请输入 Neo4j 密码',
        path_property: '路径属性',
        path_separator: '路径分隔符',
        property: '跟踪属性',
        property_type: '属性类型',
        property_type_datetime: '日期时间',
        property_type_float: '浮点数',
        property_type_int: '整数',
        property_type_string: '字符串',
        resume_from: '起始值',
        tie_breaker: '并列项字段',
        title_field: '标题字段',
        url_field: 'URL 字段',
        username: '用户名'
      },
      tooltip: {
        auth_token: '可选的 Bearer Token。如果填写，将覆盖用户名和密码。',
        connection_uri: '您的 Neo4j 数据库连接地址，例如：neo4j://localhost:7687',
        cypher: '用于获取数据的 Cypher 查询语句',
        field_mapping: '将 Neo4j 节点属性映射到文档字段',
        incremental_property: '作为水位的 Cypher 结果字段。每一行都必须包含该字段。',
        incremental_property_type: '选择水位属性的数据类型，以确保排序稳定。',
        incremental_resume_from: '可选的首次运行起始水位值。',
        incremental_sync: '启用后，将使用水位属性从上次进度继续扫描。',
        incremental_tie_breaker: '用于区分具有相同水位值记录的表达式，例如 elementId(n)。',
        page_size: '每页要获取的记录数',
        pagination: '如果数据库查询应该分页，请启用此选项。建议对大型图数据库使用此选项。',
        parameters: '可选的键值对参数，将传递给 Cypher 查询。',
        property: '用于跟踪变更的字段名',
        property_type: '跟踪属性的数据类型，用于正确比较',
        resume_from: '可选的首次同步手动起始点',
        tie_breaker: '当多个记录具有相同属性值时，用于打破平局的次要字段'
      }
    },
    network_drive: {
      error: {
        endpoint_format: '格式无效，请使用 "host:port" 或 "[ipv6]:port"',
        endpoint_invalid: '服务器端点无效',
        endpoint_required: '请输入服务器端点',
        folder_paths: '请输入文件夹路径',
        folder_paths_prefix: '文件夹路径无效，不能以 / 开头',
        share_required: '请输入共享名称',
        username_required: '请输入用户名'
      },
      labels: {
        domain: '用户域',
        endpoint: '服务器端点',
        folder_paths: '文件夹路径',
        password: '密码',
        share: '网络驱动器共享名称',
        username: '用户名'
      },
      tooltip: {
        domain: '用户的域，例如 WORKGROUP',
        endpoint: '网络驱动器服务地址，例如：127.0.0.1:445',
        folder_paths: '您要扫描的文件夹的路径',
        password: '用于网络驱动器身份验证的密码',
        share: '您要扫描的网络驱动器共享名称',
        username: '用于网络驱动器身份验证的用户名'
      }
    },
    jira: {
      error: {
        endpoint_invalid: '请输入有效的 URL',
        endpoint_prefix: 'URL 必须以 http:// 或 https:// 开头',
        endpoint_required: '请输入 Jira URL！',
        project_key_required: '请输入项目 Key！',
        token_required: '请输入密码/令牌！',
        username_required: '请输入用户名！'
      },
      labels: {
        endpoint: 'Jira 服务器 URL',
        index_attachments: '索引附件',
        index_comments: '索引评论',
        project_key: '项目 Key',
        token: '密码 / 令牌',
        username: '用户名（可选）'
      },
      tooltip: {
        endpoint: '您的 Jira 实例 URL（例如：https://your-domain.atlassian.net）',
        index_attachments: '是否索引问题附件',
        index_comments: '是否索引问题评论',
        project_key: '您要索引的 Jira 项目 Key（例如："COCO"）',
        token: '提供用户名时为密码（Basic Auth），不提供用户名时为个人访问令牌（Bearer Auth）',
        username: '用于身份验证的 Jira 账户用户名'
      }
    },
    new: {
      labels: {
        client_id: '客户端 ID',
        client_secret: '客户端密钥',
        config: '配置信息',
        connect: '连接',
        data_sync: '同步策略',
        description: '描述',
        enabled: '启用状态',
        file_extensions: '文件扩展名（可选）',
        folder_paths: '文件夹路径',
        icon: '图标',
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
        sync_enabled: '启用同步',
        tags: '标签',
        type: '数据源类型',
        webhook: '启用 Webhook',
        enrichment_pipeline: 'Enrichment Pipeline'
      },
      title: '连接 {{connector}}',
      tooltip: {
        file_extensions: '逗号分隔列表。例如 pdf、docx、txt',
        folder_paths: '您要扫描的文件夹的绝对路径。'
      }
    },
    rdbms: {
      error: {
        connection_uri_required: '请输入连接地址！',
        page_size_required: '请输入页面大小！',
        sql_required: '请输入 SQL 查询！'
      },
      labels: {
        connection_uri: '连接地址',
        data_processing: '数据加工',
        dest_field: '目标字段',
        field_mapping: '字段映射',
        last_modified_field: '最后修改字段 (可选)',
        page_size: '页面大小',
        pagination: '启用分页',
        sql: 'SQL 查询',
        src_field: '源字段'
      },
      placeholder: {
        field_name: '字段名',
        metadata_name: '元数据名称',
        payload_name: '载荷名称'
      },
      titles: {
        last_updated_by: '最后更新者',
        metadata: '元数据',
        owner: '所有者',
        payload: '载荷'
      },
      tooltip: {
        connection_uri: {
          mssql: 'Microsoft SQL Server 连接字符串，例如：sqlserver://user:password@host:1433/database',
          mysql: 'MySQL 连接字符串，例如：mysql://user:password@tcp(localhost:3306)/database',
          oracle: 'Oracle 连接字符串，例如：oracle://user:password@host:1521/service_name',
          postgresql: 'PostgreSQL 连接字符串，例如：postgresql://user:password@localhost:5432/database?sslmode=disable'
        },
        last_modified_field:
          '对于增量同步，请指定一个跟踪最后修改时间的字段（例如，updated_at）。该字段的类型应该是时间戳或日期时间。',
        page_size: '每页要获取的记录数。',
        pagination: '如果数据库查询应该分页，请启用此选项。建议对大型表使用此选项。',
        sql: '用于获取数据的 SQL 查询。'
      },
      validation: {
        field_name_required: '请输入字段名',
        metadata_name_required: '请输入名称',
        payload_name_required: '请输入名称',
        required: '请输入 {{field}}'
      }
    },
    labels: {
      owner: '所有者',
      shares: '分享',
      updated: '最后更新时间',
      size: '大小',
      externalAccount: '外部账号',
      cocoAccount: '映射 Coco 用户',
      mappingStatus: '映射状态',
      mapped: '已映射',
      unmapped: '未映射',
      enabled: '启用状态',
      permission_sync: '权限同步',
      isEnabled: '启用',
      sharesWithPermissions: '有访问权限的团队、人',
      none: '无权限',
      view: '可查看',
      comment: '可评论',
      edit: '可编辑',
      share: '可共享',
      all: '全部',
      shareToPrincipal: '共享给团队、人',
      shareTo: '共享',
      permission: '访问权限',
      you: '（你）',
      inherit: '（继承）',
      categories: '分类',
      type: '类型',
      created: '创建时间',
      createdBy: '创建人',
      updatedBy: '更新人'
    },
    file: {
      title: '文档管理'
    },
    mapping: {
      title: '映射管理'
    },
    s3: {
      error: {
        access_key_id_required: '请输入 Access Key ID！',
        bucket_required: '请输入 Bucket 名称',
        endpoint_format: '格式无效，请使用 "host", "host:port", "[ipv6]" 或 "[ipv6]:port" 格式',
        endpoint_prefix: 'Endpoint 不应包含 http:// 或 https:// 前缀',
        endpoint_required: '请输入 S3 Endpoint',
        endpoint_slash: 'Endpoint 末尾不应包含斜杠 /',
        secret_access_key_required: '请输入 Secret Access Key'
      },
      labels: {
        access_key_id: 'Access Key ID',
        bucket: 'Bucket 名称',
        prefix: '对象前缀 (可选)',
        secret_access_key: 'Secret Access Key',
        ssl: 'SSL',
        use_ssl: '使用 SSL (HTTPS)'
      },
      tooltip: {
        endpoint: '您的 S3 兼容服务的服务器地址，例如：s3.amazonaws.com 或 localhost:9000',
        prefix: '仅索引 key 以此前缀开头的对象'
      }
    },
    seconds: '秒',
    site_urls: '站点地址',
    site_urls_add: '新增站点地址'
  },
  guide: {
    llm: {
      desc: '集成大模型后，您将解锁人工智能聊天功能，还能获得智能搜索服务和高效的工作助手。',
      title: '集成大模型'
    },
    setupLater: '稍后设置',
    user: {
      desc: '设置一个新的账户以管理访问权限。',
      email: '邮箱',
      language: '语言',
      name: '姓名',
      password: '密码',
      title: '创建一个账户'
    },
    skipModal: {
      title: '确认跳过？',
      hints: {
        desc: '如果您选择跳过此步骤，进入系统后内置的 AI 功能（如 AI 助手）将无法立即使用。',
        stepDesc: '您将需要：',
        step1: '在 “模型提供商” 中添加并管理模型；',
        step2: '为每个内置 AI 助手单独选择并配置所需的大模型。'
      }
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
  integratedStoreModal: {
    buttons: {
      custom: '自定义',
      install: '安装'
    },
    hints: {
      installSuccess: '安装成功，3s后自动跳转至详情页。',
      noResults: '未找到数据'
    },
    installModal: {
      buttons: {
        install: '安装',
        return: '返回'
      },
      hints: '检测到扩展信息，确认安装此扩展到当前 Coco Server 吗？',
      title: '安装扩展'
    },
    labels: {
      aiAssistant: 'AI 助手',
      connector: '连接器',
      datasource: '数据源',
      mcpServer: 'MCP Server',
      modelProvider: '模型提供商',
      newest: '最新',
      recommend: '推荐'
    },
    title: '集成商店'
  },
  integration: {
    code: {
      desc: '将这段代码插入到你的网站<body>和</body>之间，即可开始使用组件。',
      enabled_tips: '请将组件启用状态设置为开启并更新保存，才能预览!',
      exit: '退出预览',
      preview: '预览',
      title: '添加组件到你的网站'
    },
    columns: {
      datasource: '数据源',
      description: '描述',
      enabled: '启用状态',
      name: '名称',
      operation: {
        topics: '推荐话题'
      },
      token_expire_in: 'Token 过期时间',
      type: '类型'
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
        tourist_mode: '游客模式',
        enable_module: '启用模块',
        enabled: '启用状态',
        feature_chat_history: '聊天历史',
        feature_Control: '功能控制',
        feature_search: '显示数据源搜索',
        feature_search_active: '开启数据源搜索',
        feature_think: '显示深度思考',
        feature_think_active: '开启深度思考',
        hotkey: '快捷键',
        hotkey_placeholder: '设置呼出 CocoAI 的快捷键',
        logo: '图标（浅色模式）',
        logo_mobile: '移动端图标（浅色模式）',
        logo_dark: '图标（深色模式）',
        logo_mobile_dark: 'Mobile Logo（深色模式）',
        mode: '模式',
        mode_all: '内嵌和浮动',
        mode_embedded: '内嵌的',
        mode_embedded_icon: '内嵌组件图标',
        mode_embedded_placeholder: '内嵌组件提示文本',
        mode_floating: '浮动的',
        mode_floating_icon: '浮动组件图标',
        mode_floating_placeholder: '浮动组件提示文本',
        mode_modal: '全屏弹框',
        mode_page: '全屏页面',
        module_ai_overview: 'AI Overview',
        module_ai_overview_height: '高度',
        module_ai_overview_output: '输出',
        module_ai_overview_title: '标题',
        module_ai_widgets: 'AI 组件',
        module_ai_widgets_title: '组件',
        module_chat: 'AI 聊天',
        module_chat_ai_assistant: 'AI 助手',
        module_chat_placeholder: 'AI 聊天输入框提示文本',
        module_search: '搜索',
        module_search_placeholder: '搜索输入框提示文本',
        module_search_welcome: '欢迎语',
        name: '名称',
        theme: '主题',
        theme_auto: '自动',
        theme_dark: '深色',
        theme_light: '浅色',
        language: '语言',
        type: '类型',
        type_fullscreen: 'Fullscreen',
        type_searchbox: 'SearchBox'
      },
      title: {
        edit: '编辑嵌入组件',
        new: '新增嵌入组件'
      },
      hints: {
        tourist_mode: '未登录用户将以该用户的身份进行访问'
      }
    },
    topics: {
      delete: '删除',
      label: '话题',
      new: '新增话题',
      title: '更新推荐话题'
    },
    update: {
      disable_confirm: `确定禁用嵌入组件 "{{name}}"？`,
      enable_confirm: `确定启用嵌入组件 "{{name}}"？`
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
    email: '邮箱',
    title: '欢迎'
  },
  mcpserver: {
    delete: {
      confirm: 'Are you sure you want to delete this ai MCP server "{{name}}"?'
    },
    labels: {
      category: '分类',
      config: {
        args: '参数',
        command: '命令',
        env: '环境变量'
      },
      description: '描述',
      enabled: '启用状态',
      icon: '图标',
      name: '名称',
      type: '类型'
    }
  },
  settings: {
    app_settings: {
      chat_settings: {
        labels: {
          assistant: 'AI 助手展示',
          introduction: '介绍文本',
          introduction_placeholder: '输入显示在起始页面上的欢迎文本或人工智能工具介绍（60 字符以内）',
          logo: '图标',
          logo_dark: '深色主题（白色或浅色版本图标）',
          logo_light: '浅色主题（常规版本图标）',
          logo_placeholder: '上传公司图标以显示在起始页上。',
          logo_size_placeholder: '图像尺寸限制：最大高度为 30 像素，最大宽度为 300 像素。',
          start_page: '起始页',
          start_page_placeholder:
            '你可以启用和配置聊天模式起始页，自定义公司徽标、介绍文本和常用人工智能助手，以帮助用户快速选择并开始聊天。'
        },
        title: '聊天设置'
      },
      title: '应用设置'
    },
    connector: {
      title: '连接器'
    },
    llm: {
      answering_model: '应答模型',
      defaultModel: '默认模型',
      desc: '集成大模型后，您将解锁人工智能聊天功能，还能获得智能搜索服务和高效的工作助手。',
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
      reasoning: '该模型支持推理模式',
      requestParams: '请求参数',
      temperature: '随机性',
      temperature_desc: '值越大，回复越随机',
      title: '创建一个账户',
      top_p: '核采样',
      top_p_desc: '与随机性类似，但不要和随机性一起更改',
      type: '类型'
    },
    search_settings: {
      title: '搜索设置',
      labels: {
        enabled: '启用状态',
        integration: '组件'
      }
    },
    setupLater: '稍后设置'
  },
  webhook: {
    form: {
      title: {
        edit: '编辑 Webhook',
        new: '新建 Webhook'
      }
    },
    labels: {
      ai_assistant: 'AI 助手',
      content_type: 'Content type',
      datasource: '数据源',
      file_parse_completed: '文件解析已完成',
      model_provider: '大模型提供商',
      name: '名称',
      payload_url: 'Payload URL',
      reply_completed: '回复已完成',
      secret: 'Secret',
      ssl_verify: 'SSL 验证',
      sync_completed: '同步完成',
      test: '测试',
      test_need_save: '请先保存，再进行测试',
      triggers: '触发事件'
    },
    placeholders: {
      name: '请输入',
      payload_url: '请输入',
      secret: '请输入'
    }
  },
  modelprovider: {
    labels: {
      name: '名称',
      base_url: 'Base URL',
      description: '描述',
      api_type: 'API 类型',
      models: '模型',
      enabled: '启用状态',
      icon: '图标',
      api_key: 'API 密钥',
      api_key_source: '从 {{model_provider}} 获取 API 密钥',
      api_key_source_normal: '点击这里获取 API 密钥',
      builtin: '内置',
      addModel: '添加模型',
      modelID: '模型ID',
      modelType: '模型类型',
      inferenceMode: '推理模式'
    },
    delete: {
      confirm: '您确定要删除这个模型提供商吗?'
    },
    options: {
      dialogModel: '对话模型'
    },
    hints: {
      selectOrInputModel: '选择或输入一个模型',
      inferenceMode: '这个模型是否支持推理模式'
    }
  },
  role: {
    title: '角色',
    labels: {
      name: '名称',
      description: '描述',
      permission: '权限',
      object: '操作对象',
      coco: 'Coco Server',
      generic: '通用',
      created: '创建时间'
    },
    new: {
      title: '新增角色'
    },
    edit: {
      title: '编辑角色'
    },
    delete: {
      confirm: `确定要删除角色 "{{name}}" 吗？`
    }
  },
  auth: {
    title: '授权',
    labels: {
      name: '名称',
      description: '描述',
      permission: '权限',
      object: '授权对象',
      coco: 'Coco Server',
      user: '人员',
      team: '团队',
      userRole: '团队成员角色',
      teamRole: 'APP 用户角色',
      roles: '角色',
      created: '创建时间',
      auth: '授权',
      type: '类型'
    },
    new: {
      title: '新增授权'
    },
    edit: {
      title: '编辑授权'
    },
    delete: {
      confirm: `确定要删除授权 "{{name}}" 吗？`
    }
  },
  user: {
    title: '用户',
    labels: {
      name: '姓名',
      email: '邮箱',
      roles: '角色',
      created: '创建时间',
      password: '密码'
    },
    new: {
      title: '新增用户',
      copyPassword:
        '请将此密码保存在安全且易于访问的地方。出于安全原因，你将无法通过用户管理界面再次查看它。如果你丢失了这个密码，可通过编辑用户更新密码。'
    },
    edit: {
      title: '编辑用户'
    },
    delete: {
      confirm: `确定要删除用户 "{{name}}" 吗？`
    }
  },
  search_settings: {
    labels: {
      enabled: '启用状态',
      integration: '组件'
    },
    title: '搜索设置'
  },
  setupLater: '稍后设置'
};

export default page;
