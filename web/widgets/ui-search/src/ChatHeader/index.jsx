import PropTypes from 'prop-types';
import { History, MessageSquarePlus, Search } from 'lucide-react';
import AssistantSelector from './AssistantSelector';

const ChatHeader = props => {
  const {
    activeChat,
    title,
    assistantLabel,
    assistants,
    currentAssistant,
    assistantPage,
    assistantTotal,
    showChatHistory,
    onToggleHistory,
    onNewChat,
    onBackToSearch,
    onAssistantRefresh,
    onAssistantSelect,
    onAssistantPrevPage,
    onAssistantNextPage,
    onAssistantSearch
  } = props;

  const displayTitle = title ?? activeChat?._source?.title ?? activeChat?._source?.message ?? activeChat?._id ?? '';
  const displayAssistantLabel = currentAssistant?._source?.name ?? assistantLabel;

  return (
    <div className='h-full w-full flex items-center justify-between px-4 border-b border-[#ebebeb] box-border'>
      <div className='min-w-0 flex items-center gap-2'>
        {showChatHistory && (
          <button
            className='cursor-pointer border border-[#ebebeb] rounded-xl p-2 hover:bg-gray-100 dark:hover:bg-gray-800'
            type='button'
            onClick={onToggleHistory}
          >
            <History className='h-4 w-4' />
          </button>
        )}

        <AssistantSelector
          label={displayAssistantLabel}
          assistants={assistants}
          currentAssistant={currentAssistant}
          assistantPage={assistantPage}
          assistantTotal={assistantTotal}
          onAssistantRefresh={onAssistantRefresh}
          onAssistantSelect={onAssistantSelect}
          onAssistantPrevPage={onAssistantPrevPage}
          onAssistantNextPage={onAssistantNextPage}
          onAssistantSearch={onAssistantSearch}
        />

        <button
          className='cursor-pointer rounded-lg px-2 py-1.5 border border-[#ebebeb] hover:bg-gray-100 dark:hover:bg-gray-800'
          type='button'
          onClick={onNewChat}
        >
          <MessageSquarePlus className='relative top-0.5 h-4 w-4 text-[#0387FF]' />
        </button>

        <button
          className='h-8 flex cursor-pointer items-center gap-2 border border-[rgba(235,235,235,1)] rounded-full bg-[var(--ui-search-antd-color-bg-container)] px-3 text-sm text-[#999] dark:border-[rgba(50,50,50,1)]'
          type='button'
          onClick={onBackToSearch}
        >
          <Search className='h-4 w-4 text-[#0387FF]' />
          <span>返回搜索</span>
        </button>


      </div>

      <div className='min-w-0 flex flex-1 justify-center px-4'>
        <h2 className='max-w-full truncate text-sm text-gray-900 font-medium dark:text-gray-100'>{displayTitle}</h2>
      </div>

      <div className='w-200px' />
    </div>
  );
};

ChatHeader.propTypes = {
  activeChat: PropTypes.any,
  title: PropTypes.string,
  assistantLabel: PropTypes.string,
  assistants: PropTypes.array,
  currentAssistant: PropTypes.any,
  assistantPage: PropTypes.number,
  assistantTotal: PropTypes.number,
  showChatHistory: PropTypes.bool,
  onToggleHistory: PropTypes.func,
  onNewChat: PropTypes.func,
  onBackToSearch: PropTypes.func,
  onAssistantRefresh: PropTypes.func,
  onAssistantSelect: PropTypes.func,
  onAssistantPrevPage: PropTypes.func,
  onAssistantNextPage: PropTypes.func,
  onAssistantSearch: PropTypes.func
};

ChatHeader.defaultProps = {
  assistantLabel: 'Coco AI',
  assistants: [],
  showChatHistory: true
};

export default ChatHeader;
