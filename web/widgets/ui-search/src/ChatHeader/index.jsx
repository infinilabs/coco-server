import PropTypes from 'prop-types';
import { useEffect, useMemo, useRef, useState } from 'react';
import { Input } from 'antd';
import clsx from 'clsx';
import { ChevronDown, History, MessageSquarePlus, RefreshCw, Search } from 'lucide-react';

const ChatHeader = props => {
  const { activeChat, title, assistantLabel, assistants, currentAssistant, showChatHistory, onToggleHistory, onNewChat, onBackToSearch, onAssistantRefresh, onAssistantSelect } = props;

  const displayTitle = title ?? activeChat?._source?.title ?? activeChat?._source?.message ?? activeChat?._id ?? '';
  const displayAssistantLabel = currentAssistant?._source?.name ?? assistantLabel;

  const [isAssistantOpen, setIsAssistantOpen] = useState(false);
  const [assistantKeyword, setAssistantKeyword] = useState('');
  const [isAssistantRefreshing, setIsAssistantRefreshing] = useState(false);
  const searchInputRef = useRef(null);
  const assistantPopoverRef = useRef(null);

  const filteredAssistants = useMemo(() => {
    const list = Array.isArray(assistants) ? assistants : [];
    const keyword = (assistantKeyword || '').trim().toLowerCase();
    if (!keyword) return list;
    return list.filter(item => {
      if (!item) return false;
      if (typeof item === 'string') return item.toLowerCase().includes(keyword);
      const name = item?._source?.name ?? item?.name ?? item?._id ?? '';
      return String(name).toLowerCase().includes(keyword);
    });
  }, [assistants, assistantKeyword]);

  const handleAssistantRefresh = async () => {
    setIsAssistantRefreshing(true);
    if (onAssistantRefresh) {
      await onAssistantRefresh();
    }
    setTimeout(() => {
      setIsAssistantRefreshing(false);
    }, 1000);
  };

  useEffect(() => {
    if (!isAssistantOpen) return;

    const onPointerDown = event => {
      const root = assistantPopoverRef.current;
      if (!root) return;
      if (root.contains(event.target)) return;
      setIsAssistantOpen(false);
    };

    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, [isAssistantOpen]);

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

        <div
          className='relative'
          ref={assistantPopoverRef}
        >
          <button
            className='flex cursor-pointer items-center gap-1 border border-[#ebebeb] rounded-xl px-2 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-800'
            type='button'
            onClick={() => {
              setIsAssistantOpen(open => !open);
              setTimeout(() => {
                if (searchInputRef.current) searchInputRef.current.focus();
              }, 0);
            }}
          >
            <span className='text-sm'>{displayAssistantLabel}</span>
            <ChevronDown className='h-4 w-4' />
          </button>

          {isAssistantOpen && (
            <div className='absolute left-0 top-full z-50 mt-2 w-64 rounded-xl border border-[var(--ui-search-antd-color-border-secondary)] bg-[var(--ui-search-antd-color-bg-container)] shadow-lg p-3'>
              <div className='flex items-center justify-between text-sm font-semibold'>
                <div className='truncate'>小助手（{filteredAssistants.length}）</div>
                <button
                  className='h-6 w-6 flex items-center justify-center cursor-pointer rounded hover:bg-gray-100 dark:hover:bg-gray-800'
                  type='button'
                  onClick={e => {
                    e.preventDefault();
                    e.stopPropagation();
                    handleAssistantRefresh();
                  }}
                >
                  <RefreshCw
                    className={clsx('w-4 h-4 text-[#101010] hover:text-[#0287FF]', {
                      'animate-spin': isAssistantRefreshing
                    })}
                  />
                </button>
              </div>

              <div className='mt-2'>
                <Input
                  className='h-8 rounded-full'
                  placeholder='搜索小助手...'
                  prefix={<Search className='h-4 w-4 text-[#999]' />}
                  ref={searchInputRef}
                  size='small'
                  value={assistantKeyword}
                  onChange={e => setAssistantKeyword(e.target.value)}
                  onClick={e => e.stopPropagation()}
                />
              </div>

              <div className='mt-2 max-h-60 overflow-auto'>
                {filteredAssistants.length > 0 ? (
                  <div className='flex flex-col gap-1'>
                    {filteredAssistants.map(item => {
                      const id = typeof item === 'string' ? item : item?._id;
                      const name = typeof item === 'string' ? item : (item?._source?.name ?? item?.name ?? item?._id);
                      const isActive = currentAssistant && id && currentAssistant?._id === id;
                      return (
                        <button
                          key={id || name}
                          className={clsx(
                            'w-full text-left px-2 py-1.5 rounded-lg text-sm transition-colors',
                            isActive ? 'bg-[#E5E7EB] dark:bg-[#2B3444]' : 'hover:bg-[#EDEDED] dark:hover:bg-[#353F4D]'
                          )}
                          type='button'
                          onClick={e => {
                            e.preventDefault();
                            e.stopPropagation();
                            if (onAssistantSelect) onAssistantSelect(item);
                            setIsAssistantOpen(false);
                          }}
                        >
                          <div className='truncate'>{name}</div>
                        </button>
                      );
                    })}
                  </div>
                ) : (
                  <div className='py-3 text-center text-sm text-[#999]'>暂无小助手</div>
                )}
              </div>
            </div>
          )}
        </div>

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
  showChatHistory: PropTypes.bool,
  onToggleHistory: PropTypes.func,
  onNewChat: PropTypes.func,
  onBackToSearch: PropTypes.func,
  onAssistantRefresh: PropTypes.func,
  onAssistantSelect: PropTypes.func
};

ChatHeader.defaultProps = {
  assistantLabel: 'Coco AI',
  assistants: [],
  showChatHistory: true
};

export default ChatHeader;
