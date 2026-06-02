import PropTypes from "prop-types";
import { Button } from "antd";
import { History, MessageSquarePlus, Search } from "lucide-react";
import { useTranslation } from 'react-i18next';

const LeftSvg = () => (
  <svg viewBox="0 0 48 48" width="1em" height="1em" filter="none">
    <g>
    <path d="M6 9C6 7.34315 7.34315 6 9 6H39C40.6569 6 42 7.34315 42 9V39C42 40.6569 40.6569 42 39 42H9C7.34315 42 6 40.6569 6 39V9Z" fill="none" stroke="currentColor" stroke-width="4" stroke-linejoin="round"></path><path d="M32 6V42" stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"></path><path d="M16 20L20 24L16 28" stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"></path><path d="M26 6H38" stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"></path><path d="M26 42H38" stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"></path>
    </g>
  </svg>
)

const RightSvg = () => (
  <svg viewBox="0 0 48 48" width="1em" height="1em" filter="none">
    <g>
    <rect x="6" y="6" width="36" height="36" rx="3" fill="none" stroke="currentColor" stroke-width="4" stroke-linejoin="round"></rect><path d="M18 6V42" stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"></path><path d="M11 6H36" stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"></path><path d="M11 42H36" stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"></path><path d="M32 20L28 24L32 28" stroke="currentColor" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" fill="none"></path>
    </g>
  </svg>
)

const ChatHeader = (props) => {
  const { isHistoryOpen, onToggleHistory, onNewChat, onBackToSearch, AssistantList } = props;
  const { t } = useTranslation();

  return (
    <div className="h-full w-full flex items-center justify-between px-4 border-b border-solid border-[var(--ant-color-border-secondary)] box-border">
      <div className="min-w-0 flex items-center gap-2">
        <Button
          icon={isHistoryOpen ? <RightSvg className="h-4 w-4" /> : <LeftSvg className="h-4 w-4" />}
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
