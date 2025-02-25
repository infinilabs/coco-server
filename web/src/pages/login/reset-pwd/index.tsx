import { Button, Form, Input, Space } from 'antd';

interface FormModel {
  code: string;
  confirmPassword: string;
  password: string;
  phone: string;
}

export const Component = () => {
  const { t } = useTranslation();
  const { toggleLoginModule } = useRouterPush();
  const [form] = Form.useForm<FormModel>();
  const { createConfirmPwdRule, formRules } = useFormRules();

  async function handleSubmit() {
    const params = await form.validateFields();
    console.log(params);

    // request to reset password
    window.$message?.success(t('page.login.common.validateSuccess'));
  }

  useKeyPress('enter', () => {
    handleSubmit();
  });

  return (
    <>
      <h3 className="text-18px text-primary font-medium">{t('page.login.register.title')}</h3>
      <Form
        className="pt-24px"
        form={form}
      >
        <Form.Item
          name="phone"
          rules={formRules.phone}
        >
          <Input placeholder={t('page.login.common.phonePlaceholder')} />
        </Form.Item>
        <Form.Item
          name="code"
          rules={formRules.code}
        >
          <Input placeholder={t('page.login.common.codePlaceholder')} />
        </Form.Item>
        <Form.Item
          name="password"
          rules={formRules.pwd}
        >
          <Input.Password
            autoComplete="password"
            placeholder={t('page.login.common.passwordPlaceholder')}
          />
        </Form.Item>
        <Form.Item
          name="confirmPassword"
          rules={createConfirmPwdRule(form)}
        >
          <Input.Password
            autoComplete="confirm-password"
            placeholder={t('page.login.common.confirmPasswordPlaceholder')}
          />
        </Form.Item>
        <Space
          className="w-full"
          direction="vertical"
          size={18}
        >
          <Button
            block
            shape="round"
            size="large"
            type="primary"
            onClick={handleSubmit}
          >
            {t('common.confirm')}
          </Button>

          <Button
            block
            shape="round"
            size="large"
            onClick={() => toggleLoginModule('pwd-login')}
          >
            {t('page.login.common.back')}
          </Button>
        </Space>
      </Form>
    </>
  );
};

Component.displayName = 'ResetPwd';
