import { Button, Form, Input, Select, Switch, Checkbox, Space } from 'antd';
import { useLoading } from '@sa/hooks';
import AIAssistantSelect from '@/pages/ai-assistant/modules/AIAssistantSelect';
import DataSourceSelect from '@/pages/data-source/modules/DataSourceSelect';

export const WebhookForm = (props: {
  actionText: string;
  loading?: boolean;
  record?: any;
  onSubmit: (params: any, before?: () => void, after?: () => void) => Promise<void>;
}) => {
  const { actionText, loading, record, onSubmit } = props;
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const { endLoading, startLoading } = useLoading();

  useEffect(() => {
    if (record) {
      form.setFieldsValue(record);
    } else {
      form.setFieldsValue({
        content_type: 'application/json',
        ssl_verify: true,
        triggers: {
          ai_assistant: { enabled: true, reply_completed: true },
          datasource: { enabled: false, file_parse_completed: false, sync_completed: false },
          model_provider: { enabled: false }
        }
      });
    }
  }, [record]);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    await onSubmit(
      {
        ...params
      },
      startLoading,
      endLoading
    );
  };

  return (
    <div className="px-30px">
      <Form
        form={form}
        layout="horizontal"
        labelCol={{ span: 4 }}
        wrapperCol={{ span: 14 }}
        colon={false}
      >
        <Form.Item
          label={t('page.webhook.labels.name')}
          name="name"
          rules={[{ required: true }]}
        >
          <Input placeholder={t('page.webhook.placeholders.name', '请输入')} />
        </Form.Item>

        <Form.Item
          label={t('page.webhook.labels.payload_url')}
          name="payload_url"
          rules={[{ required: true }]}
        >
          <Input placeholder={t('page.webhook.placeholders.payload_url', '请输入')} />
        </Form.Item>

        <Form.Item
          label={t('page.webhook.labels.content_type')}
          name="content_type"
        >
          <Select
            options={[
              { label: 'application/json', value: 'application/json' },
              { label: 'application/x-www-form-urlencoded', value: 'application/x-www-form-urlencoded' }
            ]}
          />
        </Form.Item>

        <Form.Item
          label={t('page.webhook.labels.secret')}
          name="secret"
        >
          <Input placeholder={t('page.webhook.placeholders.secret', '请输入')} />
        </Form.Item>

        <Form.Item
          label={t('page.webhook.labels.ssl_verify')}
          name="ssl_verify"
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>

        <Form.Item label={t('page.webhook.labels.triggers')}>
          <Space direction="vertical" className="w-full">
            <Form.Item
              label={t('page.webhook.labels.ai_assistant')}
              labelCol={{ span: 5 }}
              wrapperCol={{ span: 19 }}
              className="mb-0px"
            >
              <Space className="w-full">
                <Form.Item name={['triggers', 'ai_assistant', 'enabled']} valuePropName="checked" className="mb-0px">
                  <Switch />
                </Form.Item>
                <Form.Item
                  noStyle
                  shouldUpdate={(prev, next) => prev?.triggers?.ai_assistant?.enabled !== next?.triggers?.ai_assistant?.enabled}
                >
                  {({ getFieldValue }) => {
                    const enabled = getFieldValue(['triggers','ai_assistant','enabled']);
                    return enabled ? (
                      <Space className="w-full">
                        <Form.Item name={['triggers', 'ai_assistant', 'assistants']} className="mb-0px">
                          <AIAssistantSelect mode="multiple" width="400px" />
                        </Form.Item>
                        <Form.Item name={['triggers', 'ai_assistant', 'reply_completed']} valuePropName="checked" className="mb-0px">
                          <Checkbox>{t('page.webhook.labels.reply_completed')}</Checkbox>
                        </Form.Item>
                      </Space>
                    ) : null;
                  }}
                </Form.Item>
              </Space>
            </Form.Item>

            <Form.Item
              label={t('page.webhook.labels.datasource')}
              labelCol={{ span: 5 }}
              wrapperCol={{ span: 19 }}
              className="mb-0px"
            >
              <Space className="w-full">
                <Form.Item name={['triggers', 'datasource', 'enabled']} valuePropName="checked" className="mb-0px">
                  <Switch />
                </Form.Item>
                <Form.Item
                  noStyle
                  shouldUpdate={(prev, next) => prev?.triggers?.datasource?.enabled !== next?.triggers?.datasource?.enabled}
                >
                  {({ getFieldValue }) => {
                    const enabled = getFieldValue(['triggers','datasource','enabled']);
                    return enabled ? (
                      <Space className="w-full">
                        <Form.Item name={['triggers', 'datasource', 'datasource']} className="mb-0px">
                          <DataSourceSelect mode="multiple" width="400px" />
                        </Form.Item>
                        <Form.Item name={['triggers', 'datasource', 'file_parse_completed']} valuePropName="checked" className="mb-0px">
                          <Checkbox>{t('page.webhook.labels.file_parse_completed')}</Checkbox>
                        </Form.Item>
                        <Form.Item name={['triggers', 'datasource', 'sync_completed']} valuePropName="checked" className="mb-0px">
                          <Checkbox>{t('page.webhook.labels.sync_completed')}</Checkbox>
                        </Form.Item>
                      </Space>
                    ) : null;
                  }}
                </Form.Item>
              </Space>
            </Form.Item>

            <Form.Item
              label={t('page.webhook.labels.model_provider')}
              labelCol={{ span: 5 }}
              wrapperCol={{ span: 19 }}
              className="mb-0px"
            >
              <Form.Item name={['triggers', 'model_provider', 'enabled']} valuePropName="checked" className="mb-0px">
                <Switch />
              </Form.Item>
            </Form.Item>
          </Space>
        </Form.Item>

        <Form.Item wrapperCol={{ span: 14, offset: 4 }}>
          <Space>
            <Button
              onClick={() => {
                const id = record?.id;
                if (!id) {
                  window.$message?.warning(t('page.webhook.labels.test_need_save'));
                  return;
                }
                // 留给后台实现测试接口
                // @ts-ignore
                import('@/service/api/webhook').then(({ testWebhook }) => {
                  testWebhook(id).then(() => {
                    window.$message?.success(t('common.success'));
                  });
                });
              }}
            >
              {t('page.webhook.labels.test')}
            </Button>
            <Button type="primary" loading={loading} onClick={handleSubmit}>
              {actionText}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </div>
  );
};