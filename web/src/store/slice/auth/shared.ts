import { localStg } from '@/utils/storage';
/** Get token */
export function getToken() {
  return localStg.get('token') || '';
}

/** Get user info */
export function getUserInfo() {
  const userInfo = localStg.get('userInfo') || {
    avatar: '',
    created: '',
    email: '',
    id: '',
    name: '',
    preferences: {
      language: '',
      theme: ''
    },
    roles: [],
    updated: ''
  };

  return userInfo;
}

/** Clear auth storage */
export function clearAuthStorage() {
  localStg.remove('token');
  localStg.remove('refreshToken');
  localStg.remove('userInfo');
}
