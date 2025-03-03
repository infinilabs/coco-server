const page: App.I18n.Schema['translation']['page'] = {
  home: {
    server: {
      title: `'s `,
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
    bindWeChat: {
      title: 'Bind WeChat'
    },
    codeLogin: {
      getCode: 'Get verification code',
      imageCodePlaceholder: 'Please enter image verification code',
      reGetCode: 'Reacquire after {{time}}s',
      sendCodeSuccess: 'Verification code sent successfully',
      title: 'Verification Code Login'
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
    pwdLogin: {
      admin: 'Admin',
      forgetPassword: 'Forget password?',
      otherAccountLogin: 'Other Account Login',
      otherLoginMode: 'Other Login Mode',
      register: 'Register',
      rememberMe: 'Remember me',
      superAdmin: 'Super Admin',
      title: 'Password Login',
      user: 'User'
    },
    register: {
      agreement: 'I have read and agree to',
      policy: '《Privacy Policy》',
      protocol: '《User Agreement》',
      title: 'Register'
    },
    resetPwd: {
      title: 'Reset Password'
    }
  },
};

export default page;
