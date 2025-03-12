import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, PlusOutlined } from "@ant-design/icons";
import { GoogleDriveSVG, HugoSVG, YuqueSVG,NotionSVG} from '@/components/icons';
import WebsiteSVG from "@/assets/svg-icon/website.svg";
import CloudDiskSVG from "@/assets/svg-icon/cloud_disk.svg";
import CreatorSVG from "@/assets/svg-icon/creator.svg";
import { Button } from "antd";
import { ReactSVG } from "react-svg";

const ConnectorCategory = {
  CloudStorage: "cloud_storage",
  Website: "website",
}
const Connectors = [{
  key: "google_drive",
  name: "Google Drive",
  icon: GoogleDriveSVG,
  category: ConnectorCategory.CloudStorage,
  description: "Fetch the files metadata from Google Drive",
  tags: ["Google", "Storage"],
  author: "INFINI Labs",
},{
  name: "Hugo Site",
  key: "hugo_site",
  icon: HugoSVG,
  category: ConnectorCategory.Website,
  description: "Fetch the index.json file from a specified Hugo site",
  tags: ["hugo", "web", "static_site"],
  author: "INFINI Labs",
},{
  name: "Yuque",
  key: "yuque",
  icon: YuqueSVG,
  category: ConnectorCategory.CloudStorage,
  description: "Fetch the docs metadata for yuque",
  tags: ["docs", "yuque", "web"],
  author: "INFINI Labs",
},{
  name: "Notion",
  key: "notion",
  icon: NotionSVG,
  category: ConnectorCategory.CloudStorage,
  description: "Fetch the docs metadata for notion",
  tags: ["docs", "notion", "web"],
  author: "INFINI Labs",
}]

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const onAddClick = (key: string) => {
    nav(`/data-source/new/?type=${key}`)
  }
  const [connectors, setConnectors] = useState(Connectors);
  const onSearchClick = (query: string)=>{
    const filteredConnectors = Connectors.filter((connector) => connector.name.toLowerCase().includes(query.toLowerCase()));
    setConnectors(filteredConnectors);
  }
  return (
    <div className="h-full min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
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
        <div className="flex gap-4 flex-wrap">
          {connectors.map((connector) => (
          <div className="relative p-1em border border-gray-300 group rounded-[8px] w-[calc(33.33%-0.67rem)] hover:bg-gray-100 hover:bg-opacity-100">
          <Button type="primary" onClick={()=>{
            onAddClick(connector.key)
          }} className="absolute hidden group-hover:block top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2">
            <PlusOutlined className="font-bold text-1.4em"/>
          </Button>
            <div className="flex items-center gap-8px">
              <Icon component={connector.icon} className="font-size-2.6em"/> <span className="font-size-1.2em">{connector.name}</span>
            </div>
            <div className="flex items-center gap-2em text-gray-500 my-1em">
              {connector.category === ConnectorCategory.Website && <div className="flex items-center gap-3px"> <ReactSVG src={WebsiteSVG} className="font-size-1.2em"/> <span>Website</span></div>}
              {connector.category === ConnectorCategory.CloudStorage && <div className="flex items-center gap-3px"> <ReactSVG src={CloudDiskSVG} className="font-size-1.2em"/> <span>Cloud Storage</span></div>}
              <div className="flex items-center gap-3px">  <ReactSVG src={CreatorSVG} className="font-size-1.2em"/>  <span>{connector.author}</span></div>
            </div>
            <div className="text-gray-500 h-70px">{connector.description}</div>
            <div className="text-gray-500 text-12px flex gap-5px mt-1em mt-10px">
              {(connector.tags || []).map((tag) => <div className="border border-gray-300 rounded px-5px">{tag}</div>)}
            </div>
          </div>
        ))}
        </div>
      </ACard>
    </div>
  );
}
