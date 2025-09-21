import { request } from '../request';

/** get API Token list */
export function getTokens(params?: any) {
  const { filter = {}, ...rest } = params || {}

  return request<any>({
    method: 'get',
    params: rest,
    url: `/auth/access_token/_search`
  });
}

/** create API Token */
export function createToken(name: string) {
  return request<Api.APIToken.APIToken>({
    data: {
      name
    },
    method: 'post',
    url: '/auth/access_token'
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
