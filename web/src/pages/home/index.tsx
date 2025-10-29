import { useLoading, useRequest } from '@sa/hooks';
import { Button, Card, Col, Form, Input, Row, Spin } from 'antd';
import Clipboard from 'clipboard';

import { fetchServer, updateSettings } from '@/service/api/server';
import { selectUserInfo } from '@/store/slice/auth';
import { getDarkMode } from '@/store/slice/theme';
import { localStg } from '@/utils/storage';

const SETTINGS = [
  {
    icon: <SvgIcon icon="mdi:settings-outline" />,
    key: 'llm',
    link: '/model-provider/list',
    permissions: ['coco#model_provider/search']
  },
  {
    icon: <SvgIcon icon="mdi:plus-thick" />,
    key: 'dataSource',
    link: '/data-source',
    permissions: ['coco#datasource/search']
  },
  {
    icon: <SvgIcon icon="mdi:plus-thick" />,
    key: 'aiAssistant',
    link: '/ai-assistant',
  }
];

export function Component() {
  const userInfo = useAppSelector(selectUserInfo);
  const routerPush = useRouterPush();
  const { t } = useTranslation();
  const { hasAuth } = useAuth();
  const permissions = {
    update: true,
  }

  const domRef = useRef<HTMLDivElement | null>(null);
  const [form] = Form.useForm();
  const { endLoading, loading, startLoading } = useLoading();
  const darkMode = useAppSelector(getDarkMode);

  const { defaultRequiredRule } = useFormRules();

  const [isNameEditing, setIsNameEditing] = useState(false);
  const [isEndpointEditing, setIsEndpointEditing] = useState(false);

  const providerInfo = localStg.get('providerInfo');
  const managed = Boolean(providerInfo?.managed);

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchServer, {
    manual: true
  });

  const handleSubmit = async (field: 'endpoint' | 'name', callback?: () => void) => {
    const params = await form.validateFields([field]);
    startLoading();
    const { error } = await updateSettings({
      server: {
        ...(data || {}),
        [field]: params[field]
      }
    });
    if (error) {
      form.setFieldsValue({ [field]: data?.[field] });
    } else {
      run();
    }
    endLoading();
    if (callback) callback();
  };

  const initClipboard = (text?: string) => {
    if (!domRef.current || !text) return;

    const clipboard = new Clipboard(domRef.current, {
      text: () => text
    });

    clipboard.on('success', () => {
      window.$message?.success(t('common.copySuccess'));
    });
  };

  useMount(() => {
    run();
  });

  useEffect(() => {
    if (domRef.current) {
      initClipboard(data?.endpoint);
    }
  }, [data?.endpoint, domRef.current]);

  useEffect(() => {
    form.setFieldsValue({ endpoint: data?.endpoint, name: data?.name });
  }, [JSON.stringify(data)]);

  return (
    <Spin spinning={dataLoading || loading}>
      <Card
        className="m-b-12px px-32px py-40px"
        classNames={{ body: '!p-0' }}
      >
        <div className={`flex ${isNameEditing ? '[align-items:self-end]' : 'items-center'} m-b-48px`}>
          <div
            className={`h-40px leading-40px m-r-16px text-32px color-[var(--ant-color-text-heading)] relative ${isNameEditing ? 'w-344px' : ''}`}
          >
            {isNameEditing && (
              <Form
                className="absolute left-0 top-2px z-1 w-100%"
                form={form}
              >
                <Form.Item
                  className="m-b-0"
                  name="name"
                  rules={[defaultRequiredRule]}
                >
                  <Input
                    autoFocus
                    className="[min-width:344px] m-r-16px h-40px w-100%"
                  />
                </Form.Item>
              </Form>
            )}
            {data?.name ? data?.name : <span>{t('page.home.server.title', { user: userInfo?.name })}</span>}
          </div>
          {
            permissions.update && (
              <Button
                className="h-40px w-40px rounded-12px p-0"
                style={{ background: `${darkMode ? 'var(--ant-color-border)' : '#F7F9FC'}` }}
                type="link"
                onClick={() => {
                  if (isNameEditing) {
                    handleSubmit('name', () => setIsNameEditing(!isNameEditing));
                  } else {
                    setIsNameEditing(!isNameEditing);
                  }
                }}
              >
                <SvgIcon
                  className="text-24px"
                  icon={isNameEditing ? 'mdi:content-save' : 'mdi:square-edit-outline'}
                />
              </Button>
            )
          }
        </div>
        <div className="m-b-16px text-20px color-[var(--ant-color-text-heading)]">{t('page.home.server.address')}</div>
        <div className="m-b-16px flex">
          <div
            className="relative m-r-8px h-48px w-400px rounded-[var(--ant-border-radius)] p-r-30px color-[var(--ant-color-text-heading)] leading-48px"
            style={{ background: `${darkMode ? 'var(--ant-color-border)' : '#F7F9FC'}` }}
          >
            {isEndpointEditing ? (
              <Form form={form}>
                <Form.Item
                  name="endpoint"
                  rules={[defaultRequiredRule]}
                >
                  <Input
                    autoFocus
                    className="h-48px w-[calc(100%+30px)] p-r-32px"
                    onBlur={e => {
                      if (e.relatedTarget?.id !== 'endpoint-save') {
                        form.setFieldsValue({ endpoint: data?.endpoint });
                      }
                    }}
                  />
                </Form.Item>
              </Form>
            ) : (
              <div className="p-l-11px">{data?.endpoint}</div>
            )}
            {
              permissions.update && !managed && (
                <Button
                  className="absolute right-0 top-0 z-1 h-48px w-30px rounded-12px p-0"
                  id="endpoint-save"
                  type="link"
                  onClick={() => {
                    if (isEndpointEditing) {
                      handleSubmit('endpoint', () => setIsEndpointEditing(!isEndpointEditing));
                    } else {
                      setIsEndpointEditing(!isEndpointEditing);
                    }
                  }}
                >
                  <SvgIcon
                    className="text-24px"
                    icon={isEndpointEditing ? 'mdi:content-save' : 'mdi:square-edit-outline'}
                  />
                </Button>
              )
            }
          </div>
          <div ref={domRef}>
            <Button
              className="h-48px w-100px"
              type="primary"
            >
              <SvgIcon
                className="text-24px"
                icon="mdi:content-copy"
              />
            </Button>
          </div>
        </div>
        <div className="color-var(--ant-color-text) m-b-16px">{t('page.home.server.addressDesc')}</div>
        <Button
          className="px-0"
          type="link"
          onClick={() => window.open('https://coco.rs/#install', '_blank')}
        >
          {t('page.home.server.downloadCocoAI')}{' '}
          <SvgIcon
            className="text-16px"
            icon="mdi:external-link"
          />
        </Button>
      </Card>
      <Card
        className="p-32px"
        classNames={{ body: 'flex gap-32px justify-start !p-0 -mx-32px' }}
      >
        <Row
          className="w-100%"
          gutter={{ lg: 32, md: 24, sm: 16, xs: 8 }}
        >
          {SETTINGS.map(item => (
            <Col
              className="m-b-24px"
              key={item.key}
              lg={8}
              md={12}
            >
              <div className="m-b-16px text-20px color-[var(--ant-color-text-heading)]">
                {t(`page.home.settings.${item.key}`)}
              </div>
              <div className="color-var(--ant-color-text) m-b-45px h-60px">
                {t(`page.home.settings.${item.key}Desc`)}
              </div>
              {
                !item.permissions || hasAuth(item.permissions) ? (
                  <Button
                    className="h-40px w-40px rounded-12px p-0 text-24px"
                    disabled={!item.link}
                    type="primary"
                    onClick={() => item.link && routerPush.routerPush(item.link)}
                  >
                    {item.icon}
                  </Button>
                ) : null
              }
            </Col>
          ))}
        </Row>
      </Card>
    </Spin>
  );
}
