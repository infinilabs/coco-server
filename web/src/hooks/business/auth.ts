import { getIsLogin, selectUserInfo } from '@/store/slice/auth';

export function useAuth() {
  const userInfo = useAppSelector(selectUserInfo);
  const isLogin = useAppSelector(getIsLogin);
  function hasAuth(codes: string | string[]) {
    if (!isLogin) {
      return false;
    }

    if (typeof codes === 'string') {
      return userInfo?.permissions?.includes(codes);
    }

    return codes.every(code => userInfo?.permissions?.includes(code));
  }

  return {
    hasAuth,
    permissions: userInfo?.permissions
  };
}
