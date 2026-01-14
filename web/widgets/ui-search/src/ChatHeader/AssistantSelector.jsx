import PropTypes from 'prop-types';
import { useEffect, useMemo, useRef, useState } from 'react';
import { Input } from 'antd';
import clsx from 'clsx';
import { ChevronDown, RefreshCw, Search } from 'lucide-react';

const AssistantSelector = props => {
  const {
    label,
    assistants,
    currentAssistant,
    assistantPage,
    assistantTotal,
    onAssistantRefresh,
    onAssistantSelect,
    onAssistantPrevPage,
    onAssistantNextPage,
    onAssistantSearch
  } = props;

  const [isAssistantOpen, setIsAssistantOpen] = useState(false);
  const [assistantKeyword, setAssistantKeyword] = useState('');
  const [isAssistantRefreshing, setIsAssistantRefreshing] = useState(false);
  const searchInputRef = useRef(null);
  const assistantPopoverRef = useRef(null);

  const filteredAssistants = useMemo(() => {
    const list = Array.isArray(assistants) ? assistants : [];
    const keyword = (assistantKeyword || '').trim().toLowerCase();
    if (onAssistantSearch) return list;
    if (!keyword) return list;
    return list.filter(item => {
      if (!item) return false;
      if (typeof item === 'string') return item.toLowerCase().includes(keyword);
      const name = item?._source?.name ?? item?.name ?? item?._id ?? '';
      return String(name).toLowerCase().includes(keyword);
    });
  }, [assistants, assistantKeyword, onAssistantSearch]);

  const displayAssistantCount =
    typeof assistantTotal === 'number' ? assistantTotal : filteredAssistants.length;

  const pageSize = 5;
  const totalPages =
    typeof assistantTotal === 'number' && assistantTotal > 0
      ? Math.ceil(assistantTotal / pageSize)
      : 1;
  const currentPage = typeof assistantPage === 'number' && assistantPage > 0 ? assistantPage : 1;
  const hasPrevPage = currentPage > 1;
  const hasNextPage = currentPage < totalPages;

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
    <div className='relative' ref={assistantPopoverRef}>
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
        <span className='text-sm'>{label}</span>
        <ChevronDown className='h-4 w-4' />
      </button>

      {isAssistantOpen && (
        <div className='absolute left-0 top-full z-50 mt-2 w-64 rounded-xl border border-(--ui-search-antd-color-border-secondary) bg-white dark:bg-gray-900 shadow-lg p-3'>
          <div className='flex items-center justify-between text-sm font-semibold'>
            <div className='truncate'>小助手（{displayAssistantCount}）</div>
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
              onChange={e => {
                const value = e.target.value;
                setAssistantKeyword(value);
                if (onAssistantSearch) {
                  onAssistantSearch(value);
                }
              }}
              onClick={e => e.stopPropagation()}
            />
          </div>

          <div className='mt-2 max-h-60 overflow-auto'>
            {filteredAssistants.length > 0 ? (
              <div className='flex flex-col gap-1'>
                {filteredAssistants.map(item => {
                  const id = typeof item === 'string' ? item : item?._id;
                  const name =
                    typeof item === 'string'
                      ? item
                      : item?._source?.name ?? item?.name ?? item?._id;
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
          {assistantTotal > 0 && (
            <div className='mt-2 flex items-center justify-between text-xs text-[#999]'>
              <button
                className={clsx(
                  'px-2 py-1 rounded cursor-pointer border border-transparent',
                  hasPrevPage
                    ? 'hover:bg-[#EDEDED] dark:hover:bg-[#353F4D]'
                    : 'opacity-40 cursor-not-allowed'
                )}
                disabled={!hasPrevPage}
                type='button'
                onClick={e => {
                  e.preventDefault();
                  e.stopPropagation();
                  if (hasPrevPage && onAssistantPrevPage) {
                    onAssistantPrevPage();
                  }
                }}
              >
                上一页
              </button>
              <span>
                {currentPage}/{totalPages}
              </span>
              <button
                className={clsx(
                  'px-2 py-1 rounded cursor-pointer border border-transparent',
                  hasNextPage
                    ? 'hover:bg-[#EDEDED] dark:hover:bg-[#353F4D]'
                    : 'opacity-40 cursor-not-allowed'
                )}
                disabled={!hasNextPage}
                type='button'
                onClick={e => {
                  e.preventDefault();
                  e.stopPropagation();
                  if (hasNextPage && onAssistantNextPage) {
                    onAssistantNextPage();
                  }
                }}
              >
                下一页
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

AssistantSelector.propTypes = {
  label: PropTypes.string,
  assistants: PropTypes.array,
  currentAssistant: PropTypes.any,
  assistantPage: PropTypes.number,
  assistantTotal: PropTypes.number,
  onAssistantRefresh: PropTypes.func,
  onAssistantSelect: PropTypes.func,
  onAssistantPrevPage: PropTypes.func,
  onAssistantNextPage: PropTypes.func,
  onAssistantSearch: PropTypes.func
};

AssistantSelector.defaultProps = {
  label: 'Coco AI',
  assistants: []
};

export default AssistantSelector;

