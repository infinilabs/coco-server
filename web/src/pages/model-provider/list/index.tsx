import Search from 'antd/es/input/Search';
import Icon, {
  EllipsisOutlined,
  ExclamationCircleOutlined,
  ExportOutlined,
  FilterOutlined,
  PlusOutlined,
  SettingOutlined
} from '@ant-design/icons';
import {
  Avatar,
  Button,
  Dropdown,
  Form,
  Image,
  Input,
  List,
  MenuProps,
  Modal,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
  message
} from 'antd';
import { deleteModelProvider, searchModelPovider, updateModelProvider } from '@/service/api/model-provider';
import { formatESSearchResult } from '@/service/request/es';
import InfiniIcon from '@/components/common/icon';
import useQueryParams from '@/hooks/common/queryParams';

export function Component() {
  const type = 'table';

  const [queryParams, setQueryParams] = useQueryParams({
    size: type === 'table' ? 10 : 12,
    sort: [
      ['enabled', 'desc'],
      ['created', 'desc']
    ]
  });

  const { addSharesToData, isEditorOwner, hasEdit, isResourceShare } = useResource();
  const resourceType = 'llm-provider';

  const { t } = useTranslation();
  const nav = useNavigate();
  const [data, setData] = useState({
    total: 0,
    data: []
  });
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState();

  const { hasAuth } = useAuth();

  const permissions = {
    read: hasAuth('coco#model_provider/read'),
    create: hasAuth('coco#model_provider/create'),
    update: hasAuth('coco#model_provider/update'),
    delete: hasAuth('coco#model_provider/delete')
  };

  const fetchData = async queryParams => {
    setLoading(true);
    const res = await searchModelPovider(queryParams);
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
      setData(newData);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchData(queryParams);
  }, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query);
  }, [queryParams.query]);

  const onSearchClick = (query: string) => {
    setQueryParams(oldParams => {
      return {
        ...oldParams,
        from: query === oldParams.query ? oldParams.from : 0,
        query,
        t: new Date().valueOf()
      };
    });
  };

  const handleTableChange = pagination => {
    setQueryParams(params => {
      return {
        ...params,
        from: (pagination.current - 1) * pagination.pageSize,
        size: pagination.pageSize
      };
    });
  };

  const onPageChange = (page: number, pageSize: number) => {
    setQueryParams((oldParams: any) => {
      return {
        ...oldParams,
        from: (page - 1) * pageSize,
        size: pageSize
      };
    });
  };
  const getMenuItems = (record: any) => {
    const items = [];
    if (permissions.read && permissions.update && hasEdit(record)) {
      items.push({
        label: t('common.edit'),
        key: '1'
      });
    }
    if (permissions.update && hasEdit(record)) {
      items.push({
        label: 'API-key',
        key: '3'
      });
    }
    if (permissions.delete && record.builtin !== true && isEditorOwner(record)) {
      items.push({
        label: t('common.delete'),
        key: '2'
      });
    }
    return items;
  };

  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '2':
        window?.$modal?.confirm({
          icon: <ExclamationCircleOutlined />,
          title: t('common.tip'),
          content: t('page.modelprovider.delete.confirm'),
          onOk() {
            deleteModelProvider(record.id).then(res => {
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
      case '1':
        nav(`/model-provider/edit/${record.id}`);
        break;
      case '3':
        onAPIKeyClick(record);
        break;
    }
  };
  const onItemEnableChange = (record: any, checked: boolean) => {
    setLoading(true);
    updateModelProvider(record.id, {
      ...record,
      enabled: checked
    })
      .then(res => {
        if (res.data?.result === 'updated') {
          // update local data
          setData((oldData: any) => {
            const newData = oldData.data.map((item: any) => {
              if (item.id === record.id) {
                return {
                  ...item,
                  enabled: checked
                };
              }
              return item;
            });
            return {
              ...oldData,
              data: newData
            };
          });
          message.success(t('common.updateSuccess'));
        }
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const [editValue, setEditValue] = useState({});
  const [open, setOpen] = useState(false);
  const onOkClick = () => {
    setOpen(false);
    fetchData(queryParams);
  };
  const onCancelClick = () => {
    setOpen(false);
  };

  const onAPIKeyClick = (record: any) => {
    setEditValue(record);
    setOpen(true);
  };

  const columns = [
    {
      dataIndex: 'name',
      minWidth: 150,
      ellipsis: true,
      render: (value, record) => {
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
                onClick={() => nav(`/model-provider/edit/${record.id}`)}
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
      },
      title: t('page.integration.columns.name')
    },
    {
      dataIndex: 'owner',
      title: t('page.datasource.labels.owner'),
      width: 200,
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
      width: 150,
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
      title: t('page.assistant.labels.description'),
      minWidth: 200,
      dataIndex: 'description',
      render: (value, record) => {
        return <span title={value}>{value}</span>;
      },
      ellipsis: true
    },
    {
      dataIndex: 'enabled',
      title: t('page.assistant.labels.enabled'),
      width: 80,
      render: (value, record) => {
        return (
          <Switch
            disabled={!permissions.update || !hasEdit(record)}
            size='small'
            value={value}
            onChange={v => onItemEnableChange(record, v)}
          />
        );
      }
    },
    {
      title: t('common.operation'),
      fixed: 'right',
      width: '90px',
      hidden: !permissions.update && !permissions.delete,
      render: (_, record) => {
        const items = getMenuItems(record);
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
  const rowSelection = {
    getCheckboxProps: record => ({
      name: record.name
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows) => {}
  };

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className='flex-col-stretch sm:flex-1-hidden card-wrapper'
      >
        <div className='mb-4 mt-4 flex items-center justify-between'>
          <Search
            addonBefore={<FilterOutlined />}
            className='max-w-500px'
            enterButton={t('common.refresh')}
            value={keyword}
            onChange={e => setKeyword(e.target.value)}
            onSearch={onSearchClick}
          />
          {permissions.create && (
            <Button
              icon={<PlusOutlined />}
              type='primary'
              onClick={() => nav(`/model-provider/new`)}
            >
              {t('common.add')}
            </Button>
          )}
        </div>
        {type === 'table' ? (
          <Table
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
        ) : (
          <List
            dataSource={data.data}
            grid={{ gutter: 16, column: 3, xs: 1, sm: 2 }}
            loading={loading}
            pagination={{
              onChange: onPageChange,
              showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
              pageSize: queryParams.size,
              current: Math.floor(queryParams.from / queryParams.size) + 1,
              total: data.total?.value || data?.total,
              showSizeChanger: true,
              pageSizeOptions: [12, 24, 48, 96]
            }}
            renderItem={provider => {
              const operations = getMenuItems(provider);
              return (
                <List.Item>
                  <div className='group min-h-[132px] border border-[var(--ant-color-border)] rounded-[8px] p-1em hover:bg-[var(--ant-control-item-bg-hover)]'>
                    <div className='flex justify-between'>
                      <div className='flex gap-15px'>
                        <div className='flex items-center gap-8px'>
                          <IconWrapper className='h-40px w-40px'>
                            <InfiniIcon
                              className='font-size-2em'
                              height='2em'
                              src={provider.icon}
                              width='2em'
                            />
                          </IconWrapper>
                          {permissions.read && permissions.update && hasEdit(provider) ? (
                            <a
                              className='cursor-pointer font-size-1.2em text-[var(--ant-color-link)]'
                              onClick={() => nav(`/model-provider/edit/${provider.id}`)}
                            >
                              {provider.name}
                            </a>
                          ) : (
                            <span className='font-size-1.2em'>{provider.name}</span>
                          )}
                        </div>
                        {provider.builtin === true && (
                          <div className='flex items-center'>
                            <p className='h-[22px] rounded-[4px] bg-[#eee] px-[10px] font-size-[12px] text-[#999] line-height-[22px]'>
                              {t('page.modelprovider.labels.builtin')}
                            </p>
                          </div>
                        )}
                      </div>
                      <div>
                        <Switch
                          checked={provider.enabled}
                          disabled={!permissions.update || !hasEdit(provider)}
                          size='small'
                          onChange={v => onItemEnableChange(provider, v)}
                        />
                      </div>
                    </div>
                    <div className='line-clamp-3 my-[10px] h-[51px] text-xs text-[#999]'>{provider.description}</div>
                    <div className='flex gap-1'>
                      <div className='ml-auto flex gap-2'>
                        {permissions.update && hasEdit(provider) && (
                          <div
                            className='cursor-pointer border border-[var(--ant-color-border)] rounded-[8px] px-10px'
                            onClick={() => {
                              onAPIKeyClick(provider);
                            }}
                          >
                            API-key
                          </div>
                        )}
                        {operations?.length > 0 && (
                          <div className='inline-block cursor-pointer border border-[var(--ant-color-border)] rounded-[8px] px-4px'>
                            <Dropdown
                              menu={{ items: operations, onClick: ({ key }) => onMenuClick({ key, record: provider }) }}
                            >
                              <SettingOutlined className='text-blue-500' />
                            </Dropdown>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                </List.Item>
              );
            }}
          />
        )}
        <APIKeyComponent
          open={open}
          record={editValue}
          onCancelClick={onCancelClick}
          onOkClick={onOkClick}
        />
      </ACard>
    </ListContainer>
  );
}

const APIKeyComponent = ({ record = {}, onOkClick = () => {}, open = false, onCancelClick = () => {} }) => {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  useEffect(() => {
    form.setFieldsValue({
      api_key: record.api_key
    });
  }, [record]);
  const [loading, setLoading] = useState(false);
  let apiHref = '';
  switch (record.id) {
    case 'qianwen':
      apiHref = 'https://bailian.console.aliyun.com/?tab=model#/api-key';
      break;
    case 'deepseek':
      apiHref = 'https://platform.deepseek.com/api_keys';
      break;
    case 'gitee_ai':
      apiHref = 'https://ai.gitee.com/dashboard/settings/tokens';
      break;
    case 'openai':
      apiHref = 'https://platform.openai.com/account/api-keys';
      break;
    case 'silicon_flow':
      apiHref = 'https://cloud.siliconflow.cn/account/ak';
      break;
    case 'tencent_hunyuan':
      apiHref = 'https://console.cloud.tencent.com/hunyuan/api-key';
      break;
    case 'gemini':
      apiHref = 'https://aistudio.google.com/app/apikey';
      break;
    case 'moonshot':
      apiHref = 'https://platform.moonshot.cn/console/api-keys';
      break;
    case 'minimax':
      apiHref = 'https://platform.minimaxi.com/user-center/basic-information/interface-key';
      break;
    case 'volcanoArk':
      apiHref = 'https://console.volcengine.com/iam/keymanage/';
      break;
    case 'qianfan':
      apiHref = 'https://console.bce.baidu.com/iam/#/iam/apikey/list';
      break;
    case 'cohere':
      apiHref = 'https://dashboard.cohere.com/api-keys';
      break;
  }

  const onModalOkClick = () => {
    form.validateFields().then(values => {
      setLoading(true);
      record.api_key = values.api_key;
      updateModelProvider(record.id, record)
        .then(() => {
          setLoading(false);
          onOkClick();
        })
        .catch(() => {
          setLoading(false);
        });
    });
  };
  return (
    <Modal
      open={open}
      title={t('common.update') + t('page.modelprovider.labels.api_key')}
      onCancel={onCancelClick}
      onOk={onModalOkClick}
    >
      <Spin spinning={loading}>
        <Form
          className='my-2em'
          form={form}
          layout='vertical'
        >
          <Form.Item
            label={<span className='text-gray-500'>{t('page.modelprovider.labels.api_key')}</span>}
            name='api_key'
          >
            <Input defaultValue={record.api_key} />
          </Form.Item>
          {apiHref && (
            <div>
              <Button
                className='m-0 p-0'
                href={apiHref}
                target='_blank'
                type='link'
              >
                {t('page.modelprovider.labels.api_key_source', {
                  model_provider: record.name
                })}
                <ExportOutlined />
              </Button>
            </div>
          )}
        </Form>
      </Spin>
    </Modal>
  );
};
