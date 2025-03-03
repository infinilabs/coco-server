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

export function deleteDatasource(dataourceID: string){
  return request({
    method: 'delete',
    url: `/datasource/${dataourceID}`
  });
}