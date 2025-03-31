import Search from 'antd/es/input/Search';
import Icon, { FilterOutlined, PlusOutlined, EllipsisOutlined, ExclamationCircleOutlined} from '@ant-design/icons';
import { Button, Dropdown, Table, GetProp, message,Modal, Switch, Image } from 'antd';
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import type { SorterResult } from 'antd/es/table/interface';
import {fetchDataSourceList, deleteDatasource, updateDatasource, getConnectorByIDs} from '@/service/api'
import { formatESSearchResult } from '@/service/request/es';
import { GoogleDriveSVG, HugoSVG, YuqueSVG,NotionSVG } from '@/components/icons';
import { connect } from 'http2';

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

  const onEnabledChange = (value: boolean, record: Datasource)=>{
    record.enabled = value;
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
        if(!data.connectors) return value;
        const iconSrc = data.connectors[record.connector.id]?.icon;
        if (!iconSrc) return value;
        return (
          <a className='text-blue-500 inline-flex items-center gap-1' onClick={()=>nav(`/data-source/detail/${record.id}`, {state:{datasource_name: record.name, connector_id: record.connector?.id || ''}})}>
            <Image preview={false} height="1em" width="1em" src={iconSrc}/>
            { value }
          </a>
        )
      }
    },
    {
      title: t('page.datasource.columns.type'),
      minWidth: 100,
      render: (text: string, record: Datasource)=>{
        const type = TYPES[record?.connector?.id]
        if (!type) return data.connectors[record.connector.id]?.name || record.connector.id;
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
    {
      dataIndex: 'enabled',
      title: t('page.datasource.new.labels.enabled'),
      width: 200,
      render: (value: boolean, record: Datasource)=>{
       return <Switch value={value} onChange={(v)=>onEnabledChange(v, record)}/>
      }
    },
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

const initialData = {
  data: [],
  total: 0,
  connectors: {},
}
const [data, setData] = useState(initialData);
const [loading, setLoading] = useState(false);

const [reqParams, setReqParams] = useState({
  query: '',
  from: 0, 
  size: 10,
})
const fetchData = () => {
  setLoading(true);
  fetchDataSourceList(reqParams).then(({ data }) => {
    const newData = formatESSearchResult(data);
      setData((oldData: any) => {
        return {
          ...oldData,
          ...(newData || initialData),
        }
      });
      setLoading(false);
    });
  };

  useEffect(fetchData, [
    reqParams
  ]);

  const fetchConnectors = async (ids: string[])=>{
    const res = await getConnectorByIDs(ids);
    if(res.data){
      const newData = formatESSearchResult(res.data);
      const connectors: any = {};
      newData.data.map((item)=>{
        connectors[item.id] = item;
      });
      setData(data =>{
        return {
          ...data,
          connectors: connectors,
        }
      })
    }
  }
  useEffect(()=>{
    if(data.data?.length > 0){
      const ids = data.data.map((item)=>item.connector.id);
      fetchConnectors(ids);
    }
  }, [data.data])

  const handleTableChange: TableProps<Datasource>['onChange'] = (pagination, filters, sorter) => {
    setReqParams((params)=>{
      return {
        ...params,
        size: pagination.pageSize,
        from: (pagination.current-1) * pagination.pageSize,
      }
    })
  };
  const onRefreshClick = (query: string)=>{
    setReqParams((oldParams)=>{
      return {
        ...oldParams,
        query: query,
        from: 0,
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
          dataSource={data.data}
          pagination={
            {
              showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
              defaultPageSize: 10,
              defaultCurrent: 1,
              total: data.total?.value || data?.total,
              showSizeChanger: true,
            }
          }
          onChange={handleTableChange}
        />
      </ACard>
    </div>
  );
}
