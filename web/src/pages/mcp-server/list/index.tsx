import Icon, { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Dropdown, GetProp, Image, Modal, Switch, Table, message } from 'antd';
import type { MenuProps, TableColumnsType, TableProps } from 'antd';
import Search from 'antd/es/input/Search';

import type { IntegratedStoreModalRef } from '@/components/common/IntegratedStoreModal';
import InfiniIcon from '@/components/common/icon';
import useQueryParams from '@/hooks/common/queryParams';
import { deleteMCPServer, searchMCPServer, updateMCPServer } from '@/service/api/mcp-server';
import { formatESSearchResult } from '@/service/request/es';

type MCPServer = Api.LLM.MCPServer;

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();

  const { t } = useTranslation();

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();
  const items: MenuProps['items'] = [
    {
      key: '2',
      label: t('common.edit')
    },
    {
      key: '1',
      label: t('common.delete')
    }
  ];

  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '1':
        window?.$modal?.confirm({
          content: t('page.mcpserver.delete.confirm', { name: record.name }),
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            deleteMCPServer(record.id).then(res => {
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
        nav(`/mcp-server/edit/${record.id}`, { state: record });
        break;
    }
  };

  const onEnabledChange = (value: boolean, record: MCPServer) => {
    setLoading(true);
    updateMCPServer(record.id, {
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
  const columns: TableColumnsType<MCPServer> = [
    {
      dataIndex: 'name',
      ellipsis: true,
      minWidth: 150,
      render: (value: string, record: MCPServer) => {
        return (
          <div className="flex items-center gap-1">
            <IconWrapper className="h-20px w-20px">
              <InfiniIcon
                height="1em"
                src={record.icon}
                width="1em"
              />
            </IconWrapper>
            <span
              className="ant-table-cell-ellipsis max-w-150px cursor-pointer hover:text-blue-500"
              onClick={() => nav(`/mcp-server/edit/${record.id}`, { state: record })}
            >
              {value}
            </span>
          </div>
        );
      },
      title: t('page.mcpserver.labels.name')
    },
    {
      dataIndex: 'type',
      minWidth: 50,
      title: t('page.mcpserver.labels.type')
    },
    {
      dataIndex: 'category',
      ellipsis: true,
      minWidth: 50,
      title: t('page.mcpserver.labels.category')
    },
    {
      dataIndex: 'description',
      ellipsis: true,
      minWidth: 150,
      title: t('page.mcpserver.labels.description')
    },
    {
      dataIndex: 'enabled',
      render: (value: boolean, record: MCPServer) => {
        return (
          <Switch
            size="small"
            value={value}
            onChange={v => onEnabledChange(v, record)}
          />
        );
      },
      title: t('page.mcpserver.labels.enabled'),
      width: 80
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
  ];
  // rowSelection object indicates the need for row selection
  const rowSelection: TableProps<MCPServer>['rowSelection'] = {
    getCheckboxProps: (record: MCPServer) => ({
      name: record.id
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows: MCPServer[]) => {}
  };

  const initialData = {
    data: [],
    total: 0
  };
  const [data, setData] = useState(initialData);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState();

  const fetchData = () => {
    setLoading(true);
    searchMCPServer(queryParams).then(({ data }) => {
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
    setKeyword(queryParams.query);
  }, [queryParams.query]);

  const handleTableChange: TableProps<MCPServer>['onChange'] = (pagination, filters, sorter) => {
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

  const integratedStoreModalRef = useRef<IntegratedStoreModalRef>(null);

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
      >
        <div className="mb-4 mt-4 flex items-center justify-between">
          <Search
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
            onClick={() => {
              integratedStoreModalRef.current?.open('mcp-server');
            }}
          >
            {t('common.add')}
          </Button>
        </div>
        <Table<MCPServer>
          columns={columns}
          dataSource={data.data}
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

      <IntegratedStoreModal ref={integratedStoreModalRef} />
    </ListContainer>
  );
}
