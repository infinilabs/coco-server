import { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { useLoading } from '@sa/hooks';
import { Button, Dropdown, Input, Table, Tag, message } from 'antd';
import dayjs from 'dayjs';
import type { TableColumnsType, TableProps } from 'antd';

import useQueryParams from '@/hooks/common/queryParams';
import { deleteUser, fetchUsers } from '@/service/api/security';
import { formatESSearchResult } from '@/service/request/es';

const User = () => {
  const [queryParams, setQueryParams] = useQueryParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth();

  const permissions = {
    read: hasAuth('generic#security:user/read'),
    create: hasAuth('generic#security:user/create'),
    delete: hasAuth('generic#security:user/delete'),
    update: hasAuth('generic#security:user/update')
  };

  const nav = useNavigate();

  const [data, setData] = useState<{
    data: any[];
    total: number | { value: number };
  }>({
    data: [],
    total: 0
  });
  const { endLoading, loading, startLoading } = useLoading();
  const [keyword, setKeyword] = useState('');

  const fetchData = async (params: any) => {
    startLoading();
    const res = await fetchUsers(params);
    const newData = formatESSearchResult(res.data);
    setData(newData);
    endLoading();
  };

  const handleTableChange: TableProps<any>['onChange'] = pagination => {
    setQueryParams((params: any) => {
      const pageSize = pagination?.pageSize ?? params.size ?? 10;
      const current = pagination?.current ?? Math.floor((params.from ?? 0) / pageSize) + 1;
      return {
        ...params,
        from: (current - 1) * pageSize,
        size: pageSize
      };
    });
  };

  const onRefreshClick = (query: string) => {
    const q = (query || '').trim();
    setQueryParams((oldParams: any) => {
      return {
        ...oldParams,
        from: 0,
        query: q,
        t: new Date().valueOf()
      };
    });
  };

  const handleDelete = useCallback(
    async (id: string) => {
      startLoading();
      const res = await deleteUser(id);
      if (res.data?.result === 'deleted') {
        message.success(t('common.deleteSuccess'));
      }
      fetchData(queryParams);
      endLoading();
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [queryParams, t]
  );

  const columns: TableColumnsType<any> = [
    {
      dataIndex: 'name',
      title: t('page.user.labels.name')
    },
    {
      dataIndex: 'email',
      title: t('page.user.labels.email')
    },
    {
      dataIndex: 'roles',
      title: t('page.user.labels.roles'),
      render: (value) => {
        if (!Array.isArray(value)) return '-'
        return value.map((tag, index) => {
          return <Tag key={index}>{tag}</Tag>;
        });
      }
    },
    {
      dataIndex: 'created',
      title: t('page.user.labels.created'),
      render: (value: string) => {
        const d = dayjs(value);
        return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : value;
      }
    },
    {
      fixed: 'right',
      hidden: !permissions.update && !permissions.delete,
      render: (_, record) => {
        const items = [];
        if (permissions.read && permissions.update) {
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
        // eslint-disable-next-line @typescript-eslint/no-shadow
        const onMenuClick = ({ key, record }: any) => {
          switch (key) {
            case 'edit':
              nav(`/user/edit/${record.id}`, { state: record });
              break;
            case 'delete':
              window?.$modal?.confirm({
                content: t('page.user.delete.confirm', { name: record.name }),
                icon: <ExclamationCircleOutlined />,
                onOk() {
                  handleDelete(record.id);
                },
                title: t('common.tip')
              });
              break;
            default:
              break;
          }
        };
        return (
          <Dropdown
            menu={{
              items,
              onClick: ({ key }) => onMenuClick({ key, record })
            }}
          >
            <EllipsisOutlined />
          </Dropdown>
        );
      },
      title: t('common.operation'),
      width: '90px'
    }
  ];

  const rowSelection: TableProps<any>['rowSelection'] = {
    getCheckboxProps: record => ({
      name: record.name
    }),
    onChange: (_selectedRowKeys: React.Key[], _selectedRows) => {}
  };

  useEffect(() => {
    fetchData(queryParams);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [queryParams]);

  useEffect(() => {
    setKeyword((queryParams as any).query || '');
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
            onClick={() => nav(`/user/new`)}
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
          showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
          pageSize: queryParams.size ?? 10,
          current: Math.floor((queryParams.from ?? 0) / (queryParams.size ?? 10)) + 1,
          total: typeof data.total === 'object' ? (data.total?.value ?? 0) : (data.total ?? 0),
          showSizeChanger: true
        }}
        onChange={handleTableChange}
      />
    </ListContainer>
  );
};

export default User;
