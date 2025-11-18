const page: App.I18n.Schema['translation']['page'] = {
  apitoken: {
    columns: {
      expire_in: 'Expire In',
      name: 'Name',
      permissions: 'Permissions'
    },
    create: {
      limit: 'Access token limit exceeded. Maximum allowed: 5.',
      store_desc:
        'Please store this token in a secure and easily accessible location. For security reasons, you will not be able to view it again through the API Token management interface. If you lose this token, you will need to generate a new one.'
    },
    delete: {
      confirm: 'Are you sure you want to delete this API token?'
    }
  },
  assistant: {
    delete: {
      confirm: 'Are you sure you want to delete this ai assistant "{{name}}"?'
    },
    labels: {
      allowed_file_extensions: 'Allowed File Extensions (eg: pdf,doc)',
      answering_model: 'Answering Model',
      builtin_tools: 'Built-in',
      caller_model: 'Caller Model',
      category: 'Category',
      chat_settings: 'Chat Settings',
      datasource: 'Datasource',
      deep_think_model: 'Deep Think Model',
      default_model: 'Default Model',
      description: 'Description',
      enabled: 'Enabled',
      enabled_by_default: 'Enabled by default',
      filter: 'Filter',
      frequency_penalty: 'Frequency Penalty',
      frequency_penalty_desc:
        "Controls the AI's repetition of the same vocabulary. The higher the value, the richer the expression.",
      greeting_settings: 'Greeting Settings',
      history_message_compression_threshold: 'Historical message length compression threshold.',
      history_message_compression_threshold_desc:
        'When the uncompressed historical messages exceed this value, compression will be applied.',
      history_message_number: 'Number of historical messages included.',
      history_message_number_desc: 'Number of historical messages included in each request.',
      history_summary: 'Historical summary',
      history_summary_desc: 'Automatically compress chat history and send it as context.',
      icon: 'Icon',
      input_placeholder: 'Input placeholder',
      input_preprocessing: 'User input preprocessing',
      input_preprocessing_desc: "The user's latest message will be filled into this template.",
      input_preprocessing_placeholder: 'Preprocessing template: {{text}} will be replaced with the real-time input.',
      intent_analysis_model: 'Intent Analysis',
      keepalive: 'Keepalive',
      max_file_count: 'Max File Count',
      max_file_size_in_bytes: 'Max File Size',
      max_iterations: 'Max iterations',
      max_tokens: 'Max Tokens',
      max_tokens_desc: 'The maximum number of tokens used in a single interaction.',
      mcp_servers: 'MCP Servers',
      model_settings: 'Model Settings',
      name: 'Name',
      pick_datasource: 'Pick Datasource',
      pick_tools: 'Pick Tools',
      picking_doc_model: 'Picking Doc',
      presence_penalty: 'Presence Penalty',
      presence_penalty_desc:
        'Controls whether the AI tends to introduce new topics. The higher the value, the more the AI is inclined to introduce new content.',
      prompt_settings: 'Prompt Settings',
      prompt_settings_template: 'Template',
      reasoning: 'Reasoning Mode',
      reasoning_desc: 'Whether this model supports the reasoning mode.',
      role_prompt: 'Role Prompt',
      show_in_chat: 'Show in chat',
      suggested_chat: 'Suggested chat',
      tags: 'Tags',
      temperature: 'Temperature',
      temperature_desc:
        'Controls the randomness of the generated text. A higher value produces more diverse but less stable content, while a lower value makes the output more predictable.',
      tools: 'Call LLM Tools',
      top_p: 'Top P',
      top_p_desc:
        'The scope of selected vocabulary is limited. The lower the value, the more predictable the result; the higher the value, the more diverse the possibilities. It is not recommended to change this alongside randomness.',
      type: 'Type',
      upload: 'Upload Settings'
    },
    mode: {
      deep_think: 'Deep Think',
      simple: 'Simple',
      workflow: 'External workflow'
    }
  },
  connector: {
    columns: {
      category: 'Category',
      description: 'Description',
      name: 'Name',
      tags: 'Tags'
    },
    delete: {
      confirm: 'Are you sure you want to delete connector "{{name}}"？'
    },
    edit: {
      title: 'Edit Connector'
    },
    new: {
      labels: {
        asset_icon: 'Icon',
        asset_type: 'Type',
        assets_icons: 'Assets Icons',
        auth_url: 'Auth URI',
        category: 'Category',
        client_id: 'Client ID',
        client_secret: 'Client Secret',
        config: 'Configuration',
        description: 'Description',
        icon: 'Connector Icon',
        name: 'Name',
        path_hierarchy: 'Path Hierarchy',
        redirect_url: 'Redirect URI',
        tags: 'Tags',
        token_url: 'Token URI',
        processor: 'Processor'
      },
      title: 'New Connector',
      tooltip: {
        category: 'Please choose or input the category.',
        config: 'Configurations in JSON format.',
        path_hierarchy: 'Whether to support access documents via path hierarchy manner.'
      },
      placeholder: {
        category: 'Select or input a category'
      }
    }
  },
  datasource: {
    columns: {
      enabled: 'Enabled',
      latest_sync_time: 'Latest Sync Time',
      name: 'Name',
      searchable: 'Searchable',
      sync_policy: 'Sync Policy',
      sync_status: 'Sync Status',
      type: 'Type'
    },
    commons: {
      error: {
        extensions_format: "Invalid file extensions. Use formats like 'pdf' or '.pdf', with only letters and numbers."
      }
    },
    confluence: {
      error: {
        endpoint_invalid: 'Please enter a valid URL',
        endpoint_prefix: 'URL must start with http:// or https://',
        endpoint_required: 'Please input Confluence URL',
        endpoint_slash: 'URL should not contain a trailing slash /',
        space_required: 'Please input Confluence Space Key'
      },
      labels: {
        enable_attachments: 'Indexing attachments',
        enable_blogposts: 'Indexing blogposts',
        endpoint: 'Confluence server URL',
        space: 'Wiki Key',
        token: 'Access Token (Optional)',
        username: 'User Name (Optional)'
      },
      tooltip: {
        enable_attachments: 'Whether to index attachments',
        enable_blogposts: 'Whether to index blogposts ',
        endpoint:
          'The base URL of your Confluence instance. e.g., http://localhost:8090 or https://wiki.example.com/confluence',
        space: 'The key of the Confluence space you want to index (e.g., "DS" or "KB").',
        token: 'Your Confluence Personal Access Token (PAT).',
        username:
          'Username for authentication. Can be left empty for anonymous access or if using a Personal Access Token.'
      }
    },
    connect: 'Connect',
    missing_config_tip:
      'Google authorization parameters are not configured. Please set them before connecting. Click Confirm to go to the settings page.',
    delete: {
      confirm: 'Are you sure you want to delete this datasource?'
    },
    edit: {
      title: 'Edit Datasource'
    },
    every: 'Every',
    file_paths_add: 'Add File Path',
    hours: 'hours',
    minutes: 'minutes',
    neo4j: {
      error: {
        connection_uri_invalid: 'Invalid Neo4j URI format. Use neo4j://host:port',
        connection_uri_required: 'Please input connection URI!',
        cypher_required: 'Please input Cypher query!',
        incremental_property_required: 'Please input the incremental property!',
        incremental_resume_invalid: 'Please use an RFC3339 timestamp, e.g., 2025-01-01T00:00:00Z.',
        incremental_resume_invalid_float: 'Please enter a valid number value.',
        incremental_resume_invalid_int: 'Please enter a valid integer value.',
        incremental_tie_breaker_required: 'Please input the tie-breaker property!',
        parameter_key_required: 'Please input parameter key!',
        password_required: 'Please input password!',
        path_property_required: 'Please input path property!',
        username_required: 'Please input username!'
      },
      labels: {
        add_parameter: 'Add Parameter',
        advanced_settings: 'Advanced Settings',
        auth_token: 'Auth Token',
        auth_token_placeholder: 'Optional Neo4j auth token',
        connection_uri: 'Connection URI',
        content_field: 'Content Field',
        cypher: 'Cypher Query',
        database: 'Database',
        database_placeholder: 'Optional database name',
        enable_field_mapping: 'Enable Field Mapping',
        enable_pagination: 'Enable Pagination',
        field_mapping: 'Field Mapping',
        hashable_id: 'Hashable ID',
        hierarchy_config: 'Hierarchy Configuration',
        hierarchy_mode: 'Hierarchy Mode',
        hierarchy_mode_label: 'Label-based',
        hierarchy_mode_none: 'None (Flat)',
        hierarchy_mode_property: 'Property-based',
        hierarchy_mode_relationship: 'Relationship-based',
        id_field: 'ID Field',
        incremental_property: 'Watermark Property',
        incremental_property_type: 'Watermark Type',
        incremental_resume_from: 'Resume From Value',
        incremental_sync: 'Incremental Sync',
        incremental_tie_breaker: 'Tie-breaker Property',
        label_property: 'Label Property',
        last_modified_field: 'Last Modified Field',
        page_size: 'Page Size',
        parameter_key_placeholder: 'Parameter key',
        parameter_value_placeholder: 'Parameter value',
        parameters: 'Query Parameters',
        parent_relationship: 'Parent Relationship',
        password: 'Password',
        password_placeholder: 'Enter Neo4j password',
        path_property: 'Path Property',
        path_separator: 'Path Separator',
        property_type_datetime: 'Datetime',
        property_type_float: 'Float',
        property_type_int: 'Integer',
        property_type_string: 'String',
        title_field: 'Title Field',
        url_field: 'URL Field',
        username: 'Username'
      },
      tooltip: {
        auth_token: 'Optional bearer token. Overrides username and password if provided.',
        connection_uri: 'Your Neo4j database connection URI, e.g., neo4j://localhost:7687',
        cypher: 'The Cypher query to execute for fetching data',
        field_mapping: 'Map Neo4j node properties to document fields',
        hierarchy_mode: 'Choose how to organize document hierarchy',
        hierarchy_mode_label: 'Organize hierarchy by node labels (e.g., /Person/Employee/Manager)',
        hierarchy_mode_property: 'Organize hierarchy by node properties (e.g., /docs/api/v1)',
        hierarchy_mode_relationship:
          'Organize hierarchy by parent-child relationships (e.g., /Company/Department/Team)',
        incremental_property: 'Cypher result field used as the resume watermark. Must exist in every row.',
        incremental_property_type: 'Choose the data type of the watermark property to ensure stable ordering.',
        incremental_resume_from: 'Optional manual watermark to start from on the first run.',
        incremental_sync: 'Enable to resume scans from the last seen property value (watermark).',
        incremental_tie_breaker: 'Expression used to sort rows that share the same watermark (e.g., elementId(n)).',
        page_size: 'Number of records to fetch per page',
        pagination: 'Enable this if database queries should be paginated. Recommended for large graph databases.',
        parameters: 'Optional key/value parameters passed to the Cypher query.'
      }
    },
    jira: {
      error: {
        endpoint_invalid: 'Please enter a valid URL',
        endpoint_prefix: 'URL must start with http:// or https://',
        endpoint_required: 'Please input Jira URL!',
        project_key_required: 'Please input Project Key!',
        token_required: 'Please input Password/Token!',
        username_required: 'Please input Username!'
      },
      labels: {
        endpoint: 'Jira Server URL',
        index_attachments: 'Index Attachments',
        index_comments: 'Index Comments',
        project_key: 'Project Key',
        token: 'Password / Token',
        username: 'Username (Optional)'
      },
      tooltip: {
        endpoint: 'Your Jira instance URL (e.g., https://your-domain.atlassian.net)',
        index_attachments: 'Whether to index issue attachments',
        index_comments: 'Whether to index issue comments',
        project_key: 'The Jira project key you want to index (e.g., "COCO")',
        token: 'Your password when username is provided (Basic Auth), or your Personal Access Token when username is empty (Bearer Auth)',
        username: 'Your Jira account username for authentication'
      }
    },
    new: {
      labels: {
        client_id: 'Client ID',
        client_secret: 'Client Secret',
        config: 'Configuration',
        connect: 'Connect',
        data_sync: 'Sync  Policy',
        description: 'Description',
        enabled: 'Enabled',
        file_extensions: 'File extensions (optional)',
        folder_paths: 'Folder paths',
        icon: 'Icon',
        immediate_sync: 'Immediate Sync',
        indexing_scope: 'Indexing Scope',
        insert_doc: 'Insert Document',
        manual_sync: 'Manual Sync',
        manual_sync_desc: "Sync only when the user clicks the 'Sync' button",
        name: 'Name',
        realtime_sync: 'Real-time Sync',
        realtime_sync_desc: 'Sync immediately upon file modification',
        redirect_uri: 'Redirect URI',
        scheduled_sync: 'Scheduled Sync',
        scheduled_sync_desc: 'Sync at fixed intervals',
        site_urls: 'Site URLs',
        permission_sync_enabled: 'Permission Sync',
        sync_enabled: 'Sync',
        tags: 'Tags',
        type: 'Type',
        webhook: 'Webhook',
        enrichment_pipeline: 'Enrichment Pipeline'
      },
      title: '{{connector}} Connection',
      tooltip: {
        file_extensions: 'Comma-separated list. e.g., pdf, docx, txt',
        folder_paths: 'Absolute paths to the folders you want to scan.'
      }
    },
    rdbms: {
      placeholder: {
        field_name: 'Field Name',
        metadata_name: 'Metadata Name',
        payload_name: 'Payload Name'
      },
      titles: {
        last_updated_by: 'last_updated_by',
        metadata: 'metadata',
        owner: 'owner',
        payload: 'payload'
      },
      tooltip: {
        connection_uri: {
          mysql: 'MySQL connection string. e.g., mysql://user:password@tcp(localhost:3306)/database',
          postgresql:
            'PostgreSQL connection string. e.g., postgresql://user:password@localhost:5432/database?sslmode=disable'
        }
      },
      validation: {
        field_name_required: 'Please input field',
        metadata_name_required: 'Please input name',
        payload_name_required: 'Please input name',
        required: 'Please input {{field}}'
      }
    },
    s3: {
      error: {
        access_key_id_required: 'Please input Access Key ID',
        bucket_required: 'Please input Bucket name',
        endpoint_format: 'Invalid format, please use "host", "host:port", "[ipv6]" or "[ipv6]:port"',
        endpoint_prefix: 'Endpoint should not contain http:// or https:// prefix',
        endpoint_required: 'Please input S3 Endpoint',
        endpoint_slash: 'Endpoint should not contain a trailing slash /',
        secret_access_key_required: 'Please input Secret Access Key'
      },
      labels: {
        access_key_id: 'Access Key ID',
        bucket: 'Bucket Name',
        prefix: 'Object Prefix (optional)',
        secret_access_key: 'Secret Access Key',
        ssl: 'SSL',
        use_ssl: 'Use SSL (HTTPS)'
      },
      tooltip: {
        endpoint: 'Endpoint of your S3 server, like：s3.amazonaws.com or localhost:9000',
        prefix: 'Only index objects that begin with this prefix'
      }
    },
    seconds: 'seconds',
    site_urls: 'Site URLs',
    site_urls_add: 'Add URL',
    title: 'Datasource',
    labels: {
      owner: 'Owner',
      shares: 'Shares',
      updated: 'Last Updated At',
      size: 'Size',
      externalAccount: 'External Account',
      cocoAccount: 'Mapping Coco User',
      mappingStatus: 'Mapping Status',
      mapped: 'Mapped',
      unmapped: 'Unmapped',
      enabled: 'Enabled',
      permission_sync: 'Permission Sync',
      isEnabled: 'Enabled',
      sharesWithPermissions: 'teams or users with permissions',
      none: 'None',
      view: 'View',
      comment: 'Comment',
      edit: 'Edit',
      share: 'Share',
      all: 'All',
      shareToPrincipal: 'Share to teams or users',
      shareTo: 'Share',
      permission: 'Permission',
      you: ' (you)',
      inherit: ' (inherit)',
      categories: 'Categories',
      type: 'Type',
      created: 'Created At',
      createdBy: 'Created By',
      updatedBy: 'Updated By'
    },
    file: {
      title: 'Document Management'
    },
    mapping: {
      title: 'Mapping Management'
    }
  },
  guide: {
    llm: {
      desc: 'After integrating a large model, you will unlock the AI chat feature, providing intelligent search and an efficient work assistant.',
      title: 'Connect to a Large Model'
    },
    setupLater: 'Set Up Later',
    user: {
      desc: 'Set up a new user account to manage access and permissions.',
      email: 'Email',
      language: 'Language',
      name: 'Full Name',
      password: 'Password',
      title: 'Create a user account'
    },
    skipModal: {
      title: 'Skip setup?',
      hints: {
        desc: 'If you choose to skip this step, the built-in AI features (e.g., AI assistants) will not be available immediately, as they will be in an unconfigured state without a model.',
        stepDesc: 'You will need to:',
        step1: 'Add and manage models in the “LLM Provider” section;',
        step2: 'Individually configure and select a model for each built-in AI assistant.'
      }
    }
  },
  home: {
    server: {
      address: 'Server address',
      addressDesc:
        'In the connect settings of Coco AI, adding the Server address to the service list will allow you to access the service in Coco AI.',
      downloadCocoAI: 'Download Coco AI',
      title: `{{user}}'s Coco Server`
    },
    settings: {
      aiAssistant: 'AI Assistant',
      aiAssistantDesc:
        'Set a personalized AI assistant to handle tasks efficiently and provide intelligent suggestions.',
      dataSource: 'Data Source',
      dataSourceDesc: 'Add data sources to the service list for unified search and analysis.',
      llm: 'LLMs',
      llmDesc: 'Connect the large model to enable AI chat, intelligent search, and a work assistant.'
    }
  },
  integratedStoreModal: {
    buttons: {
      custom: 'Custom',
      install: 'Install'
    },
    hints: {
      installSuccess: 'Installed successfully. Redirecting to the details page in 3 seconds.',
      noResults: 'No results found'
    },
    installModal: {
      buttons: {
        install: 'Install',
        return: 'Return'
      },
      hints: 'Extension information detected. Do you want to install this extension to the current Coco Server?',
      title: 'Install Extension'
    },
    labels: {
      aiAssistant: 'AI Assistant',
      connector: 'Connector',
      datasource: 'Data Source',
      mcpServer: 'MCP Server',
      modelProvider: 'Model Provider',
      newest: 'Newest',
      recommend: 'Recommended'
    },
    title: 'Integration Store'
  },
  integration: {
    code: {
      desc: 'Insert this  code into your website between <body> and </body> to start using integration.',
      enabled_tips: `Please enable the integration, update and save it, than you can preview!`,
      exit: 'Exit Preview',
      preview: 'Preview',
      title: 'Add the widget to your website'
    },
    columns: {
      datasource: 'Data Source',
      description: 'Description',
      enabled: 'Enabled',
      name: 'Name',
      operation: {
        topics: 'Suggested Topics'
      },
      token_expire_in: 'Token Expire In',
      type: 'Type'
    },
    delete: {
      confirm: `Are you sure you want to delete integration "{{name}}" ?`
    },
    form: {
      labels: {
        access_control: 'Access Control',
        allow_origin: 'Allow-Origin',
        allow_origin_placeholder:
          'please enter the allowed origins that start with http:// or https://, and separate with commas. Enter * to allow access from all origins.',
        appearance: 'Appearance',
        cors: 'CORS',
        datasource: 'Data Source',
        description: 'Description',
        enable_auth: 'Enable Authentication',
        tourist_mode: 'Tourist Mode',
        enable_module: 'Enable Module',
        enabled: 'Enabled',
        feature_chat_history: 'Chat History',
        feature_Control: 'Feature Control',
        feature_search: 'Show Datasource Search',
        feature_search_active: 'Enable Datasource Search',
        feature_think: 'Show Deep Think',
        feature_think_active: 'Enable Deep Think',
        hotkey: 'Hotkey',
        hotkey_placeholder: 'Set hotkey to call out CocoAI',
        logo: 'Logo (Light)',
        logo_mobile: 'Mobile Logo (Light)',
        logo_dark: 'Logo (Dark)',
        logo_mobile_dark: 'Mobile Logo (Dark)',
        mode: 'Mode',
        mode_all: 'All',
        mode_embedded: 'Embedded',
        mode_embedded_icon: 'Embedded widget icon',
        mode_embedded_placeholder: 'Embedded widget placeholder text',
        mode_floating: 'Floating',
        mode_floating_icon: 'Floating widget icon',
        mode_floating_placeholder: 'Floating widget placeholder text',
        mode_modal: 'Modal',
        mode_page: 'Page',
        module_ai_overview: 'AI Overview',
        module_ai_overview_height: 'Height',
        module_ai_overview_output: 'Output',
        module_ai_overview_title: 'Title',
        module_ai_widgets: 'AI Widgets',
        module_ai_widgets_title: 'Widget',
        module_chat: 'AI Chat',
        module_chat_ai_assistant: 'AI Assistant',
        module_chat_placeholder: 'Chat box placeholder text',
        module_search: 'Search',
        module_search_placeholder: 'Search box placeholder text',
        module_search_welcome: 'Greeting message',
        name: 'Name',
        theme: 'Theme',
        theme_auto: 'Auto',
        theme_dark: 'Dark',
        theme_light: 'Light',
        language: 'Language',
        type: 'Type',
        type_fullscreen: 'Fullscreen',
        type_searchbox: 'SearchBox'
      },
      title: {
        edit: 'Edit Integration',
        new: 'New Integration'
      },
      hints: {
        tourist_mode: 'Unlogged users will access as this user'
      }
    },
    topics: {
      delete: 'Delete',
      label: 'Topics',
      new: 'New Topics',
      title: 'Update Suggested Topics'
    },
    update: {
      disable_confirm: `Are you sure you want to disable integration "{{name}}" ?`,
      enable_confirm: `Are you sure you want to enable integration "{{name}}" ?`
    }
  },
  login: {
    cloud: 'Sign in with INFINI Cloud',
    cocoAI: {
      autoDesc:
        'In order to continue, please click the link below if you are not redirected automatically within 5 seconds:',
      copyDesc:
        'If the redirect doesn’t work, you can copy the following URL and paste it into the Connect settings window in Coco AI:',
      enterCocoServer: 'Enter Coco Server',
      enterCocoServerDesc: 'Or, you can also:',
      launchCocoAI: 'Launch Coco AI',
      title: 'Open Coco AI'
    },
    common: {
      back: 'Back',
      codeLogin: 'Verification code login',
      codePlaceholder: 'Please enter verification code',
      confirm: 'Confirm',
      confirmPasswordPlaceholder: 'Please enter password again',
      loginOrRegister: 'Login / Register',
      loginSuccess: 'Login successfully',
      passwordPlaceholder: 'Please enter password',
      phonePlaceholder: 'Please enter phone number',
      userNamePlaceholder: 'Please enter user name',
      validateSuccess: 'Verification passed',
      welcomeBack: 'Welcome back, {{userName}} !'
    },
    desc: 'Enter your credentials to access your account.',
    password: 'Password',
    email: 'Email',
    title: 'Welcome'
  },
  mcpserver: {
    delete: {
      confirm: 'Are you sure you want to delete this ai MCP server "{{name}}"?'
    },
    labels: {
      category: 'Category',
      config: {
        args: 'Arguments',
        command: 'Command',
        env: 'Environment Variables'
      },
      description: 'Description',
      enabled: 'Enabled',
      icon: 'Icon',
      name: 'Name',
      type: 'Type'
    }
  },
  modelprovider: {
    delete: {
      confirm: 'Are you sure you want to delete this model provider?'
    },
    labels: {
      api_key: 'API Key',
      api_key_source: 'Get API Key from {{model_provider}}',
      api_key_source_normal: 'Click here to get API key',
      api_type: 'API Type',
      base_url: 'Base URL',
      builtin: 'Built-in',
      description: 'Description',
      enabled: 'Enabled',
      icon: 'Icon',
      models: 'Models',
      name: 'Name'
    }
  },
  settings: {
    app_settings: {
      chat_settings: {
        labels: {
          assistant: 'AI Assistant Display',
          introduction: 'Introduction Text',
          introduction_placeholder:
            'Enter the welcome text or AI tool introduction displayed on the start page (within 60 characters)',
          logo: 'Logo',
          logo_dark: 'Dark Theme (White or light-colored version Logo)',
          logo_light: 'Light Theme (Regular Version Logo)',
          logo_placeholder: 'Upload the company logo to be displayed in the start page.',
          logo_size_placeholder: 'Image size restrictions: Maximum height 30px, maximum width 300px.',
          start_page: 'Start Page',
          start_page_placeholder:
            'You can enable and configure the Chat mode start page, customize the company logo, introduction text, and common AI assistants to help users quickly select and start a chat.'
        },
        title: 'Chart Settings'
      },
      title: 'App Settings'
    },
    connector: {
      title: 'Connector'
    },
    llm: {
      answering_model: 'Answering Model',
      defaultModel: 'Default Model',
      desc: 'After integrating a large model, you will unlock the AI chat feature, providing intelligent search and an efficient work assistant.',
      endpoint: 'Endpoint',
      enhanced_inference: 'Enhanced Inference',
      frequency_penalty: 'Frequency Penalty',
      frequency_penalty_desc: 'the larger the value, the more likely it is to reduce repeated words',
      intent_analysis_model: 'Intent Analysis Model',
      keepalive: 'Keepalive',
      max_tokens: 'Max Tokens',
      max_tokens_desc: 'maximum number of tokens used in a single interaction',
      picking_doc_model: 'Picking Doc Model',
      presence_penalty: 'Presence Penalty',
      presence_penalty_desc: 'the larger the value, the more likely it is to expand to new topics',
      reasoning: 'This model supports reasoning mode',
      requestParams: 'Request Params',
      temperature: 'Temperature',
      temperature_desc: 'the larger the value, the more random the response',
      title: 'Connect to a Large Model',
      top_p: 'Top P',
      top_p_desc: `similar to temperature, don't change them simultaneously`,
      type: 'Type'
    },
    search_settings: {
      labels: {
        enabled: 'Enabled',
        integration: 'Integration'
      },
      title: 'Search Settings'
    },
    setupLater: 'Set Up Later'
  },
  webhook: {
    form: {
      title: {
        edit: 'Edit Webhook',
        new: 'New Webhook'
      }
    },
    labels: {
      ai_assistant: 'AI Assistant',
      content_type: 'Content type',
      datasource: 'Data Source',
      file_parse_completed: 'File Parsing Completed',
      model_provider: 'LLM Provider',
      name: 'Name',
      payload_url: 'Payload URL',
      reply_completed: 'Reply Completed',
      secret: 'Secret',
      ssl_verify: 'SSL Verify',
      sync_completed: 'Sync Completed',
      test: 'Test',
      test_need_save: 'Please save before testing',
      triggers: 'Triggers'
    },
    placeholders: {
      name: 'Please input',
      payload_url: 'Please input',
      secret: 'Please input'
    }
  },
  role: {
    title: 'Role',
    labels: {
      name: 'Name',
      description: 'Description',
      permission: 'Permission',
      object: 'Object',
      coco: 'Coco Server',
      generic: 'Generic',
      created: 'Created At'
    },
    new: {
      title: 'New Role'
    },
    edit: {
      title: 'Edit Role'
    },
    delete: {
      confirm: `Are you sure you want to delete role "{{name}}" ?`
    }
  },
  auth: {
    title: 'Authorization',
    labels: {
      name: 'Name',
      description: 'Description',
      permission: 'Permission',
      object: 'Authorization Object',
      coco: 'Coco Server',
      user: 'User',
      team: 'Team',
      userRole: 'Team Member Role',
      teamRole: 'APP User Role',
      roles: 'Roles',
      type: 'Type',
      created: 'Created At',
      auth: 'Authorization'
    },
    new: {
      title: 'New Authorization'
    },
    edit: {
      title: 'Edit Authorization'
    },
    delete: {
      confirm: `Are you sure you want to delete authorization "{{name}}" ?`
    }
  },
  user: {
    title: 'User',
    labels: {
      name: 'Full Name',
      email: 'Email',
      roles: 'Roles',
      created: 'Created At',
      password: 'Password'
    },
    new: {
      title: 'New User',
      copyPassword:
        'Please store this password in a secure and easily accessible location. For security reasons, you will not be able to view it again through the user management interface. If you lose this password, you will need to update password by editing user.'
    },
    edit: {
      title: 'Edit User'
    },
    delete: {
      confirm: `Are you sure you want to delete user "{{name}}" ?`
    }
  }
};

export default page;
