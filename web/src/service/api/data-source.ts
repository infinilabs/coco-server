import { request } from '../request';

/** get data source list */
export function fetchDataSourceList(params?: any) {
  const query: any = {
    from: params.from || 0,
    size: params.size || 10,
  }
  if (params.query) {
    query['query'] = {
      bool: {
        must: [
          {
            "query_string": {
              "fields": ["name"],
              "query": params.query,
              "fuzziness": "AUTO",
              "fuzzy_prefix_length": 2,
              "fuzzy_max_expansions": 10,
              "fuzzy_transpositions": true,
              "allow_leading_wildcard": false
            }
          }
        ]
      }
    }
  }
  return request<Api.Datasource.Datasource>({
    method: 'post',
    data: query,
    url: '/datasource/_search'
  });
}

export function fetchDatasourceDetail(params?: any){
  return request({
    method: 'get',
    params,
    url: '/document/_search'
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

export function updateDocument(documentID: string, body: any){
  return request({
    method: 'put',
    url: `/document/${documentID}`,
    data: body,
  });
}

export function batchDeleteDocument(body: any){
  return request({
    method: 'delete',
    url: `/document/`,
    data: body,
  });
}