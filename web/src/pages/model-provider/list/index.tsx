import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, PlusOutlined, SettingOutlined, ExclamationCircleOutlined } from "@ant-design/icons";
import { Button, List, Image, Switch, Tag, message, MenuProps, Modal, Dropdown} from "antd";
import { ReactSVG } from "react-svg";
import {searchModelPovider, updateModelProvider, deleteModelProvider} from "@/service/api/llm";
import { formatESSearchResult } from '@/service/request/es';
const { confirm } = Modal;
// const modelProviders = [{
//   name: "OpenAI",
//   APIKey: "xxxxxx",
//   APIEndpoint: "https://api.openai.com",
//   icon: "/assets/icons/llm/openai.svg",
//   models: ["gpt-3", "gpt-4", "gpt-5"],
//   enabled: true,
// },{
//   name: "Deepseek",
//   APIKey: "xxxxxx",
//   APIEndpoint: "https://api.deepseek.com",
//   icon: "/assets/icons/llm/deepseek.svg",
//   models: ["deepseek-r1", "deepseek-r2", "deepseek-r3"],
//   enabled: true,
// },{
//   name: "Ollama",
//   APIEndpoint: "http://127.0.0.1:11434",
//   icon: "/assets/icons/llm/ollama.svg",
//   models: ["gpt-4", "deepseek-r1", "gpt-4", "deepseek-r1"],
//   enabled: true,
// }
// ];

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
    size: 10,
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
        confirm({
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
  
  useEffect(()=>{
    fetch("http://localhost:9000/connector/_search", {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        "Authorization": "Bearer token",
        "X-API-TOKEN": "cvifokdath21us181efgzuls8ozx1kv0d6rzxw02gp4azx809mlba8n3wg14asdh2x596pjymalik2954y46", 
        "APP-INTEGRATION-ID":"cvhmcqlath272nmlj10x"
      },
    })
      .then((res) => {
        console.log(res.headers.get("Authorization")); // Debug header
        return res.json();
      })
      .then((data) => console.log(data))
      .catch((err) => console.error("Fetch error:", err));
  }, [])
  const onEditClick = (id: string)=>{
    nav(`/model-provider/edit/${id}`);
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
  return (
    <div className="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
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
            defaultPageSize: 10,
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
               <div className="p-1em min-h-[132px] border border-gray-300 group rounded-[8px] hover:bg-gray-100 hover:bg-opacity-100">
                 <div className="flex justify-between">
                    <div className="flex items-center gap-8px">
                    {provider.icon.endsWith(".svg") ? <ReactSVG src={provider.icon} className="font-size-2em"/> : <Image src={provider.icon} height="2em" width="2em" preview={false}/>}
                    <span className="font-size-1.2em">{provider.name}</span>
                    </div>
                    <div>
                      <Switch defaultChecked={provider.enabled} onChange={(v)=>onItemEnableChange(provider, v)} size="small" />
                      <div className="ml-[5px] inline-block px-4px rounded-[8px] border border-gray-200 cursor-pointer">
                      <Dropdown menu={{ items, onClick:({key})=>onMenuClick({key, record: provider}) }}>
                        <SettingOutlined className="text-blue-500"/>
                     </Dropdown>
                       
                      </div>
                    </div>
                  </div>
                  <div className="text-gray-500 text-12px mt-10px">
                    {(provider.models || []).map((model) => <Tag className="border border-gray-300 rounded px-5px mt-10px">{model}</Tag>)}
                  </div>
                </div>
            </List.Item>
          )}
        />
      </ACard>
    </div>
  );
}
