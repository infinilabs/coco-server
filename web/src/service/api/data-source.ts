import { request } from '../request';

/** get data source list */
export function fetchDataSourceList(params?: Api.SystemManage.CommonSearchParams) {
  return request<Api.Datasource.ItemList>({
    method: 'get',
    params,
    url: '/datasource/_search'
  });
}