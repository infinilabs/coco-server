import { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { useLoading } from '@sa/hooks';
import { Button, Dropdown, Input, Table, Tag, Typography, message } from 'antd';
import dayjs from 'dayjs';
import type { TableColumnsType, TableProps } from 'antd';

import useQueryParams from '@/hooks/common/queryParams';
import { deleteAuthorization, fetchAuthorizationSearch } from '@/service/api/security';
import { formatESSearchResult } from '@/service/request/es';
import { fetchPrincipalSearch } from '@/service/api/security';
import {useMemo } from 'react';

const Auth = () => {
  const [queryParams, setQueryParams] = useQueryParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth();

  const permissions = {
    read: hasAuth('generic#security:authorization/read'),
    create: hasAuth('generic#security:authorization/create'),
    delete: hasAuth('generic#security:authorization/delete'),
    update: hasAuth('generic#security:authorization/update')
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
    const res = await fetchAuthorizationSearch(params);
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
      const res = await deleteAuthorization(id);
      if (res.data?.result === 'deleted') {
        message.success(t('common.deleteSuccess'));
      }
      fetchData(queryParams);
      endLoading();
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [queryParams, t]
  );

  const {
      data: principalRes,
      loading: principalLoading,
      run: runPrincipalSearch
    } = useRequest(fetchPrincipalSearch, { manual: true });

  useEffect(() => {
    runPrincipalSearch({
      from: 0,
      size: 1000
    });
  }, []);
  const principals = useMemo(() => {
    const rs = formatESSearchResult(principalRes?.data);
    const rsMap: any = {};
    (rs.data || []).forEach(item => {
      rsMap[item.id] = item;
    });
    return rsMap;
  }, [principalRes]);

  const columns: TableColumnsType<any> = [
    {
      dataIndex: 'display_name',
      title: t('page.auth.labels.auth'),
      render: (value, record) => {
        return (
          <div className='flex items-center'>
            {(record.principal_type === 'user' || record.principal_type === 'team') && principals[record.principal_id] && <img src={principals[record.principal_id].avatar} className='rounded-full inline-block mr-8px' style={{ width: 16, height: 16 }} />}
            <Typography.Link onClick={()=>nav(`/auth/edit/${record.id}`, {state:record})}>
              {value}
            </Typography.Link>
          </div>
          
        )
      }
    },
    {
      dataIndex: 'roles',
      title: t('page.auth.labels.roles'),
      render: (value) => {
        if (!Array.isArray(value)) return '-'
        return value.map((tag, index) => {
          return <Tag key={index}>{tag}</Tag>;
        });
      }
    },
    {
      dataIndex: 'created',
      title: t('page.auth.labels.created'),
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
        const onMenuClick = ({ key, record }: any) => {
          switch (key) {
            case 'edit':
              nav(`/auth/edit/${record.id}`, { state: record });
              break;
            case 'delete':
              window?.$modal?.confirm({
                content: t('page.auth.delete.confirm', { name: record.display_name }),
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
            onClick={() => nav(`/auth/new`)}
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

export default Auth;
