import RoleSelect from '@/pages/security/modules/RoleSelect';
import { useLoading } from '@sa/hooks';
import { Button, Form, Input, Spin } from 'antd';

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
  const { defaultRequiredRule, formRules, patternRules } = useFormRules();
  const { endLoading, loading, startLoading } = useLoading();

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { roles, confirm_password, ...rest } = params;
    onSubmit({
      ...rest,
      roles: (roles || []).map((item) => item.name)
    }, startLoading, endLoading);
  };

  useEffect(() => {
    if (record && typeof record === 'object') {
      const { password, roles, ...rest } = record;
      form.setFieldsValue({
        ...rest,
        roles: roles.map((item) => ({ name: item }))
      });
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
          label={t('page.user.labels.name')}
          name="name"
          rules={[defaultRequiredRule]}
        >
          <Input className={itemClassNames} />
        </Form.Item>
        <Form.Item
          label={t('page.user.labels.email')}
          name="email"
          rules={formRules.email}
        >
          <Input className={itemClassNames} />
        </Form.Item>
        {
          record && (
            <>
              <Form.Item
                label={t('page.user.labels.password')}
                name="password"
                rules={[patternRules.pwd]}
              >
                <Input.Password className={itemClassNames}/>
              </Form.Item>
              <Form.Item
                label={t('common.confirmPassword')}
                name="confirm_password"
                rules={[
                  {
                    validator: async (rule, value) => {
                      const password = await form.getFieldValue('password');
                      if (password && password !== value) {
                        throw new Error(t('form.pwdConfirm.invalid'));
                      }
                    },
                  }
                ]}
              >
                <Input.Password className={itemClassNames} />
              </Form.Item>
            </>
          )
        }
        <Form.Item
          label={t('page.user.labels.roles')}
          name='roles'
          rules={[defaultRequiredRule]}
        >
          <RoleSelect
            className={itemClassNames}
            mode="multiple"
            width="100%"
            allowClear={true}
            placeholder={t('page.auth.labels.roles')}
          />
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
