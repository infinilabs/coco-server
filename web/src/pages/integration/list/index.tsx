import { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { useLoading } from '@sa/hooks';
import { Button, Dropdown, Input, Modal, Switch, Table, message } from 'antd';

import { deleteIntegration, fetchIntegrations, updateIntegration, renewAPIToken } from '@/service/api/integration';
import { formatESSearchResult } from '@/service/request/es';

export function Component() {
  const { t } = useTranslation();

  const { tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

  const [data, setData] = useState({
    data: [],
    total: 0
  });
  const { endLoading, loading, startLoading } = useLoading();

  const [reqParams, setReqParams] = useState({
    from: 0,
    query: '',
    size: 10
  });

  const fetchData = async reqParams => {
    startLoading();
    const res = await fetchIntegrations(reqParams);
    const newData = formatESSearchResult(res.data);
    setData(newData);
    endLoading();
  };

  const handleTableChange = pagination => {
    setReqParams(params => {
      return {
        ...params,
        from: (pagination.current - 1) * pagination.pageSize,
        size: pagination.pageSize
      };
    });
  };

  const onRefreshClick = (query: string) => {
    setReqParams(oldParams => {
      return {
        ...oldParams,
        from: 0,
        query
      };
    });
  };

  const handleDelete = async id => {
    startLoading();
    const res = await deleteIntegration(id);
    if (res.data?.result === 'deleted') {
      message.success(t('common.deleteSuccess'));
    }
    fetchData(reqParams);
    endLoading();
  };

  const handleEnabled = async record => {
    startLoading();
    const { _index, _type, ...rest } = record;
    const res = await updateIntegration(rest);
    if (res.data?.result === 'updated') {
      message.success(t('common.updateSuccess'));
    }
    fetchData(reqParams);
    endLoading();
  };

  const columns = [
    {
      dataIndex: 'name',
      title: t('page.integration.columns.name')
    },
    {
      dataIndex: 'type',
      render: value => t(`page.integration.form.labels.type_${value}`),
      title: t('page.integration.columns.type')
    },
    {
      dataIndex: 'description',
      title: t('page.integration.columns.description')
    },
    {
      dataIndex: 'datasource',
      render: (value, record) => {
        if(record.datasource?.length){
          return record.datasource?.includes('*') ? '*' : value?.length || 0;
        }
        if(record.enabled_module?.search?.datasource?.length){
          return record.enabled_module?.search?.datasource?.includes('*') ? '*' : record.enabled_module.search.datasource?.length || 0;
        }
        return 0;
      },
      title: t('page.integration.columns.datasource')
    },
    {
      dataIndex: 'enabled',
      render: (_, record) => {
        return (
          <Switch
            checked={record.enabled}
            size="small"
            onChange={checked => {
              window?.$modal?.confirm({
                content: t(`page.integration.update.${checked ? 'enable' : 'disable'}_confirm`, { name: record.name }),
                icon: <ExclamationCircleOutlined />,
                onOk() {
                  handleEnabled({ ...record, enabled: checked });
                },
                title: t('common.tip')
              });
            }}
          />
        );
      },
      title: t('page.integration.columns.enabled')
    },
    {
      dataIndex: 'token_expire_in',
      render: (value: number, record:any) => {
        return value ? new Date(value * 1000).toISOString() : '';
      },
      title: t('page.integration.columns.token_expire_in')
    },
    {
      fixed: 'right',
      render: (_, record) => {
        const items = [
          {
            key: 'edit',
            label: t('common.edit')
          },
          {
            key: 'delete',
            label: t('common.delete')
          },
          {
            key: 'renew_token',
            label: t('common.renew_token')
          }
        ];

        const onMenuClick = ({ key, record }: any) => {
          switch (key) {
            case 'edit':
              nav(`/integration/edit/${record.id}`, { state: record });
              break;
            case 'delete':
              window?.$modal?.confirm({
                content: t('page.integration.delete.confirm', { name: record.name }),
                icon: <ExclamationCircleOutlined />,
                onOk() {
                  handleDelete(record.id);
                },
                title: t('common.tip')
              });
              break;
            case 'renew_token':
              startLoading();
              renewAPIToken(record.id).then(res => {
                if (res.data?.result === 'acknowledged') {
                  message.success(t('common.updateSuccess'));
                }
              }).finally(() => {
                endLoading();
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
    fetchData(reqParams);
  }, [reqParams]);

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
      >
        <div className="mb-4 mt-4 flex items-center justify-between">
          <Input.Search
            addonBefore={<FilterOutlined />}
            className="max-w-500px"
            enterButton={t('common.refresh')}
            onSearch={onRefreshClick}
          />
          <Button
            icon={<PlusOutlined />}
            type="primary"
            onClick={() => nav(`/integration/new`)}
          >
            {t('common.add')}
          </Button>
        </div>
        <Table
          columns={columns}
          dataSource={data.data}
          loading={loading}
          rowKey="id"
          rowSelection={{ ...rowSelection }}
          size="middle"
          pagination={{
            defaultCurrent: 1,
            defaultPageSize: 10,
            showSizeChanger: true,
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
            total: data.total?.value || data?.total
          }}
          onChange={handleTableChange}
        />
      </ACard>
    </ListContainer>
  );
}