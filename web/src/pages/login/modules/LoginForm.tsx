import { Button, Form, Input } from 'antd';

import { Shield } from 'lucide-react';
import { useLogin } from '@/hooks/common/login';
import normalizeUrl from 'normalize-url';
import { getApplicationSetting } from '@/store/slice/server';

interface LoginParams {
  email: string;
  password: string;
}

function getOAuthProviders(applicationSetting: any) {
  const oauth = applicationSetting?.security?.auth?.oauth;
  if (!oauth) return [];
  return Object.entries(oauth)
    .filter(([, v]: [string, any]) => v.url)
    .map(([id, v]: [string, any]) => ({
      id,
      name: v.name || `Sign in with ${v.type || id}`,
      icon: v.icon || undefined,
      url: v.url!,
      description: v.description,
      type: v.type,
    }));
}

const LoginForm = memo(({ onProvider }: { onProvider?: () => void }) => {
  const [form] = Form.useForm<LoginParams>();
  const { loading, toLogin } = useLogin();
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();

  const applicationSetting = useAppSelector(getApplicationSetting);
  const managed = Boolean(applicationSetting?.security?.managed);
  const oauthProviders = getOAuthProviders(applicationSetting);

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
      {managed && oauthProviders.length > 0 ? (
        <div className='mt-24px flex flex-col gap-12px'>
          {oauthProviders.map((provider) => (
            <Button
              key={provider.id}
              block
              className='h-40px flex items-center justify-between border-[#0087FF] rounded-4px bg-white px-16px text-14px text-[#0087FF] font-normal leading-20px font-[PingFangSC-regular]'
              style={{ width: '440px' }}
              type='default'
              onClick={() => {
                if (window.$wujie?.props?.onExternal) {
                  window.$wujie?.props?.onExternal(
                    normalizeUrl(`${getProxyEndpoint()}${provider.url}`)
                  );
                } else {
                  window.open(normalizeUrl(`${getEndpoint()}${provider.url}`), '_self')
                }
              }}
            >
              <div className='flex items-center gap-8px'>
                {provider.icon ? (
                  <img
                    alt=''
                    className='h-20px w-20px object-contain'
                    src={provider.icon}
                  />
                ) : (
                  <Shield size={20} />
                )}
                <span>{provider.name}</span>
              </div>
              <SvgIcon icon='mdi:arrow-right' />
            </Button>
          ))}
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
