import PropTypes from "prop-types";
import { History, MessageSquarePlus, Search } from "lucide-react";

const ChatHeader = (props) => {
  const { onToggleHistory, onNewChat, onBackToSearch, AssistantList } = props;

  return (
    <div className="h-full w-full flex items-center justify-between px-4 border-b border-[#ebebeb] box-border">
      <div className="min-w-0 flex items-center gap-2">
        <button
          className="cursor-pointer border border-[#ebebeb] rounded-xl p-2 hover:bg-gray-100 dark:hover:bg-gray-800"
          type="button"
          onClick={onToggleHistory}
        >
          <History className="h-4 w-4" />
        </button>

        {AssistantList}

        <button
          className="cursor-pointer rounded-lg px-2 py-1.5 border border-[#ebebeb] hover:bg-gray-100 dark:hover:bg-gray-800"
          type="button"
          onClick={onNewChat}
        >
          <MessageSquarePlus className="relative top-0.5 h-4 w-4 text-[#0387FF]" />
        </button>

        <button
          className="h-8 flex cursor-pointer items-center gap-2 border border-[rgba(235,235,235,1)] rounded-full bg-(--ui-search-antd-color-bg-container) px-3 text-sm text-[#999] dark:border-[rgba(50,50,50,1)]"
          type="button"
          onClick={onBackToSearch}
        >
          <Search className="h-4 w-4 text-[#0387FF]" />
          <span>返回搜索</span>
        </button>
      </div>

      <div className="w-200px" />
    </div>
  );
};

ChatHeader.propTypes = {
  activeChat: PropTypes.any,
  title: PropTypes.string,
  showChatHistory: PropTypes.bool,
  onToggleHistory: PropTypes.func,
  onNewChat: PropTypes.func,
  onBackToSearch: PropTypes.func,
};

export default ChatHeader;
