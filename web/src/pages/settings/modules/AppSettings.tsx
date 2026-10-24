import { Button, Form, Input, Spin } from 'antd';
import '../index.scss';
import { fetchProviderInfo, fetchSettings, updateSettings } from '@/service/api/server';
import { getApplicationSetting } from '@/store/slice/server';
import { useLoading } from '@sa/hooks';
import ChatStartPage from './ChatStartPage';
import { useRequest } from '@sa/hooks';
import SaveBar from './SaveBar';

const AppSettings = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    update: hasAuth('coco#system/update'),
  }

  const applicationSetting = useAppSelector(getApplicationSetting);
  const managed = Boolean(applicationSetting?.security?.managed);

  const [serverName, setServerName] = useState<string>('');
  const [serverEndpoint, setServerEndpoint] = useState<string>('');

  const { endLoading, loading, startLoading } = useLoading();
  const [logo, setLogo] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined,
    darkLoading: false,
    darkList: [],
    dark: undefined,
  })
  const [dirty, setDirty] = useState(false);

  // logo 变更也计入脏状态（ChatStartPage 内部透传 setLogo）
  const changeLogo = (value: any) => {
    setLogo(value);
    setDirty(true);
  };

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  // Server connection (name + endpoint) lives in the provider info, which is a
  // different read than fetchSettings — an empty endpoint means "use the
  // current access address" so the edit box must keep the stored (possibly
  // empty) value, not the effective fallback.
  const {
    data: provider,
    loading: providerLoading,
    run: runProvider
  } = useRequest(fetchProviderInfo, {
    manual: true
  });

  useEffect(() => {
    if (provider) {
      setServerName(typeof provider?.name === 'string' ? provider.name : '');
      setServerEndpoint(typeof provider?.endpoint === 'string' ? provider.endpoint : '');
      setDirty(false);
    }
  }, [provider]);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { start_page } = params;
    startLoading();
    const payload: any = {
      app_settings: {
        chat: {
          start_page: {
            ...start_page,
            "display_assistants": start_page?.display_assistants?.map((item) => item.id),
            "logo": {
              "light": logo.light,
              "dark": logo.dark
            },
          }
        }
      }
    };
    if (provider) {
      payload.server = {
        ...(provider || {}),
        name: serverName.trim(),
        endpoint: serverEndpoint.trim()
      };
    }
    const result = await updateSettings(payload);
    if (result?.data?.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
      if (provider) runProvider();
      setDirty(false);
    }
    endLoading();
  };

  // 放弃更改：重新拉取服务端配置回填
  const handleDiscard = () => {
    setDirty(false);
    run();
    runProvider();
  };

  useMount(() => {
    run();
    runProvider();
  });

  useEffect(() => {
    if (data?.app_settings) {
      const { chat = {} } = data?.app_settings;
      const { start_page = {} } = chat || {};
      const { logo, display_assistants, ...rest } = start_page || {};
      setLogo((state) => ({ ...state, ...(logo || {}) }))
      form.setFieldsValue({
        start_page: {
          ...rest,
          display_assistants: display_assistants? display_assistants.map((item) => ({
            id: item
          })) : []
        }
      });
    } else {
      form.setFieldsValue({ 
        start_page: {
          enabled: false 
        }
      });
    }
  }, [JSON.stringify(data)]);

  return (
    <ListContainer>
      <Spin spinning={dataLoading || loading || providerLoading}>
        <Form
          className="settings-form py-24px"
          colon={false}
          form={form}
          labelAlign="left"
          onValuesChange={() => setDirty(true)}
        >
          <div className="color-[var(--ant-color-text)] font-medium mb-24px">
            {t('page.settings.app_settings.server_settings.title')}
          </div>
          <Form.Item
            colon={false}
            label={t('page.settings.app_settings.server_settings.name')}
            labelAlign="left"
          >
            <Input
              disabled={!permissions.update}
              placeholder={t('page.settings.app_settings.server_settings.name_placeholder')}
              value={serverName}
              onChange={e => { setServerName(e.target.value); setDirty(true); }}
            />
          </Form.Item>
          <Form.Item
            colon={false}
            extra={
              <div>
                {managed ? t('page.settings.app_settings.server_settings.endpoint_managed_hint') : t('page.settings.app_settings.server_settings.endpoint_hint')}{' '}
                <Button
                  className="px-0"
                  size="small"
                  type="link"
                  onClick={() => window.open('https://coco.rs/#install', '_blank')}
                >
                  {t('page.settings.app_settings.server_settings.download')}{' '}
                  <SvgIcon
                    className="text-14px"
                    icon="mdi:external-link"
                  />
                </Button>
              </div>
            }
            label={t('page.settings.app_settings.server_settings.endpoint')}
            labelAlign="left"
          >
            <Input
              disabled={!permissions.update || managed || window.__POWERED_BY_WUJIE__}
              placeholder={getEndpoint()}
              value={serverEndpoint}
              onChange={e => { setServerEndpoint(e.target.value); setDirty(true); }}
            />
          </Form.Item>
          <div className="color-[var(--ant-color-text)] font-medium mb-24px mt-8px">
            {t('page.settings.app_settings.chat_settings.title')}
          </div>
          <ChatStartPage startPageSettings={data?.app_settings?.chat?.start_page} logo={logo} setLogo={changeLogo}/>
        </Form>
      </Spin>
      {permissions.update && (
        <SaveBar dirty={dirty} loading={loading} onDiscard={handleDiscard} onSave={() => handleSubmit()} />
      )}
    </ListContainer>
  );
});

export default AppSettings;