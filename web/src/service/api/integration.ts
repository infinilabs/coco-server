import { request } from '../request';

export function fetchIntegrations(params) {
  return request({
    method: 'get',
    params,
    url: '/integration/_search'
  });
}

export function fetchIntegration(id: string) {
  return request({
    method: 'get',
    url: `/integration/${id}`
  });
}

export function createIntegration(data) {
  return request({
    data,
    method: 'post',
    url: '/integration/'
  });
}

export function updateIntegration(data) {
  const { id, ...rest } = data;
  return request({
    data: rest,
    method: 'put',
    url: `/integration/${id}`
  });
}

export function deleteIntegration(id: string) {
  return request({
    method: 'delete',
    url: `/integration/${id}`
  });
}

export function renewAPIToken(id: string) {
  return request({
    method: 'post',
    url: `/integration/${id}/_renew_token`
  });
}