import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, PlusOutlined, SettingOutlined, ExclamationCircleOutlined, ExportOutlined } from "@ant-design/icons";
import { Button, List, Image, Switch, Tag, message, MenuProps, Modal, Dropdown, Spin, Form, Input} from "antd";
import {searchModelPovider, updateModelProvider, deleteModelProvider} from "@/service/api/model-provider";
import { formatESSearchResult } from '@/service/request/es';
import InfiniIcon from '@/components/common/icon';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [data, setData] = useState({
    total: 0,
    data: [],
  });
  const [loading, setLoading] = useState(false);
  const [reqParams, setReqParams] = useState({
    from: 0,
    size: 12,
  })
  const fetchData = ()=>{
    setLoading(true)
    searchModelPovider(reqParams).then((data)=>{
      const newData = formatESSearchResult(data.data);
      setData(newData);
    }).finally(()=>{
      setLoading(false);
    });
  }
  useEffect(fetchData, [reqParams]);
  const onSearchClick = (query: string)=>{
    setReqParams({
      ...reqParams,
      query,
    })
  }
  const onPageChange = (page: number, pageSize: number) =>{
    setReqParams((oldParams: any)=>{
      return {
        ...oldParams,
        from: (page-1) * pageSize,
        size: pageSize,
      }
    })
  }
  const items: MenuProps["items"] = [
    {
      label: t('common.edit'),
      key: "1",
    },
    {
      label: t('common.delete'),
      key: "2",
    },
  ];
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
              setReqParams((old)=>{
                return {
                  ...old,
                }
              })
            });
          },
        });
       
        break;
      case "1":
        nav(`/model-provider/edit/${record.id}`);
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
    fetchData();
  }
  const onCancelClick = ()=>{
    setOpen(false);
  }

  const onAPIKeyClick = (record: any)=>{
    setEditValue(record);
    setOpen(true);
  }
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
          ></Search>
           <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/model-provider/new`)}>{t('common.add')}</Button>
        </div>
        <List
          loading={loading}
          pagination={{
            showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
            defaultPageSize: 12,
            defaultCurrent: 1,
            onChange: onPageChange,
            total: data.total || 0,
            showSizeChanger: true,
          }}
          grid={{ gutter: 16, column: 3,  xs: 1,
            sm: 2,
           }}
          dataSource={data.data}
          renderItem={(provider) => (
            <List.Item>
               <div className="p-1em min-h-[132px] border border-[var(--ant-color-border)] group rounded-[8px] hover:bg-[var(--ant-control-item-bg-hover)]">
                 <div className="flex justify-between">
                    <div className="flex gap-15px">
                      <div className="flex items-center gap-8px">
                        <IconWrapper className="w-40px h-40px">
                          <InfiniIcon src={provider.icon} height="2em" width="2em" className="font-size-2em"/>
                        </IconWrapper>
                        <span className="font-size-1.2em">{provider.name}</span>
                      </div>
                      {provider.builtin === true && <div className="flex items-center">
                        <p className="h-[22px] bg-[#eee] text-[#999] font-size-[12px] px-[10px] line-height-[22px] rounded-[4px]">{t('page.modelprovider.labels.builtin')}</p>
                      </div>}
                    </div>
                    <div>
                      <Switch checked={provider.enabled} onChange={(v)=>onItemEnableChange(provider, v)} size="small" />
                    </div>
                  </div>
                  <div className="text-[#999] h-[51px] line-clamp-3 text-xs my-[10px]">
                    {provider.description}
                  </div>
                  <div className="flex gap-1">

                    <div className="ml-auto flex gap-2">
                      <div onClick={()=>{onAPIKeyClick(provider)}} className="border border-[var(--ant-color-border)] cursor-pointer  px-10px rounded-[8px]">API-key</div>
                      <div className="inline-block px-4px rounded-[8px] border border-[var(--ant-color-border)] cursor-pointer">
                        <Dropdown menu={{ items, onClick:({key})=>onMenuClick({key, record: provider}) }}>
                            <SettingOutlined className="text-blue-500"/>
                        </Dropdown>
                      </div>
                    </div>
                  </div>
                </div>
            </List.Item>
          )}
        />
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