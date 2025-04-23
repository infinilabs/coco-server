import Icon, { EllipsisOutlined, ExclamationCircleOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Dropdown, Form, Input, Modal, Spin, Table, message } from 'antd';
import type { MenuProps, TableColumnsType } from 'antd';
import { useForm } from 'antd/es/form/Form';
import Search from 'antd/es/input/Search';
import Clipboard from 'clipboard';

import { createToken, deleteToken, getTokens, renameToken } from '@/service/api';

type APIToken = Api.APIToken.APIToken;

export function Component() {
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
              setReqParams(old => {
                return {
                  ...old
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

  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [reqParams, setReqParams] = useState({
    query: '',
    t: new Date().getTime()
  });

  const fetchData = () => {
    setLoading(true);
    getTokens().then(({ data }) => {
      setData(data || []);
      setLoading(false);
    });
  };

  useEffect(fetchData, [reqParams]);
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

  const onSearchClick = (query: string) => {
    setReqParams(old => {
      return {
        query,
        t: new Date().getTime()
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
      setReqParams(old => {
        return {
          ...old
        };
      });
    }
    setCreateState(initialCreateState);
  };

  return (
    <div className="h-full min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
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
          dataSource={data}
          loading={loading}
          rowKey="id"
          size="middle"
          pagination={{
            defaultCurrent: 1,
            defaultPageSize: 10,
            showSizeChanger: true,
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`
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
            setReqParams(old => {
              return {
                ...old
              };
            });
          }}
        />
      </ACard>
    </div>
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
