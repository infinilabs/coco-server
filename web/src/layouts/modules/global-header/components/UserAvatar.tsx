import { useRoute } from '@sa/simple-router';
import { Button, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { Suspense } from 'react';
import { useSubmit } from 'react-router-dom';

import { logout } from '@/service/api';
import { store } from '@/store';
import { resetStore, selectUserInfo } from '@/store/slice/auth';
import { localStg } from '@/utils/storage';

const PasswordModal = lazy(() => import('./PasswordModal'));

const UserAvatar = memo((props) => {
  const { className, showHome = false, showName = true } = props;
  const { t } = useTranslation();
  const userInfo = useAppSelector(selectUserInfo);
  const submit = useSubmit();
  const route = useRoute();
  const router = useRouterPush();
  const nav = useNavigate();
  const location = useLocation();

  const providerInfo = localStg.get('providerInfo');
  const managed = Boolean(providerInfo?.managed);
  const [passwordVisible, setPasswordVisible] = useState(false);

  async function handleLogout() {
    let needRedirect = false;
    if (!route.meta?.constant) needRedirect = true;
    const result = await logout();
    if (result?.data?.status === 'ok') {
      store.dispatch(resetStore());
      router.toLogin();
    }
    // submit({ needRedirect, redirectFullPath: route.fullPath }, { action: 'logout', method: 'post' });
  }

  function onLogout() {
    window?.$modal?.confirm({
      cancelText: t('common.cancel'),
      content: t('common.logoutConfirm'),
      okText: t('common.confirm'),
      onOk: () => handleLogout(),
      title: t('common.tip')
    });
  }

  function onClick({ key }: { key: string }) {
    if (key === 'home') {
      nav(`/home`)
    } else if (key === 'logout') {
      onLogout();
    } else if (key === 'password') {
      setPasswordVisible(true);
      // router.routerPushByKey('user-center');
    } else {
      // router.routerPushByKey('user-center');
    }
  }
  function loginOrRegister() {
    router.routerPushByKey('login', {
      query: { redirect: location.pathname }
    });
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

  if (managed) {
    items.splice(0, 2)
  }

  if (showHome) {
    items.unshift({
      key: 'home',
      label: (
        <div className="flex-center gap-8px">
          <SvgIcon
            className="text-icon"
            icon="mdi:settings"
          />
          {t('route.settings')}
        </div>
      )
    })
  }

  return (
    <>
      {userInfo ? (
        <Dropdown
          menu={{ items, onClick }}
          placement="bottomRight"
          trigger={['click']}
        >
          <div>
            <ButtonIcon className={`px-12px ${className}`}>
              <SvgIcon
                className="text-icon-large"
                icon="ph:user-circle"
              />
              {showName && <span className="text-16px font-medium">{userInfo.name}</span>}
            </ButtonIcon>
          </div>
        </Dropdown>
      ) : (
        <Button className={`px-12px ${className}`} onClick={loginOrRegister}>{t('page.login.common.loginOrRegister')}</Button>
      )}
      <Suspense>
        <PasswordModal
          open={passwordVisible}
          onClose={() => setPasswordVisible(false)}
          onSuccess={() => handleLogout()}
        />
      </Suspense>
    </>
  );
});

export default UserAvatar;
