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

export function searchAssistantTemplates() {
  return request<{ hits: any }>({ method: 'get', url: '/assistant-template/_search?size=100' }).then(res => {
    const raw = (res?.data?.hits?.hits || []) as any[];
    return raw.map(h => ({ id: h._id, ...(h._source || {}) }));
  });
}

export function instantiateAssistantTemplate(id: string, body: { name?: string; kb_id?: string }) {
  return request<{ _id: string }>({ method: 'post', data: body, url: `/assistant-template/${id}/_instantiate` }).then(
    res => res?.data
  );
}
