import Icon, { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Dropdown, Form, Input, Modal, Spin, Table, message } from 'antd';
import type { GetProp, MenuProps, TableColumnsType, TableProps } from 'antd';
import { useForm } from 'antd/es/form/Form';
import Search from 'antd/es/input/Search';
import Clipboard from 'clipboard';
import { formatESSearchResult } from '@/service/request/es';
import useQueryParams from '@/hooks/common/queryParams';

import { createToken, deleteToken, getTokens, renameToken } from '@/service/api';

type APIToken = Api.APIToken.APIToken;

type TablePaginationConfig = Exclude<GetProp<TableProps, 'pagination'>, boolean>;

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams({
    size: 10,
    from: 0
  });
  const { t } = useTranslation();
  const nav = useNavigate();

  const items: MenuProps['items'] = [
    {
      key: '1',
      label: t('common.rename')
    },
    {
      key: '2',
      label: t('common.delete')
    }
  ];

  const [renameState, setRenameState] = useState({
    open: false,
    tokenInfo: {}
  });

  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '2':
        window?.$modal?.confirm({
          content: t('page.apitoken.delete.confirm'),
          icon: <ExclamationCircleOutlined />,
          onCancel() {},
          onOk() {
            // delete datasource by datasource id
            deleteToken(record.id).then(res => {
              if (res.data?.result === 'deleted') {
                message.success(t('common.deleteSuccess'));
              }
              // reload data
              setQueryParams((old: any) => {
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
      case '1':
        setRenameState({
          open: true,
          tokenInfo: record
        });
        break;
    }
  };
  const columns: TableColumnsType<APIToken> = [
    {
      dataIndex: 'name',
      minWidth: 100,
      title: t('page.apitoken.columns.name')
    },
    {
      dataIndex: 'access_token',
      minWidth: 200,
      title: 'Token'
    },
    {
      dataIndex: 'expire_in',
      minWidth: 100,
      render: (value: number) => {
        return value ? new Date(value * 1000).toISOString() : '';
      },
      title: t('page.apitoken.columns.expire_in')
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

  const initialData = {
    data: [],
    total: 0
  };
  const [data, setData] = useState(initialData);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState();

  const fetchData = () => {
    setLoading(true);
    getTokens(queryParams).then(({ data }) => {
      const newData = formatESSearchResult(data);
      setData((oldData: any) => {
        return {
          ...oldData,
          ...(newData || initialData)
        };
      });
      setLoading(false);
    }).catch((error) => {
      console.error('Error fetching tokens:', error);
      setLoading(false);
    });
  };

  useEffect(fetchData, [queryParams]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modelForm] = useForm();
  const onAddClick = () => {
    // if(data.length >= 5){
    //   message.warning(t('page.apitoken.create.limit'));
    //   return;
    // }
    const tokenName = generateApiTokenName();
    setIsModalOpen(true);
    modelForm.setFieldsValue({ name: tokenName });
  };

  useEffect(() => {
    setKeyword(queryParams.query)
  }, [queryParams.query])

  const onSearchClick = (query: string) => {
    setQueryParams((oldParams: any) => {
      return {
        ...oldParams,
        from: 0,
        query,
        t: new Date().valueOf()
      };
    });
  };

  const initialCreateState = {
    loading: false,
    step: 1,
    token: ''
  };
  const [createState, setCreateState] = useState(initialCreateState);

  const buttonRef = useRef(null);
  useEffect(() => {
    if (!buttonRef.current) return;
    const clipboard = new Clipboard(buttonRef.current, {
      text: () => {
        return createState.token;
      }
    });
    clipboard.on('success', () => {
      window.$message?.success(t('common.copySuccess'));
    });

    return () => clipboard.destroy();
  }, [createState.token, buttonRef.current, isModalOpen]);

  const onModalOkClick = () => {
    if (createState.step === 1) {
      modelForm.validateFields().then(values => {
        setCreateState(old => {
          return {
            ...old,
            loading: true
          };
        });
        createToken(values.name)
          .then(({ data }) => {
            setCreateState(old => {
              return {
                ...old,
                loading: false,
                step: 2,
                token: data?.access_token || ''
              };
            });
          })
          .catch(() => {
            setCreateState(old => {
              return {
                ...old,
                loading: false
              };
            });
          });
      });
    }
  };
  const onModalCancel = () => {
    setIsModalOpen(false);
    if (createState.step === 2) {
      setQueryParams((old: any) => {
        return {
          ...old
        };
      });
    }
    setCreateState(initialCreateState);
  };

  return (
    <ListContainer>
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-4 mt-4 flex items-center justify-between">
          <Search
            addonBefore={<FilterOutlined />}
            className="max-w-500px"
            enterButton={t('common.refresh')}
            onSearch={onSearchClick}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
          />
          <Button
            icon={<PlusOutlined />}
            type="primary"
            onClick={onAddClick}
          >
            {t('common.add')}
          </Button>
        </div>
        <Table<APIToken>
          columns={columns}
          dataSource={data.data}
          loading={loading}
          rowKey="id"
          size="middle"
          pagination={{
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
            pageSize: queryParams.size,
            current: Math.floor(queryParams.from / queryParams.size) + 1,
            total: data.total?.value || data?.total,
            showSizeChanger: true
          }}
          onChange={(pagination, filters, sorter) => {
            setQueryParams((params: any) => {
              return {
                ...params,
                from: (pagination.current - 1) * pagination.pageSize,
                size: pagination.pageSize
              };
            });
          }}
        />
        <Modal
          okText={t('common.create')}
          open={isModalOpen}
          title={`${t('common.create')} API Token`}
          footer={(_, { CancelBtn, OkBtn }) => (
            <>
              {createState.step === 1 && (
                <>
                  <CancelBtn />
                  <OkBtn />
                </>
              )}
              {createState.step === 2 && (
                <>
                  <Button onClick={onModalCancel}>{t('common.close')}</Button>
                  <Button
                    ref={buttonRef}
                    type="primary"
                  >
                    {t('common.copy')}
                  </Button>
                </>
              )}
            </>
          )}
          onCancel={onModalCancel}
          onOk={onModalOkClick}
        >
          <Spin spinning={createState.loading}>
            {createState.step === 1 && (
              <Form
                className="my-2em"
                form={modelForm}
                layout="vertical"
              >
                <Form.Item
                  label={<span className="text-gray-500">{t('page.apitoken.columns.name')}</span>}
                  name="name"
                >
                  <Input />
                </Form.Item>
              </Form>
            )}
            {createState.step === 2 && (
              <div>
                <div className="my-[15px] font-size-[12px] text-gray-500">{t('page.apitoken.create.store_desc')}</div>
                <div className="rounded bg-gray-100 py-[3px] pl-1em text-gray-500 leading-[1.4em]">
                  {createState.token}
                </div>
              </div>
            )}
          </Spin>
        </Modal>
        <RenameComponent
          open={renameState.open}
          tokenInfo={renameState.tokenInfo}
          onCancelClick={() => {
            setRenameState({
              open: false,
              tokenInfo: {}
            });
          }}
          onOkClick={() => {
            setRenameState({
              open: false,
              tokenInfo: {}
            });
            setQueryParams((old: any) => {
              return {
                ...old
              };
            });
          }}
        />
      </ACard>
    </ListContainer>
  );
}

function generateApiTokenName(prefix = 'token') {
  const timestamp = Date.now(); // Current timestamp in milliseconds
  const randomString = Math.random().toString(36).substring(2, 10); // Random alphanumeric string
  return `${prefix}_${timestamp}_${randomString}`;
}

const RenameComponent = ({ onCancelClick = () => {}, onOkClick = () => {}, open = false, tokenInfo = {} }) => {
  const { t } = useTranslation();
  const [form] = useForm();
  const [loading, setLoading] = useState(false);

  const onModalOkClick = () => {
    form.validateFields().then(values => {
      setLoading(true);
      renameToken(tokenInfo.id, values.name)
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
      title={`${t('common.rename')} API Token`}
      onCancel={onCancelClick}
      onOk={onModalOkClick}
    >
      <Spin spinning={loading}>
        <Form
          className="my-2em"
          form={form}
          layout="vertical"
        >
          <Form.Item
            label={<span className="text-gray-500">{t('page.apitoken.columns.name')}</span>}
            name="name"
          >
            <Input defaultValue={tokenInfo.name} />
          </Form.Item>
        </Form>
      </Spin>
    </Modal>
  );
};
