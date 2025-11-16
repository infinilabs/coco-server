import DropdownList from "@/common/src/DropdownList";
import { useMemo, useState } from "react";
import { formatESSearchResult } from "@/service/request/es";
import { fetchDataSourceList, getDatasource } from "@/service/api";
import { useRequest } from "@sa/hooks";
import { getLocale } from "@/store/slice/app";

export default (props) => {

    const { value, onChange, width, className, mode } = props;

    const { t } = useTranslation();
    const locale = useAppSelector(getLocale);
    const { hasAuth } = useAuth()

    const permissions = {
      read: hasAuth('coco#datasource/read'),
      search: hasAuth('coco#datasource/search'),
      create: hasAuth('coco#datasource/create'),
    };

    const {
      data: res,
      loading,
      run: fetchData
    } = useRequest(fetchDataSourceList, {
      manual: true
    });

    const { data: itemRes, loading: itemLoading, run: fetchItem } = useRequest(getDatasource, {
      manual: true,
    });

    const { data: itemsRes, loading: itemsLoading, run: fetchItems } = useRequest(fetchDataSourceList, {
      manual: true,
    });

    const [queryParams, setQueryParams] = useState({
      query: '',
      from: 0, 
      size: 10,
    })

    const [sorter, setSorter] = useState([])

    useEffect(() => {
      fetchData({
        ...queryParams,
        sort: sorter.map((item) => `${item[0]}:${item[1]}`).join(',') || 'created:desc',
      })
    }, [queryParams, sorter, permissions.search])

    useEffect(() => {
      if (mode === 'multiple') {
        if (value && value.some((item) => !!(item?.id && !item?.name)) && permissions.search) {
          fetchItems({
            filter: value.map((item) => `id:${item.id}`),
            from: 0, 
            size: 10000,
          })
        }
      } else {
        if (value?.id && !value?.name && permissions.read) {
          fetchItem(value.id)
        }
      }
    }, [JSON.stringify(value), mode, permissions.search, permissions.read])

    const result = useMemo(() => {
      return formatESSearchResult(res);
    }, [res, value])

    const formatValue = useMemo(() => {
      if (mode === 'multiple') {
        if (value && value.some((item) => !!(item?.id && !item?.name)) && itemsRes) {
          return itemsRes?.hits?.hits ? itemsRes?.hits?.hits.map((item) => item._source) : []
        }
        return value || []
      } else {
        if (value?.id && !value?.name && itemRes) {
          return itemRes._source
        }
        return value
      }
    }, [value, itemRes, itemsRes, mode])

    const { data, total } = result;

    return (
        <DropdownList
          mode={mode}
          getPopupContainer={(triggerNode) => triggerNode.parentNode}
          className={className}
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
            fetchData({
              ...queryParams,
              sort: sorter.map((item) => ({
                [item[0]]: {
                  "order": item[1]
                }
              }))
            })
          } : undefined}
          locale={locale}
          actions={permissions.create ? [
            <a
              onClick={() => {
                window.open(
                  `#/data-source/new-first`,
                  "_blank"
                );
              }}
            >
              {t('common.create')}
            </a>
          ] : []}
        />
    )
}