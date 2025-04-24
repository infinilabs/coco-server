import Search from 'antd/es/input/Search';
import Icon, { FilterOutlined, PlusOutlined, EllipsisOutlined, ExclamationCircleOutlined} from '@ant-design/icons';
import { Button, Dropdown, Table, GetProp, message,Modal, Switch, Image } from 'antd';
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import {searchAssistant, deleteAssistant, updateAssistant, cloneAssistant} from '@/service/api/assistant';
import { formatESSearchResult } from '@/service/request/es';
import InfiniIcon from '@/components/common/icon';

type Assistant = Api.LLM.Assistant;

export function Component() {
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
    {
      label: t('common.clone'),
      key: "3",
    },
  ];

  const onMenuClick = ({key, record}: any)=>{
    switch(key){
      case "1":
        window?.$modal?.confirm({
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
        nav(`/ai-assistant/edit/${record.id}`);
        break;
      case "3":
        cloneAssistant(record.id).then((res)=>{
          if(res.data?.result === "created"){
            nav(`/ai-assistant/edit/${res.data?._id}`);
          }else{
            message.error(res.data?.error?.reason);
          }
        });
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
      width: 300,
      render: (value: string, record: Assistant)=>{
        return (
          <div className='flex items-center gap-1'>
            <IconWrapper className="w-20px h-20px">
              <InfiniIcon height="1em" width="1em" src={record.icon} />
            </IconWrapper>
            <span className='max-w-150px ant-table-cell-ellipsis'>{ value }</span>
            {record.builtin === true && <div className="flex items-center ml-[5px]">
              <p className="h-[22px] bg-[#eee] text-[#999] font-size-[12px] px-[10px] line-height-[22px] rounded-[4px]">{t('page.modelprovider.labels.builtin')}</p>
            </div>}
          </div>
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
      minWidth: 50,
      dataIndex: ["mcp_servers", "enabled"],
      render: (value: boolean, record: Assistant)=>{
        return t('common.enableOrDisable.'+ (value ? 'enable' : 'disable'));
      }
    },
    {
      title: t('page.assistant.labels.description'),
      minWidth: 200,
      dataIndex: "description",
      render: (value: string, record: Assistant)=>{
        return <span title={value}>{value}</span>;
      },
      ellipsis: true,
    },
    {
      dataIndex: 'enabled',
      title: t('page.assistant.labels.enabled'),
      width: 80,
      render: (value: boolean, record: Assistant)=>{
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
