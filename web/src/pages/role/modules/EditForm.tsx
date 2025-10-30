import { useLoading } from '@sa/hooks';
import { Button, Form, Input, Spin } from 'antd';

import Permissions from './Permissions';

interface EditFormProps {
  actionText: string;
  loading?: boolean;
  onSubmit: (params: any, before?: () => void, after?: () => void) => Promise<void>;
  record?: any;
}

export const EditForm = memo((props: EditFormProps) => {
  const { actionText, onSubmit, record } = props;
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { defaultRequiredRule, patternRules } = useFormRules();
  const { endLoading, loading, startLoading } = useLoading();

  const handleSubmit = async () => {
    const params = await form.validateFields();
    onSubmit(params, startLoading, endLoading);
  };

  // eslint-disable-next-line @typescript-eslint/no-shadow
  const initValue = (record: any) => {
    const initRecord = {
      name: record.name || '',
      description: record.description || '',
      permission: {
        feature: record.grants?.permissions || []
      }
    };
    form.setFieldsValue(initRecord);
  };

  useEffect(() => {
    if (record && typeof record === 'object') {
      initValue(record);
    }
  }, [record]);

  const itemClassNames = '!w-496px';

  return (
    <Spin spinning={props.loading || loading || false}>
      <Form
        colon={false}
        form={form}
        labelAlign='left'
        layout='horizontal'
        labelCol={{
          style: { maxWidth: 200, minWidth: 200, textAlign: 'left' }
        }}
      >
        <Form.Item
          label={t('page.role.labels.name')}
          name='name'
          rules={[
            defaultRequiredRule,
            patternRules.noSpecial
          ]}
        >
          <Input className={itemClassNames} />
        </Form.Item>
        <Form.Item
          label={t('page.role.labels.description')}
          name='description'
        >
          <Input.TextArea
            className={itemClassNames}
            rows={4}
          />
        </Form.Item>
        <Form.Item
          label={t('page.role.labels.permission')}
          name='permission'
        >
          <Permissions />
        </Form.Item>
        <Form.Item label=' '>
          <Button
            type='primary'
            onClick={() => handleSubmit()}
          >
            {actionText}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});
