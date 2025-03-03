import { type LoaderFunctionArgs, useLoaderData } from "react-router-dom";
import type { TableColumnsType, TableProps, MenuProps } from "antd";
import { Switch, Table, Dropdown} from "antd";
import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, DownOutlined, FolderOpenOutlined, FileWordOutlined, FileOutlined, EllipsisOutlined } from "@ant-design/icons";
import {fetchDatasourceDetail} from '@/service/api';
import { formatESSearchResult } from '@/service/request/es';

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
}

const columns: TableColumnsType<DataType> = [
  {
    title: "Name",
    dataIndex: "title",
    render: (text: string, record: DataType) =>{
      if(record.type == "web_page"){
        return <span><FileOutlined className="mr-3px" /><a target="_blank" href={record.url} className="text-blue-500">{text}</a></span>
      }
      if(record.is_dir){
        return <span><FolderOpenOutlined className="mr-3px text-yellow-500" /><a>{text}</a></span>
      }
      return <span>{record.type =="word" ? <FileWordOutlined className="mr-3px text-blue-500"/>: <FileOutlined className="mr-3px"/>}{text}</span>
    },
  },
  {
    title: "Searchable",
    dataIndex: "searchable",
    render: (text: boolean) => {
      return <Switch value={text} />;
    },
  },
  {
    title: "Operations",
    fixed: 'right',
    width: "90px",
    render: () => {
      return <Dropdown menu={{ items }}>
        <EllipsisOutlined/>
      </Dropdown>
    },
  },
];

// rowSelection object indicates the need for row selection
const rowSelection: TableProps<DataType>["rowSelection"] = {
  onChange: (selectedRowKeys: React.Key[], selectedRows: DataType[]) => {
    console.log(
      `selectedRowKeys: ${selectedRowKeys}`,
      "selectedRows: ",
      selectedRows,
    );
  },
  getCheckboxProps: (record: DataType) => ({// Column configuration not to be checked
    name: record.title,
  }),
};

const items: MenuProps["items"] = [
  {
    label: "Delete",
    key: "1",
  },
];

export function Component() {
  const datasourceID = useLoaderData();

  const { t } = useTranslation();
  const nav = useNavigate();
  const location = useLocation();
  const { datasource_name} = location.state || {}; 

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
            placeholder="input search text"
            enterButton="Refresh"
            onSearch={onRefreshClick}
          />
          <div>
            <Dropdown.Button
              icon={<DownOutlined />}
              menu={{ items }}
              type="primary"
              onClick={() => {}}
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
