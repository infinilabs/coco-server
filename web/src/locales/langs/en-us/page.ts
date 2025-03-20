const page: App.I18n.Schema['translation']['page'] = {
  home: {
    server: {
      title: `{{user}}'s Coco Server`,
      address: 'Server address',
      addressDesc: 'In the connect settings of Coco AI, adding the Server address to the service list will allow you to access the service in Coco AI.',
      downloadCocoAI: 'Download Coco AI'
    },
    settings: {
      llm: 'LLMs',
      llmDesc: 'Connect the large model to enable AI chat, intelligent search, and a work assistant.',
      dataSource: 'Data Source',
      dataSourceDesc: 'Add data sources to the service list for unified search and analysis.',
      aiAssistant: 'AI Assistant',
      aiAssistantDesc: 'Set a personalized AI assistant to handle tasks efficiently and provide intelligent suggestions.'
    },
  },
  settings: {
    llm: {
      type: 'Type',
      endpoint: 'Endpoint',
      defaultModel: 'Default Model',
      keepalive: 'Keepalive',
      requestParams: 'Request Params',
      temperature: 'Temperature',
      temperature_desc: 'the larger the value, the more random the response',
      top_p: 'Top P',
      top_p_desc: `similar to temperature, don't change them simultaneously`,
      max_tokens: 'Max Tokens',
      max_tokens_desc: 'maximum number of tokens used in a single interaction',
      presence_penalty: 'Presence Penalty',
      presence_penalty_desc: 'the larger the value, the more likely it is to expand to new topics',
      frequency_penalty: 'Frequency Penalty',
      frequency_penalty_desc: 'the larger the value, the more likely it is to reduce repeated words',
      enhanced_inference: 'Enhanced Inference',
      intent_analysis_model: "Intent Analysis Model",
      picking_doc_model: "Picking Doc Model",
      answering_model: "Answering Model",
    }
  },
  login: {
    title: 'Welcome',
    desc: 'Enter your credentials to access your account.',
    password: 'Password',
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
    cocoAI: {
      title: 'Open Coco AI',
      autoDesc: 'In order to continue, please click the link below if you are not redirected automatically within 5 seconds:',
      launchCocoAI: 'Launch Coco AI',
      copyDesc: 'If the redirect doesn’t work, you can copy the following URL and paste it into the Connect settings window in Coco AI:',
      enterCocoServer: 'Enter Coco Server',
      enterCocoServerDesc: 'Or, you can also:'
    }
  },
  guide: {
    user: {
      title: 'Create a user account',
      desc: 'Set up a new user account to manage access and permissions.',
      name: 'Full Name',
      email: 'Email',
      password: 'Password'
    },
    llm: {
      title: 'Connect to a Large Model',
      desc: 'After integrating a large model, you will unlock the AI chat feature, providing intelligent search and an efficient work assistant.',
    },
    setupLater: 'Set Up Later',
  },
  datasource: {
    title: "Datasource",
    columns: {
      name: "Name",
      type: "Type",
      sync_policy: "Sync Policy",
      latest_sync_time: "Latest Sync Time",
      sync_status: "Sync Status",
      enabled: "Enabled",
      searchable: "Searchable",
    },
    new:{
      title: "{{connector}} Connection",
      labels: {
        name: "Name",
        type: "Type",
        indexing_scope: "Indexing Scope",
        data_sync: "Data Synchronization",
        manual_sync: "Manual Sync",
        manual_sync_desc: "Sync only when the user clicks the 'Sync' button",
        scheduled_sync: "Scheduled Sync",
        scheduled_sync_desc: "Sync at fixed intervals",
        realtime_sync: "Real-time Sync",
        realtime_sync_desc: "Sync immediately upon file modification",
        immediate_sync: "Immediate Sync",
        client_id: "Client ID",
        client_secret: "Client Secret",
        redirect_uri: "Redirect URI",
        sync_enabled: "Sync Enabled",
        enabled: "Enabled",
        site_urls: "Site URLs",
        connect: "Connect",
      }
    },
    edit:{
      title: "Edit Datasource",
    },
    delete: {
      confirm: 'Are you sure you want to delete this datasource?'
    },
    every: 'Every',
    seconds: 'seconds',
    minutes: 'minutes',
    hours: 'hours',
    connect: 'Connect',
    site_urls: 'Site URLs',
    site_urls_add: 'Add URL'
  },
  apitoken: {
    columns: {
      name: "Name",
      expire_in: "Expire In",
    },
    delete: {
      confirm: 'Are you sure you want to delete this API token?'
    },
    create: {
      store_desc: 'Please store this token in a secure and easily accessible location. For security reasons, you will not be able to view it again through the API Token management interface. If you lose this token, you will need to generate a new one.',
      limit: "Access token limit exceeded. Maximum allowed: 5.",
    }
  },
  connector: {
    columns: {
      name: "Name",
      category: "Category",
      description: "Description",
      tags: "Tags",
    },
    delete: {
      confirm: 'Are you sure you want to delete connector "{{name}}"？'
    },
    edit: {
      title: "Edit Connector",
    },
    new: {
      title: "New Connector",
      labels: {
        name: "Name",
        category: "Category",
        description: "Description",
        tags: "Tags",
        assets_icons: "Assets Icons",
        icon: "Connector Icon",
        client_id: "Client ID",
        client_secret: "Client Secret",
        redirect_url: "Redirect URI",
        auth_url: "Auth URI",
        token_url: "Token URI",
        asset_icon: "Icon",
        asset_type: "Type",
      }
    },
  }
};

export default page;
