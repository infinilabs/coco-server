import { request } from '../request';
import { formatSearchFilter } from '../request/es';

export function fetchWebhooks(params) {
  const { filter = {}, ...rest } = params || {};
  return request({
    method: 'get',
    params: rest,
    url: `/webhook/_search?${formatSearchFilter(filter)}`
  });
}

export function fetchWebhook(id: string) {
  return request({
    method: 'get',
    url: `/webhook/${id}`
  });
}

export function createWebhook(data) {
  return request({
    data,
    method: 'post',
    url: '/webhook/'
  });
}

export function updateWebhook(data) {
  const { id, ...rest } = data;
  return request({
    data: rest,
    method: 'put',
    url: `/webhook/${id}`
  });
}

export function deleteWebhook(id: string) {
  return request({
    method: 'delete',
    url: `/webhook/${id}`
  });
}

export function testWebhook(id: string) {
  return request({
    method: 'post',
    url: `/webhook/${id}/_test`
  });
}