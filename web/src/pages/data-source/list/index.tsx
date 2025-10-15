import Icon, { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Dropdown, Image, Modal, Switch, Table, message } from 'antd';
import type { GetProp, MenuProps, TableColumnsType, TableProps } from 'antd';
import Search from 'antd/es/input/Search';
import type { SorterResult } from 'antd/es/table/interface';

import InfiniIcon from '@/components/common/icon';
import { GoogleDriveSVG, HugoSVG, NotionSVG, YuqueSVG } from '@/components/icons';
import { deleteDatasource, fetchDataSourceList, getConnectorByIDs, updateDatasource } from '@/service/api';
import { formatESSearchResult } from '@/service/request/es';
import useQueryParams from '@/hooks/common/queryParams';

type Datasource = Api.Datasource.Datasource;

type TablePaginationConfig = Exclude<GetProp<TableProps, 'pagination'>, boolean>;

interface TableParams {
  filters?: Parameters<GetProp<TableProps, 'onChange'>>[1];
  pagination?: TablePaginationConfig;
  sortField?: SorterResult<any>['field'];
  sortOrder?: SorterResult<any>['order'];
}

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

  const { hasAuth } = useAuth()

  const permissions = {
    create: hasAuth('coco:datasource/create'),
    update: hasAuth('coco:datasource/update'),
    delete: hasAuth('coco:datasource/delete'),
  }

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

    console.log('Updating sync config:', updatedSync); // Debug log

    updateDatasource(record.id, {
      ...record,
      sync: updatedSync
    })
      .then(res => {
        if (res.data?.result === 'updated') {
          message.success(t('common.updateSuccess'));
          console.log('Sync update response:', res.data); // Debug log
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
      minWidth: 200,
      render: (value: string, record: Datasource) => {
        let iconSrc = record.icon;
        if(!iconSrc && data.connectors) {
          iconSrc = data.connectors[record.connector.id]?.icon;
        }
        if (!iconSrc) return value;
        return (
          <a
            className="inline-flex items-center gap-1 text-blue-500"
            onClick={() =>
              nav(`/data-source/detail/${record.id}`, {
                state: { connector_id: record.connector?.id || '', datasource_name: record.name }
              })
            }
          >
            <IconWrapper className="w-20px h-20px">
              <InfiniIcon
                height="1em"
                src={iconSrc}
                width="1em"
              />
            </IconWrapper>
            {value}
          </a>
        );
      },
      title: t('page.datasource.columns.name')
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
    {
      dataIndex: ['sync', 'enabled'],
      render: (value: boolean, record: Datasource) => {
        return (
          <Switch
            size="small"
            checked={value}
            onChange={v => onSyncEnabledChange(v, record)}
            disabled={!permissions.update}
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
            size="small"
            value={value}
            onChange={v => onEnabledChange(v, record)}
            disabled={!permissions.update}
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
        if (permissions.update) {
          items.push({
            key: '2',
            label: t('common.edit')
          })
        }
        if (permissions.delete) {
          items.push({
            key: '1',
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

  const fetchData = () => {
    setLoading(true);
    fetchDataSourceList(queryParams).then(({ data }) => {
      const newData = formatESSearchResult(data);
      setData((oldData: any) => {
        return {
          ...oldData,
          ...(newData || initialData)
        };
      });
      setLoading(false);
    });
  };

  useEffect(fetchData, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query)
  }, [queryParams.query])

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
        from: 0,
        query,
        t: new Date().valueOf()
      };
    });
  };

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
      >
        <div className="mb-4 mt-4 flex items-center justify-between">
          <Search
            value={keyword} 
            onChange={(e) => setKeyword(e.target.value)} 
            addonBefore={<FilterOutlined />}
            className="max-w-500px"
            enterButton={t('common.refresh')}
            onSearch={onRefreshClick}
          />
          {
            permissions.create && (
              <Button
                icon={<PlusOutlined />}
                type="primary"
                onClick={() => nav(`/data-source/new-first`)}
              >
                {t('common.add')}
              </Button>
            )
          }
        </div>
        <Table<Datasource>
          columns={columns}
          dataSource={data.data}
          loading={loading}
          rowKey="id"
          rowSelection={{ ...rowSelection }}
          size="middle"
          pagination={{
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
            pageSize: queryParams.size,
            current: Math.floor(queryParams.from / queryParams.size) + 1,
            total: data.total?.value || data?.total,
            showSizeChanger: true,
          }}
          onChange={handleTableChange}
        />
      </ACard>
    </ListContainer>
  );
}
