import { request } from '../request';
import { formatSearchFilter } from '../request/es';

export function searchAssistant(params?: any, option?: any) {
  const { filter = {}, ...rest } = params || {}
  return request({
    method: 'get',
    params: rest,
    url: `/assistant/_search?${formatSearchFilter(filter)}`,
    ...(option || {})
  })
}

export function createAssistant(body: any){
  return request({
    method: 'post',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: '/assistant/'
  });
}

export function updateAssistant(id:string, body: any){
  return request({
    method: 'put',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: `/assistant/${id}`
  });
}

export function deleteAssistant(assistantID: string){
  return request({
    method: 'delete',
    url: `/assistant/${assistantID}`
  });
}

export function getAssistant(assistantID: string){
  return request({
    method: 'get',
    url: `/assistant/${assistantID}`
  });
}

export function cloneAssistant(assistantID: string){
  return request({
    method: 'post',
    url: `/assistant/${assistantID}/_clone`
  });
}

export function getAssistantCategory() {
  const query = {
    aggs: {
      categories: {
        terms: {
          field: 'category',
          size: 100
        }
      }
    },
    size: 0
  };
  return request({
    data: query,
    method: 'post',
    url: '/assistant/_search'
  });
}


/** Count-only chat session query, optionally narrowed to a `created` window.
 *  IMPORTANT: the window must use the orm `filter` syntax — a `query.created`
 *  param is silently ignored by the backend (only fulltext `query` and `filter`
 *  params are parsed). e.g. ['created>=now-7d'] for the last 7 days, or
 *  ['created>=now-14d', 'created<now-7d'] for the previous 7-day window. */
export function fetchChatSessionCount(created?: string[]) {
  const qs = (created ?? []).map(r => `filter=${encodeURIComponent(r)}`).join('&');
  return request({
    method: 'get',
    params: { size: 0 },
    url: `/chat/_history${qs ? `?${qs}` : ''}`
  });
}
