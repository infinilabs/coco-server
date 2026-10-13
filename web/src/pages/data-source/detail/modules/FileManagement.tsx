import { CheckCircleFilled, CloseCircleFilled, DatabaseOutlined, DownOutlined, EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, RightOutlined } from '@ant-design/icons';
import type { MenuProps, TableColumnsType, TableProps } from 'antd';
import { Avatar, Button, Dropdown, Popover, Table, Typography, message } from 'antd';
import Search from 'antd/es/input/Search';
import { useNavigate } from 'react-router-dom';

import FontIcon from '@/components/common/font_icon';
import InfiniIcon from '@/components/common/icon';
import ProcessingTimeline from '@/components/DocumentDrawer/ProcessingTimeline';
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
import { getServer } from '@/store/slice/server';

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
  const navigate = useNavigate();

  const server = useAppSelector(getServer);

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
      case 'view_detail':
        navigate(`/data-source/doc/${record.id}`);
        break;
      case '1':
        window?.$modal?.confirm({
          content: t('page.datasource.labels.deleteConfirm'),
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            deleteDocument(record.id).then(res => {
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
        message.success(t('common.updateSuccess'));
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
  // 批量操作菜单只含删除；行级菜单另加「查看详情」（跳文档详情整页）
  const items: MenuProps['items'] = [
    {
      key: '1',
      label: t('common.delete')
    }
  ];
  const rowMenuItems: MenuProps['items'] = [
    {
      key: 'view_detail',
      label: t('page.datasource.doc.viewDetail')
    },
    { type: 'divider' },
    ...items
  ];
  const onBatchMenuClick = useCallback(
    ({ key }: any) => {
      switch (key) {
        case '1':
          window?.$modal?.confirm({
            content: t('page.datasource.labels.deleteAllConfirm'),
            icon: <ExclamationCircleOutlined />,
            onCancel() {},
            onOk() {
              if (state.selectedRowKeys?.length === 0) {
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
                    message.success(t('common.deleteSuccess'));
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
                      server={server}
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
                  server={server}
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
      title: t('common.status'),
      dataIndex: 'processed',
      width: '80px',
      render: (processed: boolean, record: any) => {
        const icon =
          processed === true ? (
            <CheckCircleFilled className='text-[var(--ant-color-success)]' />
          ) : (
            <CloseCircleFilled className='text-[var(--ant-color-error)]' />
          );
        // the parsing popover (W13b artifact face): lazy-mounted on first
        // open, destroyed on close so each visit refetches fresh state.
        // 固定高度 + 内部滚动：antd 默认 placement=top，若浮层高度随异步
        // 内容变化（小→大），autoAdjustOverflow 会在渲染后把它从上方翻到
        // 下方产生可见跳动；首帧尺寸恒定则只定位一次
        return (
          <Popover
            arrow={false}
            content={
              <div className='flex h-400px w-360px flex-col'>
                <div className='mb-8px text-14px font-bold'>{t('page.datasource.doc.parsingTitle')}</div>
                <div className='min-h-0 flex-1 overflow-y-auto'>
                  <ProcessingTimeline docId={record?.id} maxRuns={3} />
                </div>
                <div className='mt-10px flex shrink-0 justify-end border-t-1 border-t-[var(--ant-color-border-secondary)] pt-8px'>
                  <Typography.Link
                    className='text-12px'
                    onClick={() => navigate(`/data-source/doc/${record?.id}?tab=runs`)}
                  >
                    {t('page.preview.processing.viewFullRuns')} <RightOutlined className='text-10px' />
                  </Typography.Link>
                </div>
              </div>
            }
            destroyOnHidden
            title={null}
            trigger='click'
          >
            <span className='cursor-pointer'>{icon}</span>
          </Popover>
        );
      }
    },
    {
      fixed: 'right',
      hidden: !permissions.delete,
      render: (_, record) => {
        if (!isEditorOwner(record)) return null;
        return (
          <Dropdown menu={{ items: rowMenuItems, onClick: ({ key }) => onMenuClick({ key, record }) }}>
            <EllipsisOutlined className='cursor-pointer'/>
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
      // 数据源名称已在顶部基本信息卡展示：面包屑只保留根图标入口 + 文件夹路径
      return (
        <>
          <DatabaseOutlined
            className='cursor-pointer transition-colors hover:text-[var(--ant-color-primary)]'
            onClick={() => {
              setQueryParams(old => {
                const newParams = { ...old };
                delete newParams.path;
                return {
                  ...newParams,
                  from: 0,
                  size: 10
                };
              });
            }}
          />
          <span className='mx-10px'>&gt;</span>
          {paths.map((item, index) => {
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
                        return {
                          ...old,
                          path: JSON.stringify(paths.slice(0, index + 1)),
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
          })}
        </>
      );
    }
    return null;
  };

  if (!datasourceID) return <LookForward />;

  return (
    <ListContainer>
      {renderTitle(datasource) ? (
        <div className='ml--16px mt-12px flex items-center text-lg font-bold'>
          <div className='absolute mr-6px h-1.2em w-10px bg-[#1677FF]' />
          <div className='pl-16px'>{renderTitle(datasource)}</div>
        </div>
      ) : null}
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
            <Dropdown 
              menu={{ items: items.map((item) => ({
                ...item,
                disabled: state.selectedRowKeys?.length === 0
              })), onClick: onBatchMenuClick }}
            >
              <Button type='primary' icon={<DownOutlined />} iconPlacement="end">
                {t('common.operation')}
              </Button>
            </Dropdown>
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
