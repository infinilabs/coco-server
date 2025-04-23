import { Button, Form, Spin } from 'antd';
import '../index.scss';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading } from '@sa/hooks';
import ChartStartPage from './ChartStartPage';

const AppSettings = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const { endLoading, loading, startLoading } = useLoading();
  const [logo, setLogo] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined,
    darkLoading: false,
    darkList: [],
    dark: undefined,
  })

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { start_page } = params;
    startLoading();
    const result = await updateSettings({
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
    if (data?.data?.app_settings) {
      const { chat = {} } = data?.data?.app_settings;
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
    <div className="h-full min-h-500px">
      <Spin spinning={dataLoading || loading}>
        <Form
          className="settings-form py-24px"
          colon={false}
          form={form}
          labelAlign="left"
        >
          <div className="color-#333 font-medium mb-24px">
            {t('page.settings.app_settings.chat_settings.title')}
          </div>
          <ChartStartPage startPageSettings={data?.data?.app_settings?.chat?.start_page} logo={logo} setLogo={setLogo}/>
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
    </div>
  );
});

export default AppSettings;