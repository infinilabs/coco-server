import Search from 'antd/es/input/Search';
import Icon, { FilterOutlined, PlusOutlined, EllipsisOutlined, ExclamationCircleOutlined} from '@ant-design/icons';
import { Button, Dropdown, Table, GetProp, message,Modal, Switch, Image } from 'antd';
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import {searchMCPServer, deleteMCPServer, updateMCPServer} from '@/service/api/mcp-server';
import { formatESSearchResult } from '@/service/request/es';
import InfiniIcon from '@/components/common/icon';
import useQueryParams from '@/hooks/common/search';

type MCPServer = Api.LLM.MCPServer;

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();
  
  const { t } = useTranslation();

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();
  const items: MenuProps["items"] = [
    {
      label: t('common.edit'),
      key: "2",
    },
    {
      label: t('common.delete'),
      key: "1",
    },
  ];

  const onMenuClick = ({key, record}: any)=>{
    switch(key){
      case "1":
        window?.$modal?.confirm({
          icon: <ExclamationCircleOutlined />,
          title: t('common.tip'),
          content: t('page.mcpserver.delete.confirm', { name: record.name }),
          onOk() {
            deleteMCPServer(record.id).then((res)=>{
              if(res.data?.result === "deleted"){
                message.success(t('common.deleteSuccess'))
              }
              //reload data
              setQueryParams((old)=>{
                return {
                  ...old,
                  t: new Date().valueOf()
                }
              })
            });
          },
          onCancel() {
          },
        });
       
        break;
      case "2":
        nav(`/mcp-server/edit/${record.id}`, {state:record});
        break;
    }
  }

  const onEnabledChange = (value: boolean, record: MCPServer)=>{
    setLoading(true);
    updateMCPServer(record.id, {
      ...record,
      enabled: value
    }).then((res)=>{
      if(res.data?.result === "updated"){
        message.success(t('common.updateSuccess'))
      }
      //reload data
      setQueryParams((old)=>{
        return {
          ...old,
          t: new Date().valueOf()
        }
      })
    }).finally(()=>{
      setLoading(false);
    });
  }
  const columns: TableColumnsType<MCPServer> = [
    {
      title: t('page.mcpserver.labels.name'),
      dataIndex: "name",
      minWidth: 150,
      ellipsis: true,
      render: (value: string, record: MCPServer)=>{
        return (
          <div className='flex items-center gap-1'>
            <IconWrapper className="w-20px h-20px">
              <InfiniIcon height="1em" width="1em" src={record.icon} />
            </IconWrapper>
            <span className='max-w-150px ant-table-cell-ellipsis'>{ value }</span>
          </div>
        )
      }
    },
    {
      title: t('page.mcpserver.labels.type'),
      minWidth: 50,
      dataIndex: "type",
    },
    {
      title: t('page.mcpserver.labels.category'),
      minWidth: 50,
      dataIndex: "category",
      ellipsis: true,
    },
    {
      title: t('page.mcpserver.labels.description'),
      minWidth: 150,
      dataIndex: "description",
      ellipsis: true,
    },
    {
      dataIndex: 'enabled',
      title: t('page.mcpserver.labels.enabled'),
      width: 80,
      render: (value: boolean, record: MCPServer)=>{
       return <Switch size="small" value={value} onChange={(v)=>onEnabledChange(v, record)}/>
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
const rowSelection: TableProps<MCPServer>["rowSelection"] = {
  onChange: (selectedRowKeys: React.Key[], selectedRows: MCPServer[]) => {
  },
  getCheckboxProps: (record: MCPServer) => ({
    name: record.id,
  }),
};

const initialData = {
  data: [],
  total: 0,
}
const [data, setData] = useState(initialData);
const [loading, setLoading] = useState(false);
const [keyword, setKeyword] = useState();

const fetchData = () => {
  setLoading(true);
  searchMCPServer(queryParams).then(({ data }) => {
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

  useEffect(fetchData, []);

  useEffect(() => {
    setKeyword(queryParams.query)
  }, [queryParams.query])

  const handleTableChange: TableProps<MCPServer>['onChange'] = (pagination, filters, sorter) => {
    setQueryParams((params)=>{
      return {
        ...params,
        size: pagination.pageSize,
        from: (pagination.current-1) * pagination.pageSize,
      }
    })
  };
  const onRefreshClick = (query: string)=>{
    setQueryParams((oldParams)=>{
      return {
        ...oldParams,
        query: query,
        from: 0,
        t: new Date().valueOf()
      }
    })
  }

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
      >
      <div className='mb-4 mt-4 flex items-center justify-between'>
        <Search value={keyword} onChange={(e) => setKeyword(e.target.value)} addonBefore={<FilterOutlined />} className='max-w-500px' onSearch={onRefreshClick} enterButton={ t('common.refresh')}></Search>
        <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/mcp-server/new`)}>{t('common.add')}</Button>
      </div>
      <Table<MCPServer>
          rowKey="id"
          loading={loading}
          size="middle"
          rowSelection={{ ...rowSelection }}
          columns={columns}
          dataSource={data.data}
          pagination={
            {
              showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
              pageSize: queryParams.size,
              current: queryParams.from + 1,
              total: data.total?.value || data?.total,
              showSizeChanger: true,
            }
          }
          onChange={handleTableChange}
        />
      </ACard>
    </ListContainer>
  );
}
