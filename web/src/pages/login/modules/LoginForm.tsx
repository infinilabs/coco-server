import { Button, Form, Input } from 'antd';

import { useLogin } from '@/hooks/common/login';

type AccountKey = 'admin' | 'super' | 'user';
interface Account {
  key: AccountKey;
  label: string;
  password: string;
  userName: string;
}

type LoginParams = Pick<Account, 'password' | 'userName'>;

const LoginForm = memo(({ onProvider } : { onProvider?: () => void }) => {
  const [form] = Form.useForm<LoginParams>();
  const { loading, toLogin } = useLogin();
  const { t } = useTranslation();
  const { formRules } = useFormRules();

  async function handleSubmit() {
    const params = await form.validateFields();
    if (onProvider) {
      toLogin(params, false);
      onProvider()
    } else {
      toLogin(params);
    }
  }

  useKeyPress('enter', () => {
    handleSubmit();
  });

  return (
    <>
      <div className="text-32px color-#333 m-b-16px">
        {t('page.login.title')}
      </div>
      <div className="text-16px color-#999 m-b-64px">
        {t('page.login.desc')}
      </div>
      <Form
        form={form}
        layout="vertical"
      >
        <Form.Item
          name="password"
          label={t('page.login.password')}
          className="m-b-32px"
          rules={formRules.pwd}
        >
            <Input.Password className="h-40px"/>
        </Form.Item>
        <div className="text-right">
            <Button type="primary" loading={loading} size="large" className="w-56px h-56px text-24px" onClick={handleSubmit}>
                <SvgIcon icon="mdi:arrow-right" />
            </Button>
        </div>
      </Form>
    </>
  );
})

export default LoginForm
