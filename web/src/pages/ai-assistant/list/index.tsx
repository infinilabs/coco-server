import Search from 'antd/es/input/Search';
import { FilterOutlined, PlusOutlined, EllipsisOutlined, ExclamationCircleOutlined} from '@ant-design/icons';
import { Button, Dropdown, Table, message, Switch, Avatar } from 'antd';
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import {searchAssistant, deleteAssistant, updateAssistant, cloneAssistant} from '@/service/api/assistant';
import { formatESSearchResult } from '@/service/request/es';
import InfiniIcon from '@/components/common/icon';
import useQueryParams from '@/hooks/common/queryParams';

type Assistant = Api.LLM.Assistant;

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();

  const { t } = useTranslation();

  const { addSharesToData, isEditorOwner, hasEdit, isResourceShare } = useResource()
  const resourceType = 'assistant'

  const { hasAuth } = useAuth()

  const permissions = {
    read: hasAuth('coco#assistant/read'),
    create: hasAuth('coco#assistant/create'),
    update: hasAuth('coco#assistant/update'),
    delete: hasAuth('coco#assistant/delete'),
  }

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

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
        nav(`/ai-assistant/edit/${record.id}`);
        break;
      case "3":
        cloneAssistant(record.id).then((res)=>{
          if(res.data?.result === "created"){
            if (permissions.update && hasEdit(record)) {
              nav(`/ai-assistant/edit/${res.data?._id}`);
            } else {
              nav(`/ai-assistant/list`);
            }
          }else{
            message.error(res.data?.error?.reason);
          }
        });
        break;
    }
  }

  const onEnabledChange = (value: boolean, record: Assistant)=>{
    setLoading(true);
    updateAssistant(record.id, {
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
  const columns: TableColumnsType<Assistant> = [
    {
      title: t('page.assistant.labels.name'),
      dataIndex: "name",
      minWidth: 150,
      ellipsis: true,
      render: (value: string, record: Assistant)=>{
        const isShare = isResourceShare(record)

        let shareIcon;

        if (isShare) {
          shareIcon = (
            <div className='flex-grow-0 flex-shrink-0'>
              <SvgIcon localIcon='share' className='text-#999'/>
            </div>
          )
        }

        return (
          <div className='flex items-center gap-1'>
            {
              record.icon && (
                <IconWrapper className="flex-grow-0 flex-shrink-0 flex-basis-auto w-20px h-20px">
                  <InfiniIcon height="1em" width="1em" src={record.icon} />
                </IconWrapper>
              )
            }
            { permissions.read && permissions.update && hasEdit(record) ? (
              <a className='max-w-150px ant-table-cell-ellipsis cursor-pointer text-[var(--ant-color-link)]' onClick={()=>nav(`/ai-assistant/edit/${record.id}`)}>{ value }</a>
            ) : (
              <span className='max-w-150px ant-table-cell-ellipsis'>{ value }</span>
            )}
            {record.builtin === true && <div className="flex items-center ml-5px">
              <p className="h-[22px] bg-[#eee] text-[#999] font-size-[12px] px-[10px] line-height-[22px] rounded-[4px]">{t('page.modelprovider.labels.builtin')}</p>
            </div>}
            {shareIcon}
          </div>
        )
      }
    },
    {
      dataIndex: 'owner',
      title: t('page.datasource.labels.owner'),
      render: (value, record) => {
        if (!value) return '-'
        return (
          <div className='flex'>
            <Avatar.Group max={{ count: 1 }} size={"small"}>
              <AvatarLabel data={value} showCard={true}/>
            </Avatar.Group>
          </div>
        )
      }
    },
    {
      dataIndex: 'shares',
      title: t('page.datasource.labels.shares'),
      render: (value, record) => {
        if (!value) return '-'
        return (
          <Shares 
            record={record} 
            title={record.name} 
            onSuccess={() => fetchData(queryParams)}
            resource={{
              'resource_type': resourceType,
              'resource_id': record.id,
            }}
          />
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
       return <Switch size="small" value={value} onChange={(v)=>onEnabledChange(v, record)} disabled={!permissions.update || !hasEdit(record)}/>
      }
    },
    {
      title: t('common.operation'),
      fixed: 'right',
      width: "90px",
      hidden: !permissions.update && !permissions.delete,
      render: (_, record) => {
        const items: MenuProps["items"] = [];
        if (permissions.read && permissions.update && hasEdit(record)) {
          items.push({
            label: t('common.edit'),
            key: "2",
          });
        }
        if (permissions.delete && record.builtin !== true && isEditorOwner(record)) {
          items.push({
            label: t('common.delete'),
            key: "1",
          });
        }
        if (permissions.create && hasEdit(record)) {
          items.push({
            label: t('common.clone'),
            key: "3",
          })
        }
        if (items.length === 0) return null;
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

  const [keyword, setKeyword] = useState();

  const fetchData = async (queryParams) => {
    setLoading(true);
    const res = await searchAssistant(queryParams)
    if (res?.data) {
      const newData = formatESSearchResult(res?.data);
      if (newData.data.length > 0) {
        const resources = newData.data.map((item) => ({
          "resource_id": item.id,
          "resource_type": resourceType,
        }))
        const dataWithShares = await addSharesToData(newData.data, resources)
        if (dataWithShares) {
          newData.data = dataWithShares
        }
      }
      setData((oldData: any) => {
        return {
          ...oldData,
          ...(newData || initialData),
        }
      });
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData(queryParams)
  }, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query)
  }, [queryParams.query])

  const handleTableChange: TableProps<Assistant>['onChange'] = (pagination, filters, sorter) => {
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
        { permissions.create && <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/ai-assistant/new`)}>{t('common.add')}</Button> }
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
              pageSize: queryParams.size,
              current: Math.floor(queryParams.from / queryParams.size) + 1,
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
