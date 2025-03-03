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
      requestParams: 'Request Params',
      temperature: 'Temperature',
      temperatureDesc: 'the larger the value, the more random the response',
      topP: 'Top P',
      topPDesc: `similar to temperature, don't change them simultaneously`,
      maxTokens: 'Max Tokens',
      maxTokensDesc: 'maximum number of tokens used in a single interaction',
      presencePenalty: 'Presence Penalty',
      presencePenaltyDesc: 'the larger the value, the more likely it is to expand to new topics',
      frequencyPenalty: 'Frequency Penalty',
      frequencyPenaltyDesc: 'the larger the value, the more likely it is to reduce repeated words',
      enhancedInference: 'Enhanced Inference',
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
    columns: {
      name: "Name",
      type: "Type",
      sync_policy: "Sync Policy",
      latest_sync_time: "Latest Sync Time",
      sync_status: "Sync Status",
      enabled: "Enabled",
    },
    new:{
      title: "Data Source Connection",
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
        immediate_sync: "Immediate Sync"
      }
    }
  }
};

export default page;
