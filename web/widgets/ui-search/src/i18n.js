import i18n from "i18next";
import { initReactI18next } from "react-i18next";

const resources = {
  en: {
    translation: {
      "assistant.chat.timedout": "Request timed out. Please try again later.",
      "assistant.message.logo": "Assistant Logo",
      "history_list.search.placeholder": "Search history...",
      "history_list.no_history": "No history",
      "history_list.date.today": "Today",
      "history_list.date.yesterday": "Yesterday",
      "history_list.date.last7Days": "Previous 7 days",
      "history_list.date.last30Days": "Previous 30 days",
      "history_list.menu.rename": "Rename",
      "history_list.menu.delete": "Delete",
      "history_list.delete_modal.title": "Delete Chat",
      "history_list.delete_modal.button.delete": "Delete",
      "history_list.delete_modal.button.cancel": "Cancel",
      "history_list.delete_modal.description": "Are you sure you want to delete \"{{title}}\"? This action cannot be undone."
    }
  },
  zh: {
    translation: {
      "assistant.chat.timedout": "请求超时，请稍后再试。",
      "assistant.message.logo": "助手图标",
      "history_list.search.placeholder": "搜索历史...",
      "history_list.no_history": "暂无历史记录",
      "history_list.date.today": "今天",
      "history_list.date.yesterday": "昨天",
      "history_list.date.last7Days": "过去 7 天",
      "history_list.date.last30Days": "过去 30 天",
      "history_list.menu.rename": "重命名",
      "history_list.menu.delete": "删除",
      "history_list.delete_modal.title": "删除会话",
      "history_list.delete_modal.button.delete": "删除",
      "history_list.delete_modal.button.cancel": "取消",
      "history_list.delete_modal.description": "确定要删除会话 \"{{title}}\" 吗？此操作无法撤销。"
    }
  }
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng: "en",
    fallbackLng: "en",
    interpolation: {
      escapeValue: false
    }
  });

export default i18n;
