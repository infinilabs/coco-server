import { Button, Form, Spin, Switch } from 'antd';
import '../index.scss';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading, useRequest } from '@sa/hooks';
import IntegrationSelect from '@/pages/integration/modules/IntegrationSelect';

const SearchSettings = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const { endLoading, loading, startLoading } = useLoading();

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  useEffect(() => {
    run({
      from: 0,
      size: 10000
    });
  }, []);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { enabled, integration } = params;
    startLoading();
    const result = await updateSettings({
       search_settings: {
        enabled,
        integration: integration?.id
       } 
    });
    if (result?.data?.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
    }
    endLoading();
  };

  useMount(() => {
    run();
  });

  useEffect(() => {
    if (data?.search_settings) {
      form.setFieldsValue({
        ...data?.search_settings
      });
    } else {
      form.setFieldsValue({ 
        enabled: false
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
          <Form.Item label={t('page.settings.search_settings.labels.integration')} name={['integration']}>
            <IntegrationSelect filter={{ enabled: [true], type: ['fullscreen', 'page', 'modal']}}/>
          </Form.Item>
          <Form.Item label=" " >
            <Button
              type="primary"
              onClick={() => handleSubmit()}
            >
              {t('common.update')}
            </Button>
          </Form.Item>
        </Form>
      </Spin>
    </ListContainer>
  );
});

export default SearchSettings;