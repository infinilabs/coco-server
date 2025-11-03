import { Avatar, Button, Dropdown, Form, Image, Input, Modal, Spin, Table, Tag, message } from 'antd';

import '../index.scss';
import type { MenuProps, TableColumnsType } from 'antd';
import Search from 'antd/es/input/Search';

import InfiniIcon from '@/components/common/icon';
import { GoogleDriveSVG, HugoSVG, NotionSVG, YuqueSVG } from '@/components/icons';
import { deleteConnector, searchConnector } from '@/service/api/connector';

import Icon, {
  EllipsisOutlined,
  ExclamationCircleOutlined,
  FilterOutlined,
  PlusOutlined,
} from '@ant-design/icons';

import { formatESSearchResult } from '@/service/request/es';
import useQueryParams from '@/hooks/common/queryParams';

type Connector = Api.Datasource.Connector;

const ConnectorSettings = memo(() => {
  const [queryParams, setQueryParams] = useQueryParams();
  
  const { t } = useTranslation();

  const { addSharesToData, isEditorOwner, hasEdit } = useResource()
  const resourceType = 'connector'

  const { hasAuth } = useAuth()

  const permissions = {
    read: hasAuth('coco#connector/read'),
    create: hasAuth('coco#connector/create'),
    update: hasAuth('coco#connector/update'),
    delete: hasAuth('coco#connector/delete'),
  }

  const nav = useNavigate();

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
            <IconWrapper className="w-20px h-20px">
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
      dataIndex: 'category',
      minWidth: 200,
      title: t('page.connector.columns.category')
    },
    {
      dataIndex: 'description',
      minWidth: 100,
      title: t('page.connector.columns.description'),
      ellipsis: true,
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
      hidden: !permissions.update && !permissions.delete,
      render: (_, record) => {
        const items: MenuProps['items'] = [];
        if (permissions.read && permissions.update && hasEdit(record)) {
          items.push({
            key: '1',
            label: t('common.edit')
          })
        }
        if (permissions.delete && isEditorOwner(record)) {
          items.push({
            key: '2',
            label: t('common.delete')
          })
        }
        if (items.length === 0) return null;
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

  const fetchData = async (queryParams) => {
    setLoading(true);
    const res = await searchConnector(queryParams)
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

  const onAddClick = () => {
    nav(`/connector/new`);
  };

  const handleTableChange = (pagination, filters, sorter) => {
      setQueryParams((params)=>{
        return {
          ...params,
          size: pagination.pageSize,
          from: (pagination.current-1) * pagination.pageSize,
        }
      })
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
  return (
    <ListContainer>
      <div className="mb-4 mt-4 flex items-center justify-between">
        <Search
          value={keyword} 
          onChange={(e) => setKeyword(e.target.value)} 
          addonBefore={<FilterOutlined />}
          className="max-w-500px"
          enterButton={t('common.refresh')}
          onSearch={onSearchClick}
        />
        {
          permissions.create && (
            <Button
              icon={<PlusOutlined />}
              type="primary"
              onClick={onAddClick}
            >
              {t('common.add')}
            </Button>
          )
        }
      </div>
      <Table<Connector>
        columns={columns}
        dataSource={data.data}
        loading={loading}
        rowKey="id"
        size="middle"
        pagination={{
          pageSize: queryParams.size,
          current: Math.floor(queryParams.from / queryParams.size) + 1,
          showSizeChanger: true,
          total: data?.total?.value || data?.total,
          showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`
        }}
        onChange={handleTableChange}
      />
    </ListContainer>
  );

});

export default ConnectorSettings;
