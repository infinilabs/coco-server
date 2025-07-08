import { request } from '../request';
import { formatSearchFilter } from '../request/es';

export function searchAssistant(params?: any) {
  const { filter = {}, ...rest } = params || {}
  return request({
    method: 'get',
    params: rest,
    url: `/assistant/_search?${formatSearchFilter(filter)}`
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