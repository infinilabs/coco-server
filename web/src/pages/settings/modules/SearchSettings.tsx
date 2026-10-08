import { Divider, Form, Select, Spin, Switch } from 'antd';
import '../index.scss';
import { fetchIntegration, updateIntegration } from '@/service/api/integration';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading, useRequest } from '@sa/hooks';
import { getApplicationSetting, setApplicationSetting, updateRootRouteIfSearch } from '@/store/slice/server';
import { initAuthRoute, initConstantRoute, selectFilterPaths, setFilterPaths } from '@/store/slice/route';
import { resetAuth } from '@/store/slice/auth';
import BuiltinSearchForm from './BuiltinSearchForm';

// the built-in AI search integration (backend: core.DefaultSearchIntegrationID).
// the app's own search page always reuses this fullscreen widget — it is hidden
// from the integration list, and this settings tab is its only editing UI
const BUILTIN_SEARCH_INTEGRATION_ID = 'full-screen-widget-default';

const SearchSettings = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    update: hasAuth('coco#system/update'),
  }

  const { endLoading, loading, startLoading } = useLoading();

  const dispatch = useAppDispatch();
  const applicationSetting = useAppSelector(getApplicationSetting);
  const filterPaths = useAppSelector(selectFilterPaths);

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  // the built-in fullscreen widget whose parameters this page edits through
  // the same form components the integration editor uses
  const {
    data: builtin,
    loading: builtinLoading,
    run: runBuiltin
  } = useRequest(fetchIntegration, {
    manual: true
  });

  useEffect(() => {
    run();
  }, []);

  useMount(() => {
    run();
    runBuiltin(BUILTIN_SEARCH_INTEGRATION_ID);
  });

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { enabled, search_type } = params;
    startLoading();
    const search_settings = {
      enabled,
      search_type: search_type || 'keyword'
    }
    const result = await updateSettings({
       search_settings
    });
    if (result?.data?.acknowledged) {
      const newApplicationSetting = {
        ...applicationSetting,
        search_settings
      }
      await dispatch(setApplicationSetting(newApplicationSetting));
      await dispatch(updateRootRouteIfSearch(newApplicationSetting));
      if (search_settings.enabled) {
        await dispatch(setFilterPaths(filterPaths.filter(path => path !== '/search')));
      }
      await dispatch(initConstantRoute());
      await dispatch(resetAuth());
      await dispatch(initAuthRoute());
      window.$message?.success(t('common.updateSuccess'));
    }
    endLoading();
  };

  // same submit contract as the integration edit page — the payload lands on
  // the built-in integration, so the app search and an embedded widget behave
  // identically
  const handleUpdateBuiltin = async (params: any, before?: () => void, after?: () => void) => {
    if (before) before();
    const res = await updateIntegration({ id: BUILTIN_SEARCH_INTEGRATION_ID, ...params });
    if (res?.data?.result === 'updated') {
      window.$message?.success(t('common.updateSuccess'));
    }
    if (after) after();
  };

  useEffect(() => {
    if (data?.search_settings) {
      form.setFieldsValue({
        ...data?.search_settings,
        search_type: data?.search_settings?.search_type || 'keyword'
      });
    } else {
      form.setFieldsValue({
        enabled: false,
        search_type: 'keyword'
      });
    }
  }, [JSON.stringify(data)]);

  return (
    <ListContainer>
      <Spin spinning={dataLoading || loading}>
        <Form
          className="settings-form py-24px"
          colon={false}
          form={form}
          labelAlign="left"
        >
          <Form.Item
              label={t('page.settings.search_settings.labels.enabled')}
              name={['enabled']}
            >
            <Switch size="small" />
          </Form.Item>
          <Form.Item
            label={t('page.settings.search_settings.labels.search_type')}
            name={['search_type']}
            tooltip={t('page.settings.search_settings.labels.search_type_desc')}
          >
            <Select
              size="small"
              className="w-120px"
              options={['keyword', 'semantic', 'hybrid', 'hybrid_rrf'].map((value) => ({
                value,
                label: t(`page.settings.search_settings.options.${value}`)
              }))}
            />
          </Form.Item>

          <Divider />

          <div className="color-[var(--ant-color-text)] font-medium mb-8px">
            {t('page.settings.search_settings.labels.builtin_widget')}
          </div>
          <div className="mb-24px settings-form-help">
            {t('page.settings.search_settings.labels.builtin_widget_hint')}
          </div>
          {
            builtin ? (
              <BuiltinSearchForm
                loading={builtinLoading}
                record={builtin?._source}
                onSubmit={handleUpdateBuiltin}
              />
            ) : (
              <Spin />
            )
          }
        </Form>
      </Spin>
    </ListContainer>
  );
});

export default SearchSettings;
