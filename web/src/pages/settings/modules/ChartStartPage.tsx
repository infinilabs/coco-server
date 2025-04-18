import { Avatar, Button, Form, Input, Spin, Switch, Upload } from 'antd';
import '../index.scss';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading } from '@sa/hooks';
import "./ChartStartPage.scss"
import AIAssistantSelect from '@/pages/ai-assistant/modules/AIAssistantSelect';
import { PlusOutlined } from '@ant-design/icons';

const ChartStartPage = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();

  const { endLoading, loading, startLoading } = useLoading();
  const [lightFileList, setLightFileList] = useState([]);
  const [lightLogo, setLightLogo] = useState({ loading: false });
  const [darkFileList, setDarkFileList] = useState([]);
  const [darkLogo, setDarkLogo] = useState({ loading: false });

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  const handleSubmit = async () => {
    const params = await form.validateFields();
    startLoading();
    const result = await updateSettings({
      chat_start_page: {
        ...params,
        "display_assistants": params?.display_assistants?.map((item) => item.id),
        "logo": {
          "light": lightLogo.base64,
          "dark": darkLogo.base64
        },
      }
    });
    if (result?.data?.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
    }
    endLoading();
  };

  const renderIcon = (base64) => {
    if (base64) {
      return (
        <div className="chart-start-page-image css-var-r0 ant-btn">
          <Avatar shape="square" src={base64} />
        </div>
      );
    }
    return null;
  }

  useMount(() => {
    run();
  });

  useEffect(() => {
    if (data?.data?.chat_start_page) {
      const { logo, display_assistants, ...rest } = data?.data?.chat_start_page;
      if (logo?.light) {
        setLightLogo((state) => ({ ...state, base64: logo?.light }))
      }
      if (logo?.dark) {
        setDarkLogo((state) => ({ ...state, base64: logo?.dark }))
      }
      form.setFieldsValue({
        ...rest,
        display_assistants: display_assistants? display_assistants.map((item) => ({
          id: item
        })) : []
      });
    } else {
      form.setFieldsValue({ enabled: true  });
    }
  }, [JSON.stringify(data)]);

  const uploadProps = {
    name: "file",
    action: "",
    accept: "image/*,.svg",
  };

  return (
    <Spin spinning={dataLoading || loading}>
      <Form
        className="settings-form py-24px"
        colon={false}
        form={form}
        labelAlign="left"
      >
        <Form.Item
          label={t('page.chart_start_page.labels.start_page')}
          name="enabled"
          help={t('page.chart_start_page.labels.start_page_placeholder')}
          className="mb-88px"
        >
          <Switch size="small" />
        </Form.Item>
        <Form.Item
          label={t('page.chart_start_page.labels.logo')}
          name="logo"
        >
          <div className='settings-form-help mb-16px'>
            <div>{t('page.chart_start_page.labels.logo_placeholder')}</div>
            <div>{t('page.chart_start_page.labels.logo_size_placeholder')}</div>
          </div>
          <Form.Item className="sub-form-item mb-48px" layout="vertical" name={['logo', 'light']} label={t('page.chart_start_page.labels.logo_light')}>
            <div style={{ display: "flex", gap: 22 }}>
              {renderIcon(lightLogo.base64)}
              <Upload
                {...uploadProps}
                showUploadList={false}
                fileList={lightFileList}
                beforeUpload={(file) => {
                  setLightFileList([file]);
                  setLightLogo({ loading: true });
                  const reader = new FileReader();
                  reader.readAsDataURL(file);
                  reader.onload = () => {
                    const base64 = reader.result
                    setLightLogo({
                      loading: false,
                      base64,
                    });
                  };
                  return false
                }}
              >
                <Button loading={lightLogo.loading} icon={<SvgIcon className="text-12px" icon="mdi:upload" />}>{t('common.upload')}</Button>
              </Upload>
              <Button className="px-0" type="link" onClick={() => {
                setLightLogo({
                  loading: false,
                });
              }}>{t('common.reset')}</Button>
            </div>
          </Form.Item>
          <Form.Item className="sub-form-item mb-32px" layout="vertical" name={['logo', 'dark']} label={t('page.chart_start_page.labels.logo_dark')}>
          <div style={{ display: "flex", gap: 22 }}>
              {renderIcon(darkLogo.base64)}
              <Upload
                {...uploadProps}
                showUploadList={false}
                fileList={darkFileList}
                beforeUpload={(file) => {
                  setDarkFileList([file]);
                  setDarkLogo({ loading: true });
                  const reader = new FileReader();
                  reader.readAsDataURL(file);
                  reader.onload = () => {
                    const base64 = reader.result
                    setDarkLogo({
                      loading: false,
                      base64,
                    });
                  };
                  return false
                }}
              >
                <Button loading={darkLogo.loading} icon={<SvgIcon className="text-12px" icon="mdi:upload" />}>{t('common.upload')}</Button>
              </Upload>
              <Button className="px-0" type="link" onClick={() => {
                setDarkLogo({
                  loading: false,
                });
              }}>{t('common.reset')}</Button>
            </div>
          </Form.Item>
        </Form.Item>
        <Form.Item
          label={t('page.chart_start_page.labels.introduction')}
          name="introduction"
          help={t('page.chart_start_page.labels.introduction_placeholder')}
          className="mb-64px"
        >
          <Input.TextArea rows={3} maxLength={60}/>
        </Form.Item>
        <Form.Item
          label={t('page.chart_start_page.labels.assistant')}
        >
          <Form.List name="display_assistants">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field, index) => {
                  return (
                    <Form.Item key={index} className="m-0">
                      <div className="flex items-center gap-6px">
                        <Form.Item
                          {...field}
                          rules={[defaultRequiredRule]}
                          className="flex-1"
                        >
                          <AIAssistantSelect />
                        </Form.Item>
                        <Form.Item>
                          <span onClick={() => remove(field.name)}><SvgIcon className="text-16px cursor-pointer" icon="mdi:minus-circle-outline" /></span>
                        </Form.Item>
                      </div>
                    </Form.Item>
                  )
                })}
                <Form.Item>
                  <Button className="!w-80px" type="primary" disabled={fields.length >= 8} icon={<PlusOutlined />} onClick={() => add()}></Button>
                </Form.Item>
              </>
            )}
          </Form.List>
        </Form.Item>
        <Form.Item label=" ">
            <Button
              type="primary"
              onClick={() => handleSubmit()}
            >
              {t('common.update')}
            </Button>
          </Form.Item>
      </Form>
    </Spin>
  );
});

export default ChartStartPage;