import Search from 'antd/es/input/Search';
import Icon, { FilterOutlined, PlusOutlined, EllipsisOutlined, ExclamationCircleOutlined} from '@ant-design/icons';
import { Button, Dropdown, Table, GetProp, message,Modal, Switch } from 'antd';
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import type { SorterResult } from 'antd/es/table/interface';
import {fetchDataSourceList, deleteDatasource, updateDatasource} from '@/service/api'
import { formatESSearchResult } from '@/service/request/es';
import { GoogleDriveSVG, HugoSVG, YuqueSVG,NotionSVG } from '@/components/icons';

const { confirm } = Modal;
type Datasource = Api.Datasource.Datasource;


type TablePaginationConfig = Exclude<GetProp<TableProps, 'pagination'>, boolean>;

interface TableParams {
  pagination?: TablePaginationConfig;
  sortField?: SorterResult<any>['field'];
  sortOrder?: SorterResult<any>['order'];
  filters?: Parameters<GetProp<TableProps, 'onChange'>>[1];
}

const TYPES = {
  'google_drive': {
    name: 'Google Drive',
    icon: GoogleDriveSVG
  },
  'hugo_site': {
    name: 'Hugo Site',
    icon: HugoSVG
  },
  'yuque': {
    name: 'Yuque',
    icon: YuqueSVG
  },
  'notion': {
    name: 'Notion',
    icon: NotionSVG
  },
}


export function Component() {
  const { t } = useTranslation();

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();
  const items: MenuProps["items"] = [
    {
      label: t('common.delete'),
      key: "1",
    },
    {
      label: t('common.edit'),
      key: "2",
    },
  ];

  const onMenuClick = ({key, record}: any)=>{
    switch(key){
      case "1":
        confirm({
          icon: <ExclamationCircleOutlined />,
          title: t('common.tip'),
          content: t('page.datasource.delete.confirm'),
          onOk() {
             //delete datasource by datasource id
            deleteDatasource(record.id).then((res)=>{
              if(res.data?.result === "deleted"){
                message.success(t('common.deleteSuccess'))
              }
              //reload data
              setReqParams((old)=>{
                return {
                  ...old,
                }
              })
            });
          },
          onCancel() {
          },
        });
       
        break;
      case "2":
        nav(`/data-source/edit/${record.id}`, {state:record});
        break;
    }
  }
  const onSyncEnabledChange = (value: boolean, record: Datasource)=>{
    record.sync_enabled = value;
    setLoading(true);
    updateDatasource(record.id, record).then((res)=>{
      if(res.data?.result === "updated"){
        message.success(t('common.updateSuccess'))
      }
      //reload data
      setReqParams((old)=>{
        return {
          ...old,
        }
      })
    }).finally(()=>{
      setLoading(false);
    });
  }
  const columns: TableColumnsType<Datasource> = [
    {
      title: t('page.datasource.columns.name'),
      dataIndex: "name",
      minWidth: 200,
      render: (value: string, record: Datasource)=>{
        const type = TYPES[record?.connector?.id]
        return (
          <a className='text-blue-500' onClick={()=>nav(`/data-source/detail/${record.id}`, {state:{datasource_name: record.name, connector_id: record.connector?.id || ''}})}>
            { type && <Icon component={type.icon} className='m-r-6px'/> }
            {value}
          </a>
        )
      }
    },
    {
      title: t('page.datasource.columns.type'),
      minWidth: 100,
      render: (text: string, record: Datasource)=>{
        const type = TYPES[record?.connector?.id]
        if (!type) return record?.connector?.id
        return type.name
      },
    },
    {
      dataIndex: 'sync_enabled',
      title: t('page.datasource.new.labels.sync_enabled'),
      width: 200,
      render: (value: boolean, record: Datasource)=>{
       return <Switch value={value} onChange={(v)=>onSyncEnabledChange(v, record)}/>
      }
    },
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
        <Search addonBefore={<FilterOutlined />} className='max-w-500px' onSearch={onRefreshClick} enterButton={ t('common.refresh')}></Search>
        <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/data-source/new-first`)}>{t('common.add')}</Button>
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
