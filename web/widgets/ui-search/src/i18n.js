import i18n from "i18next";

const resources = {
  'en-US': {
    translation: {
      labels: {
        deepThink: "DeepThink",
        deepResearch: "DeepResearch",
        hybrid: "Hybrid Search",
        semantic: "Semantic Search",
        keyword: "Keyword Search",
        accountInfo: "Account Info",
        guestTip: "Guest mode — login to unlock full experience",
        pleaseLogin: "Please log in to start.",
        login: "Login",
        inputPlaceholder: "Type a message...",
        fuzziness: "Fuzziness",
        bestMatch: "Best match",
        pastYear: "Past year",
        backToSearch: "Back to search",
        all: "All",
        document: "Document",
        image: "Image",
        searchTips: "Search Tips",
        advancedFilterTip: "Press / to enable advanced field filters, or enter fieldName: to convert to condition",
        advancedFilterTipPart1: "Press",
        advancedFilterTipPart1Suffix: "to enable advanced field filters,",
        advancedFilterTipOr: "or directly enter",
        fieldName: "fieldName",
        advancedFilterTipConvert: "to convert to condition",
        satisfyAll: "Match all conditions",
        conditionGroup: "Condition Group",
        excludeCondition: "Exclude condition",
        quickFind: "Quick find | Direct to files and results",
        deepThinkShort: "DeepThink | AI summary, conclusion first",
        deepResearchShort: "DeepResearch | Multi-step reasoning, comprehensive analysis",
        suggestionsTitle: "Search Suggestions",
        filterTitle: "Filter conditions",
        relatedSearch: "Related searches",
        aiOverview: "AI Insights",
        resultsWithTime: "Found {{count}} record ({{took}} millisecond)",
        resultsWithTime_plural: "Found {{count}} records ({{took}} milliseconds)",
      },
      history_list: {
        search: {
          placeholder: "Search history...",
        },
        date: {
          today: "Today",
          yesterday: "Yesterday",
          last7Days: "Previous 7 Days",
          last30Days: "Previous 30 Days",
        },
        menu: {
          rename: "Rename",
          delete: "Delete",
        },
        delete_modal: {
          title: "Delete Chat",
          description: "Are you sure you want to delete '{{item}}'?",
          button: {
            cancel: "Cancel",
            delete: "Delete",
          },
        },
        operate: {
          rename_success: "Rename successful",
          rename_error: "Rename failed",
          renaming: "Renaming...",
          delete_success: "Delete successful",
          delete_error: "Delete failed",
          deleting: "Deleting...",
        },
      },
      assistant_list: {
        default_name: "Assistant",
        title: "Assistants",
        search: {
          placeholder: "Search assistant",
        },
        no_data: "No data",
      },
      assistant: {
        chat: {
          greetings: "Hello! How can I help you today?",
          timedout: "Request timed out. Please try again.",
        },
      },
      app: {
        input: {
          placeholder: "Ask whatever you want...",
        },
        disclaimer: "AI generated content may be inaccurate.",
        new_chat: "New Chat",
        logo: {
          chat: "Chat",
        },
      },
      search: {
        input: {
          attachment: "Attachment",
          attachment_remove: "Remove",
          attachment_upload_failed: "Upload failed",
          voice: "Voice",
          send: "Send",
          stop: "Stop",
          deepThink: "DeepThink",
          deepResearch: "DeepResearch",
          search: "Search",
          MCP: "MCP",
          searchPopover: {
            title: "Select",
            placeholder: "Search...",
            allScope: "All Scope",
          },
        },
        textarea: {
          placeholder: "Ask whatever you want...",
          ariaLabel: "Chat Input",
        },
      },
    },
  },
  "zh-CN": {
    translation: {
      labels: {
        deepThink: "深度思考",
        deepResearch: "深度研究",
        hybrid: "混合搜索",
        semantic: "语义搜索",
        keyword: "关键词搜索",
        accountInfo: "账户信息",
        guestTip: "游客模式，登录解锁完整体验",
        pleaseLogin: "请登录您的账户以开始。",
        login: "登录",
        inputPlaceholder: "请输入问题...",
        fuzziness: "模糊程度",
        bestMatch: "最佳匹配",
        pastYear: "最近一年",
        backToSearch: "返回搜索",
        all: "全部",
        document: "文档",
        image: "图片",
        searchTips: "搜索 Tips",
        advancedFilterTip: "按 / 启用高级字段过滤，或直接输入 字段名 + : 转为条件",
        advancedFilterTipPart1: "按",
        advancedFilterTipPart1Suffix: "启用高级字段过滤，",
        advancedFilterTipOr: "或直接输入",
        fieldName: "字段名",
        advancedFilterTipConvert: "转为条件",
        satisfyAll: "满足全部条件",
        conditionGroup: "条件组合",
        excludeCondition: "排除条件",
        quickFind: "快速查找 | 直达文件与结果",
        deepThinkShort: "深度思考 | AI 提炼，结论优先",
        deepResearchShort: "深度研究 | 多步推理，综合分析",
        suggestionsTitle: "搜索建议",
        filterTitle: "过滤条件",
        relatedSearch: "相关搜索",
        aiOverview: "智能解读",
        resultsWithTime: "共找到 {{count}} 条记录（{{took}} 毫秒）",
      },
      history_list: {
        search: {
          placeholder: "搜索历史记录...",
        },
        date: {
          today: "今天",
          yesterday: "昨天",
          last7Days: "过去 7 天",
          last30Days: "过去 30 天",
        },
        menu: {
          rename: "重命名",
          delete: "删除",
        },
        delete_modal: {
          title: "删除对话",
          description: "确定要删除 '{{item}}' 吗？",
          button: {
            cancel: "取消",
            delete: "删除",
          },
        },
        operate: {
          rename_success: "重命名成功",
          rename_error: "重命名失败",
          renaming: "正在重命名...",
          delete_success: "删除成功",
          delete_error: "删除失败",
          deleting: "正在删除...",
        },
      },
      assistant_list: {
        default_name: "助手",
        title: "助手列表",
        search: {
          placeholder: "搜索助手",
        },
        no_data: "暂无数据",
      },
      assistant: {
        chat: {
          greetings: "你好！今天有什么可以帮你的吗？",
          timedout: "请求超时，请重试。",
        },
      },
      app: {
        input: {
          placeholder: "问点什么...",
        },
        disclaimer: "AI 生成的内容可能不准确。",
        new_chat: "新对话",
        logo: {
          chat: "对话",
        },
      },
      search: {
        input: {
          attachment: "附件",
          attachment_remove: "移除",
          attachment_upload_failed: "上传失败",
          voice: "语音",
          send: "发送",
          stop: "停止",
          deepThink: "深度思考",
          deepResearch: "深度研究",
          search: "联网搜索",
          MCP: "MCP",
          searchPopover: {
            title: "选择",
            placeholder: "搜索...",
            allScope: "全部范围",
          },
        },
        textarea: {
          placeholder: "问点什么...",
          ariaLabel: "聊天输入",
        },
      },
    },
  },
};

const i18nInstance = i18n.createInstance();

i18nInstance.init({
  resources,
  lng: "en-US",
  fallbackLng: "en-US",
  interpolation: {
    escapeValue: false,
  },
});

export default i18nInstance;
