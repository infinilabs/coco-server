import { request } from '../request';

/**
 * Login
 *
 * @param password Password
 */
export function fetchLogin(password: string) {
  return request<Api.Auth.LoginToken>({
    data: {
      password
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
      new_password,
      old_password
    },
    method: 'put',
    url: '/account/password'
  });
}

/** Logout */
export function logout() {
  return request({
    method: 'post',
    url: '/account/logout'
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

export function fetchAccessToken() {
  return request<Api.Auth.LoginToken>({
    url: '/auth/access_token',
    method: 'post'
  });
}
