import { Button, Form, Input } from 'antd';

import INFINICloud from '@/assets/svg-icon/INFINICloud.svg';
import { useLogin } from '@/hooks/common/login';
import { localStg } from '@/utils/storage';
import normalizeUrl from 'normalize-url';

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
  const { defaultRequiredRule } = useFormRules();

  const providerInfo = localStg.get('providerInfo');
  const managed = Boolean(providerInfo?.managed);

  async function handleSubmit() {
    const params = await form.validateFields();
    if (onProvider) {
      toLogin(params, false, onProvider);
    } else {
      toLogin(params);
    }
  }

  useKeyPress('enter', () => {
    handleSubmit();
  });

  return (
    <>
      <div className='m-b-16px text-32px color-[var(--ant-color-text-heading)]'>
        {t('page.login.title')}
      </div>
      <div className='m-b-64px text-16px color-[var(--ant-color-text)]'>
        {t('page.login.desc')}
      </div>
      {managed ? (
        <div className='mt-24px'>
          <Button
            block
            className='h-40px flex items-center justify-between border-[#0087FF] rounded-4px bg-white px-16px text-14px text-[#0087FF] font-normal leading-20px font-[PingFangSC-regular]'
            style={{ width: '440px' }}
            type='default'
            onClick={() => {
              const sso_url = providerInfo?.provider?.auth_provider?.sso?.url;

              if (window.$wujie?.props?.onExternal) {
                window.$wujie?.props?.onExternal(
                  normalizeUrl(`${getProxyEndpoint()}/${sso_url}`)
                );
              } else {
                window.open(normalizeUrl(`${getEndpoint()}/${sso_url}&redirect_url=${encodeURIComponent(window.location.href)}`), '_self')
              }
            }}
          >
            <div className='flex items-center gap-8px'>
              <img
                alt='infini cloud'
                className='h-20px w-20px'
                src={INFINICloud}
              />
              <span>{t('page.login.cloud')}</span>
            </div>
            <SvgIcon icon='mdi:arrow-right' />
          </Button>
        </div>
      ) : (
        <Form form={form} layout='vertical'>
          <Form.Item
            className='m-b-32px'
            label={t('page.login.email')}
            name='email'
            rules={[defaultRequiredRule]}
          >
            <Input className='h-40px' />
          </Form.Item>
          <Form.Item
            className='m-b-32px'
            label={t('page.login.password')}
            name='password'
            rules={[defaultRequiredRule]}
          >
            <Input.Password className='h-40px' />
          </Form.Item>
          <div className='text-right'>
            <Button
              className='h-56px w-56px text-24px'
              loading={loading}
              size='large'
              type='primary'
              onClick={handleSubmit}
            >
              <SvgIcon icon='mdi:arrow-right' />
            </Button>
          </div>
        </Form>
      )}
    </>
  );
});

export default LoginForm;
