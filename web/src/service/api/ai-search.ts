import { request } from '../request';

export function querySearch(data: any, search: string, options?: any) {
  return request({
    ...(options || {}),
    data,
    method: 'post',
    url: `/query/_search?${search || ''}`
  });
}

export function assistantAsk(data: any, id: string, options?: any) {
  return request({
    ...(options || {}),
    data,
    method: 'post',
    url: `/assistant/${id}/_ask`
  });
}

export function fetchSuggestions(tag: string | undefined, params: any, options?: any) {
  return request({
    ...(options || {}),
    params,
    method: 'get',
    url: `/query/_suggest${tag ? `/${tag}` : ''}`
  });
}

export function fetchRecommends(tag: string | undefined, options?: any) {
  return request({
    ...(options || {}),
    method: 'get',
    url: `/query/_recommend${tag ? `/${tag}` : ''}`
  });
}

export function fetchFieldsMeta(fields: string[], options?: any) {
  return request({
    ...(options || {}),
    method: 'get',
    url: `/field_meta/${fields.join(',')}`
  });
}