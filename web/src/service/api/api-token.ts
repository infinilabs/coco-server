import { request } from '../request';

/** get API Token list */
export function getTokens() {
  return request<Api.APIToken.APIToken>({
    method: 'get',
    url: '/auth/access_token/_cat'
  });
}

/** create API Token */
export function createToken(name: string) {
  return request<Api.APIToken.APIToken>({
    data: {
      name
    },
    method: 'post',
    url: '/auth/request_access_token'
  });
}

/** delete API Token */
export function deleteToken(tokenID: string) {
  return request({
    method: 'delete',
    url: `/auth/access_token/${tokenID}`
  });
}

/** rename API Token */
export function renameToken(tokenID: string, name: string) {
  return request<Api.APIToken.APIToken>({
    data: {
      name
    },
    method: 'post',
    url: `/auth/access_token/${tokenID}/_rename`
  });
}
