import DropdownList from "@/common/src/DropdownList";
import { useMemo, useState } from "react";
import { getAssistant, searchAssistant } from "@/service/api/assistant";
import { formatESSearchResult } from "@/service/request/es";

export default (props) => {

    const { value, onChange, width, className, mode, assistants, excluded = [] } = props;

    const { t } = useTranslation();

    const {
      data: res,
      loading,
      run: fetchData
    } = useRequest(searchAssistant, {
      manual: true
    });

    const { data: itemRes, loading: itemLoading, run: fetchItem } = useRequest(getAssistant, {
      manual: true,
    });

    const { data: itemsRes, loading: itemsLoading, run: fetchItems } = useRequest(searchAssistant, {
      manual: true,
    });

    const [queryParams, setQueryParams] = useState({
      query: '',
      from: 0, 
      size: 10,
    })

    const [sorter, setSorter] = useState([])

    const fetchFilterData = (queryParams, sorter, assistants) => {
      if (typeof assistants === 'undefined' || assistants.length !== 0) {
        fetchData({
          ...queryParams,
          sort: sorter.map((item) => `${item[0]}:${item[1]}`).join(',') || 'created:desc',
          filter: assistants ? {
            id: assistants.map((item) => item.id)
          } : {}
        })
      }
    }

    useEffect(() => {
      fetchFilterData(queryParams, sorter, assistants)
    }, [queryParams, sorter, assistants])

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
          fetchItem(value.id)
        }
      }
    }, [JSON.stringify(value), mode])

    const result = useMemo(() => {
      const rs = formatESSearchResult(res?.data)
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
        if (value && value.some((item) => !!(item?.id && !item?.name)) && itemsRes?.data) {
          return itemsRes?.data?.hits?.hits ? itemsRes?.data?.hits?.hits.map((item) => item._source) : []
        }
        return value || []
      } else {
        if (value?.id && !value?.name && itemRes?.data) {
          return itemRes?.data._source
        }
        return value
      }
    }, [value, itemRes?.data, itemsRes?.data, mode, excluded])

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
          onRefresh={() => {
            fetchFilterData(queryParams, sorter, assistants)
          }}
          action={[
            <a
              onClick={() => {
                window.open(
                  `/#/ai-assistant/new`,
                  "_blank"
                );
              }}
            >
              {t('common.create')}
            </a>
          ]}
        />
      </div>
    )
}