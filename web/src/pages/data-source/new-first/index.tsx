import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, PlusOutlined } from "@ant-design/icons";
import WebsiteSVG from "@/assets/svg-icon/website.svg";
import CloudDiskSVG from "@/assets/svg-icon/cloud_disk.svg";
import CreatorSVG from "@/assets/svg-icon/creator.svg";
import { Button, List, Image } from "antd";
import { ReactSVG } from "react-svg";
import {searchConnector} from "@/service/api/connector";
import { formatESSearchResult } from '@/service/request/es';

const ConnectorCategory = {
  CloudStorage: "cloud_storage",
  Website: "website",
}

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const onAddClick = (key: string) => {
    nav(`/data-source/new/?type=${key}`)
  }
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
    searchConnector(reqParams).then((data)=>{
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
        </div>
        <List
          pagination={{
            showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
            defaultPageSize: 10,
            defaultCurrent: 1,
            onChange: onPageChange,
            total: data.total || 0,
            showSizeChanger: true,
          }}
          grid={{ gutter: 16, column: 3 }}
          dataSource={data.data}
          renderItem={(connector) => (
            <List.Item>
               <div className="relative p-1em border border-gray-300 group rounded-[8px] hover:bg-gray-100 hover:bg-opacity-100">
                <Button type="primary" onClick={()=>{
                  onAddClick(connector.id)
                }} className="absolute hidden group-hover:block top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2">
                  <PlusOutlined className="font-bold text-1.4em"/>
                </Button>
                  <div className="flex items-center gap-8px">
                  <Image src={connector.icon} height="2.6em" width="2.6em" preview={false}/><span className="font-size-1.2em">{connector.name}</span>
                    {/* <Icon component={connector.icon} className="font-size-2.6em"/> <span className="font-size-1.2em">{connector.name}</span> */}
                  </div>
                  <div className="flex items-center gap-2em text-gray-500 my-1em">
                    {connector.category === ConnectorCategory.Website && <div className="flex items-center gap-3px"> <ReactSVG src={WebsiteSVG} className="font-size-1.2em"/> <span>Website</span></div>}
                    {connector.category === ConnectorCategory.CloudStorage && <div className="flex items-center gap-3px"> <ReactSVG src={CloudDiskSVG} className="font-size-1.2em"/> <span>Cloud Storage</span></div>}
                    <div className="flex items-center gap-3px">  <ReactSVG src={CreatorSVG} className="font-size-1.2em"/>  <span>{connector.author || "INFINI Labs"}</span></div>
                  </div>
                  <div className="text-gray-500 h-70px">{connector.description}</div>
                  <div className="text-gray-500 text-12px flex gap-5px mt-1em mt-10px">
                    {(connector.tags || []).map((tag) => <div className="border border-gray-300 rounded px-5px">{tag}</div>)}
                  </div>
                </div>
            </List.Item>
          )}
        />
      </ACard>
    </div>
  );
}
