import { Button, Dropdown, Form, Image, Input, Modal, Spin, Table, Tag, message } from 'antd';

import '../index.scss';
import type { MenuProps, TableColumnsType } from 'antd';
import Search from 'antd/es/input/Search';

import InfiniIcon from '@/components/common/icon';
import { GoogleDriveSVG, HugoSVG, NotionSVG, YuqueSVG } from '@/components/icons';
import { deleteConnector, searchConnector } from '@/service/api/connector';
import { fetchSettings, updateSettings } from '@/service/api/server';

import Icon, {
  EllipsisOutlined,
  ExclamationCircleOutlined,
  FilterOutlined,
  PlusOutlined,
  createFromIconfontCN
} from '@ant-design/icons';

import { formatESSearchResult } from '@/service/request/es';

const { confirm } = Modal;
type Connector = Api.Datasource.Connector;

export const GoogleDriveSettings = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const { defaultRequiredRule, formRules } = useFormRules();
  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });
  useMount(() => {
    run();
  });

  useEffect(() => {
    if (data?.data?.connector?.google_drive) {
      form.setFieldsValue(data.data.connector.google_drive || {});
    }
  }, [JSON.stringify(data)]);

  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    setLoading(true);
    const params = await form.validateFields();
    const result = await updateSettings({
      connector: {
        google_drive: params
      }
    });
    setLoading(false);
    if (result.data.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
    }
  };

  return (
    <Spin spinning={loading}>
      <Form
        className="settings-form"
        colon={false}
        form={form}
        labelAlign="left"
      >
        <Form.Item
          label="Client ID"
          name="client_id"
          rules={[defaultRequiredRule]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label="Client Secret"
          name="client_secret"
          rules={[defaultRequiredRule]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label="Redirect URI"
          name="redirect_url"
          rules={formRules.endpoint}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label="Auth URI"
          name="auth_url"
          rules={formRules.endpoint}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label="Token URI"
          name="token_url"
          rules={formRules.endpoint}
        >
          <Input />
        </Form.Item>
        <Form.Item label=" ">
          <Button
            type="primary"
            onClick={() => handleSubmit()}
          >
            {t('common.update')}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});

const ConnectorSettings = memo(() => {
  const { t } = useTranslation();
  const nav = useNavigate();

  const items: MenuProps['items'] = [
    {
      key: '1',
      label: t('common.edit')
    },
    {
      key: '2',
      label: t('common.delete')
    }
  ];

  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '2':
        confirm({
          content: t('page.connector.delete.confirm', { name: record.name }),
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            deleteConnector(record.id).then(res => {
              if (res.data?.result === 'deleted') {
                message.success(t('common.deleteSuccess'));
              }
              // reload data
              setReqParams(old => {
                return {
                  ...old
                };
              });
            });
          },
          title: t('common.tip')
        });

        break;
      case '1':
        nav(`/connector/edit/${record.id}`, { state: record });
        break;
    }
  };
  const columns: TableColumnsType<Connector> = [
    {
      dataIndex: 'name',
      minWidth: 100,
      render: (name, record) => {
        let svgIcon = null;
        switch (record.id) {
          case 'google_drive':
            svgIcon = GoogleDriveSVG;
            break;
          case 'yuque':
            svgIcon = YuqueSVG;
            break;
          case 'notion':
            svgIcon = NotionSVG;
            break;
          case 'hugo_site':
            svgIcon = HugoSVG;
            break;
        }
        return (
          <div className="flex items-center">
            {svgIcon ? (
              <Icon component={svgIcon} />
            ) : (
              <InfiniIcon
                height="1em"
                src={record.icon}
                width="1em"
              />
            )}
            <span className="ml-2">{name}</span>
          </div>
        );
      },
      title: t('page.connector.columns.name')
    },
    {
      dataIndex: 'category',
      minWidth: 200,
      title: t('page.connector.columns.category')
    },
    {
      dataIndex: 'description',
      minWidth: 100,
      title: t('page.connector.columns.description')
    },
    {
      dataIndex: 'tags',
      minWidth: 100,
      render: (value: string[]) => {
        return (value || []).map(tag => {
          return <Tag>{tag}</Tag>;
        });
      },
      title: t('page.connector.columns.tags')
    },
    {
      fixed: 'right',
      render: (_, record) => {
        return (
          <Dropdown menu={{ items, onClick: ({ key }) => onMenuClick({ key, record }) }}>
            <EllipsisOutlined />
          </Dropdown>
        );
      },
      title: t('common.operation'),
      width: '90px'
    }
  ];

  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [reqParams, setReqParams] = useState({
    query: '',
    t: new Date().getTime()
  });
  const [tableParams, setTableParams] = useState({
    pagination: {
      current: 1,
      pageSize: 10
    }
  });

  const fetchData = () => {
    setLoading(true);
    searchConnector(reqParams).then(({ data }: { data: any }) => {
      const newData = formatESSearchResult(data);
      setData(newData?.data || []);
      setLoading(false);
      setTableParams(oldParams => {
        return {
          ...oldParams,
          pagination: {
            ...oldParams.pagination,
            total: newData.total?.value || newData.total
          }
        };
      });
    });
  };

  useEffect(fetchData, [reqParams]);
  const onAddClick = () => {
    nav(`/connector/new`);
  };

  const onSearchClick = (query: string) => {
    setReqParams(old => {
      return {
        query,
        t: new Date().getTime()
      };
    });
  };
  return (
    <div className="h-full min-h-500px flex-col-stretch overflow-hidden lt-sm:overflow-auto">
      <div className="mb-4 mt-4 flex items-center justify-between">
        <Search
          addonBefore={<FilterOutlined />}
          className="max-w-500px"
          enterButton={t('common.refresh')}
          onSearch={onSearchClick}
        />
        <Button
          icon={<PlusOutlined />}
          type="primary"
          onClick={onAddClick}
        >
          {t('common.add')}
        </Button>
      </div>
      <Table<Connector>
        columns={columns}
        dataSource={data}
        loading={loading}
        rowKey="id"
        size="middle"
        pagination={{
          defaultCurrent: 1,
          defaultPageSize: 10,
          showSizeChanger: true,
          showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`
        }}
      />
    </div>
  );

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
