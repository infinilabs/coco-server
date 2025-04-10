import Search from 'antd/es/input/Search';
import Icon, { FilterOutlined, PlusOutlined, EllipsisOutlined, ExclamationCircleOutlined} from '@ant-design/icons';
import { Button, Dropdown, Table, GetProp, message,Modal, Switch, Image } from 'antd';
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import {searchAssistant, deleteAssistant, updateAssistant} from '@/service/api/assistant';
import { formatESSearchResult } from '@/service/request/es';
import InfiniIcon from '@/components/common/icon';

const { confirm } = Modal;
type Assistant = Api.LLM.Assistant;
// const mockData = {
//   total: 1,
//   data: [{
//     name: "searchkit 小助手",
//     type: "自定义助手",
//     icon: "/assets/icons/llm/assistant.svg",
//     description: "这是一个自定义助手",
//     answering_model: "deepseek-v1",
//   }],
// };


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
          content: t('page.assistant.delete.confirm', { name: record.name }),
          onOk() {
            deleteAssistant(record.id).then((res)=>{
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
        nav(`/ai-assistant/edit/${record.id}`, {state:record});
        break;
    }
  }

  const onEnabledChange = (value: boolean, record: Assistant)=>{
    record.enabled = value;
    setLoading(true);
    updateAssistant(record.id, record).then((res)=>{
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
  const columns: TableColumnsType<Assistant> = [
    {
      title: t('page.assistant.labels.name'),
      dataIndex: "name",
      minWidth: 150,
      ellipsis: true,
      render: (value: string, record: Assistant)=>{
        if(!data.connectors) return value;
        const iconSrc = data.connectors[record.connector.id]?.icon;
        if (!iconSrc) return value;
        return (
          <a className='text-blue-500 inline-flex items-center gap-1' onClick={()=>nav(`/data-source/detail/${record.id}`, {state:{datasource_name: record.name, connector_id: record.connector?.id || ''}})}>
            <InfiniIcon height="1em" width="1em" src={iconSrc}/>
            { value }
          </a>
        )
      }
    },
    {
      title: t('page.assistant.labels.type'),
      minWidth: 50,
      dataIndex: "type",
    },
    {
      title: t('page.assistant.labels.datasource'),
      minWidth: 50,
      dataIndex: ["datasource", "enabled"],
      render: (value: boolean, record: Assistant)=>{
        return t('common.enableOrDisable.'+ (value ? 'enable' : 'disable'));
      }
    },
    {
      title: t('page.assistant.labels.mcp_servers'),
      minWidth: 80,
      dataIndex: ["mcp_servers", "enabled"],
      render: (value: boolean, record: Assistant)=>{
        return t('common.enableOrDisable.'+ (value ? 'enable' : 'disable'));
      }
    },
    {
      title: t('page.assistant.labels.description'),
      minWidth: 150,
      dataIndex: "description",
      ellipsis: true,
    },
    {
      dataIndex: 'enabled',
      title: t('page.assistant.labels.enabled'),
      width: 80,
      render: (value: boolean, record: Assistant)=>{
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
const rowSelection: TableProps<Assistant>["rowSelection"] = {
  onChange: (selectedRowKeys: React.Key[], selectedRows: Assistant[]) => {
  },
  getCheckboxProps: (record: Assistant) => ({
    name: record.name,
  }),
};

const initialData = {
  data: [],
  total: 0,
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
  searchAssistant(reqParams).then(({ data }) => {
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

  const handleTableChange: TableProps<Assistant>['onChange'] = (pagination, filters, sorter) => {
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
        <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/ai-assistant/new`)}>{t('common.add')}</Button>
      </div>
      <Table<Assistant>
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
