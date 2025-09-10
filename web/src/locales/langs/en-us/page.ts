import { t } from "i18next";

const page: App.I18n.Schema['translation']['page'] = {
  apitoken: {
    columns: {
      expire_in: 'Expire In',
      name: 'Name'
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
        description: 'Description',
        icon: 'Connector Icon',
        name: 'Name',
        config: 'Configuration',
        path_hierarchy: 'Path Hierarchy',
        redirect_url: 'Redirect URI',
        tags: 'Tags',
        token_url: 'Token URI'
      },
      title: 'New Connector'
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
    connect: 'Connect',
    missing_config_tip: "Google authorization parameters are not configured. Please set them before connecting. Click Confirm to go to the settings page.",
    delete: {
      confirm: 'Are you sure you want to delete this datasource?'
    },
    edit: {
      title: 'Edit Datasource'
    },
    every: 'Every',
    hours: 'hours',
    minutes: 'minutes',
    new: {
      labels: {
        client_id: 'Client ID',
        client_secret: 'Client Secret',
        connect: 'Connect',
        data_sync: 'Data Synchronization',
        enabled: 'Enabled',
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
        sync_enabled: 'Sync Enabled',
        type: 'Type',
        folder_paths: 'Folder paths',
        file_extensions: 'File extensions (optional)'
      },
      title: '{{connector}} Connection',
      tooltip: {
        file_extensions: 'Comma-separated list. e.g., pdf, docx, txt',
        folder_paths: 'Absolute paths to the folders you want to scan.'
      }
    },
    seconds: 'seconds',
    site_urls: 'Site URLs',
    site_urls_add: 'Add URL',
    file_paths_add: 'Add File Path',
    title: 'Datasource',
    commons: {
      error: {
        extensions_format: "Invalid file extensions. Use formats like 'pdf' or '.pdf', with only letters and numbers."
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
        endpoint: 'The base URL of your Confluence instance. e.g., http://localhost:8090 or https://wiki.example.com/confluence',
        space: 'The key of the Confluence space you want to index (e.g., "DS" or "KB").',
        token: 'Your Confluence Personal Access Token (PAT).',
        username: 'Username for authentication. Can be left empty for anonymous access or if using a Personal Access Token.'
      }
    },
    rdbms: {
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
      },
      placeholder: {
        field_name: 'Field Name',
        metadata_name: 'Metadata Name',
        payload_name: 'Payload Name'
      }
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
      name: 'Full Name',
      password: 'Password',
      title: 'Create a user account',
      language: 'Language',
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
  integration: {
    code: {
      desc: 'Insert this  code into your website between <body> and </body> to start using integration.',
      exit: 'Exit Preview',
      preview: 'Preview',
      title: 'Add the widget to your website',
      enabled_tips: `Please enable the integration, update and save it, than you can preview!`
    },
    columns: {
      datasource: 'Data Source',
      description: 'Description',
      enabled: 'Enabled',
      name: 'Name',
      type: 'Type',
      operation: {
        topics: 'Suggested Topics'
      },
      token_expire_in: 'Token Expire In',
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
        enable_module: 'Enable Module',
        feature_chat_history: 'Chat History',
        feature_Control: 'Feature Control',
        feature_search: 'Show Datasource Search',
        feature_search_active: 'Enable Datasource Search',
        feature_think: 'Show Deep Think',
        feature_think_active: 'Enable Deep Think',
        hotkey: 'Hotkey',
        hotkey_placeholder: 'Set hotkey to call out CocoAI',
        module_ai_widgets: 'AI Widgets',
        module_ai_widgets_title: 'Widget',
        module_ai_overview: 'AI Overview',
        module_ai_overview_title: 'Title',
        module_ai_overview_height: 'Height',
        module_ai_overview_output: 'Output',
        module_chat: 'AI Chat',
        module_chat_ai_assistant: 'AI Assistant',
        module_chat_placeholder: 'Chat box placeholder text',
        module_search: 'Search',
        module_search_placeholder: 'Search box placeholder text',
        module_search_welcome: 'Greeting message',
        logo: 'Logo',
        logo_mobile: 'Mobile Logo',
        name: 'Name',
        enabled: 'Enabled',
        theme: 'Theme',
        theme_auto: 'Auto',
        theme_dark: 'Dark',
        theme_light: 'Light',
        type: 'Type',
        type_searchbox: 'SearchBox',
        type_fullscreen: 'Fullscreen',
        mode: 'Mode',
        mode_all: 'All',
        mode_embedded: 'Embedded',
        mode_embedded_placeholder: 'Embedded widget placeholder text',
        mode_embedded_icon: 'Embedded widget icon',
        mode_floating: 'Floating',
        mode_floating_placeholder: 'Floating widget placeholder text',
        mode_floating_icon: 'Floating widget icon',
      },
      title: {
        edit: 'Edit Integration',
        new: 'New Integration'
      }
    },
    update: {
      disable_confirm: `Are you sure you want to disable integration "{{name}}" ?`,
      enable_confirm: `Are you sure you want to enable integration "{{name}}" ?`
    },
    topics: {
      title: 'Update Suggested Topics',
      label: 'Topics',
      new: 'New Topics',
      delete: 'Delete'
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
    title: 'Welcome'
  },
  settings: {
    llm: {
      title: 'Connect to a Large Model',
      desc: 'After integrating a large model, you will unlock the AI chat feature, providing intelligent search and an efficient work assistant.',
      answering_model: 'Answering Model',
      defaultModel: 'Default Model',
      reasoning: 'Reasoning Mode',
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
      requestParams: 'Request Params',
      temperature: 'Temperature',
      temperature_desc: 'the larger the value, the more random the response',
      top_p: 'Top P',
      top_p_desc: `similar to temperature, don't change them simultaneously`,
      type: 'Type'
    },
    connector: {
      title: "Connector"
    },
    app_settings: {
      title: "App Settings",
      chat_settings: {
        title: "Chart Settings",
        labels: {
          start_page: 'Start Page',
          start_page_placeholder: 'You can enable and configure the Chat mode start page, customize the company logo, introduction text, and common AI assistants to help users quickly select and start a chat.',
          logo: 'Logo',
          logo_placeholder: 'Upload the company logo to be displayed in the start page.',
          logo_size_placeholder: 'Image size restrictions: Maximum height 30px, maximum width 300px.',
          logo_light: 'Light Theme (Regular Version Logo)',
          logo_dark: 'Dark Theme (White or light-colored version Logo)',
          introduction: 'Introduction Text',
          introduction_placeholder: 'Enter the welcome text or AI tool introduction displayed on the start page (within 60 characters)',
          assistant: 'AI Assistant Display',
        }
      }
    },
    setupLater: 'Set Up Later'
  },
  modelprovider: {
    labels: {
      name: "Name",
      base_url: "Base URL",
      description: "Description",
      api_type: "API Type",
      models: "Models",
      enabled: "Enabled",
      icon: "Icon",
      api_key: "API Key",
      api_key_source: "Get API Key from {{model_provider}}",
      api_key_source_normal: "Click here to get API key",
      builtin: "Built-in",
    },
    delete: {
      confirm: 'Are you sure you want to delete this model provider?'
    },
  },
  assistant: {
    labels: {
      name: "Name",
      type: "Type",
      category: "Category",
      tags: "Tags",
      default_model: "Default Model",
      enabled: "Enabled",
      icon: "Icon",
      description: "Description",
      intent_analysis_model: "Intent Analysis",
      picking_doc_model: "Picking Doc",
      deep_think_model: "Deep Think Model",
      answering_model: "Answering Model",
      keepalive: "Keepalive",
      role_prompt: "Role Prompt",
      temperature: "Temperature",
      temperature_desc: "Controls the randomness of the generated text. A higher value produces more diverse but less stable content, while a lower value makes the output more predictable.",
      top_p: "Top P",
      top_p_desc: "The scope of selected vocabulary is limited. The lower the value, the more predictable the result; the higher the value, the more diverse the possibilities. It is not recommended to change this alongside randomness.",
      max_tokens: "Max Tokens",
      max_tokens_desc: "The maximum number of tokens used in a single interaction.",
      presence_penalty: "Presence Penalty",
      presence_penalty_desc: "Controls whether the AI tends to introduce new topics. The higher the value, the more the AI is inclined to introduce new content.",
      frequency_penalty: "Frequency Penalty",
      frequency_penalty_desc: "Controls the AI's repetition of the same vocabulary. The higher the value, the richer the expression.",
      reasoning: "Reasoning Mode",
      reasoning_desc: "Whether this model supports the reasoning mode.",
      chat_settings: "Chat Settings",
      greeting_settings: "Greeting Settings",
      suggested_chat: "Suggested chat",
      input_preprocessing: "User input preprocessing",
      input_preprocessing_desc: "The user's latest message will be filled into this template.",
      input_preprocessing_placeholder: "Preprocessing template: {{text}} will be replaced with the real-time input.",
      input_placeholder: "Input placeholder",
      history_message_number: "Number of historical messages included.",
      history_message_number_desc: "Number of historical messages included in each request.",
      history_message_compression_threshold: "Historical message length compression threshold.",
      history_message_compression_threshold_desc: "When the uncompressed historical messages exceed this value, compression will be applied.",
      history_summary: "Historical summary",
      history_summary_desc: "Automatically compress chat history and send it as context.",
      model_settings: "Model Settings",
      datasource: "Datasource",
      mcp_servers: "MCP Servers",
      upload: "Upload Settings",
      show_in_chat: "Show in chat",
      enabled_by_default: "Enabled by default",
      pick_datasource: "Pick Datasource",
      pick_tools: "Pick Tools",
      max_iterations: "Max iterations",
      caller_model: "Caller Model",
      filter: "Filter",
      tools: "Call LLM Tools",
      builtin_tools: "Built-in",
      prompt_settings: "Prompt Settings",
      prompt_settings_template: "Template",
      allowed_file_extensions: "Allowed File Extensions (eg: pdf,doc)",
      max_file_size_in_bytes: "Max File Size",
      max_file_count: "Max File Count"
    },
    mode: {
      simple: "Simple",
      deep_think: "Deep Think",
      workflow: "External workflow",
    },
    delete: {
      confirm: 'Are you sure you want to delete this ai assistant "{{name}}"?'
    },
  },
  mcpserver: {
    labels: {
      name: "Name",
      type: "Type",
      enabled: "Enabled",
      description: "Description",
      icon: "Icon",
      category: "Category",
      config: {
        command: "Command",
        args: "Arguments",
        env: "Environment Variables",
      }
    },
    delete: {
      confirm: 'Are you sure you want to delete this ai MCP server "{{name}}"?'
    },
  },
};

export default page;
