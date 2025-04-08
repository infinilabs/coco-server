import { Button, Form, Input } from 'antd';

import INFINICloud from '@/assets/svg-icon/INFINICloud.svg';
import { useLogin } from '@/hooks/common/login';

type AccountKey = 'admin' | 'super' | 'user';
interface Account {
  key: AccountKey;
  label: string;
  password: string;
  userName: string;
}

type LoginParams = Pick<Account, 'password' | 'userName'>;

const LoginForm = memo(({ onProvider }: { onProvider?: () => void }) => {
  const [form] = Form.useForm<LoginParams>();
  const { loading, toLogin } = useLogin();
  const { t } = useTranslation();
  const { formRules } = useFormRules();

  const managed = true;

  async function handleSubmit() {
    const params = await form.validateFields();
    if (onProvider) {
      toLogin(params, false);
      onProvider();
    } else {
      toLogin(params);
    }
  }

  useKeyPress('enter', () => {
    handleSubmit();
  });

  return (
    <>
      <div className="m-b-16px text-32px color-[var(--ant-color-text-heading)]">{t('page.login.title')}</div>
      <div className="m-b-64px text-16px color-[var(--ant-color-text)]">{t('page.login.desc')}</div>
      {managed ? (
        <div className="mt-24px">
          <Button
            block
            className="h-40px flex items-center justify-between border-[#0087FF] rounded-4px bg-white px-16px text-14px text-[#0087FF] font-normal leading-20px font-[PingFangSC-regular]"
            style={{ width: '440px' }}
            type="default"
            onClick={() => window.open('https://cloud.infini.com/login', '_blank')}
          >
            <div className="flex items-center gap-8px">
              <img
                alt="infini cloud"
                className="h-20px w-20px"
                src={INFINICloud}
              />
              <span>{t('page.login.cloud')}</span>
            </div>
            <SvgIcon icon="mdi:arrow-right" />
          </Button>
        </div>
      ) : (
        <Form
          form={form}
          layout="vertical"
        >
          <Form.Item
            className="m-b-32px"
            label={t('page.login.password')}
            name="password"
            rules={formRules.pwd}
          >
            <Input.Password className="h-40px" />
          </Form.Item>
          <div className="text-right">
            <Button
              className="h-56px w-56px text-24px"
              loading={loading}
              size="large"
              type="primary"
              onClick={handleSubmit}
            >
              <SvgIcon icon="mdi:arrow-right" />
            </Button>
          </div>
        </Form>
      )}
    </>
  );
});

export default LoginForm;
