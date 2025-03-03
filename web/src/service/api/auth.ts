import { request } from '../request';

/**
 * Login
 *
 * @param password Password
 */
export function fetchLogin(password: string) {
  return request<Api.Auth.LoginToken>({
    data: {
      password,
    },
    method: 'post',
    url: '/account/login'
  });
}

/** Get user info */
export function fetchGetUserInfo() {
  return request<Api.Auth.UserInfo>({ url: '/account/profile' });
}

/**
 * Modify password
 *
 * @param old_password Password
 * @param new_password Password
 */
export function modifyPassword(old_password: string, new_password: string) {
  return request({
    data: {
      old_password,
      new_password
    },
    method: 'put',
    url: '/account/password'
  });
}

/**
 * Refresh token
 *
 * @param refreshToken Refresh token
 */
export function fetchRefreshToken(refreshToken: string) {
  return request<Api.Auth.LoginToken>({
    data: {
      refreshToken
    },
    method: 'post',
    url: '/auth/refreshToken'
  });
}

/**
 * return custom backend error
 *
 * @param code error code
 * @param msg error message
 */
export function fetchCustomBackendError(code: string, msg: string) {
  return request({ params: { code, msg }, url: '/auth/error' });
}
