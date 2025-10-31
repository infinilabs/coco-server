import Icon, { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined, UserOutlined } from '@ant-design/icons';
import { Avatar, Button, Dropdown, Image, Modal, Switch, Table, message } from 'antd';
import type { GetProp, MenuProps, TableColumnsType, TableProps } from 'antd';
import Search from 'antd/es/input/Search';
import type { SorterResult } from 'antd/es/table/interface';

import InfiniIcon from '@/components/common/icon';
import { GoogleDriveSVG, HugoSVG, NotionSVG, YuqueSVG } from '@/components/icons';
import { deleteDatasource, fetchDataSourceList, getConnectorByIDs, updateDatasource } from '@/service/api';
import { formatESSearchResult } from '@/service/request/es';
import useQueryParams from '@/hooks/common/queryParams';
import Shares from '../modules/Shares';
import { fetchBatchShares } from '@/service/api/share';
import { fetchBatchEntityLabels } from '@/service/api/entity';
import AvatarLabel from '../modules/AvatarLabel';
import { groupBy, keys, map, uniq } from "lodash";
import { selectUserInfo } from '@/store/slice/auth';

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

export function formatDataForShare(item: any, shares: any, entities: any, currentUser: any) {
  const hasEntities = entities?.length > 0
  if (hasEntities) {
    if (shares?.length > 0) {
      item.shares = shares.filter((s) => s.resource_id === item.id).map((item) => ({
        ...item,
        entity: entities.find((o) => o.id === item.principal_id)
      }))
    } else {
      item.shares = []
    }
    if (item._system?.owner_id) {
      item.owner = entities.find((o) => o.id === item._system?.owner_id)
    }
    if (currentUser?.id) {
      item.editor = entities.find((o) => o.id === currentUser?.id)
    }
  }
}

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();

  const { t } = useTranslation();

  const userInfo = useAppSelector(selectUserInfo);

  const { hasAuth } = useAuth()

  const permissions = {
    read: hasAuth('coco#datasource/read'),
    create: hasAuth('coco#datasource/create'),
    update: hasAuth('coco#datasource/update'),
    delete: hasAuth('coco#datasource/delete'),
    shares: hasAuth('generic#sharing/search'),
    entityLabel: hasAuth('generic#entity:label/read')
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
      minWidth: 200,
      render: (value: string, record: Datasource) => {
        let iconSrc = record.icon;
        if(!iconSrc && data.connectors) {
          iconSrc = data.connectors[record.connector.id]?.icon;
        }
        if (!iconSrc) return value;
        const content = (
          <>
            <IconWrapper className="w-20px h-20px">
              <InfiniIcon
                height="1em"
                src={iconSrc}
                width="1em"
              />
            </IconWrapper>
            {value}
          </>
        )
        if (permissions.read) {
          return (
            <a
              className="inline-flex items-center gap-1 text-blue-500"
              onClick={() =>
                nav(`/data-source/detail/${record.id}`, {
                  state: { connector_id: record.connector?.id || '', datasource_name: record.name }
                })
              }
            >
              {content}
            </a>
          );
        }
        return (
          <span className="inline-flex items-center gap-1" >
            {content}
          </span>
        );
      },
      title: t('page.datasource.columns.name')
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
            onSuccess={() => fetchData()}
            resourceType={'datasource'}
            resourceID={record.id}
          />
        )
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
      },
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
        if (permissions.read && permissions.update) {
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
  ].filter((item) => !!item);
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
      const resources = newData.data.filter((item) => !!item.connector?.id).map((item) => ({
        "resource_id": item.id,
        "resource_type": 'datasource'
      }))
      let shareRes: any;
      if (permissions.shares) {
        shareRes = await fetchBatchShares(resources)
      }
      let entityRes: any
      if (permissions.entityLabel) {
        const entities = newData.data.filter((item) => !!item._system?.owner_id).map((item) => ({
          type: 'user',
          id: item._system.owner_id
        }))
        if (shareRes?.data?.length > 0) {
          entities.push(...shareRes?.data.map((item) => ({ type: item.principal_type, id: item.principal_id })))
        }
        if (userInfo?.id) {
          entities.push({
            type: 'user',
            id: userInfo.id
          })
        }
        const grouped = groupBy(entities, 'type');
        const body = map(keys(grouped), (type) => ({
            type,
            id: uniq(map(grouped[type], 'id')) 
        }))
        entityRes = await fetchBatchEntityLabels(body)
      }
      newData.data.forEach((item, index) => {
        formatDataForShare(item, shareRes?.data, entityRes?.data, userInfo)
      })
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
    fetchData()
  }, [queryParams]);

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
