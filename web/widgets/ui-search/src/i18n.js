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
      }
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
      }
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
