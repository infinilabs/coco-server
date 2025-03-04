import { request } from '../request';

/** get data source list */
export function fetchDataSourceList(params?: any) {
  return request<Api.Datasource.Datasource>({
    method: 'get',
    params,
    url: '/datasource/_search'
  });
}

export function fetchDatasourceDetail(params?: any){
  return request({
    method: 'get',
    params,
    url: '/query/_search'
  });
}

export function createDatasource(body: any){
  return request({
    method: 'post',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: '/datasource/'
  });
}

export function updateDatasource(id:string, body: any){
  return request({
    method: 'put',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: `/datasource/${id}`
  });
}

export function deleteDatasource(dataourceID: string){
  return request({
    method: 'delete',
    url: `/datasource/${dataourceID}`
  });
}

export function deleteDocument(documentID: string){
  return request({
    method: 'delete',
    url: `/document/${documentID}`
  });
}