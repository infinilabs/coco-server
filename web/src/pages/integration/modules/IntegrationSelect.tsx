import DropdownList from "@/common/src/DropdownList";
import { useMemo, useState } from "react";
import { fetchIntegration, fetchIntegrations } from "@/service/api/integration";
import { formatESSearchResult } from "@/service/request/es";
import { useRequest } from "@sa/hooks";
import { getLocale } from "@/store/slice/app";

export default (props) => {

    const { value, onChange, width, className, mode, filter, excluded = [] } = props;

    const { t } = useTranslation();
    const locale = useAppSelector(getLocale);
    const { hasAuth } = useAuth()

    const permissions = {
      read: hasAuth('coco#integration/read'),
      search: hasAuth('coco#integration/search'),
      create: hasAuth('coco#integration/create'),
    };

    const {
      data: res,
      loading,
      run: fetchData
    } = useRequest(fetchIntegrations, {
      manual: true
    });

    const { data: itemRes, loading: itemLoading, run: fetchItem } = useRequest(fetchIntegration, {
      manual: true,
    });

    const { data: itemsRes, loading: itemsLoading, run: fetchItems } = useRequest(fetchIntegrations, {
      manual: true,
    });

    const [queryParams, setQueryParams] = useState({
      query: '',
      from: 0, 
      size: 10,
    })

    const [sorter, setSorter] = useState([])

    const fetchFilterData = (queryParams, sorter, filter) => {
      fetchData({
        ...queryParams,
        sort: sorter.map((item) => `${item[0]}:${item[1]}`).join(',') || 'created:desc',
        filter
      })
    }

    useEffect(() => {
      if (!permissions.search) return;
      fetchFilterData(queryParams, sorter, filter)
    }, [queryParams, sorter, filter, permissions.search])

    useEffect(() => {
      if (mode === 'multiple') {
        if (value && value.some((item) => !!(item?.id && !item?.name)) && permissions.search) {
          fetchItems({
            filter,
            from: 0, 
            size: 10000,
          })
        }
      } else {
        if (value?.id && !value?.name && permissions.read) {
          fetchItem(value.id)
        }
      }
    }, [JSON.stringify(value), mode, filter, permissions.search, permissions.read])

    const result = useMemo(() => {
      const rs = formatESSearchResult(res)
      return {
        ...rs,
        data: rs.data.map((item) => ({
          ...item,
          disabled: item.id === value?.id ? false : excluded?.includes(item.id)
        }))
      };
    }, [res, value, excluded])

    const formatValue = useMemo(() => {
      if (mode === 'multiple') {
        if (value && value.some((item) => !!(item?.id && !item?.name)) && itemsRes) {
          return itemsRes?.hits?.hits ? itemsRes?.hits?.hits.map((item) => item._source) : []
        }
        return value || []
      } else {
        if (value?.id && !value?.name && itemRes) {
          return itemRes?._source
        }
        return value
      }
    }, [value, itemRes, itemsRes, mode, excluded])

    const { data, total } = result;

    return (
      <div className="flex gap-2 items-center">
        <DropdownList
          mode={mode}
          getPopupContainer={(triggerNode) => triggerNode.parentNode}
          className={`ai-assistant-select ${className}`}
          value={formatValue}
          loading={loading || itemLoading || itemsLoading}
          onChange={onChange}
          placeholder="Please select"
          rowKey="id"
          data={data}
          renderItem={(item) => (
              item.name
          )}
          width={width || "100%"}
          dropdownWidth={width}
          renderLabel={(item) => item.name}
          searchKey="name"
          onSearchChange={(value) => {
            setQueryParams((params) => ({
              ...params,
              query: value
            }))
          }}
          sorter={sorter}
          onSorterChange={setSorter}
          sorterOptions={[
            { label: "Name", key: "name" },
          ]}
          pagination={{
            currentPage: total
              ? Math.floor(queryParams.from / queryParams.size) + 1
              : 0,
            total,
            onChange: (page) => {
              setQueryParams({
                ...queryParams,
                from: (page - 1) * queryParams.size,
              });
            },
          }}
          onRefresh={permissions.search ? () => {
            fetchFilterData(queryParams, sorter, filter)
          } : undefined}
          actions={permissions.create ? [
            <a
              onClick={() => {
                window.open(
                  `#/integration/new`,
                  "_blank"
                );
              }}
            >
              {t('common.create')}
            </a>
          ] : []}
          locale={locale}
        />
      </div>
    )
}