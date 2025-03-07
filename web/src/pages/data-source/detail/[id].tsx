import { type LoaderFunctionArgs, useLoaderData } from "react-router-dom";
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import { Switch, Table, Dropdown, message, Modal} from "antd";
import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, DownOutlined, ExclamationCircleOutlined, EllipsisOutlined } from "@ant-design/icons";
import {fetchDatasourceDetail, deleteDocument, updateDocument, batchDeleteDocument} from '@/service/api';
import { formatESSearchResult } from '@/service/request/es';
import {ConnectorImageIcon} from '@/components/icons/connector';
const { confirm } = Modal;

interface DataType {
  id: string;
  title: string;
  searchable: boolean;
  url: string;
  tags: string[];
  category: string;
  subcategory: string;
  icon: string;
  is_dir: boolean;
  type: string;
  disabled: boolean;
}

export function Component() {
  const datasourceID = useLoaderData();

  const { t } = useTranslation();
  const nav = useNavigate();
  const location = useLocation();
  const { datasource_name, connector_id} = location.state || {}; 
  const onMenuClick = ({key, record}: any)=>{
    switch(key){
      case "1":
        confirm({
          icon: <ExclamationCircleOutlined />,
          content: 'Are you sure you want to delete this document?',
          onOk() {
            deleteDocument(record.id).then((res)=>{
              if(res.data?.result === "deleted"){
                message.success("deleted success")
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
    }
  }
const [state, setState] = useState({
  selectedRowKeys: [],
});
// rowSelection object indicates the need for row selection
const rowSelection: TableProps<DataType>["rowSelection"] = {
  selectedRowKeys: state.selectedRowKeys,
  onChange: (selectedRowKeys: React.Key[], selectedRows: DataType[]) => {
    setState((st: any)=>{
      return {
        ...st,
        selectedRowKeys: selectedRowKeys,
      }
    })
  },
  getCheckboxProps: (record: DataType) => ({// Column configuration not to be checked
    name: record.title,
  }),
};
  const onSearchableChange = (checked: boolean, record: DataType)=>{
    //update searchable status
    record.disabled = !checked
    updateDocument(record.id, record).then((res)=>{
      if(res.data?.result === "updated"){
        message.success("updated success")
      }
      //reload data
      setReqParams((old)=>{
        return {
          ...old,
        }
      })
    });
  }
  const items: MenuProps["items"] = [
    {
      label: t('common.delete'),
      key: "1",
    },
  ];
  const onBatchMenuClick = useCallback(({key}: any)=>{
    switch(key){
      case "1":
        confirm({
          icon: <ExclamationCircleOutlined />,
          content: 'Are you sure you want to delete theses documents?',
          onOk() {
            if (state.selectedRowKeys?.length === 0) {
              message.error("Please select at least one document")
              return;
            }
            setLoading(true);
            batchDeleteDocument(state.selectedRowKeys).then((res)=>{
              if(res.data?.result === "acknowledged"){
                setState((st: any)=>{
                  return {
                    ...st,
                    selectedRowKeys: [],
                  }
                })
                message.success("submit success")
              }
              //reload data
              setTimeout(()=>{
                setReqParams((old)=>{
                  return {
                    ...old,
                  }
                })
              }, 1000);
            }).finally(()=>{
              setLoading(false);
            });
          },
          onCancel() {
          },
        });
       
        break;
    }
  }, [state.selectedRowKeys]);

  const columns: TableColumnsType<DataType> = useMemo(()=>[
    {
      title: t('page.datasource.columns.name'),
      dataIndex: "title",
      render: (text: string, record: DataType) =>{
        return <span> <Icon component={() => <ConnectorImageIcon connector={connector_id} doc_type={record.icon}/>} className="mr-3px" /><a target="_blank" href={record.url} className="text-blue-500">{text}</a></span>
      },
    },
    {
      title: t('page.datasource.columns.searchable'),
      dataIndex: "disabled",
      render: (text: boolean, record: DataType) => {
        return <Switch value={!text} onChange={(v)=>{
          onSearchableChange(v, record)
        }}/>;
      },
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
  ], [connector_id, t]);

  if (!datasourceID) return <LookForward />;

  const [reqParams, setReqParams] = useState({
    from: 0,
    size: 20,
    datasource: datasourceID,
  })
  const [data, setData] = useState({});
  const [loading, setLoading] = useState(false);

  const fetchData = ()=>{
    setLoading(true)
    fetchDatasourceDetail(reqParams).then((data)=>{
      const newData = formatESSearchResult(data.data);
      setData(newData);
    }).finally(()=>{
      setLoading(false);
    });
  }

  useEffect(fetchData, [reqParams]);

  const onTableChange = (pagination, filters, sorter, extra: { currentDataSource: [], action })=>{
    setReqParams((params)=>{
      return {
        ...params,
        size: pagination.pageSize,
        from: (pagination.current-1) * pagination.pageSize,
      }
    })
  }
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
    <div className="bg-white pt-15px pb-15px">
      <div>
        <div className="mb-4 flex items-center text-lg font-bold">
          <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
          <div>{datasource_name}</div>
        </div>
      </div>
      <div className="p-5 pt-2">
        <div className="mb-4 mt-4 flex items-center justify-between">
          <Search
            addonBefore={<FilterOutlined />}
            className="max-w-500px"
            enterButton= {t('common.refresh')}
            onSearch={onRefreshClick}
          />
          <div>
            <Dropdown.Button
              icon={<DownOutlined />}
              menu={{ items, onClick: onBatchMenuClick }}
              type="primary"
            >
              {t('common.operation')}
            </Dropdown.Button>
          </div>
        </div>
        <Table<DataType>
          size="middle"
          rowKey="id"
          rowSelection={{ ...rowSelection }}
          columns={columns}
          loading={loading}
          onChange={onTableChange}
          pagination={
            {
              showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
              defaultPageSize:20,
              defaultCurrent: 1,
              total: data.total?.value || data?.total,
              showSizeChanger: true,
            }
          }
          dataSource={data.data || []}
        />
      </div>
    </div>
  );
}

export async function loader({ params, ...rest }: LoaderFunctionArgs) {
  const datasourceID = params.id;
  //todo fetch datasource info by id
  return datasourceID;
}
