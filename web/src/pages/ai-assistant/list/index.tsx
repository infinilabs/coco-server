import Icon, { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Dropdown, GetProp, Image, Modal, Switch, Table, message } from 'antd';
import type { MenuProps, TableColumnsType, TableProps } from 'antd';
import Search from 'antd/es/input/Search';

import type { IntegratedStoreModalRef } from '@/components/common/IntegratedStoreModal';
import InfiniIcon from '@/components/common/icon';
import useQueryParams from '@/hooks/common/queryParams';
import { cloneAssistant, deleteAssistant, searchAssistant, updateAssistant } from '@/service/api/assistant';
import { formatESSearchResult } from '@/service/request/es';

type Assistant = Api.LLM.Assistant;

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();

  const { t } = useTranslation();

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

  const getMenuItems = useCallback((record: Assistant): MenuProps['items'] => {
    const items: MenuProps['items'] = [
      {
        key: '2',
        label: t('common.edit')
      }
    ];
    if (record.builtin !== true) {
      items.push({
        key: '1',
        label: t('common.delete')
      });
    }
    items.push({
      key: '3',
      label: t('common.clone')
    });
    return items;
  }, []);

  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '1':
        window?.$modal?.confirm({
          content: t('page.assistant.delete.confirm', { name: record.name }),
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            deleteAssistant(record.id).then(res => {
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
        nav(`/ai-assistant/edit/${record.id}`);
        break;
      case '3':
        cloneAssistant(record.id).then(res => {
          if (res.data?.result === 'created') {
            nav(`/ai-assistant/edit/${res.data?._id}`);
          } else {
            message.error(res.data?.error?.reason);
          }
        });
        break;
    }
  };

  const onEnabledChange = (value: boolean, record: Assistant) => {
    setLoading(true);
    updateAssistant(record.id, {
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
  const columns: TableColumnsType<Assistant> = [
    {
      dataIndex: 'name',
      render: (value: string, record: Assistant) => {
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
              onClick={() => nav(`/ai-assistant/edit/${record.id}`)}
            >
              {value}
            </span>
            {record.builtin === true && (
              <div className="ml-[5px] flex items-center">
                <p className="h-[22px] rounded-[4px] bg-[#eee] px-[10px] font-size-[12px] text-[#999] line-height-[22px]">
                  {t('page.modelprovider.labels.builtin')}
                </p>
              </div>
            )}
          </div>
        );
      },
      title: t('page.assistant.labels.name'),
      width: 300
    },
    {
      dataIndex: 'type',
      minWidth: 50,
      title: t('page.assistant.labels.type')
    },
    {
      dataIndex: ['datasource', 'enabled'],
      minWidth: 50,
      render: (value: boolean, record: Assistant) => {
        return t(`common.enableOrDisable.${value ? 'enable' : 'disable'}`);
      },
      title: t('page.assistant.labels.datasource')
    },
    {
      dataIndex: ['mcp_servers', 'enabled'],
      minWidth: 50,
      render: (value: boolean, record: Assistant) => {
        return t(`common.enableOrDisable.${value ? 'enable' : 'disable'}`);
      },
      title: t('page.assistant.labels.mcp_servers')
    },
    {
      dataIndex: 'description',
      ellipsis: true,
      minWidth: 200,
      render: (value: string, record: Assistant) => {
        return <span title={value}>{value}</span>;
      },
      title: t('page.assistant.labels.description')
    },
    {
      dataIndex: 'enabled',
      render: (value: boolean, record: Assistant) => {
        return (
          <Switch
            size="small"
            value={value}
            onChange={v => onEnabledChange(v, record)}
          />
        );
      },
      title: t('page.assistant.labels.enabled'),
      width: 80
    },
    {
      fixed: 'right',
      render: (_, record) => {
        return (
          <Dropdown menu={{ items: getMenuItems(record), onClick: ({ key }) => onMenuClick({ key, record }) }}>
            <EllipsisOutlined />
          </Dropdown>
        );
      },
      title: t('common.operation'),
      width: '90px'
    }
  ];
  // rowSelection object indicates the need for row selection
  const rowSelection: TableProps<Assistant>['rowSelection'] = {
    getCheckboxProps: (record: Assistant) => ({
      name: record.name
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows: Assistant[]) => {}
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
    searchAssistant(queryParams).then(({ data }) => {
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

  const handleTableChange: TableProps<Assistant>['onChange'] = (pagination, filters, sorter) => {
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
              integratedStoreModalRef.current?.open('ai-assistant');
            }}
          >
            {t('common.add')}
          </Button>
        </div>
        <Table<Assistant>
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
