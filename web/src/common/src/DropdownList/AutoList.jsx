import { useEffect, useMemo, useState } from "react";
import { DEFAULT_PAGE_SIZE, ORDER_DESC } from ".";
import List from "./List";
import { cloneDeep } from "lodash";

const AutoList = (props) => {
    const { visible, data = [], pagination, searchKey, searchValue, sorter, filters, groups } = props;

    const [currentPagination, setCurrentPagination] = useState();

    const handlePageChange = (newPage) => {
        const { pages } = currentPagination;
        if (newPage < 0 || newPage > pages) return;
        setCurrentPagination({
            ...currentPagination,
            currentPage: newPage
        })
    }

    const filterData = useMemo(() => {
        let newData = cloneDeep(data);
        if (searchKey && searchValue) {
            newData = newData.filter((item) => `${item[searchKey] || ''}`.toLowerCase().indexOf(searchValue.toLowerCase()) !== -1)
        }
        groups.filter((group) => group.value !== undefined && group.value !== '' && !!group.key).forEach((group) => {
            newData = newData.filter((item) => {
                if (item.hasOwnProperty(group.key)) {
                    return item[group.key] === group.value
                } else {
                    return true;
                }
            })
        })
        const keys = Object.keys(filters);
        if (keys.length > 0) {
            keys.filter((key) => (filters[key] || []).length !== 0).forEach((key) => {
                newData = newData.filter((item) => filters[key].indexOf(item[key]) !== -1)
            })
        }
        if (sorter.length >= 2 && sorter[0] && sorter[1]) {
            const key = sorter[0];
            const order = sorter[1];
            newData = newData.sort((a, b) => {
                if (typeof a[key] ==='string') {
                    return order === ORDER_DESC ? b[key].localeCompare(a[key]) : a[key].localeCompare(b[key])
                } else if (!isNaN(a[key])) {
                    return order === ORDER_DESC ? b[key] - a[key] : a[key] - b[key]
                }
                return 0
            })
        }
        return newData;
    }, [JSON.stringify(data), searchKey, searchValue, JSON.stringify(sorter), JSON.stringify(filters), JSON.stringify(groups)])

    useEffect(() => {
        if (pagination === false) {
            setCurrentPagination(false)
            return;
        }
        const total = filterData.length;
        const pageSize = pagination?.pageSize || DEFAULT_PAGE_SIZE;
        setCurrentPagination({
            currentPage: total ? 1 : 0,
            pageSize,
            total,
            pages: Math.ceil(total / pageSize)
        })
    }, [filterData, pagination])

    const pageData = useMemo(() => {
        if (!currentPagination) return filterData;
        const { currentPage, pageSize } = currentPagination;
        return filterData.filter((item, index) => index >= (currentPage - 1) * pageSize && index < currentPage * pageSize)
    }, [filterData, currentPagination])

    return (
        <List 
            {...props}  
            visible={visible}
            data={pageData}
            pagination={currentPagination === false ? false : {
                ...currentPagination,
                onChange: handlePageChange
            }}
        />
    )
}

export default AutoList 