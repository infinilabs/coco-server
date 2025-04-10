import { localStg } from '@/utils/storage';
/** Get token */
export function getToken() {
  return localStg.get('token') || '';
}

/** Get user info */
export function getUserInfo() {
  return localStg.get('userInfo');
}

/** Clear auth storage */
export function clearAuthStorage() {
  localStg.remove('userInfo');
}
