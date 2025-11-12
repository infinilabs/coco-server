import DropdownList from '@/common/src/DropdownList';
import { useMemo, useState } from 'react';
import { getAssistant, searchAssistant } from '@/service/api/assistant';
import { formatESSearchResult } from '@/service/request/es';
import { useRequest } from '@sa/hooks';
import { getLocale } from '@/store/slice/app';

export default props => {
  const { value, onChange, width, className, mode, assistants, excluded = [] } = props;

  const { t } = useTranslation();
  const locale = useAppSelector(getLocale);

  const { hasAuth } = useAuth();

  const permissions = {
    read: hasAuth('coco#assistant/read'),
    search: hasAuth('coco#assistant/search'),
    create: hasAuth('coco#assistant/create'),
  };

  const {
    data: res,
    loading,
    run: fetchData
  } = useRequest(searchAssistant, {
    manual: true
  });

  const {
    data: itemRes,
    loading: itemLoading,
    run: fetchItem
  } = useRequest(getAssistant, {
    manual: true
  });

  const {
    data: itemsRes,
    loading: itemsLoading,
    run: fetchItems
  } = useRequest(searchAssistant, {
    manual: true
  });

  const [queryParams, setQueryParams] = useState({
    query: '',
    from: 0,
    size: 10
  });

  const [sorter, setSorter] = useState([]);

  const fetchFilterData = (queryParams, sorter, assistants) => {
    if (typeof assistants === 'undefined' || assistants.length !== 0) {
      fetchData({
        ...queryParams,
        sort: sorter.map(item => `${item[0]}:${item[1]}`).join(',') || 'created:desc',
        filter: assistants
          ? {
              id: assistants.map(item => item.id)
            }
          : {}
      });
    }
  };

  useEffect(() => {
    if (permissions.search) {
      fetchFilterData(queryParams, sorter, assistants);
    }
  }, [queryParams, sorter, assistants, permissions.search]);

  useEffect(() => {
    if (mode === 'multiple') {
      if (value && value.some(item => Boolean(item?.id && !item?.name)) && permissions.search) {
        fetchItems({
          filter: {
            id: value.map(item => item.id)
          },
          from: 0,
          size: 10000
        });
      }
    } else if (value?.id && !value?.name && permissions.read) {
      fetchItem(value.id);
    }
  }, [JSON.stringify(value), mode, permissions.search, permissions.read]);

  const result = useMemo(() => {
    const rs = formatESSearchResult(res);
    return {
      ...rs,
      data: rs.data.map(item => ({
        ...item,
        disabled: item.id === value?.id ? false : excluded?.includes(item.id)
      }))
    };
  }, [res, value, excluded]);

  const formatValue = useMemo(() => {
    if (mode === 'multiple') {
      if (value && value.some(item => Boolean(item?.id && !item?.name)) && itemsRes) {
        return itemsRes?.hits?.hits ? itemsRes?.hits?.hits.map(item => item._source) : [];
      }
      return value || [];
    }
    if (value?.id && !value?.name && itemRes) {
      return itemRes?._source;
    }
    return value;
  }, [value, itemRes, itemsRes, mode, excluded]);

  const { data, total } = result;

  return (
    <div className='flex items-center gap-2'>
      <DropdownList
        className={`ai-assistant-select ${className}`}
        data={data}
        dropdownWidth={width}
        getPopupContainer={triggerNode => triggerNode.parentNode}
        loading={loading || itemLoading || itemsLoading}
        mode={mode}
        placeholder='Please select'
        renderItem={item => item.name}
        renderLabel={item => item.name}
        rowKey='id'
        searchKey='name'
        sorter={sorter}
        sorterOptions={[{ label: 'Name', key: 'name' }]}
        value={formatValue}
        width={width || '100%'}
        locale={locale}
        actions={permissions.create ? [
          <a
            onClick={() => {
              window.open(`#/ai-assistant/new`, '_blank');
            }}
          >
            {t('common.create')}
          </a>
        ] : []}
        pagination={{
          currentPage: total ? Math.floor(queryParams.from / queryParams.size) + 1 : 0,
          total,
          onChange: page => {
            setQueryParams({
              ...queryParams,
              from: (page - 1) * queryParams.size
            });
          }
        }}
        onChange={onChange}
        onSorterChange={setSorter}
        onRefresh={permissions.search ? () => {
          fetchFilterData(queryParams, sorter, assistants);
        } : undefined}
        onSearchChange={value => {
          setQueryParams(params => ({
            ...params,
            query: value
          }));
        }}
      />
    </div>
  );
};
