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
export function createToken(data) {
  return request<Api.APIToken.APIToken>({
    data,
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

/** update API Token */
export function updateToken(data) {
  const { id, ...rest } = data
  return request<Api.APIToken.APIToken>({
    data: rest,
    method: 'put',
    url: `/auth/access_token/${id}`
  });
}
