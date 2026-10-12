import { request } from '../request';
import { formatSearchFilter } from '../request/es';

/** get data source list */
export function fetchDataSourceList(params?: any) {
  const { filter = {}, ...rest } = params || {}

  return request<Api.Datasource.Datasource>({
    method: 'get',
    params: rest,
    url: `/datasource/_search?${formatSearchFilter(filter)}`
  });
}

export function fetchDatasourceDetail(params?: any) {
  const { filter = {}, ...rest } = params || {}
  return request({
    method: 'get',
    params: rest,
    url: `/document/_search?${formatSearchFilter(filter)}`
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

export function getDocument(documentID: string) {
  return request({
    method: 'get',
    url: `/document/${documentID}`
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

/** 数据源能力概览（W4）：文档总数 + 向量化/FAQ/Wiki 覆盖情况 */
export function fetchDatasourceCapabilities(datasourceID: string) {
  return request<{ datasource_id: string; documents: number; vector: boolean; faq: boolean; wiki: boolean }>({
    method: 'get',
    url: `/datasource/${datasourceID}/capabilities`
  });
}

/** 单文档重放（失败文档无视指纹守卫，force 语义与后端 reprocessurl 一致） */
export function reprocessDocument(documentID: string) {
  return request({
    method: 'post',
    url: `/document/${documentID}/_reprocess?force=true`
  });
}
