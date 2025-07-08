import { request } from '../request';
import { formatSearchFilter } from '../request/es';

/** get data source list */
export function fetchDataSourceList(params?: any) {
  const { filter = {}, ...rest } = params || {}

  return request<Api.Datasource.Datasource>({
    method: 'post',
    params: rest,
    url: `/datasource/_search?${formatSearchFilter(filter)}`
  });
}

export function fetchDatasourceDetail(params?: any) {
  return request({
    method: 'get',
    params,
    url: '/document/_search'
  });
}

export function createDatasource(body: any) {
  return request({
    data: body,
    headers: {
      'Content-Type': 'application/json'
    },
    method: 'post',
    url: '/datasource/'
  });
}

export function updateDatasource(id: string, body: any) {
  return request({
    data: body,
    headers: {
      'Content-Type': 'application/json'
    },
    method: 'put',
    url: `/datasource/${id}`
  });
}

export function deleteDatasource(dataourceID: string) {
  return request({
    method: 'delete',
    url: `/datasource/${dataourceID}`
  });
}

export function getDatasource(dataourceID: string) {
  return request({
    method: 'get',
    url: `/datasource/${dataourceID}`
  });
}

export function deleteDocument(documentID: string) {
  return request({
    method: 'delete',
    url: `/document/${documentID}`
  });
}

export function updateDocument(documentID: string, body: any) {
  return request({
    data: body,
    method: 'put',
    url: `/document/${documentID}`
  });
}

export function batchDeleteDocument(body: any) {
  return request({
    data: body,
    method: 'delete',
    url: `/document/`
  });
}
