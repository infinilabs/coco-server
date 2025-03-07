import { Button, Form, Input, InputNumber, Spin, Switch, Tabs} from "antd";
import "../index.scss"
import { fetchSettings, updateSettings } from "@/service/api/server";


export const GoogleDriveSettings = memo(() => {
    const [form] = Form.useForm();
    const { t } = useTranslation();

    const { defaultRequiredRule, formRules } = useFormRules();
    const { data, run, loading: dataLoading } = useRequest(fetchSettings, {
      manual: true
  });
  useMount(() => {
      run();
  });

  useEffect(() => {
    if (data?.data?.connector?.google_drive) {
      form.setFieldsValue(data.data.connector.google_drive || { });
    }
  }, [JSON.stringify(data)]);

    const [loading, setLoading] = useState(false);

    const handleSubmit = async () => {
      setLoading(true);
        const params = await form.validateFields();
        const result = await updateSettings({
            connector: {
              google_drive: params,
            }
        });
        setLoading(false);
        if (result.data.acknowledged) {
          window.$message?.success(t('common.updateSuccess'));
        }
    }

    return (
        <Spin spinning={loading}>
            <Form 
                form={form}
                labelAlign="left"
                className="settings-form"
                colon={false}
            >
                <Form.Item
                    name="client_id"
                    label="Client ID"
                    rules={[defaultRequiredRule]}
                >
                  <Input />
                </Form.Item>
                <Form.Item
                    name="client_secret"
                    label="Client Secret"
                    rules={[defaultRequiredRule]}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="redirect_url"
                    label="Redirect URI"
                    rules={formRules.endpoint}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="auth_url"
                    label="Auth URI"
                    rules={formRules.endpoint}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="token_url"
                    label="Token URI"
                    rules={formRules.endpoint}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    label=" "
                >
                    <Button type="primary" onClick={() => handleSubmit()}>{t('common.update')}</Button>
                </Form.Item>
            </Form>
        </Spin>
    )
});

const ConnectorSettings = memo(() => {
  const items = [
    {
      key: 'google_drive',
      label: 'Gogole Drive',
      children: <GoogleDriveSettings />,
    },
  ];

  return  <Tabs items={items}/>
});

export default ConnectorSettings;