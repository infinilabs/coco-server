import Icon, { DownOutlined, EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined } from '@ant-design/icons';
import type { MenuProps, TableColumnsType, TableProps } from 'antd';
import { Dropdown, Image, Modal, Switch, Table, message } from 'antd';
import Search from 'antd/es/input/Search';
import { type LoaderFunctionArgs, useLoaderData } from 'react-router-dom';

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
import { useRoute } from '@sa/simple-router';

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

export function Component() {
  const route = useRoute();
  const datasourceID = route.params.id
  const [queryParams, setQueryParams] = useQueryParams();

  const { t } = useTranslation();
  
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
    () => [
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
              setQueryParams(old => {
                return {
                  ...old,
                  filter: {
                    ...(old.filter || {}),
                    category: record.category ? [`${record.category}/${record.title}`] : ['/'],
                  }
                }
              })
            }
          } else if (record.url) {
            aProps.href = record.url;
            aProps.target = "_blank";
          }
          return (
            <span className="inline-flex items-center gap-1">
              {imgSrc && (
                <IconWrapper className="w-20px h-20px">
                  <InfiniIcon
                    height="1em"
                    src={imgSrc}
                    width="1em"
                  />
                </IconWrapper>
              )}
              { record.url || record.type === 'folder' ? <a {...aProps}>{text}</a> : <span>{text}</span> }
            </span>
          );
        },
        title: t('page.datasource.columns.name')
      },
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
            />
          );
        },
        title: t('page.datasource.columns.searchable')
      },
      {
        fixed: 'right',
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
    ],
    [connector]
  );

  if (!datasourceID) return <LookForward />;

  const [data, setData] = useState({});
  const [loading, setLoading] = useState(false);

  const [keyword, setKeyword] = useState();

  const fetchData = () => {
    setLoading(true);
    const { filter = {} } = queryParams || {};
    fetchDatasourceDetail({
      ...queryParams,
      filter: {
        ...filter,
        'category': filter.category || ['/'],
        'source.id': [datasourceID],
      }
    })
      .then(data => {
        const newData = formatESSearchResult(data.data);
        setData(newData);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(fetchData, []);

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
    if (Array.isArray(datasource?.categories)) {
      return datasource?.categories.map((item, index) => {
        const isLast = index === datasource?.categories.length - 1;
        return (
          <span key={index} style={{ opacity: isLast ? 1 : 0.5 }}>
            {
              isLast ? (
                <span>{item}</span>
              ) : (
                <a onClick={() => {
                  const category = index === 0 ? '/' : `/${datasource?.categories.slice(0, index + 1).join('/')}`;
                  setQueryParams(old => {
                    return {
                      ...old,
                      filter: {
                        ...(old.filter || {}),
                        category: [category],
                      }
                    }
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
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="sm:flex-1-auto min-h-full flex-col-stretch card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{renderTitle(datasource)}</div>
        </div>
        <div className="p-5 pt-2">
          <div className="mb-4 mt-4 flex items-center justify-between">
            <Search
              value={keyword} 
              onChange={(e) => setKeyword(e.target.value)} 
              addonBefore={<FilterOutlined />}
              className="max-w-500px"
              enterButton={t('common.refresh')}
              onSearch={onRefreshClick}
            />
            <div>
              <Dropdown.Button
                icon={<DownOutlined />}
                menu={{ items, onClick: onBatchMenuClick }}
                type="primary"
              >
                {t('common.operation')}
              </Dropdown.Button>
            </div>
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
        </div>
      </ACard>
    </div>
  );
}
