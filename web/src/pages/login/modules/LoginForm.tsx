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

const LoginForm = memo(() => {
  const [form] = Form.useForm<LoginParams>();
  const { loading, toLogin } = useLogin();

  async function handleSubmit() {
    const params = await form.validateFields();
    toLogin(params);
  }

  useKeyPress('enter', () => {
    handleSubmit();
  });

  return (
    <>
      <div className="text-28px color-#333 m-b-16px">
        Welcome
      </div>
      <div className="text-14px color-#999 m-b-64px">
        Enter your credentials to access your account.
      </div>
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          password: '123456',
        }}
      >
        <Form.Item
          name="password"
          label="Password"
          className="m-b-32px"
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
