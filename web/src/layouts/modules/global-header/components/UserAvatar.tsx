import { useRoute } from '@sa/simple-router';
import { Button, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { useSubmit } from 'react-router-dom';

import { selectToken, selectUserInfo } from '@/store/slice/auth';
import { Suspense } from 'react';

const PasswordModal = lazy(() => import('./PasswordModal'));

const UserAvatar = memo(() => {
  const token = useAppSelector(selectToken);
  const { t } = useTranslation();
  const userInfo = useAppSelector(selectUserInfo);
  const submit = useSubmit();
  const route = useRoute();
  const router = useRouterPush();

  const [passwordVisible, setPasswordVisible] = useState(false)

  function handleLogout() {
    let needRedirect = false;
    if (!route.meta?.constant) needRedirect = true;
    submit({ needRedirect, redirectFullPath: route.fullPath }, { action: '/account/logout', method: 'post' });
  }

  function logout() {
    window?.$modal?.confirm({
      cancelText: t('common.cancel'),
      content: t('common.logoutConfirm'),
      okText: t('common.confirm'),
      onOk: () => handleLogout(),
      title: t('common.tip')
    });
  }

  function onClick({ key }: { key: string }) {
    if (key === 'logout') {
      logout();
    } else if (key === 'password') {
      setPasswordVisible(true)
      // router.routerPushByKey('user-center');
    } else {
      // router.routerPushByKey('user-center');
    }
  }
  function loginOrRegister() {
    router.routerPushByKey('login');
  }

  const items: MenuProps['items'] = [
    {
      key: 'password',
      label: (
        <div className="flex-center gap-8px">
          <SvgIcon
            className="text-icon"
            icon="mdi:password"
          />
          {t('common.password')}
        </div>
      )
    },
    {
      type: 'divider'
    },
    {
      key: 'logout',
      label: (
        <div className="flex-center gap-8px">
          <SvgIcon
            className="text-icon"
            icon="ph:sign-out"
          />
          {t('common.logout')}
        </div>
      )
    }
  ];
  
  return (
    <>
      {
        token ? (
          <Dropdown
            menu={{ items, onClick }}
            placement="bottomRight"
            trigger={['click']}
          >
            <div>
              <ButtonIcon className="px-12px">
                <SvgIcon
                  className="text-icon-large"
                  icon="ph:user-circle"
                />
                <span className="text-16px font-medium">{userInfo.username}</span>
              </ButtonIcon>
            </div>
          </Dropdown>
        ) : (
          <Button onClick={loginOrRegister}>{t('page.login.common.loginOrRegister')}</Button>
        )
      }
      <Suspense>
        <PasswordModal open={passwordVisible} onClose={() => setPasswordVisible(false)} onSuccess={() => handleLogout()}/>
      </Suspense>
    </>
  )
});

export default UserAvatar;
