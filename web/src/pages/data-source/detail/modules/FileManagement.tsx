import { DownOutlined, EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, UserOutlined } from '@ant-design/icons';
import type { MenuProps, TableColumnsType, TableProps } from 'antd';
import { Avatar, Dropdown, Switch, Table, message } from 'antd';
import Search from 'antd/es/input/Search';

import FontIcon from '@/components/common/font_icon';
import InfiniIcon from '@/components/common/icon';
import {
  batchDeleteDocument,
  deleteDocument,
  fetchDatasourceDetail,
  getConnector,
  getDatasource,
  updateDocument
} from '@/service/api';
import { formatESSearchResult } from '@/service/request/es';
import useQueryParams from '@/hooks/common/queryParams';
import { fetchBatchShares } from '@/service/api/share';
import { fetchBatchEntity } from '@/service/api/entity';
import Shares from '../../modules/Shares';

interface DataType {
  category: string;
  disabled: boolean;
  icon: string;
  id: string;
  is_dir: boolean;
  searchable: boolean;
  subcategory: string;
  tags: string[];
  title: string;
  type: string;
  url: string;
}

const FileManagement = (props) => {
  const datasourceID = props.id
  const [queryParams, setQueryParams] = useQueryParams();

  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    update: hasAuth('coco#datasource/update'),
    delete: hasAuth('coco#datasource/delete'),
  }
  
  const [connector, setConnector] = useState<any>({});
  const [datasource, setDatasource] = useState<any>();

  useEffect(() => {
    if (!datasourceID) return;
    getDatasource(datasourceID).then(res => {
      if (res.data?.found === true) {
        setDatasource(res.data._source || {});
      }
    });
  }, [datasourceID]);
  useEffect(() => {
    if (!datasource?.connector?.id) return;
    getConnector(datasource?.connector?.id).then(res => {
      if (res.data?.found === true) {
        setConnector(res.data._source || {});
      }
    });
  }, [datasource?.connector?.id]);
  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '1':
        window?.$modal?.confirm({
          content: 'Are you sure you want to delete this document?',
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            deleteDocument(record.id).then(res => {
              if (res.data?.result === 'deleted') {
                message.success('deleted success');
              }
              // reload data
              setQueryParams(old => {
                return {
                  ...old,
                  t: new Date().valueOf()
                };
              });
            });
          }
        });

        break;
    }
  };
  const [state, setState] = useState({
    selectedRowKeys: []
  });
  // rowSelection object indicates the need for row selection
  const rowSelection: TableProps<DataType>['rowSelection'] = {
    getCheckboxProps: (record: DataType) => ({
      // Column configuration not to be checked
      name: record.title
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows: DataType[]) => {
      setState((st: any) => {
        return {
          ...st,
          selectedRowKeys
        };
      });
    },
    selectedRowKeys: state.selectedRowKeys
  };
  const onSearchableChange = (checked: boolean, record: DataType) => {
    // update searchable status
    record.disabled = !checked;
    updateDocument(record.id, record).then(res => {
      if (res.data?.result === 'updated') {
        message.success('updated success');
      }
      // reload data
      setQueryParams(old => {
        return {
          ...old,
          t: new Date().valueOf()
        };
      });
    });
  };
  const items: MenuProps['items'] = [
    {
      key: '1',
      label: t('common.delete')
    }
  ];
  const onBatchMenuClick = useCallback(
    ({ key }: any) => {
      switch (key) {
        case '1':
          window?.$modal?.confirm({
            content: 'Are you sure you want to delete theses documents?',
            icon: <ExclamationCircleOutlined />,
            onCancel() {},
            onOk() {
              if (state.selectedRowKeys?.length === 0) {
                message.error('Please select at least one document');
                return;
              }
              setLoading(true);
              batchDeleteDocument(state.selectedRowKeys)
                .then(res => {
                  if (res.data?.result === 'acknowledged') {
                    setState((st: any) => {
                      return {
                        ...st,
                        selectedRowKeys: []
                      };
                    });
                    message.success('submit success');
                  }
                  // reload data
                  setTimeout(() => {
                    setQueryParams(old => {
                      return {
                        ...old,
                        t: new Date().valueOf()
                      };
                    });
                  }, 1000);
                })
                .finally(() => {
                  setLoading(false);
                });
            }
          });

          break;
      }
    },
    [state.selectedRowKeys]
  );

  const columns: TableColumnsType<DataType> = useMemo(
    () => {
      return [
        {
          dataIndex: 'title',
          render: (text: string, record: DataType) => {
            let imgSrc = '';
            if (connector?.assets?.icons) {
              imgSrc = connector.assets.icons[record.icon];
            }
            const aProps = {
              className: "text-blue-500",
              rel: "noreferrer",
            }
            if (record.type === 'folder') {
              aProps.onClick = () => {
                const categories = (record.categories || []).concat([record.title])
                setQueryParams(old => {
                  return {
                    ...old,
                    path: JSON.stringify(categories),
                  }
                })
              } 
            } else if (record.url) {
              aProps.href = record.url;
              aProps.target = "_blank";
            }
            return (
              <span className="inline-flex items-center gap-1">
                {imgSrc ? (
                  <IconWrapper className="w-20px h-20px">
                    <InfiniIcon
                      height="1em"
                      src={imgSrc}
                      width="1em"
                    />
                  </IconWrapper>
                ) : <FontIcon name={record.icon} />}
                { record.url || record.type === 'folder' ? <a {...aProps}>{text}</a> : <span>{text}</span> }
              </span>
            );
          },
          title: t('page.datasource.columns.name')
        },
        {
          dataIndex: 'owner',
          title: t('page.datasource.labels.owner'),
          render: (value, record) => {
            return <Avatar size={"small"} icon={<UserOutlined />} />
          }
        },
        {
          dataIndex: 'shares',
          title: t('page.datasource.labels.shares'),
          render: (value, record) => {
            const isFolder = record.type === 'folder';
            return (
              <Shares
                datasource={datasource} 
                record={record} 
                title={record.title} 
                onSuccess={() => fetchData(queryParams, datasource)}
                resourceType={datasource?.connector?.id}
                resourceID={isFolder ? datasource?.id : record.id}
                resourcePath={isFolder ? `/${(record.categories || []).concat([record.title]).join('/')}` : undefined}
              />
            )
          }
        },
        // {
        //   dataIndex: 'updated',
        //   title: t('page.datasource.labels.updated')
        // },
        // {
        //   dataIndex: 'size',
        //   title: t('page.datasource.labels.size')
        // },
        {
          dataIndex: 'disabled',
          render: (text: boolean, record: DataType) => {
            return (
              <Switch
                size="small"
                value={!text}
                onChange={v => {
                  onSearchableChange(v, record);
                }}
                disabled={!permissions.update}
              />
            );
          },
          title: t('page.datasource.columns.searchable')
        },
        {
          fixed: 'right',
          hidden: !permissions.delete,
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
      ].filter((item) => !!item)
    },
    [connector]
  );

  if (!datasourceID) return <LookForward />;

  const [data, setData] = useState({});
  const [loading, setLoading] = useState(false);

  const [keyword, setKeyword] = useState();

  const fetchData = async (queryParams, datasource) => {
    if (!datasource) return;
    setLoading(true);
    const { filter = {} } = queryParams || {};
    const res = await fetchDatasourceDetail({
      ...queryParams,
      filter: {
        ...filter,
        'source.id': [datasourceID],
      }
    })
    if (res?.data) {
      const newData = formatESSearchResult(res.data);
      if (datasource.connector?.id) {
        const resources = newData.data.map((item) => ({
          "resource_id": item.id,
          "resource_type": datasource.connector.id
        }))
        const shareRes = await fetchBatchShares(resources)
        const ownerRes = await fetchBatchEntity([{
          type: 'user',
          id: newData.data.filter((item) => !!item._system?.owner_id).map((item) => item._system.owner_id)
        }])
        newData.data.forEach((item, index) => {
          if (shareRes?.data?.length > 0) {
            item.shares = shareRes.data.filter((s) => s.resource_id === item.id)
          }
          if (ownerRes?.data && item._system?.owner_id) {
            item.owner = ownerRes?.data[item._system.owner_id]
          }
        })
      }
      setData(newData);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData(queryParams, datasource)
  }, [queryParams, datasource]);

  useEffect(() => {
    setKeyword(queryParams.query)
  }, [queryParams.query])

  const onTableChange = (pagination, filters, sorter, extra: { action; currentDataSource: [] }) => {
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
  
  const renderTitle = (datasource) => {
    let paths
    try {
      paths = JSON.parse(queryParams?.path);
    } catch (error) {
      paths = [];
    }
    if (Array.isArray(paths) && paths.length > 0) {
      if (datasource?.name) {
        paths.unshift(datasource?.name);
      }
      return paths.map((item, index) => {
        const isLast = index === paths.length - 1;
        return (
          <span key={index} style={{ opacity: isLast ? 1 : 0.5 }}>
            {
              isLast ? (
                <span>{item}</span>
              ) : (
                <a onClick={() => {
                  setQueryParams(old => {
                    const newParams = Object.assign({}, old);
                    if (index !== 0) {
                      const path = paths.slice(1, index + 1);
                      newParams.path = JSON.stringify(path);
                    } else {
                      delete newParams.path;
                    }
                    return newParams;
                  })
                }}>{item}</a>
              )
            }
            { !isLast && <span className='mx-10px'>&gt;</span> }
          </span>
        );
      });
    }
    return datasource?.name;
  }

  return (
    <ListContainer>
      <div className="mt-12px ml--16px flex items-center text-lg font-bold">
        <div className="absolute mr-6px h-1.2em w-10px bg-[#1677FF]" />
        <div className="pl-16px">{renderTitle(datasource)}</div>
      </div>
      <div className="mb-4 mt-6 flex items-center justify-between">
        <Search
          value={keyword} 
          onChange={(e) => setKeyword(e.target.value)} 
          addonBefore={<FilterOutlined />}
          className="max-w-500px"
          enterButton={t('common.refresh')}
          onSearch={onRefreshClick}
        />
        {
          permissions.delete && (
            <div>
              <Dropdown.Button
                icon={<DownOutlined />}
                menu={{ items, onClick: onBatchMenuClick }}
                type="primary"
              >
                {t('common.operation')}
              </Dropdown.Button>
            </div>
          )
        }
      </div>
      <Table<DataType>
        columns={columns}
        dataSource={data.data || []}
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
        onChange={onTableChange}
      />
    </ListContainer>
  );
}

export default FileManagement
