import Icon, {
  ExclamationCircleOutlined,
  ExportOutlined,
  FilterOutlined,
  PlusOutlined,
  SettingOutlined
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { Button, Dropdown, Form, Image, Input, List, Modal, Spin, Switch, Tag, message } from 'antd';
import Search from 'antd/es/input/Search';

import type { IntegratedStoreModalRef } from '@/components/common/IntegratedStoreModal';
import InfiniIcon from '@/components/common/icon';
import useQueryParams from '@/hooks/common/queryParams';
import { deleteModelProvider, searchModelPovider, updateModelProvider } from '@/service/api/model-provider';
import { formatESSearchResult } from '@/service/request/es';

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams({
    size: 12,
    sort: [
      ['enabled', 'desc'],
      ['created', 'desc']
    ]
  });
  const { t } = useTranslation();
  const nav = useNavigate();
  const [data, setData] = useState({
    data: [],
    total: 0
  });
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState();

  const fetchData = () => {
    setLoading(true);
    searchModelPovider(queryParams)
      .then(data => {
        const newData = formatESSearchResult(data.data);
        setData(newData);
      })
      .finally(() => {
        setLoading(false);
      });
  };
  useEffect(fetchData, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query);
  }, [queryParams.query]);

  const onSearchClick = (query: string) => {
    setQueryParams({
      ...queryParams,
      query,
      t: new Date().valueOf()
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
  const getMenuItems = useCallback((record: any): MenuProps['items'] => {
    const items: MenuProps['items'] = [
      {
        key: '1',
        label: t('common.edit')
      }
    ];
    if (record.builtin !== true) {
      items.push({
        key: '2',
        label: t('common.delete')
      });
    }
    return items;
  }, []);

  const onMenuClick = ({ key, record }: any) => {
    switch (key) {
      case '2':
        window?.$modal?.confirm({
          content: t('page.modelprovider.delete.confirm'),
          icon: <ExclamationCircleOutlined />,
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
          },
          title: t('common.tip')
        });

        break;
      case '1':
        nav(`/model-provider/edit/${record.id}`);
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
    fetchData();
  };
  const onCancelClick = () => {
    setOpen(false);
  };

  const onAPIKeyClick = (record: any) => {
    setEditValue(record);
    setOpen(true);
  };

  const integratedStoreModalRef = useRef<IntegratedStoreModalRef>(null);

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
            value={keyword}
            onChange={e => setKeyword(e.target.value)}
            onSearch={onSearchClick}
          />
          <Button
            icon={<PlusOutlined />}
            type="primary"
            onClick={() => {
              integratedStoreModalRef.current?.open('model-provider');
            }}
          >
            {t('common.add')}
          </Button>
        </div>
        <List
          dataSource={data.data}
          grid={{ column: 3, gutter: 16, sm: 2, xs: 1 }}
          loading={loading}
          pagination={{
            current: Math.floor(queryParams.from / queryParams.size) + 1,
            onChange: onPageChange,
            pageSize: queryParams.size,
            pageSizeOptions: [12, 24, 48, 96],
            showSizeChanger: true,
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
            total: data.total?.value || data?.total
          }}
          renderItem={provider => (
            <List.Item>
              <div className="group min-h-[132px] border border-[var(--ant-color-border)] rounded-[8px] p-1em hover:bg-[var(--ant-control-item-bg-hover)]">
                <div className="flex justify-between">
                  <div className="flex gap-15px">
                    <div className="flex items-center gap-8px">
                      <IconWrapper className="h-40px w-40px">
                        <InfiniIcon
                          className="font-size-2em"
                          height="2em"
                          src={provider.icon}
                          width="2em"
                        />
                      </IconWrapper>
                      <span
                        className="cursor-pointer font-size-1.2em hover:text-blue-500"
                        onClick={() => nav(`/model-provider/edit/${provider.id}`)}
                      >
                        {provider.name}
                      </span>
                    </div>
                    {provider.builtin === true && (
                      <div className="flex items-center">
                        <p className="h-[22px] rounded-[4px] bg-[#eee] px-[10px] font-size-[12px] text-[#999] line-height-[22px]">
                          {t('page.modelprovider.labels.builtin')}
                        </p>
                      </div>
                    )}
                  </div>
                  <div>
                    <Switch
                      checked={provider.enabled}
                      size="small"
                      onChange={v => onItemEnableChange(provider, v)}
                    />
                  </div>
                </div>
                <div className="line-clamp-3 my-[10px] h-[51px] text-xs text-[#999]">{provider.description}</div>
                <div className="flex gap-1">
                  <div className="ml-auto flex gap-2">
                    <div
                      className="cursor-pointer border border-[var(--ant-color-border)] rounded-[8px] px-10px"
                      onClick={() => {
                        onAPIKeyClick(provider);
                      }}
                    >
                      API-key
                    </div>
                    <div className="inline-block cursor-pointer border border-[var(--ant-color-border)] rounded-[8px] px-4px">
                      <Dropdown
                        menu={{
                          items: getMenuItems(provider),
                          onClick: ({ key }) => onMenuClick({ key, record: provider })
                        }}
                      >
                        <SettingOutlined className="text-blue-500" />
                      </Dropdown>
                    </div>
                  </div>
                </div>
              </div>
            </List.Item>
          )}
        />
        <APIKeyComponent
          open={open}
          record={editValue}
          onCancelClick={onCancelClick}
          onOkClick={onOkClick}
        />
      </ACard>

      <IntegratedStoreModal ref={integratedStoreModalRef} />
    </ListContainer>
  );
}

const APIKeyComponent = ({ onCancelClick = () => {}, onOkClick = () => {}, open = false, record = {} }) => {
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
          className="my-2em"
          form={form}
          layout="vertical"
        >
          <Form.Item
            label={<span className="text-gray-500">{t('page.modelprovider.labels.api_key')}</span>}
            name="api_key"
          >
            <Input defaultValue={record.api_key} />
          </Form.Item>
          {apiHref && (
            <div>
              <Button
                className="m-0 p-0"
                href={apiHref}
                target="_blank"
                type="link"
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
