import Search from 'antd/es/input/Search';
import { FilterOutlined, PlusOutlined, EllipsisOutlined} from '@ant-design/icons';
import { Button, Dropdown, Table, GetProp, message } from 'antd';
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import type { SorterResult } from 'antd/es/table/interface';
import {fetchDataSourceList, deleteDatasource} from '@/service/api'
import { formatESSearchResult } from '@/service/request/es';
type Datasource = Api.Datasource.Datasource;


type TablePaginationConfig = Exclude<GetProp<TableProps, 'pagination'>, boolean>;

interface TableParams {
  pagination?: TablePaginationConfig;
  sortField?: SorterResult<any>['field'];
  sortOrder?: SorterResult<any>['order'];
  filters?: Parameters<GetProp<TableProps, 'onChange'>>[1];
}

export function Component() {
  const { t } = useTranslation();

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();
  const items: MenuProps["items"] = [
    {
      label: t('common.delete'),
      key: "1",
      onClick: ()=>{

      }
    },
  ];

  const onMenuClick = ({key, record}: any)=>{
    switch(key){
      case "1":
        //todo delete datasource
        deleteDatasource(record.id).then((res)=>{
          if(res.data?.result === "deleted"){
            message.success("deleted success")
          }
          //reload data
          setReqParams((old)=>{
            return {
              ...old,
            }
          })
        })
        
    }
  }
  const columns: TableColumnsType<Datasource> = [
    {
      title: t('page.datasource.columns.name'),
      dataIndex: "name",
      minWidth: 200,
      render: (value: string, record: Datasource)=>{
        return <a className='text-blue-500' onClick={()=>nav(`/data-source/detail/${record.id}`, {state:{datasource_name: record.name}})}>{value}</a>
      }
    },
    {
      title: t('page.datasource.columns.type'),
      minWidth: 100,
      render: (text: string, record: Datasource)=>{
        return record.connector.id
      },
    },
    // {
    //   dataIndex: 'sync_config.type',
    //   width: 200,
    //   title: t('page.datasource.columns.sync_policy')
    // },
    // {
    //   dataIndex: 'latest_sync_time',
    //   key: 'latest_sync_time',
    //   title: t('page.datasource.columns.latest_sync_time'),
    //   width: 200
    // },
    // {
    //   dataIndex: 'sync_status',
    //   key: 'sync_status',
    //   width: 200,
    //   title: t('page.datasource.columns.sync_status')
    // },
    // {
    //   align: 'center',
    //   dataIndex: 'enabled',
    //   key: 'enabled',
    //   width: 200,
    //   title: t('page.datasource.columns.enabled')
    // },
    {
      title: t('common.operation'),
      fixed: 'right',
      width: "90px",
      render: (_, record) => {
        return <Dropdown menu={{ items, onClick:({key})=>onMenuClick({key, record}) }}>
          <EllipsisOutlined/>
        </Dropdown>
      },
    },
  ];
  // rowSelection object indicates the need for row selection
const rowSelection: TableProps<Datasource>["rowSelection"] = {
  onChange: (selectedRowKeys: React.Key[], selectedRows: Datasource[]) => {
  },
  getCheckboxProps: (record: Datasource) => ({
    name: record.name,
  }),
};

const [data, setData] = useState<Datasource[]>();
const [loading, setLoading] = useState(false);
const [tableParams, setTableParams] = useState<TableParams>({
  pagination: {
    current: 1,
    pageSize: 10,
  },
});

const [reqParams, setReqParams] = useState({
  query: '',
  from: 0, 
  size: 10,
})
const fetchData = () => {
  setLoading(true);
  fetchDataSourceList(reqParams).then(({ data }) => {
    const newData = formatESSearchResult(data);
      setData(newData?.data || []);
      setLoading(false);
      if(newData?.data?.length == reqParams.size){
        setReqParams((oldParams)=>{
          return {
            ...oldParams,
            from: oldParams.from+oldParams.size,
          }
        })
      }
      setTableParams(oldParams=>{
        return {
          ...oldParams,
          pagination: {
            ...oldParams.pagination,
            total: newData.total?.value || newData.total,
          },
        }
      });
    });
  };

  useEffect(fetchData, [
    reqParams
  ]);

  const handleTableChange: TableProps<Datasource>['onChange'] = (pagination, filters, sorter) => {
    setTableParams({
      pagination,
      filters,
      sortOrder: Array.isArray(sorter) ? undefined : sorter.order,
      sortField: Array.isArray(sorter) ? undefined : sorter.field,
    });

    // `dataSource` is useless since `pageSize` changed
    if (pagination.pageSize !== tableParams.pagination?.pageSize) {
      setData([]);
    }
  };
  const onRefreshClick = (query: string)=>{
    setReqParams((oldParams)=>{
      return {
        ...oldParams,
        query: query,
        form: 0,
      }
    })
  }

  return (
    <div className="h-full min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
      >
      <div className='mb-4 mt-4 flex items-center justify-between'>
        <Search addonBefore={<FilterOutlined />} className='max-w-500px' placeholder="input search text" onSearch={onRefreshClick} enterButton={ t('common.refresh')}></Search>
        <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/data-source/new`)}>{t('common.add')}</Button>
      </div>
      <Table<Datasource>
          rowKey="id"
          loading={loading}
          size="middle"
          rowSelection={{ ...rowSelection }}
          columns={columns}
          dataSource={data}
          onChange={handleTableChange}
        />
      </ACard>
    </div>
  );
}
