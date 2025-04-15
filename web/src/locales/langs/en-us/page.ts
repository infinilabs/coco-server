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
        type: 'Type'
      },
      title: '{{connector}} Connection'
    },
    seconds: 'seconds',
    site_urls: 'Site URLs',
    site_urls_add: 'Add URL',
    title: 'Datasource'
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
      title: 'Create a user account'
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
      desc: 'Insert this  code into your website between <body> and </body> to start searching and chatting.',
      exit: 'Exit Preview',
      preview: 'Preview',
      title: 'Add the widget to your website'
    },
    columns: {
      datasource: 'Data Source',
      description: 'Description',
      enabled: 'Enabled',
      name: 'Name',
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
        enable_module: 'Enable Module',
        feature_chat_history: 'Chat History',
        feature_Control: 'Feature Control',
        feature_search: 'Show Datasource Search',
        feature_search_active: 'Enable Datasource Search',
        feature_think: 'Show Deep Think',
        feature_think_active: 'Enable Deep Think',
        hotkey: 'Hotkey',
        hotkey_placeholder: 'Set hotkey to call out CocoAI',
        module_chat: 'AI Chat',
        module_chat_placeholder: 'Chat box placeholder text',
        module_search: 'Search',
        module_search_placeholder: 'Search box placeholder text',
        name: 'Name',
        theme: 'Theme',
        theme_auto: 'Auto',
        theme_dark: 'Dark',
        theme_light: 'Light',
        type: 'Type',
        type_all: 'All',
        type_embedded: 'Embedded',
        type_embedded_placeholder: 'Embedded widget placeholder text',
        type_embedded_icon: 'Embedded widget icon',
        type_floating: 'Floating',
        type_floating_placeholder: 'Floating widget placeholder text',
        type_floating_icon: 'Floating widget icon',
      },
      title: {
        edit: 'Edit Integration',
        new: 'New Integration'
      }
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
    title: 'Welcome'
  },
  settings: {
    llm: {
      answering_model: 'Answering Model',
      defaultModel: 'Default Model',
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
    }
  }
};

export default page;
