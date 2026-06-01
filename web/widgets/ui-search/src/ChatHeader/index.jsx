import PropTypes from "prop-types";
import { Button } from "antd";
import { History, MessageSquarePlus, Search } from "lucide-react";
import { useTranslation } from 'react-i18next';

const ChatHeader = (props) => {
  const { onToggleHistory, onNewChat, onBackToSearch, AssistantList } = props;
  const { t } = useTranslation();

  return (
    <div className="h-full w-full flex items-center justify-between px-4 border-b border-solid border-[var(--ant-color-border-secondary)] box-border">
      <div className="min-w-0 flex items-center gap-2">
        <Button
          icon={<History className="h-4 w-4" />}
          onClick={onToggleHistory}
          className="!rounded-12px border-[#F0F0F0] dark:border-[#303030]"
        />

        <div className="border-[#F0F0F0] dark:border-[#303030]">
          {AssistantList}
        </div>

        <Button
          icon={<MessageSquarePlus className="h-4 w-4 !text-[var(--ant-color-primary)]" />}
          onClick={onNewChat}
          className="!rounded-12px border-[#F0F0F0] dark:border-[#303030]"
        />

        <Button
          shape="round"
          icon={<Search className="h-4 w-4 !text-[var(--ant-color-primary)]" />}
          onClick={onBackToSearch}
          className="!rounded-12px border-[#F0F0F0] dark:border-[#303030]"
        >
          {t('labels.backToSearch')}
        </Button>
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
