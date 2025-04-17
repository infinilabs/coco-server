import { CloseOutlined, EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { useLoading } from '@sa/hooks';
import { Button, Dropdown, Form, Input, Modal, Spin, Switch, Table, message } from 'antd';

import { deleteIntegration, fetchIntegrations, fetchIntegrationTopics, updateIntegration, updateIntegrationTopics } from '@/service/api/integration';
import { formatESSearchResult } from '@/service/request/es';

const { confirm } = Modal;

export function Component() {
  const { t } = useTranslation();

  const { tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

  const [data, setData] = useState({
    data: [],
    total: 0
  });
  const { endLoading, loading, startLoading } = useLoading();
  const [topicsState, setTopicsState] = useState({
    open: false,
    record: undefined
  })

  const [reqParams, setReqParams] = useState({
    from: 0,
    query: '',
    size: 10
  });

  const fetchData = async reqParams => {
    startLoading();
    const res = await fetchIntegrations(reqParams);
    const newData = formatESSearchResult(res.data);
    setData(newData);
    endLoading();
  };

  const handleTableChange = pagination => {
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
        from: 0,
        query
      };
    });
  };

  const handleDelete = async id => {
    startLoading();
    const res = await deleteIntegration(id);
    if (res.data?.result === 'deleted') {
      message.success(t('common.deleteSuccess'));
    }
    fetchData(reqParams);
    endLoading();
  };

  const handleEnabled = async record => {
    startLoading();
    const { _index, _type, ...rest } = record;
    const res = await updateIntegration(rest);
    if (res.data?.result === 'updated') {
      message.success(t('common.updateSuccess'));
    }
    fetchData(reqParams);
    endLoading();
  };

  const columns = [
    {
      dataIndex: 'name',
      title: t('page.integration.columns.name')
    },
    {
      dataIndex: 'type',
      render: value => t(`page.integration.form.labels.type_${value}`),
      title: t('page.integration.columns.type')
    },
    {
      dataIndex: 'description',
      title: t('page.integration.columns.description')
    },
    {
      dataIndex: 'datasource',
      render: (value, record) => {
        return value?.includes('*') ? '*' : value?.length || 0;
      },
      title: t('page.integration.columns.datasource')
    },
    {
      dataIndex: 'enabled',
      render: (_, record) => {
        return (
          <Switch
            checked={record.enabled}
            size="small"
            onChange={checked => {
              confirm({
                content: t(`page.integration.update.${checked ? 'enable' : 'disable'}_confirm`, { name: record.name }),
                icon: <ExclamationCircleOutlined />,
                onOk() {
                  handleEnabled({ ...record, enabled: checked });
                },
                title: t('common.tip')
              });
            }}
          />
        );
      },
      title: t('page.integration.columns.enabled')
    },
    {
      fixed: 'right',
      render: (_, record) => {
        const items = [
          {
            key: 'edit',
            label: t('common.edit')
          },
          {
            key: 'topics',
            label: t('page.integration.columns.operation.topics')
          },
          {
            key: 'delete',
            label: t('common.delete')
          }
        ];

        const onMenuClick = ({ key, record }: any) => {
          switch (key) {
            case 'edit':
              nav(`/integration/edit/${record.id}`, { state: record });
              break;
            case 'topics':
              setTopicsState({
                open: true,
                record
              })
              break;
            case 'delete':
              confirm({
                content: t('page.integration.delete.confirm', { name: record.name }),
                icon: <ExclamationCircleOutlined />,
                onOk() {
                  handleDelete(record.id);
                },
                title: t('common.tip')
              });
              break;
          }
        };
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
  const rowSelection = {
    getCheckboxProps: record => ({
      name: record.name
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows) => {}
  };

  useEffect(() => {
    fetchData(reqParams);
  }, [reqParams]);

  return (
    <div className="h-full min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
      >
        <div className="mb-4 mt-4 flex items-center justify-between">
          <Input.Search
            addonBefore={<FilterOutlined />}
            className="max-w-500px"
            enterButton={t('common.refresh')}
            onSearch={onRefreshClick}
          />
          <Button
            icon={<PlusOutlined />}
            type="primary"
            onClick={() => nav(`/integration/new`)}
          >
            {t('common.add')}
          </Button>
        </div>
        <Table
          columns={columns}
          dataSource={data.data}
          loading={loading}
          rowKey="id"
          rowSelection={{ ...rowSelection }}
          size="middle"
          pagination={{
            defaultCurrent: 1,
            defaultPageSize: 10,
            showSizeChanger: true,
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
            total: data.total?.value || data?.total
          }}
          onChange={handleTableChange}
        />
        <ModalTopics
          open={topicsState.open}
          record={topicsState.record}
          onCancel={() => {
            setTopicsState({
              open: false,
              record: undefined
            })
          }}
          onOk={() => {
            setTopicsState({
              open: false,
              record: undefined
            })
          }}
        />
      </ACard>
    </div>
  );
}

const ModalTopics = ({ onCancel = () => {}, onOk = () => {}, open = false, record = {} }) => {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const { defaultRequiredRule } = useFormRules();

  const fetchTopics = async (id) => {
    if (!id) return;
    setLoading(true)
    const res = await fetchIntegrationTopics(id)
    form.setFieldsValue({ topics: res?.data?.length > 0 ? res.data : [''] })
    setLoading(false)
  }

  useEffect(() => {
    fetchTopics(record?.id)
  }, [record?.id])

  const onModalOkClick = () => {
    if (!record?.id) return;
    form.validateFields().then(async values => {
      setLoading(true);
      const { topics } = values;
      const res = await updateIntegrationTopics({
        id: record.id,
        topics
      });
      if (res.data?.acknowledged) {
        window.$message?.success(t('common.updateSuccess'));
        onOk()
      }
      setLoading(false);
    });
  };
  return (
    <Modal
      open={open}
      title={`${t('page.integration.topics.title')}`}
      onCancel={onCancel}
      onOk={onModalOkClick}
      destroyOnClose
      width={650}
    >
      <Spin spinning={loading}>
        <Form
          className="my-2em"
          form={form}
          layout="vertical"
          layout={'horizontal'}
          colon={false}
        >
          <Form.List name="topics">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field, index) => {
                  return (
                    <Form.Item key={field.key} className="m-0">
                      <div className="flex items-center gap-6px">
                        <Form.Item
                          {...field}
                          rules={[defaultRequiredRule]}
                          className="flex-1"
                        >
                          <Input placeholder={`${t(`page.integration.topics.label`)} ${index+1}`}/>
                        </Form.Item>
                        <Form.Item>
                          <Button disabled={fields.length <= 1} danger onClick={() => remove(field.name)}>{t(`page.integration.topics.delete`)}</Button>
                        </Form.Item>
                      </div>
                    </Form.Item>
                  )
                })}
                <Form.Item>
                  <Button disabled={fields.length >= 5} type="dashed" onClick={() => add()} block>{t(`page.integration.topics.new`)}</Button>
                </Form.Item>
              </>
            )}
          </Form.List>
        </Form>
      </Spin>
    </Modal>
  );
};