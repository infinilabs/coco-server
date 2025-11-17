import { useLoading } from '@sa/hooks';
import { useTranslation } from 'react-i18next';

import { getUerName, login, resetStore } from '@/store/slice/auth';
import { initAuthRoute } from '@/store/slice/route';

import { useAppDispatch } from '../business/useStore';

import { useRouterPush } from './routerPush';

export function useLogin() {
  const { endLoading, loading, startLoading } = useLoading();
  const { redirectFromLogin } = useRouterPush();
  const { t } = useTranslation();

  const dispatch = useAppDispatch();

  async function toLogin(params: { password: string; email: string }, redirect = true, onSuccess: () => void = () => {}) {
    startLoading();
    dispatch(login(params)).then(async (result) => {
      if (result.payload) {
        const userName = dispatch(getUerName());
  
        if (userName) {
          await dispatch(initAuthRoute());
  
          if (redirect) {
            await redirectFromLogin(redirect);
          }
  
          if (onSuccess) onSuccess()
  
          window.$notification?.success({
            description: t('page.login.common.welcomeBack', { userName }),
            message: t('page.login.common.loginSuccess')
          });
        } else {
          dispatch(resetStore());
        }
      }
      endLoading();
    })
  }

  return {
    loading,
    toLogin
  };
}
