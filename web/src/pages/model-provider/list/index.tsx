import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, PlusOutlined, SettingOutlined, ExclamationCircleOutlined, ExportOutlined, EllipsisOutlined } from "@ant-design/icons";
import { Button, List, Image, Switch, Tag, message, MenuProps, Modal, Dropdown, Spin, Form, Input, Table, Typography, Avatar} from "antd";
import {searchModelPovider, updateModelProvider, deleteModelProvider} from "@/service/api/model-provider";
import { formatESSearchResult } from '@/service/request/es';
import InfiniIcon from '@/components/common/icon';
import useQueryParams from "@/hooks/common/queryParams";

export function Component() {
  const type = 'table'

  const [queryParams, setQueryParams] = useQueryParams({
    size: type === 'table' ? 10 : 12,
    sort: [['enabled', 'desc'], ['created', 'desc']]
  });

  const { addSharesToData, isEditorOwner, hasEdit, isResourceShare } = useResource()
  const resourceType = 'llm-provider'

  const { t } = useTranslation();
  const nav = useNavigate();
  const [data, setData] = useState({
    total: 0,
    data: [],
  });
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState();

  const { hasAuth } = useAuth()

  const permissions = {
    read: hasAuth('coco#model_provider/read'),
    create: hasAuth('coco#model_provider/create'),
    update: hasAuth('coco#model_provider/update'),
    delete: hasAuth('coco#model_provider/delete'),
  }

  const fetchData = async (queryParams) => {
    setLoading(true);
    const res = await searchModelPovider(queryParams)
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
      setData(newData);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData(queryParams)
  }, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query)
  }, [queryParams.query])
  
  const onSearchClick = (query: string)=>{
    setQueryParams({
      ...queryParams,
      query,
      t: new Date().valueOf()
    })
  }

  const handleTableChange = pagination => {
    setQueryParams(params => {
      return {
        ...params,
        from: (pagination.current - 1) * pagination.pageSize,
        size: pagination.pageSize
      };
    });
  };

  const onPageChange = (page: number, pageSize: number) =>{
    setQueryParams((oldParams: any)=>{
      return {
        ...oldParams,
        from: (page-1) * pageSize,
        size: pageSize,
      }
    })
  }
  const getMenuItems = (record: any) => {
    const items = [];
    if (permissions.read && permissions.update && hasEdit(record)) {
      items.push({
        label: t('common.edit'),
        key: "1",
      });
    }
    if (permissions.update && hasEdit(record)) {
      items.push({
        label: 'API-key',
        key: "3"
      })
    }
    if(permissions.delete && record.builtin !== true && isEditorOwner(record)) {
      items.push({
        label: t('common.delete'),
        key: "2",
      });
    }
    return items;
  };
  
  const onMenuClick = ({key, record}: any)=>{
    switch(key){
      case "2":
        window?.$modal?.confirm({
          icon: <ExclamationCircleOutlined />,
          title: t('common.tip'),
          content: t('page.modelprovider.delete.confirm'),
          onOk() {
             deleteModelProvider(record.id).then((res)=>{
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
        });
       
        break;
      case "1":
        nav(`/model-provider/edit/${record.id}`);
        break;
      case "3":
        onAPIKeyClick(record)
        break;
    }
  }
  const onItemEnableChange = (record: any, checked: boolean)=>{
    setLoading(true);
    updateModelProvider(record.id, {
      ...record,
      enabled: checked,
    }).then((res)=>{
      if(res.data?.result === "updated"){
        // update local data
        setData((oldData: any)=>{
          const newData = oldData.data.map((item: any)=>{
            if(item.id === record.id){
              return {
                ...item,
                enabled: checked,
              }
            }
            return item;
          });
          return {
            ...oldData,
            data: newData,
          }
        })
        message.success(t('common.updateSuccess'))
      }
    }).finally(()=>{
      setLoading(false);
    })
  }

  const [editValue, setEditValue] = useState({});
  const [open, setOpen] = useState(false);
  const onOkClick = ()=>{
    setOpen(false);
    fetchData(queryParams);
  }
  const onCancelClick = ()=>{
    setOpen(false);
  }

  const onAPIKeyClick = (record: any)=>{
    setEditValue(record);
    setOpen(true);
  }

  const columns = [
    {
      dataIndex: 'name',
      minWidth: 150,
      ellipsis: true,
      render: (value, record) => {

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
              <a className='max-w-150px ant-table-cell-ellipsis cursor-pointer text-[var(--ant-color-link)]' onClick={()=>nav(`/model-provider/edit/${record.id}`)}>{ value }</a>
            ) : (
              <span className='max-w-150px ant-table-cell-ellipsis'>{ value }</span>
            )}
            {record.builtin === true && <div className="flex items-center ml-5px">
              <p className="h-[22px] bg-[#eee] text-[#999] font-size-[12px] px-[10px] line-height-[22px] rounded-[4px]">{t('page.modelprovider.labels.builtin')}</p>
            </div>}
            {shareIcon}
          </div>
        )
      },
      title: t('page.integration.columns.name')
    },
    {
      dataIndex: 'owner',
      title: t('page.datasource.labels.owner'),
      width: 200,
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
      width: 150,
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
      title: t('page.assistant.labels.description'),
      minWidth: 200,
      dataIndex: "description",
      render: (value, record)=>{
        return <span title={value}>{value}</span>;
      },
      ellipsis: true,
    },
    {
      dataIndex: 'enabled',
      title: t('page.assistant.labels.enabled'),
      width: 80,
      render: (value, record)=>{
        return <Switch size="small" value={value} onChange={(v)=> onItemEnableChange(record, v)} disabled={!permissions.update || !hasEdit(record)}/>
      }
    },
    {
      title: t('common.operation'),
      fixed: 'right',
      width: "90px",
      hidden: !permissions.update && !permissions.delete,
      render: (_, record) => {
        const items = getMenuItems(record);
        if (items.length === 0) return null;
        return <Dropdown menu={{ items, onClick:({key})=>onMenuClick({key, record}) }}>
          <EllipsisOutlined/>
        </Dropdown>
      },
    },
  ];
  // rowSelection object indicates the need for row selection
  const rowSelection = {
    getCheckboxProps: record => ({
      name: record.name
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows) => {}
  };

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-4 mt-4 flex items-center justify-between">
          <Search
            addonBefore={<FilterOutlined />}
            className="max-w-500px"
            onSearch={onSearchClick}
            enterButton={t("common.refresh")}
            value={keyword} 
            onChange={(e) => setKeyword(e.target.value)} 
          ></Search>
          { permissions.create && <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/model-provider/new`)}>{t('common.add')}</Button> }
        </div>
        {
          type === 'table' ? (
            <Table
              columns={columns}
              dataSource={data.data}
              loading={loading}
              rowKey="id"
              rowSelection={{ ...rowSelection }}
              size="middle"
              pagination={{
                showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
                pageSize: queryParams.size,
                current: Math.floor(queryParams.from / queryParams.size) + 1,
                total: data.total?.value || data?.total,
                showSizeChanger: true,
              }}
              onChange={handleTableChange}
            />
          ) : (
            <List
              loading={loading}
              pagination={{
                onChange: onPageChange,
                showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
                pageSize: queryParams.size,
                current: Math.floor(queryParams.from / queryParams.size) + 1,
                total: data.total?.value || data?.total,
                showSizeChanger: true,
                pageSizeOptions: [12, 24, 48, 96]
              }}
              grid={{ gutter: 16, column: 3,  xs: 1,
                sm: 2,
              }}
              dataSource={data.data}
              renderItem={(provider) => {
                const operations = getMenuItems(provider)
                return (
                  <List.Item>
                    <div className="p-1em min-h-[132px] border border-[var(--ant-color-border)] group rounded-[8px] hover:bg-[var(--ant-control-item-bg-hover)]">
                      <div className="flex justify-between">
                          <div className="flex gap-15px">
                            <div className="flex items-center gap-8px">
                              <IconWrapper className="w-40px h-40px">
                                <InfiniIcon src={provider.icon} height="2em" width="2em" className="font-size-2em"/>
                              </IconWrapper>
                              { permissions.read && permissions.update && hasEdit(provider) ? (
                                <a className='font-size-1.2em cursor-pointer text-[var(--ant-color-link)]' onClick={()=>nav(`/model-provider/edit/${provider.id}`)}>{ provider.name }</a>
                              ) : (
                                <span className='font-size-1.2em'>{ provider.name }</span>
                              )}
                            </div>
                            {provider.builtin === true && <div className="flex items-center">
                              <p className="h-[22px] bg-[#eee] text-[#999] font-size-[12px] px-[10px] line-height-[22px] rounded-[4px]">{t('page.modelprovider.labels.builtin')}</p>
                            </div>}
                          </div>
                          <div>
                            <Switch checked={provider.enabled} onChange={(v)=>onItemEnableChange(provider, v)} size="small" disabled={!permissions.update || !hasEdit(provider)}/>
                          </div>
                        </div>
                        <div className="text-[#999] h-[51px] line-clamp-3 text-xs my-[10px]">
                          {provider.description}
                        </div>
                        <div className="flex gap-1">

                          <div className="ml-auto flex gap-2">
                            { permissions.update && hasEdit(provider) && <div onClick={()=>{onAPIKeyClick(provider)}} className="border border-[var(--ant-color-border)] cursor-pointer  px-10px rounded-[8px]">API-key</div> }
                            {
                              operations?.length > 0 && (
                                <div className="inline-block px-4px rounded-[8px] border border-[var(--ant-color-border)] cursor-pointer">
                                  <Dropdown menu={{ items: operations, onClick:({key})=>onMenuClick({key, record: provider}) }}>
                                      <SettingOutlined className="text-blue-500"/>
                                  </Dropdown>
                                </div>
                              )
                            }
                          </div>
                        </div>
                      </div>
                  </List.Item>
                )
              }}
            />
          )
        }
        <APIKeyComponent
         open={open}
         onOkClick={onOkClick}
         onCancelClick={onCancelClick}
         record={editValue} />
      </ACard>
    </ListContainer>
  );
}

const APIKeyComponent = ({
  record = {},
  onOkClick = ()=>{},
  open = false,
  onCancelClick = ()=>{},
})=>{
  const { t } = useTranslation();
  const [form] = Form.useForm();
  useEffect(()=>{
    form.setFieldsValue({
      api_key: record.api_key,
    });
  }, [record])
  const [loading, setLoading] = useState(false);
  let apiHref = "";
  switch(record.id){
    case "qianwen":
      apiHref = "https://bailian.console.aliyun.com/?tab=model#/api-key";
      break;
    case "deepseek":
      apiHref = "https://platform.deepseek.com/api_keys";
      break;
    case "gitee_ai":
      apiHref = "https://ai.gitee.com/dashboard/settings/tokens";
      break;
    case "openai":
      apiHref = "https://platform.openai.com/account/api-keys";
      break;
    case "silicon_flow":
      apiHref = "https://cloud.siliconflow.cn/account/ak";
      break;
    case "tencent_hunyuan":
      apiHref = "https://console.cloud.tencent.com/hunyuan/api-key";
      break;
    case "gemini":
      apiHref = "https://aistudio.google.com/app/apikey";
      break;
    case "moonshot":
      apiHref = "https://platform.moonshot.cn/console/api-keys";
      break;
    case "minimax":
      apiHref = "https://platform.minimaxi.com/user-center/basic-information/interface-key";
      break;
    case "volcanoArk":
      apiHref = "https://console.volcengine.com/iam/keymanage/";
      break;
    case "qianfan":
      apiHref = "https://console.bce.baidu.com/iam/#/iam/apikey/list";
      break;
    case "cohere":
      apiHref = "https://dashboard.cohere.com/api-keys";
      break;
  }

  const onModalOkClick = ()=>{
    form.validateFields().then((values)=>{
      setLoading(true);
      record.api_key = values.api_key;
      updateModelProvider(record.id, record).then(()=>{
        setLoading(false);
        onOkClick();
      }).catch(()=>{
        setLoading(false);
      });
    })
  }
  return (<Modal title={t('common.update')+t('page.modelprovider.labels.api_key')}
  open={open} 
  onOk={onModalOkClick} 
  onCancel={onCancelClick}>
  <Spin spinning={loading}>
    <Form form={form} layout="vertical" className="my-2em">
      <Form.Item label={<span className="text-gray-500">{t('page.modelprovider.labels.api_key')}</span>} name="api_key">
        <Input defaultValue={record.api_key}/>
      </Form.Item>
    {apiHref && <div><Button className="m-0 p-0" href={apiHref} target="_blank" type="link">{t('page.modelprovider.labels.api_key_source', {
        model_provider: record.name,
      })}
      <ExportOutlined/></Button></div>}
    </Form>
  </Spin>
</Modal>)
}