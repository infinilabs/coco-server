import { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Avatar, Button, Dropdown, Switch, Table, message } from 'antd';
import type { MenuProps, TableColumnsType, TableProps } from 'antd';
import Search from 'antd/es/input/Search';

import InfiniIcon from '@/components/common/icon';
import { GoogleDriveSVG, HugoSVG, NotionSVG, YuqueSVG } from '@/components/icons';
import { deleteDatasource, fetchDataSourceList, getConnectorByIDs, updateDatasource } from '@/service/api';
import { formatESSearchResult } from '@/service/request/es';
import useQueryParams from '@/hooks/common/queryParams';

type Datasource = Api.Datasource.Datasource;

const TYPES = {
  google_drive: {
    icon: GoogleDriveSVG,
    name: 'Google Drive'
  },
  hugo_site: {
    icon: HugoSVG,
    name: 'Hugo Site'
  },
  notion: {
    icon: NotionSVG,
    name: 'Notion'
  },
  yuque: {
    icon: YuqueSVG,
    name: 'Yuque'
  }
};

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();

  const { t } = useTranslation();

  const { addSharesToData, isEditorOwner, hasEdit, hasView, isResourceShare } = useResource();
  const resourceType = 'datasource';
  const resourceCategoryType = 'connector';

  const { hasAuth } = useAuth();

  const permissions = {
    read: hasAuth('coco#datasource/read'),
    readConnector: hasAuth('coco#connector/read'),
    create: hasAuth('coco#datasource/create'),
    update: hasAuth('coco#datasource/update'),
    delete: hasAuth('coco#datasource/delete')
  };

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '1':
        window?.$modal?.confirm({
          content: t('page.datasource.delete.confirm'),
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            // delete datasource by datasource id
            deleteDatasource(record.id).then(res => {
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
      case '2':
        nav(`/data-source/edit/${record.id}`, { state: record });
        break;
    }
  };
  const onSyncEnabledChange = (value: boolean, record: Datasource) => {
    setLoading(true);

    // Ensure we preserve all existing sync configuration values
    const currentSync = record.sync || {
      enabled: false,
      strategy: 'interval',
      interval: '1h'
    };

    // Ensure all required fields are present
    const completeSync = {
      enabled: currentSync.enabled || false,
      strategy: currentSync.strategy || 'interval',
      interval: currentSync.interval || '1h'
    };

    // Create the updated sync config, preserving existing values
    const updatedSync = {
      ...completeSync,
      enabled: value
    };

    updateDatasource(record.id, {
      ...record,
      sync: updatedSync
    })
      .then(res => {
        if (res.data?.result === 'updated') {
          message.success(t('common.updateSuccess'));
        }
        // reload data
        setQueryParams(old => {
          return {
            ...old,
            t: new Date().valueOf()
          };
        });
      })
      .catch(err => {
        console.error('Sync update failed:', err);
        message.error(t('common.updateFailed'));
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const onEnabledChange = (value: boolean, record: Datasource) => {
    setLoading(true);
    updateDatasource(record.id, {
      ...record,
      enabled: value
    })
      .then(res => {
        if (res.data?.result === 'updated') {
          message.success(t('common.updateSuccess'));
        }
        // reload data
        setQueryParams(old => {
          return {
            ...old,
            t: new Date().valueOf()
          };
        });
      })
      .finally(() => {
        setLoading(false);
      });
  };
  const columns: TableColumnsType<Datasource> = [
    {
      dataIndex: 'name',
      minWidth: 150,
      ellipsis: true,
      render: (value: string, record: Datasource) => {
        let iconSrc = record.icon;
        if (!iconSrc && data.connectors) {
          iconSrc = data.connectors[record.connector.id]?.icon;
        }

        const isShare = isResourceShare(record);

        let shareIcon;

        if (isShare) {
          shareIcon = (
            <div className='flex-shrink-0 flex-grow-0'>
              <SvgIcon
                className='text-#999'
                localIcon='share'
              />
            </div>
          );
        }

        return (
          <div className='flex items-center gap-1'>
            {iconSrc && (
              <IconWrapper className='h-20px w-20px flex-shrink-0 flex-grow-0 flex-basis-auto'>
                <InfiniIcon
                  height='1em'
                  src={iconSrc}
                  width='1em'
                />
              </IconWrapper>
            )}
            {permissions.read && permissions.readConnector ? (
              <a
                className='ant-table-cell-ellipsis max-w-150px cursor-pointer text-[var(--ant-color-link)]'
                onClick={() =>
                  nav(`/data-source/detail/${record.id}${isEditorOwner(record) ? '' : '?view=list'}`, {
                    state: { connector_id: record.connector?.id || '', datasource_name: record.name }
                  })
                }
              >
                {value}
              </a>
            ) : (
              <span className='ant-table-cell-ellipsis max-w-150px'>{value}</span>
            )}
            {shareIcon}
          </div>
        );
      },
      title: t('page.datasource.columns.name')
    },
    {
      dataIndex: 'owner',
      title: t('page.datasource.labels.owner'),
      render: (value, record) => {
        if (!value) return '-';
        return (
          <div className='flex overflow-hidden'>
            <Avatar.Group
              max={{ count: 1 }}
              size='small'
            >
              <AvatarLabel
                data={value}
                showCard={true}
              />
            </Avatar.Group>
          </div>
        );
      }
    },
    {
      dataIndex: 'shares',
      title: t('page.datasource.labels.shares'),
      render: (value, record) => {
        if (!value) return '-';
        return (
          <Shares
            record={record}
            title={record.name}
            resource={{
              resource_category_type: resourceCategoryType,
              resource_category_id: record.connector?.id,
              resource_type: resourceType,
              resource_id: record.id
            }}
            onSuccess={() => fetchData()}
          />
        );
      }
    },
    {
      minWidth: 100,
      render: (text: string, record: Datasource) => {
        const type = TYPES[record?.connector?.id];
        if (!type) return data.connectors[record.connector.id]?.name || record.connector.id;
        return type.name;
      },
      title: t('page.datasource.columns.type')
    },
    // {
    //   dataIndex: ['sync', 'enabled'],
    //   title: t('page.datasource.labels.permission_sync'),
    //   render: (value: number) => {
    //     return value ? t('page.datasource.labels.isEnabled') : '-'
    //   },
    // },
    // {
    //   dataIndex: 'strategy',
    //   title: t('page.datasource.columns.sync_policy'),
    // },
    {
      dataIndex: 'updated',
      title: t('page.datasource.labels.updated'),
      render: (value: number) => {
        return value ? new Date(value).toISOString() : '';
      }
    },
    // {
    //   dataIndex: 'sync_status',
    //   title: t('page.datasource.columns.sync_status'),
    // },
    {
      dataIndex: ['sync', 'enabled'],
      render: (value: boolean, record: Datasource) => {
        return (
          <Switch
            checked={value}
            disabled={!permissions.update || !hasEdit(record)}
            size='small'
            onChange={v => onSyncEnabledChange(v, record)}
          />
        );
      },
      title: t('page.datasource.new.labels.sync_enabled'),
      width: 200
    },
    {
      dataIndex: 'enabled',
      render: (value: boolean, record: Datasource) => {
        return (
          <Switch
            disabled={!permissions.update || !hasEdit(record)}
            size='small'
            value={value}
            onChange={v => onEnabledChange(v, record)}
          />
        );
      },
      title: t('page.datasource.new.labels.enabled'),
      width: 200
    },
    {
      fixed: 'right',
      hidden: !permissions.update && !permissions.delete,
      render: (_, record) => {
        const items: MenuProps['items'] = [];
        if (permissions.read && permissions.update && hasEdit(record)) {
          items.push({
            key: '2',
            label: t('common.edit')
          });
        }
        if (permissions.delete && isEditorOwner(record)) {
          items.push({
            key: '1',
            label: t('common.delete')
          });
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
  ].filter(item => Boolean(item));
  // rowSelection object indicates the need for row selection
  const rowSelection: TableProps<Datasource>['rowSelection'] = {
    getCheckboxProps: (record: Datasource) => ({
      name: record.name
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows: Datasource[]) => {}
  };

  const initialData = {
    connectors: {},
    data: [],
    total: 0
  };
  const [data, setData] = useState(initialData);
  const [loading, setLoading] = useState(false);

  const [keyword, setKeyword] = useState();

  const fetchData = async () => {
    setLoading(true);
    const res = await fetchDataSourceList(queryParams);
    if (res?.data) {
      const newData = formatESSearchResult(res?.data);
      if (newData.data.length > 0) {
        const resources = newData.data.map(item => ({
          resource_id: item.id,
          resource_type: resourceType,
          resource_category_type: resourceCategoryType,
          resource_category_id: item.connector?.id
        }));
        const dataWithShares = await addSharesToData(newData.data, resources);
        if (dataWithShares) {
          newData.data = dataWithShares;
        }
      }
      setData((oldData: any) => {
        return {
          ...oldData,
          ...(newData || initialData)
        };
      });
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData();
  }, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query);
  }, [queryParams.query]);

  const fetchConnectors = async (ids: string[]) => {
    const res = await getConnectorByIDs(ids);
    if (res.data) {
      const newData = formatESSearchResult(res.data);
      const connectors: any = {};
      newData.data.map(item => {
        connectors[item.id] = item;
      });
      setData(data => {
        return {
          ...data,
          connectors
        };
      });
    }
  };
  useEffect(() => {
    if (data.data?.length > 0) {
      const ids = data.data.map(item => item.connector.id);
      fetchConnectors(ids);
    }
  }, [data.data]);

  const handleTableChange: TableProps<Datasource>['onChange'] = (pagination, filters, sorter) => {
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
        from: query === oldParams.query ? oldParams.from : 0,
        query,
        t: new Date().valueOf()
      };
    });
  };

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className='flex-col-stretch sm:flex-1-hidden card-wrapper'
        ref={tableWrapperRef}
      >
        <div className='mb-4 mt-4 flex items-center justify-between'>
          <Search
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
              onClick={() => nav(`/data-source/new-first`)}
            >
              {t('common.add')}
            </Button>
          )}
        </div>
        <Table<Datasource>
          columns={columns}
          dataSource={data.data}
          loading={loading}
          rowKey='id'
          rowSelection={{ ...rowSelection }}
          size='middle'
          pagination={{
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
            pageSize: queryParams.size,
            current: Math.floor(queryParams.from / queryParams.size) + 1,
            total: data.total?.value || data?.total,
            showSizeChanger: true
          }}
          onChange={handleTableChange}
        />
      </ACard>
    </ListContainer>
  );
}
