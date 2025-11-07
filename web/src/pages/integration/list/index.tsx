import { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { useLoading } from '@sa/hooks';
import { Button, Dropdown, Input, Switch, Table, Tabs, message } from 'antd';
import { useSearchParams } from 'react-router-dom';

import SvgIcon from '@/components/stateless/custom/SvgIcon';
import useQueryParams from '@/hooks/common/queryParams';
import { deleteIntegration, fetchIntegrations, renewAPIToken, updateIntegration } from '@/service/api/integration';
import { formatESSearchResult } from '@/service/request/es';

import { isFullscreen } from '../modules/EditForm';

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();
  const { t } = useTranslation();

  const { tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

  const [data, setData] = useState({
    data: [],
    total: 0
  });
  const { endLoading, loading, startLoading } = useLoading();
  const [keyword, setKeyword] = useState();

  // 用于判断 Webhooks 类型
  const isWebhook = (type?: string) => ['webhook', 'webhooks'].includes(String(type || '').toLowerCase());

  const fetchData = async params => {
    startLoading();
    const res = await fetchIntegrations(params);
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
    const res = await deleteIntegration(id);
    if (res.data?.result === 'deleted') {
      message.success(t('common.deleteSuccess'));
    }
    fetchData(queryParams);
    endLoading();
  };

  const handleEnabled = async record => {
    startLoading();
    const { _index, _type, ...rest } = record;
    const res = await updateIntegration(rest);
    if (res.data?.result === 'updated') {
      message.success(t('common.updateSuccess'));
    }
    fetchData(queryParams);
    endLoading();
  };

  const columns = [
    {
      dataIndex: 'name',
      render: value => (
        <div className="flex items-center gap-2">
          <SvgIcon
            className="text-icon-small text-gray-500"
            icon="mdi:puzzle-outline"
          />
          <span>{value}</span>
        </div>
      ),
      title: t('page.integration.columns.name')
    },
    {
      dataIndex: 'type',
      render: value => {
        // Webhooks 优先匹配
        if (isWebhook(value)) {
          return t('page.integration.tabs.webhooks');
        }
        return isFullscreen(value)
          ? t('page.integration.form.labels.type_fullscreen')
          : t('page.integration.form.labels.type_searchbox');
      },
      title: t('page.integration.columns.type')
    },
    {
      dataIndex: 'description',
      title: t('page.integration.columns.description')
    },
    {
      dataIndex: 'datasource',
      render: (value, record) => {
        if (record.datasource?.length) {
          return record.datasource?.includes('*') ? '*' : value?.length || 0;
        }
        if (record.enabled_module?.search?.datasource?.length) {
          return record.enabled_module?.search?.datasource?.includes('*')
            ? '*'
            : record.enabled_module.search.datasource?.length || 0;
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
      render: (value: number, record: any) => {
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

        // eslint-disable-next-line @typescript-eslint/no-shadow
        const onMenuClick = ({ key, record }: any) => {
          // eslint-disable-next-line default-case
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
              renewAPIToken(record.id)
                .then(res => {
                  if (res.data?.result === 'acknowledged') {
                    message.success(t('common.updateSuccess'));
                  }
                })
                .finally(() => {
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
    fetchData(queryParams);
  }, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query);
  }, [queryParams.query]);

  // 新增：Tabs 状态（与 settings 页面一致）
  const [searchParams, setSearchParams] = useSearchParams();
  const items = [
    {
      key: 'searchbox',
      label: t('page.integration.form.labels.type_searchbox')
    },
    {
      key: 'fullscreen',
      label: t('page.integration.form.labels.type_fullscreen')
    }
    // {
    //   key: 'webhooks',
    //   label: t('page.integration.tabs.webhooks'),
    // }
  ];
  const activeKey = useMemo(() => {
    return searchParams.get('tab') || items[0].key;
  }, [searchParams]);
  const onTabChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  // 新增：按当前 tab 过滤数据源
  const filteredData = useMemo(() => {
    const list = data?.data || [];
    if (activeKey === 'fullscreen') {
      return { ...data, data: list.filter((item: any) => isFullscreen(item?.type)) };
    }
    if (activeKey === 'searchbox') {
      return { ...data, data: list.filter((item: any) => !isFullscreen(item?.type)) };
    }
    if (activeKey === 'webhooks') {
      return { ...data, data: list.filter((item: any) => isWebhook(item?.type)) };
    }
    return data;
  }, [data, activeKey]);

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
      >
        {/* 新增：Tabs 切换 SearchBox / Fullscreen / Webhooks */}
        <Tabs
          activeKey={activeKey}
          className="settings-tabs"
          items={items}
          onChange={onTabChange}
        />

        <div className="mb-4 mt-4 flex items-center justify-between">
          <Input.Search
            addonBefore={<FilterOutlined />}
            className="max-w-500px"
            enterButton={t('common.refresh')}
            value={keyword}
            onChange={e => setKeyword(e.target.value)}
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
          dataSource={filteredData.data}
          loading={loading}
          rowKey="id"
          rowSelection={{ ...rowSelection }}
          size="middle"
          pagination={{
            current: Math.floor(queryParams.from / queryParams.size) + 1,
            pageSize: queryParams.size,
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
