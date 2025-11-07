import { PlusCircleFilled, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Col, Flex, Input, Menu, Modal, Row, Space, Spin, Typography, message } from 'antd';
import type { AnyObject } from 'antd/es/_util/type';
import type { ItemType, MenuItemType } from 'antd/es/menu/interface';
import classNames from 'classnames';
import { castArray, isEmpty } from 'lodash';
import { Eye, FolderDown, SquareArrowOutUpRight } from 'lucide-react';
import type { ReactNode } from 'react';

// @ts-ignore
import FontIcon from '@/components/common/font_icon';
import { request } from '@/service/request';
import { formatESSearchResult } from '@/service/request/es';
import { localStg } from '@/utils/storage';

type Category = 'ai-assistant' | 'connector' | 'data-source' | 'mcp-server' | 'model-provider';

export interface DataSource {
  _system: {
    owner_id: string;
    tenant_id: string;
  };
  category: string;
  created: string;
  description: string;
  developer: {
    _system: {
      owner_id: string;
      tenant_id: string;
    };
    avatar: string;
    created: string;
    github_handle: string;
    id: string;
    location: string;
    name: string;
    updated: string;
    website: string;
  };
  icon: string;
  id: string;
  name: string;
  platforms: string[];
  screenshots: {
    title: string;
    url: string;
  }[];
  stats: {
    installs: number;
    views: number;
  };
  tags: string[];
  type: string;
  updated: string;
  url: {
    code: string;
  };
  version: {
    number: string;
  };
}

export interface IntegratedStoreModalRef {
  open: (category: Category) => void;
}

const IntegratedStoreModal = forwardRef<IntegratedStoreModalRef>((_, ref) => {
  const [open, setOpen] = useState(false);
  const [category, setCategory] = useState<Category>('ai-assistant');
  const navigate = useNavigate();
  const [loadingData, setLoadingData] = useState(true);
  const [data, setData] = useState<DataSource[]>([]);
  const [installation, setInstallation] = useState(false);
  const [searchKeyword, setSearchKeyboard] = useState<string>();
  const { t } = useTranslation();
  const [fromClipboard, setFromClipboard] = useState(false);

  useImperativeHandle(ref, () => ({
    async open(category) {
      setOpen(true);
      setCategory(category);

      try {
        const text = await navigator.clipboard.readText();

        const { id } = JSON.parse(text);

        const { data } = await request({
          method: 'get',
          url: `/store/server/${id}`
        });

        const dataSource = data._source;

        if (dataSource) {
          setFromClipboard(true);

          setData(castArray(dataSource));
        }
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error('读取剪贴板失败：', error);
      }
    }
  }));

  const tabItems = [
    {
      label: t('page.integratedStoreModal.labels.recommend'),
      value: 'recommend'
    },
    {
      label: t('page.integratedStoreModal.labels.newest'),
      value: 'newest'
    }
  ];

  const [currentTab, setCurrentTab] = useState(tabItems[0].value);

  const requestType = useMemo(() => {
    switch (category) {
      case 'ai-assistant':
        return 'assistant';
      case 'data-source':
        return 'datasource';
      case 'mcp-server':
        return 'mcp';
      case 'model-provider':
        return 'llm-provider';
      default:
        return category;
    }
  }, [category]);

  useAsyncEffect(async () => {
    try {
      if (!open || fromClipboard) return;

      setLoadingData(true);

      const params: AnyObject = {};

      if (!isEmpty(searchKeyword)) {
        params.query = searchKeyword;
      }

      if (currentTab === 'newest') {
        params.sort = 'created:desc';
      }
      const providerInfo = localStg.get('providerInfo');
      let storeUrl = `/store/server/_search?filter=type:${requestType}`;
      if(providerInfo.store?.local === false){
        storeUrl = providerInfo.store.endpoint + storeUrl
      }
      const searchParams = new URLSearchParams();
      Object.entries(params).forEach(([key, value]) => {
        searchParams.append(key, value);
      });
  
      storeUrl += `&${searchParams.toString()}`; 

      const res = await fetch(storeUrl, {
        method: 'get',
      });
      const result = await res.json();
      const { data } = formatESSearchResult(result);

      setData(data);
    } catch(error) {
      console.error(error);
      // do something
    } finally {
      setLoadingData(false);
    }
  }, [open, requestType, searchKeyword, currentTab, fromClipboard]);

  const renderTitle = () => {
    if (fromClipboard) {
      return t('page.integratedStoreModal.installModal.title');
    }

    return (
      <Space>
        <span>{t('page.integratedStoreModal.title')}</span>

        <Typography.Link
          href="https://coco.rs/zh/integration"
          target="_blank"
        >
          <SquareArrowOutUpRight className="size-4 text-primary" />
        </Typography.Link>
      </Space>
    );
  };

  const menuItems: ItemType<MenuItemType & { key: Category }>[] = [
    {
      key: 'ai-assistant',
      label: t('page.integratedStoreModal.labels.aiAssistant')
    },
    {
      key: 'model-provider',
      label: t('page.integratedStoreModal.labels.modelProvider')
    },
    {
      key: 'data-source',
      label: t('page.integratedStoreModal.labels.datasource')
    },
    {
      key: 'mcp-server',
      label: t('page.integratedStoreModal.labels.mcpServer')
    },
    {
      key: 'connector',
      label: t('page.integratedStoreModal.labels.connector')
    }
  ];

  const handleCancel = () => {
    if (fromClipboard) {
      setFromClipboard(false);
    } else {
      setOpen(false);
    }
  };

  const handleNew = useCallback(() => {
    let link = `/${category}/new`;

    if (category === 'data-source') {
      link += '-first';
    }
    setOpen(false)
    navigate(link);
  }, [category, navigate]);

  const handleInstall = async (id: string) => {
    try {
      setInstallation(true);

      const { data } = await request({
        method: 'post',
        url: `/store/server/${id}/_install`
      });

      if (data.acknowledged) {
        message.success(t('page.integratedStoreModal.hints.installSuccess'));

        setTimeout(() => {
          navigate(data.redirect_url);
        }, 3000);
      }
    } catch (error) {
      message.error(String(error));
    } finally {
      setInstallation(false);
    }
  };

  const handleOk = () => {
    if (!fromClipboard) return;

    handleInstall(data[0].id);
  };

  const renderDataCard = (data: DataSource, children?: ReactNode) => {
    return (
      <Card
        className={classNames('group text-xs text-color-3', {
          'hover:(border-primary bg-primary-50)': !fromClipboard,
          'mt-2 h-50 [&_.ant-card-body]:h-full': fromClipboard
        })}
      >
        <Flex
          vertical
          className="h-full"
          justify="space-between"
        >
          <Flex
            vertical
            gap={12}
            justify="space-between"
          >
            {data?.icon?.startsWith('font_') ? (
              <FontIcon
                className="text-8"
                name={data?.icon}
              />
            ) : (
              <img
                className="size-8"
                src={data?.icon}
              />
            )}

            <div className="truncate text-sm text-color-1">{data?.name}</div>

            <Space>
              <img
                className="size-4"
                src={data?.developer?.avatar}
              />

              <span>{data?.developer?.name}</span>
            </Space>

            <div className="line-clamp-3">{data?.description}</div>
          </Flex>

          <Flex
            align="center"
            justify="space-between"
            className={classNames({
              'group-hover:hidden': !fromClipboard
            })}
          >
            <Space>
              <Eye className="size-4" />

              <span>{data?.stats?.views || '-'}</span>
            </Space>

            <Space>
              <FolderDown className="size-4" />

              <span>{data?.stats?.installs || '-'}</span>
            </Space>
          </Flex>

          {children}
        </Flex>
      </Card>
    );
  };

  const renderData = () => {
    return (
      <Row
        className="max-h-100 overflow-auto pr-3 [&_.ant-card-body]:h-full [&_.ant-card]:(h-full cursor-pointer transition) children:h-55"
        gutter={[8, 8]}
      >
        <Col span={6}>
          <Card
            className="border-primary"
            classNames={{
              body: 'flex flex-col items-center justify-center gap-6'
            }}
            onClick={handleNew}
          >
            <PlusCircleFilled className="text-8 text-primary" />

            <span className="text-primary">{t('page.integratedStoreModal.buttons.custom')}</span>
          </Card>
        </Col>

        {data.map(item => {
          const { id } = item;

          return (
            <Col
              key={id}
              span={6}
            >
              {renderDataCard(
                item,
                <div className="hidden -mx-2 -my-1 group-hover:block">
                  <Button
                    block
                    size="small"
                    type="primary"
                    onClick={() => {
                      handleInstall(id);
                    }}
                  >
                    {t('page.integratedStoreModal.buttons.install')}
                  </Button>
                </div>
              )}
            </Col>
          );
        })}
      </Row>
    );
  };

  const renderEmpty = () => {
    return (
      <Flex
        vertical
        align="center"
        className="h-90 pr-3"
        gap={24}
        justify="center"
      >
        <NoDataIcon size={96} />

        <span className="text-color-3">{t('page.integratedStoreModal.hints.noResults')}</span>

        <Button
          type="primary"
          onClick={handleNew}
        >
          <PlusOutlined />

          <span>{t('page.integratedStoreModal.buttons.custom')}</span>
        </Button>
      </Flex>
    );
  };

  return (
    <>
      <Modal
        centered
        cancelText={fromClipboard && t('page.integratedStoreModal.installModal.buttons.return')}
        footer={fromClipboard ? undefined : null}
        maskClosable={false}
        okText={fromClipboard && t('page.integratedStoreModal.installModal.buttons.install')}
        open={open}
        title={renderTitle()}
        width={fromClipboard ? 450 : 860}
        classNames={{
          body: fromClipboard ? undefined : '-mx-5'
        }}
        onCancel={handleCancel}
        onOk={handleOk}
      >
        {fromClipboard ? (
          <>
            <span>{t('page.integratedStoreModal.installModal.hints')}</span>

            {renderDataCard(data[0])}
          </>
        ) : (
          <Flex>
            <Menu
              items={menuItems}
              selectedKeys={[category]}
              style={{ width: 150 }}
              onSelect={({ key }) => {
                setCategory(key as Category);
              }}
            />

            <div className="flex-1 pl-4">
              <Flex
                align="center"
                className="w-full pr-3"
                justify="space-between"
              >
                <Flex align="center">
                  {tabItems.map(item => {
                    const { label, value } = item;

                    return (
                      <div
                        key={value}
                        className={classNames('cursor-pointer rounded-full px-4 lh-8 transition hover:text-primary', {
                          'text-primary bg-primary-50 dark:bg-primary-900': value === currentTab
                        })}
                        onClick={() => {
                          setCurrentTab(value);
                        }}
                      >
                        {label}
                      </div>
                    );
                  })}
                </Flex>

                <Input.Search
                  className="w-60"
                  onSearch={setSearchKeyboard}
                />
              </Flex>

              <Spin spinning={loadingData}>
                <div className="pt-4">{data.length > 0 ? renderData() : renderEmpty()}</div>
              </Spin>
            </div>
          </Flex>
        )}
      </Modal>

      <Spin
        fullscreen
        rootClassName="z-10000"
        spinning={installation}
      />
    </>
  );
});

export default IntegratedStoreModal;
