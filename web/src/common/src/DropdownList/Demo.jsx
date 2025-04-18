import { useEffect, useState } from 'react';
import DropdownList, { ORDER_DESC } from '.';
import { cloneDeep } from 'lodash';

const mock_data = ((size) => {
    return [...new Array(size).keys()].map((item, index) => {
        const num = index + 1
        const type = index % 2 === 0 ? `elasticsearch` : `easysearch`
        return (
            {
                cluster_id: `${num} - ${type}`,
                cluster_name: `${num} - ${type}`,
                type,
                version: `7.0.${index}`,
                status: 'green'
            }
        )
    })
})(1000)

const Demo = () => {

    const pageSize = 10;
    const currentPage = undefined;
    // const [currentPage, setCurrentPage] = useState(1)
    const [data, setData] = useState([]);
    // const [total, setTotal] = useState();
    const [loading, setLoading] = useState(false)
    const [value, setValue] = useState([
        {
            cluster_id: '1 - elasticsearch',
            cluster_name: '1 - elasticsearch',
            type: 'elasticsearch',
            version: '7.0.2',
            status: 'green'
        }
    ])
    const [searchValue, setSearchValue] = useState()
    const [sorter, setSorter] = useState(['cluster_name', 'desc'])
    const [filters, setFilters] = useState({ })
    const [groups, setGroups] = useState([{ key: 'category', value: 'all' }, { key: 'type', value: 'easysearch'}])

    const fetchData = (currentPage, pageSize, searchValue, sorter, filters, groups) => {
        setLoading(true);
        setTimeout(() => {
            if (!currentPage) {
                setData(mock_data)
                setLoading(false)
                return mock_data
            };
            let newData = cloneDeep(mock_data)
            if (searchValue) {
                newData = newData.filter((item) => {
                    return item.cluster_name.toLowerCase().indexOf(searchValue.toLowerCase()) !== -1
                })
            }
            const keys = Object.keys(filters);
            if (keys.length > 0) {
                keys.filter((key) => (filters[key] || []).length !== 0).forEach((key) => {
                    newData = newData.filter((item) => filters[key].indexOf(item[key]) !== -1)
                })
            }

            if (groups.length > 0) {
                groups.forEach((group) => {
                    newData = newData.filter((item) => {
                        if (item.hasOwnProperty(group.key)) {
                            return item[group.key] === group.value
                        } else {
                            return true;
                        }
                    })
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
            // const total = newData.length
            setData(newData.filter((item, index) => index >= (currentPage - 1) * pageSize && index < currentPage * pageSize))
            // setTotal(total)
            setLoading(false)
        }, 1000)
    }

    //async
    useEffect(() => {
        fetchData(currentPage, pageSize, searchValue, sorter, filters, groups)
    }, [currentPage, pageSize, searchValue, sorter, filters, groups])

    // useEffect(() => {
    //     fetchData()
    // }, [])

    return (
        <DropdownList 
            width={300}
            allowClear
            mode="multiple"
            value={value}
            onChange={setValue}
            loading={loading}
            rowKey="cluster_id"
            data={data}
            renderItem={(item) => (
                <>
                    <span style={{ marginRight: 4 }}>
                        <span style={{
                            width: 12,
                            height: 12,
                            borderRadius: 6,
                            backgroundColor: 'green',
                            display: 'inline-block',
                        }}></span>
                    </span>
                    {item.cluster_name}
                </>
            )}
            renderTag={(item) => <div>{item.type} {item.version}</div>}
            // pagination={false}
            // pagination={{
            //     pageSize: 4,
            // }}
            // pagination={{
            //     currentPage,
            //     pageSize: pageSize,
            //     total,
            //     onChange: (page) => setCurrentPage(page)
            // }}
            // searchKey="cluster_name"
            onSearchChange={setSearchValue}
            sorter={sorter}
            onSorterChange={setSorter}
            sorterOptions={[
                { label: "Cluster Name", key: "cluster_name" },
            ]}
            filters={filters}
            onFiltersChange={setFilters}
            filterOptions={[
                { 
                    label: "DISTRIBUTION", 
                    key: "type", 
                    list: [
                        {
                            value: "elasticsearch"
                        },
                        {
                            value: "easysearch"
                        }
                    ]
                }
            ]}
            groups={groups}
            onGroupsChange={setGroups}
            groupOptions={[
                {
                    key: 'category',
                    label: 'All',
                    value: 'all',
                    list: [
                        {
                            key: 'type',
                            label: 'Easysearch',
                            value: 'easysearch'
                        },
                        {
                            key: 'type',
                            label: 'Elasticsearch',
                            value: 'elasticsearch'
                        },
                    ]
                },
                {
                    key: 'category',
                    label: 'InfiniLabs',
                    value: 'infinilabs',
                    list: [
                        {
                            key: 'type',
                            label: 'Easysearch',
                            value: 'easysearch'
                        },
                    ]
                },
            ]}
        />
    )
}

export default Demo;