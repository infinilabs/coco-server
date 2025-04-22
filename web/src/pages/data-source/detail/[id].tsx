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
  updateDocument
} from '@/service/api';
import { formatESSearchResult } from '@/service/request/es';

const { confirm } = Modal;

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
  const datasourceID = useLoaderData();

  const { t } = useTranslation();
  const nav = useNavigate();
  const location = useLocation();
  const { connector_id, datasource_name } = location.state || {};
  const [connector, setConnector] = useState<any>({});
  useEffect(() => {
    getConnector(connector_id).then(res => {
      if (res.data?.found === true) {
        setConnector(res.data._source || {});
      }
    });
  }, [connector_id]);
  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '1':
        confirm({
          content: 'Are you sure you want to delete this document?',
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            deleteDocument(record.id).then(res => {
              if (res.data?.result === 'deleted') {
                message.success('deleted success');
              }
              // reload data
              setReqParams(old => {
                return {
                  ...old
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
      setReqParams(old => {
        return {
          ...old
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
          confirm({
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
                    setReqParams(old => {
                      return {
                        ...old
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
          return (
            <span className="inline-flex items-center gap-1">
              {imgSrc && (
                <InfiniIcon
                  className="mr-3px"
                  height="1em"
                  src={imgSrc}
                  width="1em"
                />
              )}
              <a
                className="text-blue-500"
                href={record.url}
                rel="noreferrer"
                target="_blank"
              >
                {text}
              </a>
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
    [connector_id, connector]
  );

  if (!datasourceID) return <LookForward />;

  const [reqParams, setReqParams] = useState({
    datasource: datasourceID,
    from: 0,
    size: 20
  });
  const [data, setData] = useState({});
  const [loading, setLoading] = useState(false);

  const fetchData = () => {
    setLoading(true);
    fetchDatasourceDetail(reqParams)
      .then(data => {
        const newData = formatESSearchResult(data.data);
        setData(newData);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(fetchData, [reqParams]);

  const onTableChange = (pagination, filters, sorter, extra: { action; currentDataSource: [] }) => {
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
        form: 0,
        query
      };
    });
  };

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="sm:flex-1-auto min-h-full flex-col-stretch card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{datasource_name}</div>
        </div>
        <div className="p-5 pt-2">
          <div className="mb-4 mt-4 flex items-center justify-between">
            <Search
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
              defaultCurrent: 1,
              defaultPageSize: 20,
              showSizeChanger: true,
              showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
              total: data.total?.value || data?.total
            }}
            onChange={onTableChange}
          />
        </div>
      </ACard>
    </div>
  );
}

export async function loader({ params, ...rest }: LoaderFunctionArgs) {
  const datasourceID = params.id;
  // todo fetch datasource info by id
  return datasourceID;
}
