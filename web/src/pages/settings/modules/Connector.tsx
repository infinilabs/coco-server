import { Button, Dropdown, Form, Image, Input, Modal, Spin, Table, Tag, message } from 'antd';

import '../index.scss';
import type { MenuProps, TableColumnsType } from 'antd';
import Search from 'antd/es/input/Search';

import type { IntegratedStoreModalRef } from '@/components/common/IntegratedStoreModal';
import InfiniIcon from '@/components/common/icon';
import { GoogleDriveSVG, HugoSVG, NotionSVG, YuqueSVG } from '@/components/icons';
import useQueryParams from '@/hooks/common/queryParams';
import { deleteConnector, searchConnector } from '@/service/api/connector';

import Icon, {
  EllipsisOutlined,
  ExclamationCircleOutlined,
  FilterOutlined,
  PlusOutlined,
  createFromIconfontCN
} from '@ant-design/icons';

import { formatESSearchResult } from '@/service/request/es';

type Connector = Api.Datasource.Connector;

const ConnectorSettings = memo(() => {
  const [queryParams, setQueryParams] = useQueryParams();

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
        window?.$modal?.confirm({
          content: t('page.connector.delete.confirm', { name: record.name }),
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            deleteConnector(record.id).then(res => {
              if (res.data?.result === 'deleted') {
                message.success(t('common.deleteSuccess'));
              }
              // reload data
              setQueryParams(old => {
                return {
                  ...old,
                  t: new Date().valueOf()
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
            <IconWrapper className="h-20px w-20px">
              {svgIcon ? (
                <Icon component={svgIcon} />
              ) : (
                <InfiniIcon
                  height="1em"
                  src={record.icon}
                  width="1em"
                />
              )}
            </IconWrapper>
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
      ellipsis: true,
      minWidth: 100,
      title: t('page.connector.columns.description')
    },
    {
      dataIndex: 'tags',
      minWidth: 100,
      render: (value: string[]) => {
        return (value || []).map((tag, index) => {
          return <Tag key={index}>{tag}</Tag>;
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

  const [keyword, setKeyword] = useState();

  const fetchData = () => {
    setLoading(true);
    searchConnector(queryParams).then(({ data }: { data: any }) => {
      const newData = formatESSearchResult(data);
      setData(newData);
      setLoading(false);
    });
  };

  useEffect(fetchData, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query);
  }, [queryParams.query]);

  const onAddClick = () => {
    nav(`/connector/new`);
  };

  const handleTableChange = (pagination, filters, sorter) => {
    setQueryParams(params => {
      return {
        ...params,
        from: (pagination.current - 1) * pagination.pageSize,
        size: pagination.pageSize
      };
    });
  };

  const onSearchClick = (query: string) => {
    setQueryParams(old => {
      return {
        ...old,
        query,
        t: new Date().getTime()
      };
    });
  };

  const integratedStoreModalRef = useRef<IntegratedStoreModalRef>(null);

  return (
    <ListContainer>
      <div className="mb-4 mt-4 flex items-center justify-between">
        <Search
          addonBefore={<FilterOutlined />}
          className="max-w-500px"
          enterButton={t('common.refresh')}
          value={keyword}
          onChange={e => setKeyword(e.target.value)}
          onSearch={onSearchClick}
        />
        <Button
          icon={<PlusOutlined />}
          type="primary"
          onClick={() => {
            integratedStoreModalRef.current?.open('connector');
          }}
        >
          {t('common.add')}
        </Button>

        <IntegratedStoreModal ref={integratedStoreModalRef} />
      </div>
      <Table<Connector>
        columns={columns}
        dataSource={data.data}
        loading={loading}
        rowKey="id"
        size="middle"
        pagination={{
          current: Math.floor(queryParams.from / queryParams.size) + 1,
          pageSize: queryParams.size,
          showSizeChanger: true,
          showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
          total: data?.total?.value || data?.total
        }}
        onChange={handleTableChange}
      />
    </ListContainer>
  );
});

export default ConnectorSettings;
