import { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Avatar, Button, Dropdown, Switch, Table, message } from 'antd';
import type { MenuProps, TableColumnsType, TableProps } from 'antd';
import Search from 'antd/es/input/Search';

import type { IntegratedStoreModalRef } from '@/components/common/IntegratedStoreModal';
import InfiniIcon from '@/components/common/icon';
import useQueryParams from '@/hooks/common/queryParams';
import { cloneAssistant, deleteAssistant, searchAssistant, updateAssistant } from '@/service/api/assistant';
import { formatESSearchResult } from '@/service/request/es';
import { Api } from '@/types/api';

type Assistant = Api.LLM.Assistant;

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();

  const { t } = useTranslation();

  const { addSharesToData, isEditorOwner, hasEdit, isResourceShare } = useResource();
  const resourceType = 'assistant';

  const { hasAuth } = useAuth();

  const permissions = {
    read: hasAuth('coco#assistant/read'),
    create: hasAuth('coco#assistant/create'),
    update: hasAuth('coco#assistant/update'),
    delete: hasAuth('coco#assistant/delete')
  };

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

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
            if (permissions.update && hasEdit(record)) {
              nav(`/ai-assistant/edit/${res.data?._id}`);
            } else {
              nav(`/ai-assistant/list`);
            }
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
      title: t('page.assistant.labels.name'),
      dataIndex: 'name',
      minWidth: 150,
      ellipsis: true,
      render: (value: string, record: Assistant) => {
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
            {record.icon && (
              <IconWrapper className='h-20px w-20px flex-shrink-0 flex-grow-0 flex-basis-auto'>
                <InfiniIcon
                  height='1em'
                  src={record.icon}
                  width='1em'
                />
              </IconWrapper>
            )}
            {permissions.read && permissions.update && hasEdit(record) ? (
              <a
                className='ant-table-cell-ellipsis max-w-150px cursor-pointer text-[var(--ant-color-link)]'
                onClick={() => nav(`/ai-assistant/edit/${record.id}`)}
              >
                {value}
              </a>
            ) : (
              <span className='ant-table-cell-ellipsis max-w-150px'>{value}</span>
            )}
            {record.builtin === true && (
              <div className='ml-5px flex items-center'>
                <p className='h-[22px] rounded-[4px] bg-[#eee] px-[10px] font-size-[12px] text-[#999] line-height-[22px]'>
                  {t('page.modelprovider.labels.builtin')}
                </p>
              </div>
            )}
            {shareIcon}
          </div>
        );
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
        if (!value) return '-';
        return (
          <Shares
            record={record}
            title={record.name}
            resource={{
              resource_type: resourceType,
              resource_id: record.id
            }}
            onSuccess={() => fetchData(queryParams)}
          />
        );
      }
    },
    {
      dataIndex: 'type',
      minWidth: 50,
      dataIndex: 'type'
    },
    {
      dataIndex: ['datasource', 'enabled'],
      minWidth: 50,
      dataIndex: ['datasource', 'enabled'],
      render: (value: boolean, record: Assistant) => {
        return t(`common.enableOrDisable.${value ? 'enable' : 'disable'}`);
      }
    },
    {
      dataIndex: ['mcp_servers', 'enabled'],
      minWidth: 50,
      dataIndex: ['mcp_servers', 'enabled'],
      render: (value: boolean, record: Assistant) => {
        return t(`common.enableOrDisable.${value ? 'enable' : 'disable'}`);
      }
    },
    {
      dataIndex: 'description',
      ellipsis: true,
      minWidth: 200,
      dataIndex: 'description',
      render: (value: string, record: Assistant) => {
        return <span title={value}>{value}</span>;
      },
      ellipsis: true
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
      width: 80,
      render: (value: boolean, record: Assistant) => {
        return (
          <Switch
            disabled={!permissions.update || !hasEdit(record)}
            size='small'
            value={value}
            onChange={v => onEnabledChange(v, record)}
          />
        );
      }
    },
    {
      fixed: 'right',
      width: '90px',
      hidden: !permissions.update && !permissions.delete,
      render: (_, record) => {
        const items: MenuProps['items'] = [];
        if (permissions.read && permissions.update && hasEdit(record)) {
          items.push({
            label: t('common.edit'),
            key: '2'
          });
        }
        if (permissions.delete && record.builtin !== true && isEditorOwner(record)) {
          items.push({
            label: t('common.delete'),
            key: '1'
          });
        }
        if (permissions.create && hasEdit(record)) {
          items.push({
            label: t('common.clone'),
            key: '3'
          });
        }
        if (items.length === 0) return null;
        return (
          <Dropdown menu={{ items, onClick: ({ key }) => onMenuClick({ key, record }) }}>
            <EllipsisOutlined />
          </Dropdown>
        );
      }
    }
  ];

  // rowSelection object indicates the need for row selection
  const rowSelection: TableProps<Assistant>['rowSelection'] = {
    onChange: (selectedRowKeys: React.Key[], selectedRows: Assistant[]) => {},
    getCheckboxProps: (record: Assistant) => ({
      name: record.name
    })
  };

  const initialData = {
    data: [],
    total: 0
  };
  const [data, setData] = useState(initialData);
  const [loading, setLoading] = useState(false);

  const [keyword, setKeyword] = useState();

  const fetchData = async queryParams => {
    setLoading(true);
    const res = await searchAssistant(queryParams);
    if (res?.data) {
      const newData = formatESSearchResult(res?.data);
      if (newData.data.length > 0) {
        const resources = newData.data.map(item => ({
          resource_id: item.id,
          resource_type: resourceType
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
    fetchData(queryParams);
  }, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query);
  }, [queryParams.query]);

  const handleTableChange: TableProps<Assistant>['onChange'] = (pagination, filters, sorter) => {
    setQueryParams(params => {
      return {
        ...params,
        size: pagination.pageSize,
        from: (pagination.current - 1) * pagination.pageSize
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

  const integratedStoreModalRef = useRef<IntegratedStoreModalRef>(null);

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
              onClick={() => integratedStoreModalRef.current?.open('ai-assistant')}
            >
              {t('common.add')}
            </Button>
          )}
        </div>
        <Table<Assistant>
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

      <IntegratedStoreModal ref={integratedStoreModalRef} />
    </ListContainer>
  );
}
