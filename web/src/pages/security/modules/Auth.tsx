import { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { useLoading } from '@sa/hooks';
import { Button, Dropdown, Input, Table, message } from 'antd';

import useQueryParams from '@/hooks/common/queryParams';
import { deleteRole, fetchRoles } from '@/service/api/role';
import { formatESSearchResult } from '@/service/request/es';

const Auth = () => {
  const [queryParams, setQueryParams] = useQueryParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth();

  const permissions = {
    create: hasAuth('coco:role/create'),
    delete: hasAuth('coco:role/delete'),
    update: hasAuth('coco:role/update')
  };

  const nav = useNavigate();

  const [data, setData] = useState({
    data: [],
    total: 0
  });
  const { endLoading, loading, startLoading } = useLoading();
  const [keyword, setKeyword] = useState();

  const fetchData = async params => {
    startLoading();
    const res = await fetchRoles(params);
    const newData = formatESSearchResult(res.data);
    setData(newData);
    endLoading();
  };

  const handleTableChange = pagination => {
    setQueryParams(params => {
      return {
        ...params,
        from: (pagination.current - 1) * pagination.pageSize,
        size: pagination.pageSize
      };
    });
  };

  const onRefreshClick = (query: string) => {
    setQueryParams(oldParams => {
      return {
        ...oldParams,
        from: 0,
        query,
        t: new Date().valueOf()
      };
    });
  };

  const handleDelete = async id => {
    startLoading();
    const res = await deleteRole(id);
    if (res.data?.result === 'deleted') {
      message.success(t('common.deleteSuccess'));
    }
    fetchData(queryParams);
    endLoading();
  };

  const columns = [
    {
      dataIndex: 'name',
      title: '授权'
    },
    {
      dataIndex: 'description',
      title: '角色'
    },
    {
      dataIndex: 'description',
      title: '创建时间'
    },
    {
      dataIndex: 'description',
      title: '启用状态'
    },
    {
      fixed: 'right',
      render: (_, record) => {
        const items = [];
        if (permissions.update) {
          items.push({
            key: 'edit',
            label: t('common.edit')
          });
        }
        if (permissions.delete) {
          items.push({
            key: 'delete',
            label: t('common.delete')
          });
        }
        if (items.length === 0) return null;
        const onMenuClick = ({ key, record }: any) => {
          switch (key) {
            case 'edit':
              nav(`/role/edit/${record.id}`, { state: record });
              break;
            case 'delete':
              window?.$modal?.confirm({
                content: t('page.role.delete.confirm', { name: record.name }),
                icon: <ExclamationCircleOutlined />,
                onOk() {
                  handleDelete(record.id);
                },
                title: t('common.tip')
              });
              break;
          }
        };
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
  // rowSelection object indicates the need for row selection
  const rowSelection = {
    getCheckboxProps: record => ({
      name: record.name
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows) => {}
  };

  useEffect(() => {
    fetchData(queryParams);
  }, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query);
  }, [queryParams.query]);

  return (
    <ListContainer>
      <div className='mb-4 mt-4 flex items-center justify-between'>
        <Input.Search
          addonBefore={<FilterOutlined />}
          className='max-w-500px'
          enterButton={t('common.refresh')}
          value={keyword}
          onChange={e => setKeyword(e.target.value)}
          onSearch={onRefreshClick}
        />
        {permissions.create && (
          <Button
            icon={<PlusOutlined />}
            type='primary'
            onClick={() => nav(`/role/new`)}
          >
            {t('common.add')}
          </Button>
        )}
      </div>
      <Table
        columns={columns}
        dataSource={data.data}
        loading={loading}
        rowKey='id'
        rowSelection={{ ...rowSelection }}
        size='middle'
        pagination={{
          current: Math.floor(queryParams.from / queryParams.size) + 1,
          pageSize: queryParams.size,
          showSizeChanger: true,
          showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
          total: data.total?.value || data?.total
        }}
        onChange={handleTableChange}
      />
    </ListContainer>
  );
};

export default Auth;
