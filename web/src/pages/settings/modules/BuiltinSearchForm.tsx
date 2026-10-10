import { Avatar, Button, Form, Input, InputNumber, Select, Spin, Switch, Upload } from 'antd';
import { useLoading, useRequest } from '@sa/hooks';

import AIAssistantSelect from '@/pages/ai-assistant/modules/AIAssistantSelect';
import { fetchDataSourceList } from '@/service/api';

/**
 * Dedicated form for the built-in AI search integration (the fullscreen widget
 * the app's own search page reuses). A trimmed copy of the integration
 * editor's FullscreenForm: only the fields that make sense for the in-app
 * search — datasource / placeholder / welcome, banner logos, AI Overview and
 * the deep-think/research assistants. Everything else on the integration doc
 * (name, type, cors, appearance, guest…) is left untouched by the save below.
 */

interface Props {
  loading?: boolean;
  record?: Record<string, any>;
  onSubmit: (params: any, before?: () => void, after?: () => void) => void;
}

const BuiltinSearchForm = memo(({ loading, onSubmit, record }: Props) => {
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();
  const { endLoading, loading: submitLoading, startLoading } = useLoading();

  const { hasAuth } = useAuth();
  const permissions = {
    fetchDataSources: hasAuth('coco#datasource/search')
  };

  const [searchLogos, setSearchLogos] = useState<Record<string, any>>({
    darkList: [],
    darkLoading: false,
    dark: undefined,
    lightList: [],
    lightLoading: false,
    light: undefined
  });

  const [aiOverviewLogo, setAIOverviewLogo] = useState<Record<string, any>>({
    lightList: [],
    lightLoading: false,
    light: undefined
  });

  const [searchBackground, setSearchBackground] = useState<Record<string, any>>({
    darkList: [],
    darkLoading: false,
    dark: undefined,
    lightList: [],
    lightLoading: false,
    light: undefined
  });

  const [bannerHeight, setBannerHeight] = useState<number | undefined>(64);

  const [welcomeFontSize, setWelcomeFontSize] = useState<number | undefined>(30);
  const [welcomeGradient, setWelcomeGradient] = useState(true);

  const [aiOverviewEnabled, setAIOverviewEnabled] = useState(false);

  const {
    data: result,
    loading: dataSourceLoading,
    run
  } = useRequest(fetchDataSourceList, {
    manual: true
  });

  const dataSource = useMemo(() => {
    // the raw ES response isn't reflected in the Datasource type — same
    // shape the integration editor consumes
    return ((result as any)?.hits?.hits?.map((item: any) => ({ ...item._source })) || []);
  }, [JSON.stringify(result)]);

  useEffect(() => {
    if (permissions.fetchDataSources) {
      run({
        from: 0,
        size: 10000
      });
    }
  }, [permissions.fetchDataSources]);

  useEffect(() => {
    if (!record) return;
    setSearchLogos(state => ({ ...state, ...(record.payload?.logo || {}) }));
    setAIOverviewLogo(state => ({ ...state, ...(record.payload?.ai_overview?.logo || {}) }));
    setSearchBackground(state => ({ ...state, ...(record.payload?.background || {}) }));
    setBannerHeight(record.payload?.banner_height || 64);
    setWelcomeFontSize(record.payload?.welcome_font_size || 30);
    setWelcomeGradient(record.payload?.welcome_gradient !== false);
    setAIOverviewEnabled(record.payload?.ai_overview?.enabled ?? false);
    form.setFieldsValue({
      datasource: record.enabled_module?.search?.datasource?.includes('*')
        ? ['*']
        : record.enabled_module?.search?.datasource,
      placeholder: record.enabled_module?.search?.placeholder,
      welcome: record.payload?.welcome,
      ai_overview: record.payload?.ai_overview
        ? {
          ...record.payload.ai_overview,
          assistant: record.payload.ai_overview.assistant ? { id: record.payload.ai_overview.assistant } : undefined
        }
        : undefined,
      deep_think_assistant: record.deep_think_assistant ? { id: record.deep_think_assistant } : undefined,
      deep_research_assistant: record.deep_research_assistant ? { id: record.deep_research_assistant } : undefined
    });
  }, [record]);

  const uploadProps = {
    name: 'file',
    action: '',
    accept: 'image/*,.svg'
  };

  const renderIcon = (base64?: string) => {
    if (base64) {
      return (
        <div className="chart-start-page-image css-var-r0 ant-btn">
          <Avatar shape="square" src={base64} />
        </div>
      );
    }
    return null;
  };

  const renderLogoUpload = (
    fieldName: string | string[],
    label: string,
    preview: string | undefined,
    logoState: Record<string, any>,
    setLogoState: React.Dispatch<React.SetStateAction<any>>,
    valueKey: string,
    loadingKey: string,
    listKey: string
  ) => (
    <>
      <div className="mb-8px">{label}</div>
      <Form.Item className="mb-8px" name={fieldName}>
        <div style={{ display: 'flex', gap: 22 }}>
          {renderIcon(preview)}
          <Upload
            {...uploadProps}
            beforeUpload={(file) => {
              setLogoState((state: any) => ({
                ...state,
                [listKey]: [file],
                [loadingKey]: true
              }));
              const reader = new FileReader();
              reader.readAsDataURL(file);
              reader.onload = () => {
                setLogoState((state: any) => ({
                  ...state,
                  [loadingKey]: false,
                  [valueKey]: reader.result
                }));
              };
              return false;
            }}
            fileList={logoState[listKey]}
            showUploadList={false}
          >
            <Button icon={<SvgIcon className="text-12px" icon="mdi:upload" />} loading={logoState[loadingKey]}>
              {t('common.upload')}
            </Button>
          </Upload>
          <Button
            className="px-0"
            type="link"
            onClick={() => {
              setLogoState((state: any) => ({
                ...state,
                [loadingKey]: false,
                [valueKey]: ''
              }));
            }}
          >
            {t('common.reset')}
          </Button>
        </div>
      </Form.Item>
    </>
  );

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { ai_overview, datasource, deep_research_assistant, deep_think_assistant, placeholder, welcome } = params;
    startLoading();
    onSubmit(
      {
        // only the keys this form edits — the backend merges the delta, so
        // name/type/cors/appearance/guest/description stay untouched
        deep_research_assistant: deep_research_assistant?.id ?? record?.deep_research_assistant,
        deep_think_assistant: deep_think_assistant?.id ?? record?.deep_think_assistant,
        enabled_module: {
          search: {
            ...(record?.enabled_module?.search || {}),
            datasource: datasource?.includes('*') ? ['*'] : datasource,
            placeholder
          }
        },
        payload: {
          ...(record?.payload || {}),
          background: {
            dark: searchBackground?.dark || '',
            light: searchBackground?.light || ''
          },
          banner_height: bannerHeight || 64,
          welcome_font_size: welcomeFontSize || 30,
          welcome_gradient: welcomeGradient,
          logo: {
            dark: searchLogos?.dark || '',
            light: searchLogos?.light || ''
          },
          welcome,
          ai_overview: {
            ...(record?.payload?.ai_overview || {}),
            ...ai_overview,
            assistant: ai_overview?.assistant?.id || '',
            logo: {
              light: aiOverviewLogo?.light || ''
            }
          }
        }
      },
      undefined,
      endLoading
    );
  };

  const itemClassNames = '!w-496px';

  return (
    <Spin spinning={loading || submitLoading || false}>
      <Form
        colon={false}
        form={form}
        labelAlign="left"
        labelCol={{
          style: { maxWidth: 200, minWidth: 200, textAlign: 'left' }
        }}
        wrapperCol={{
          style: { maxWidth: 528, minWidth: 528, textAlign: 'left' }
        }}
      >
        <Form.Item label={t('page.integration.form.labels.search_settings')}>
          <div className="mb-8px">
            <span className="mr-4px text-[var(--ant-color-error)]">*</span>
            {t('page.integration.form.labels.datasource')}
          </div>
          <Form.Item
            className="mb-8px"
            name="datasource"
            rules={[defaultRequiredRule]}
          >
            <Select
              allowClear
              className={itemClassNames}
              loading={dataSourceLoading}
              mode="multiple"
              options={[{ label: '*', value: '*' }].concat(
                dataSource.map((item: any) => ({
                  label: item.name,
                  value: item.id
                }))
              )}
            />
          </Form.Item>
          <div className="mb-8px">{t('page.integration.form.labels.module_search_placeholder')}</div>
          <Form.Item className="mb-8px" name="placeholder">
            <Input className={itemClassNames} />
          </Form.Item>
          <div className="mb-8px">{t('page.integration.form.labels.module_search_welcome')}</div>
          <Form.Item className="mb-8px" name="welcome">
            <Input.TextArea className={itemClassNames} rows={3} />
          </Form.Item>
          <div className="mb-8px">{t('page.settings.search_settings.labels.welcome_font_size')}</div>
          <Form.Item className="mb-8px">
            <InputNumber
              className={itemClassNames}
              max={100}
              min={12}
              onChange={v => setWelcomeFontSize(v || undefined)}
              placeholder="30"
              value={welcomeFontSize}
            />
          </Form.Item>
          <div className="flex items-center gap-8px">
            <Switch checked={welcomeGradient} onChange={setWelcomeGradient} size="small" />
            <span className="text-12px color-[var(--ant-color-text-description)]">
              {t('page.settings.search_settings.labels.welcome_gradient')}
            </span>
          </div>
        </Form.Item>

        <Form.Item label=" ">
          {renderLogoUpload(
            ['logo', 'light'],
            t('page.integration.form.labels.logo'),
            searchLogos.light,
            searchLogos,
            setSearchLogos,
            'light',
            'lightLoading',
            'lightList'
          )}
          {renderLogoUpload(
            ['logo', 'dark'],
            t('page.integration.form.labels.logo_dark'),
            searchLogos.dark,
            searchLogos,
            setSearchLogos,
            'dark',
            'darkLoading',
            'darkList'
          )}
          <div className="mb-8px mt-8px">
            {t('page.settings.search_settings.labels.banner_height')}
          </div>
          <Form.Item className="mb-8px">
            <InputNumber
              className={itemClassNames}
              max={300}
              min={32}
              onChange={v => setBannerHeight(v || undefined)}
              placeholder="64"
              value={bannerHeight}
            />
          </Form.Item>
          <div className="text-12px color-[var(--ant-color-text-description)]">
            {t('page.settings.search_settings.labels.banner_height_hint')}
          </div>
        </Form.Item>

        <Form.Item label={t('page.settings.search_settings.labels.search_background')}>
          {renderLogoUpload(
            ['background', 'light'],
            t('page.settings.search_settings.labels.search_background_light'),
            searchBackground.light,
            searchBackground,
            setSearchBackground,
            'light',
            'lightLoading',
            'lightList'
          )}
          {renderLogoUpload(
            ['background', 'dark'],
            t('page.settings.search_settings.labels.search_background_dark'),
            searchBackground.dark,
            searchBackground,
            setSearchBackground,
            'dark',
            'darkLoading',
            'darkList'
          )}
          <div className="text-12px color-[var(--ant-color-text-description)]">
            {t('page.settings.search_settings.labels.search_background_hint')}
          </div>
        </Form.Item>

        <Form.Item label={t('page.integration.form.labels.module_ai_overview')}>
          <Form.Item
            className="mb-0px"
            name={['ai_overview', 'enabled']}
            valuePropName="checked"
          >
            <Switch onChange={setAIOverviewEnabled} size="small" />
          </Form.Item>
          {aiOverviewEnabled && (
            <>
              <div className="mb-8px pt-8px">{t('page.integration.form.labels.module_ai_overview_title')}</div>
              <Form.Item className="mb-8px" name={['ai_overview', 'title']}>
                <Input className={itemClassNames} />
              </Form.Item>
              <div className="mb-8px">{t('page.integration.form.labels.logo')}</div>
              <Form.Item className="mb-8px" name={['ai_overview', 'logo']}>
                <div style={{ display: 'flex', gap: 22 }}>
                  {renderIcon(aiOverviewLogo?.light)}
                  <Upload
                    {...uploadProps}
                    beforeUpload={(file) => {
                      setAIOverviewLogo(state => ({
                        ...state,
                        lightList: [file],
                        lightLoading: true
                      }));
                      const reader = new FileReader();
                      reader.readAsDataURL(file);
                      reader.onload = () => {
                        setAIOverviewLogo(state => ({
                          ...state,
                          lightLoading: false,
                          light: reader.result
                        }));
                      };
                      return false;
                    }}
                    fileList={aiOverviewLogo.lightList}
                    showUploadList={false}
                  >
                    <Button icon={<SvgIcon className="text-12px" icon="mdi:upload" />} loading={aiOverviewLogo?.lightLoading}>
                      {t('common.upload')}
                    </Button>
                  </Upload>
                  <Button
                    className="px-0"
                    type="link"
                    onClick={() => {
                      setAIOverviewLogo(state => ({
                        ...state,
                        lightLoading: false,
                        light: ''
                      }));
                    }}
                  >
                    {t('common.reset')}
                  </Button>
                </div>
              </Form.Item>
              <div className="mb-8px">{t('page.integration.form.labels.module_ai_overview_height')}</div>
              <Form.Item className="mb-8px" name={['ai_overview', 'height']}>
                <InputNumber className={itemClassNames} min={0} step={1} />
              </Form.Item>
              <div className="mb-8px">
                <span className="mr-4px text-[var(--ant-color-error)]">*</span>
                {t('page.integration.form.labels.module_chat_ai_assistant')}
              </div>
              <Form.Item
                className="mb-8px"
                name={['ai_overview', 'assistant']}
                rules={[
                  {
                    validator: (_rule: any, value: any) => {
                      if (value?.id) return Promise.resolve();
                      return Promise.reject((defaultRequiredRule as any).message);
                    }
                  }
                ]}
              >
                <AIAssistantSelect allowClear className={itemClassNames} />
              </Form.Item>
              <div className="mb-8px">{t('page.integration.form.labels.module_ai_overview_output')}</div>
              <Form.Item className="mb-0px" name={['ai_overview', 'output']}>
                <Select className={itemClassNames}>
                  <Select.Option value="markdown">Markdown</Select.Option>
                  <Select.Option value="text">Text</Select.Option>
                </Select>
              </Form.Item>
            </>
          )}
        </Form.Item>

        <Form.Item label={t('page.integration.form.labels.conversation_settings')}>
          <div className="mb-8px">{t('page.integration.form.labels.deep_think_assistant')}</div>
          <Form.Item className="mb-8px" name="deep_think_assistant">
            <AIAssistantSelect allowClear className={itemClassNames} filter={{ type: ['deep_think'] }} />
          </Form.Item>
          <div className="mb-8px">{t('page.integration.form.labels.deep_research_assistant')}</div>
          <Form.Item className="mb-0px" name="deep_research_assistant">
            <AIAssistantSelect allowClear className={itemClassNames} filter={{ type: ['deep_research'] }} />
          </Form.Item>
        </Form.Item>

        <Form.Item label=" ">
          <Button type="primary" onClick={handleSubmit}>
            {t('common.update')}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});

export default BuiltinSearchForm;
