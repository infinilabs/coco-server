import DropdownList from "@/common/src/DropdownList";
import { useMemo, useState } from "react";
import { formatESSearchResult } from "@/service/request/es";
import { useRequest } from "@sa/hooks";
import { fetchPrincipals } from "@/service/api/share";
import { getLocale } from "@/store/slice/app";

export default (props) => {

    const { value, onChange, width, className, mode, excluded = [], children, onDropdownVisibleChange } = props;

    const { t } = useTranslation();
      const locale = useAppSelector(getLocale);

    const { hasAuth } = useAuth()

    const permissions = {
        search: hasAuth('generic#security:principal/search'),
    }

    const {
      data: res,
      loading,
      run: fetchData
    } = useRequest(fetchPrincipals, {
      manual: true
    });

    const { data: itemsRes, loading: itemsLoading, run: fetchItems } = useRequest(fetchPrincipals, {
      manual: true,
    });

    const [queryParams, setQueryParams] = useState({
      query: '',
      from: 0, 
      size: 10,
    })

    const [sorter, setSorter] = useState([])

    const fetchFilterData = (queryParams, sorter, excluded) => {
      if (!permissions.search) return;
      fetchData({
        ...queryParams,
        sort: sorter.map((item) => `${item[0]}:${item[1]}`).join(',') || 'created:desc',
        filter: Array.isArray(excluded) ? {
            '!id': excluded
          } : {}
      })
    }

    useEffect(() => {
      fetchFilterData(queryParams, sorter, excluded)
    }, [JSON.stringify(queryParams), JSON.stringify(sorter), JSON.stringify(excluded)])

    useEffect(() => {
      if (mode === 'multiple') {
        if (value && value.some((item) => !!(item?.id && !item?.name))) {
          fetchItems({
            filter: {
              id: value.map((item) => item.id)
            },
            from: 0, 
            size: 10000,
          })
        }
      } else {
        if (value?.id && !value?.name) {
          fetchItems({
            filter: {
              id: [value?.id]
            },
            from: 0, 
            size: 10000,
          })
        }
      }
    }, [JSON.stringify(value), mode])

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
        if (value?.id && !value?.name && itemsRes) {
          return itemsRes?.hits?.hits?.[0] ? itemsRes?.hits?.hits[0]._source : []
        }
        return value
      }
    }, [value, itemsRes, mode, excluded])

    const { data, total } = result;

    return (
      <DropdownList
        mode={mode}
        getPopupContainer={(triggerNode) => triggerNode.parentNode}
        className={`ai-assistant-select ${className}`}
        value={formatValue}
        loading={loading || itemsLoading}
        onDropdownVisibleChange={onDropdownVisibleChange}
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
          fetchFilterData(queryParams, sorter, excluded)
        } : undefined}
        locale={locale}
      >
        {children}
      </DropdownList>
    )
}