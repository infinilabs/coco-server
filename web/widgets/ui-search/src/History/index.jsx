import { useMemo, useRef, useState } from 'react';
import PropTypes from 'prop-types';
import { Button, Input } from 'antd';
import { PanelLeftClose, RefreshCw, Search } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import debounce from 'lodash/debounce';
import clsx from 'clsx';
import HistoryListContent from './HistoryListContent';

const HistoryList = props => {
  const { historyPanelId, chats, active, onSearch, onRefresh, onSelect, onRename, onRemove, onClose } = props;
  const { t } = useTranslation();
  const searchInputRef = useRef(null);
  const [isRefresh, setIsRefresh] = useState(false);

  const debouncedSearch = useMemo(() => {
    return debounce(value => onSearch && onSearch(value), 300);
  }, [onSearch]);

  const handleRefresh = async () => {
    setIsRefresh(true);
    if (onRefresh) {
      await onRefresh();
    }
    setTimeout(() => {
      setIsRefresh(false);
    }, 1000);
  };

  return (
    <div
      className={clsx('flex flex-col h-full text-sm bg-[#F3F4F6] dark:bg-[#1F2937] pt-1')}
      id={historyPanelId}
    >
      <div className='flex items-center gap-2 border-b border-[var(--ui-search-antd-color-border-secondary)] p-2'>
        <Input
          className='h-8 w-8 flex-1 rounded-full'
          placeholder={t('history_list.search.placeholder', 'Search history...')}
          prefix={<Search className='h-4 w-4 text-[#999]' />}
          ref={searchInputRef}
          size='small'
          onChange={e => debouncedSearch(e.target.value)}
        />

        <Button
          className='h-8 w-8 flex items-center justify-center'
          size='small'
          type='text'
          onClick={handleRefresh}
        >
          <RefreshCw
            className={clsx('w-4 h-4 text-[#101010] hover:text-[#0287FF]', {
              'animate-spin': isRefresh
            })}
          />
        </Button>
      </div>

      <div className='custom-scrollbar flex-1 overflow-auto p-2'>
        <HistoryListContent
          active={active}
          chats={chats}
          onRemove={onRemove}
          onRename={onRename}
          onSelect={onSelect}
        />
      </div>

      {onClose && (
        <div className='flex justify-end border-t border-[var(--ui-search-antd-color-border-secondary)] p-2'>
          <Button
            size='small'
            type='text'
            onClick={onClose}
          >
            <PanelLeftClose className='h-4 w-4 text-gray-500' />
          </Button>
        </div>
      )}
    </div>
  );
};

export default HistoryList;

HistoryList.propTypes = {
  historyPanelId: PropTypes.string,
  chats: PropTypes.array,
  active: PropTypes.any,
  onSearch: PropTypes.func,
  onRefresh: PropTypes.func,
  onSelect: PropTypes.func,
  onRename: PropTypes.func,
  onRemove: PropTypes.func,
  onClose: PropTypes.func
};
