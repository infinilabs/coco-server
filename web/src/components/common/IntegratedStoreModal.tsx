import { PlusCircleFilled, PlusOutlined, UserOutlined } from '@ant-design/icons';
import { Avatar, Button, Card, Col, Flex, Input, Menu, Modal, Row, Space, Spin, Typography, message } from 'antd';
import type { ItemType, MenuItemType } from 'antd/es/menu/interface';
import classNames from 'classnames';
import { Eye, FolderDown, SquareArrowOutUpRight } from 'lucide-react';

// @ts-ignore
import FontIcon from '@/components/common/font_icon';
import { request } from '@/service/request';
import { formatESSearchResult } from '@/service/request/es';

type Category = 'ai-assistant' | 'connector' | 'data-source' | 'mcp-server' | 'model-provider';

export interface IntegratedStoreModalRef {
  open: (category: Category) => void;
}

interface DataSource {
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

const IntegratedStoreModal = forwardRef<IntegratedStoreModalRef>((_, ref) => {
  const [open, setOpen] = useState(false);
  const [category, setCategory] = useState<Category>('ai-assistant');
  const navigate = useNavigate();
  const [loadingData, setLoadingData] = useState(true);
  const [data, setData] = useState<DataSource[]>([]);
  const [installation, setInstallation] = useState(false);
  const [searchKeyword, setSearchKeyboard] = useState<string>();
  const { t } = useTranslation();

  useImperativeHandle(ref, () => ({
    open(category) {
      setOpen(true);
      setCategory(category);
    }
  }));

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
      if (!open) return;

      setLoadingData(true);

      const result = await request({
        method: 'get',
        url: `/store/server/${requestType}/_search?query=${searchKeyword}`
      });

      const { data } = formatESSearchResult(result);

      setData(data);
    } catch {
      // do something
    } finally {
      setLoadingData(false);
    }
  }, [open, requestType, searchKeyword]);

  const renderTitle = () => {
    return (
      <Space>
        <span>集成商店</span>

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
      label: 'AI 助手'
    },
    {
      key: 'model-provider',
      label: '模型提供商'
    },
    {
      key: 'data-source',
      label: '数据源'
    },
    {
      key: 'mcp-server',
      label: 'MCP Server'
    },
    {
      key: 'connector',
      label: '连接器'
    }
  ];

  const handleCancel = () => {
    setOpen(false);
  };

  const tabItems = [
    {
      label: '推荐',
      value: '推荐'
    },
    {
      label: '最新',
      value: '最新'
    }
  ];

  const [currentTab, setCurrentTab] = useState(tabItems[0].value);

  const handleNew = useCallback(() => {
    let link = `/${category}/new`;

    if (category === 'data-source') {
      link += '-first';
    }

    navigate(link);
  }, [category, navigate]);

  const handleInstall = async (id: string) => {
    try {
      setInstallation(true);

      const { data } = await request({
        method: 'post',
        url: ` /store/${requestType}/${id}/_install`
      });

      if (data.acknowledged) {
        message.success(t('common.installSuccess'));

        navigate(data.redirect_url);
      }
    } catch (error) {
      message.error(String(error));
    } finally {
      setInstallation(false);
    }
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

            <span className="text-primary">自定义</span>
          </Card>
        </Col>

        {data.map(item => {
          const { description, developer, icon, id, name, stats } = item;

          return (
            <Col
              key={id}
              span={6}
            >
              <Card className="group text-xs text-color-3 hover:(border-primary bg-primary-50)">
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
                    {icon.startsWith('font_') ? (
                      <FontIcon
                        className="text-8"
                        name={icon}
                      />
                    ) : (
                      <img
                        className="size-8"
                        src={icon}
                      />
                    )}

                    <div className="truncate text-sm text-color-1">{name}</div>

                    <Space>
                      <Avatar
                        icon={<UserOutlined />}
                        size={20}
                      />
                      <img
                        className="size-4"
                        src={developer.avatar}
                      />

                      <span>{developer.name}</span>
                    </Space>

                    <div className="line-clamp-3">{description}</div>
                  </Flex>

                  <Flex
                    align="center"
                    className="group-hover:hidden"
                    justify="space-between"
                  >
                    <Space>
                      <Eye className="size-4" />

                      <span>{stats.views}</span>
                    </Space>

                    <Space>
                      <FolderDown className="size-4" />

                      <span>{stats.installs}</span>
                    </Space>
                  </Flex>

                  <div className="hidden -mx-2 -my-1 group-hover:block">
                    <Button
                      block
                      size="small"
                      type="primary"
                      onClick={() => {
                        handleInstall(id);
                      }}
                    >
                      {t('common.install')}
                    </Button>
                  </div>
                </Flex>
              </Card>
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

        <span className="text-color-3">No Results</span>

        <Button
          type="primary"
          onClick={handleNew}
        >
          <PlusOutlined />

          <span>自定义</span>
        </Button>
      </Flex>
    );
  };

  return (
    <>
      <Modal
        centered
        footer={null}
        maskClosable={false}
        open={open}
        title={renderTitle()}
        width={860}
        classNames={{
          body: '-mx-5'
        }}
        onCancel={handleCancel}
      >
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
      </Modal>

      <Spin
        fullscreen
        spinning={installation}
      />
    </>
  );
});

export default IntegratedStoreModal;
