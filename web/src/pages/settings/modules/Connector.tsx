import { Button, Form, Input, Spin, Table, Modal, Dropdown, message, Tag, Image} from "antd";
import "../index.scss"
import { fetchSettings, updateSettings } from "@/service/api/server";
import {searchConnector, deleteConnector} from "@/service/api/connector";
import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, PlusOutlined, ExclamationCircleOutlined, EllipsisOutlined, createFromIconfontCN } from "@ant-design/icons";
import type { TableColumnsType, MenuProps } from "antd";
import { formatESSearchResult } from '@/service/request/es';
import { GoogleDriveSVG, HugoSVG, YuqueSVG,NotionSVG } from '@/components/icons';
import InfiniIcon from '@/components/common/icon';
const { confirm } = Modal;
type Connector = Api.Datasource.Connector;

export const GoogleDriveSettings = memo(() => {
    const [form] = Form.useForm();
    const { t } = useTranslation();

    const { defaultRequiredRule, formRules } = useFormRules();
    const { data, run, loading: dataLoading } = useRequest(fetchSettings, {
      manual: true
  });
  useMount(() => {
      run();
  });

  useEffect(() => {
    if (data?.data?.connector?.google_drive) {
      form.setFieldsValue(data.data.connector.google_drive || { });
    }
  }, [JSON.stringify(data)]);

    const [loading, setLoading] = useState(false);

    const handleSubmit = async () => {
      setLoading(true);
        const params = await form.validateFields();
        const result = await updateSettings({
            connector: {
              google_drive: params,
            }
        });
        setLoading(false);
        if (result.data.acknowledged) {
          window.$message?.success(t('common.updateSuccess'));
        }
    }

    return (
        <Spin spinning={loading}>
            <Form 
                form={form}
                labelAlign="left"
                className="settings-form"
                colon={false}
            >
                <Form.Item
                    name="client_id"
                    label="Client ID"
                    rules={[defaultRequiredRule]}
                >
                  <Input />
                </Form.Item>
                <Form.Item
                    name="client_secret"
                    label="Client Secret"
                    rules={[defaultRequiredRule]}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="redirect_url"
                    label="Redirect URI"
                    rules={formRules.endpoint}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="auth_url"
                    label="Auth URI"
                    rules={formRules.endpoint}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="token_url"
                    label="Token URI"
                    rules={formRules.endpoint}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    label=" "
                >
                    <Button type="primary" onClick={() => handleSubmit()}>{t('common.update')}</Button>
                </Form.Item>
            </Form>
        </Spin>
    )
});

const ConnectorSettings = memo(() => {
  const { t } = useTranslation();
  const nav = useNavigate();

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
          content: t('page.connector.delete.confirm', {name: record.name}),
          onOk() {
            deleteConnector(record.id).then((res)=>{
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
      case "1":
        nav(`/connector/edit/${record.id}`, {state:record});
        break;
    }
  }
  const columns: TableColumnsType<Connector> = [
    {
      title: t('page.connector.columns.name'),
      dataIndex: "name",
      minWidth: 100,
      render: (name, record) => {
        let svgIcon = null;
        switch(record.id){
          case "google_drive":
            svgIcon = GoogleDriveSVG;
            break;
          case "yuque":
            svgIcon = YuqueSVG;
            break;
          case "notion":
            svgIcon = NotionSVG;
            break;
          case "hugo_site":
            svgIcon = HugoSVG;
            break;
        }
        return (
          <div className="flex items-center">
            {svgIcon ? <Icon component={svgIcon} /> : 
            <InfiniIcon src={record.icon} width="1em" height="1em"/>}
            <span className="ml-2">{name}</span>
          </div>
        )
      }
    },
    {
      title: t('page.connector.columns.category'),
      dataIndex: "category",
      minWidth: 200,
    },
    {
      title: t('page.connector.columns.description'),
      minWidth: 100,
      dataIndex: "description",
    },
    {
      title: t('page.connector.columns.tags'),
      minWidth: 100,
      dataIndex: "tags",
      render: (value: string[])=>{
        return (value || []).map((tag)=>{
          return <Tag>{tag}</Tag>
        })
      },
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

  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [reqParams, setReqParams] = useState({
    query: '',
    t: new Date().getTime(),
  })
  const [tableParams, setTableParams] = useState({
    pagination: {
      current: 1,
      pageSize: 10,
    },
  });

  const fetchData = () => {
    setLoading(true);
    searchConnector(reqParams).then(({ data }: {data: any}) => {
      const newData = formatESSearchResult(data);
      setData(newData?.data || []);
      setLoading(false);
      setTableParams(oldParams=>{
        return {
          ...oldParams,
          pagination: {
            ...oldParams.pagination,
            total: newData.total?.value || newData.total,
          },
        }
      });
    });
  };
  
  useEffect(fetchData, [reqParams]);
  const onAddClick = ()=>{
    nav(`/connector/new`)
  }

  const onSearchClick = (query: string)=>{
    setReqParams((old)=>{
      return {
        query: query,
        t: new Date().getTime(),
      }
    })
  }
  return (
    <div className="h-full min-h-500px flex-col-stretch overflow-hidden lt-sm:overflow-auto">
      <div className="mb-4 mt-4 flex items-center justify-between">
        <Search
          addonBefore={<FilterOutlined />}
          className="max-w-500px"
          onSearch={onSearchClick}
          enterButton={t("common.refresh")}
        ></Search>
        <Button type='primary' icon={<PlusOutlined/>}  onClick={onAddClick}>{t('common.add')}</Button>
      </div>
      <Table<Connector>
          rowKey="id"
          loading={loading}
          size="middle"
          columns={columns}
          dataSource={data}
          pagination={{ 
            showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
            defaultPageSize: 10,
            defaultCurrent: 1,
            showSizeChanger: true,
          }}
        />
      </div>
  )

  // const items = [
  //   {
  //     key: 'google_drive',
  //     label: 'Gogole Drive',
  //     children: <GoogleDriveSettings />,
  //   },
  // ];

  // return  <Tabs items={items}/>
});

export default ConnectorSettings;