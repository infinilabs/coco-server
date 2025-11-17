import { DownOutlined, EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined } from '@ant-design/icons';
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

const FileManagement = props => {
  const datasourceID = props.id;
  const [queryParams, setQueryParams] = useQueryParams();

  const responsive = useResponsive();
  const { t } = useTranslation();

  const { addSharesToData, isEditorOwner, hasEdit, isResourceShare } = useResource();

  const { hasAuth } = useAuth();

  const permissions = {
    readDatasource: hasAuth('coco#datasource/read'),
    readConnector: hasAuth('coco#connector/read'),
    update: hasAuth('coco#document/update'),
    delete: hasAuth('coco#document/delete')
  };

  const [connector, setConnector] = useState<any>({});
  const [datasource, setDatasource] = useState<any>();
  const [selectedRecord, setSelectedRecord] = useState<any>();

  useEffect(() => {
    if (!datasourceID) return;
    if (permissions.readDatasource) {
      getDatasource(datasourceID).then(res => {
        if (res.data?.found === true) {
          setDatasource(res.data._source || {});
        }
      });
    }
  }, [datasourceID]);
  useEffect(() => {
    if (!datasource?.connector?.id) return;
    if (permissions.readConnector) {
      getConnector(datasource?.connector?.id).then(res => {
        if (res.data?.found === true) {
          setConnector(res.data._source || {});
        }
      });
    }
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
      name: record.title,
      disabled: !(isEditorOwner(record) && permissions.delete)
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

  const columns: TableColumnsType<DataType> = [
    {
      dataIndex: 'title',
      render: (text: string, record: DataType) => {
        const isShare = isResourceShare(record);

        let imgSrc = '';
        if (connector?.assets?.icons) {
          imgSrc = connector.assets.icons[record.icon];
        }
        const aProps = {
          className: 'text-blue-500',
          rel: 'noreferrer'
        };

        const pathHierarchy = connector?.path_hierarchy && record.type === 'folder';

        if (pathHierarchy) {
          aProps.onClick = () => {
            const categories = (record.categories || []).concat([record.title]);
            setQueryParams(old => {
              return {
                ...old,
                from: 0,
                size: 10,
                path: JSON.stringify(categories),
                view: isEditorOwner(record) ? 'auto' : 'list'
              };
            });
          };
        } else {
          aProps.onClick = () => {
            setSelectedRecord(record)
          }
        }

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

        if (connector?.path_hierarchy && queryParams.view === 'list') {
          return (
            <span className='flex-col inline-flex'>
              <span className='h-16px inline-flex items-center gap-1 text-12px'>
                {imgSrc ? (
                  <IconWrapper className='h-1em w-1em text-16px'>
                    <InfiniIcon
                      height='1em'
                      src={imgSrc}
                      width='1em'
                    />
                  </IconWrapper>
                ) : (
                  <FontIcon name={record.icon} />
                )}
                <a {...aProps}>{text}</a>
                {shareIcon}
              </span>
              <span className='h-14px text-10px text-#999'>
                {record.categories?.length > 0 ? `/${record.categories.join('/')}` : '/'}
              </span>
            </span>
          );
        }

        return (
          <span className='inline-flex items-center gap-1'>
            {imgSrc ? (
              <IconWrapper className='h-20px w-20px'>
                <InfiniIcon
                  height='1em'
                  src={imgSrc}
                  width='1em'
                />
              </IconWrapper>
            ) : (
              <FontIcon name={record.icon} />
            )}
            <a {...aProps}>{text}</a>
            {shareIcon}
          </span>
        );
      },
      title: t('page.datasource.columns.name')
    },
    {
      dataIndex: 'categories',
      title: t('page.datasource.labels.categories'),
      hidden: connector?.path_hierarchy && queryParams.view === 'list',
      render: (value) => {
        return Array.isArray(value) ? `/${value.join('/')}` : '/'
      }
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
        return (
          <Shares
            record={record}
            title={record.title}
            resource={{
              resource_category_type: 'datasource',
              resource_category_id: datasourceID,
              resource_type: 'document',
              resource_id: record.id,
              resource_parent_path: record.categories?.length > 0 ? `/${record.categories.join('/')}/` : '/',
              resource_full_path:
                (record.categories?.length > 0 ? `/${record.categories.join('/')}/` : '/') + record.title,
              resource_is_folder: record?.type === 'folder'
            }}
            onSuccess={() => fetchData(queryParams, datasourceID)}
          />
        );
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
    // {
    //   dataIndex: 'disabled',
    //   render: (text: boolean, record: DataType) => {
    //     return (
    //       <Switch
    //         disabled={!permissions.update || !hasEdit(record)}
    //         size='small'
    //         value={!text}
    //         onChange={v => {
    //           onSearchableChange(v, record);
    //         }}
    //       />
    //     );
    //   },
    //   title: t('page.datasource.columns.searchable')
    // },
    {
      fixed: 'right',
      hidden: !permissions.delete,
      render: (_, record) => {
        if (!isEditorOwner(record)) return null;
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

  const [data, setData] = useState({});
  const [loading, setLoading] = useState(false);

  const [keyword, setKeyword] = useState();

  const fetchData = async (queryParams, datasourceID) => {
    if (!datasourceID) return;
    setLoading(true);
    const { filter = {} } = queryParams || {};
    const res = await fetchDatasourceDetail({
      ...queryParams,
      filter: {
        ...filter,
        'source.id': [datasourceID]
      }
    });
    if (res?.data) {
      const newData = formatESSearchResult(res.data);
      if (newData.data.length > 0) {
        const resources = newData.data.map(item => ({
          resource_id: item.id,
          resource_type: 'document',
          resource_category_type: 'datasource',
          resource_category_id: datasourceID,
          resource_parent_path: item.categories?.length > 0 ? `/${item.categories.join('/')}/` : '/'
        }));
        const dataWithShares = await addSharesToData(newData.data, resources);
        if (dataWithShares) {
          newData.data = dataWithShares;
        }
      }
      setData(newData);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData(queryParams, datasourceID);
  }, [queryParams, datasourceID]);

  useEffect(() => {
    setKeyword(queryParams.query);
  }, [queryParams.query]);

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
        from: query === oldParams.query ? oldParams.from : 0,
        query,
        t: new Date().valueOf()
      };
    });
  };

  const renderTitle = datasource => {
    let paths;
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
          <span
            key={index}
            style={{ opacity: isLast ? 1 : 0.5 }}
          >
            {isLast ? (
              <span>{item}</span>
            ) : (
              <a
                onClick={() => {
                  setQueryParams(old => {
                    const newParams = { ...old };
                    if (index !== 0) {
                      const path = paths.slice(1, index + 1);
                      newParams.path = JSON.stringify(path);
                    } else {
                      delete newParams.path;
                    }
                    return {
                      ...newParams,
                      from: 0,
                      size: 10
                    };
                  });
                }}
              >
                {item}
              </a>
            )}
            {!isLast && <span className='mx-10px'>&gt;</span>}
          </span>
        );
      });
    }
    return datasource?.name;
  };

  if (!datasourceID) return <LookForward />;

  return (
    <ListContainer>
      <div className='ml--16px mt-12px flex items-center text-lg font-bold'>
        <div className='absolute mr-6px h-1.2em w-10px bg-[#1677FF]' />
        <div className='pl-16px'>{renderTitle(datasource)}</div>
      </div>
      <div className='mb-4 mt-6 flex items-center justify-between'>
        <Search
          addonBefore={<FilterOutlined />}
          className='max-w-500px'
          enterButton={t('common.refresh')}
          value={keyword}
          onChange={e => setKeyword(e.target.value)}
          onSearch={onRefreshClick}
        />
        {permissions.delete && (
          <div>
            <Dropdown.Button
              icon={<DownOutlined />}
              menu={{ items, onClick: onBatchMenuClick }}
              type='primary'
            >
              {t('common.operation')}
            </Dropdown.Button>
          </div>
        )}
      </div>
      <Table<DataType>
        columns={columns}
        dataSource={data.data || []}
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
        onChange={onTableChange}
      />
      <DocumentDrawer 
        data={selectedRecord}
        open={!!selectedRecord}
        onClose={() => setSelectedRecord(undefined)}
        isMobile={!responsive.sm}
      />
    </ListContainer>
  );
};

export default FileManagement;
