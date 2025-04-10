import { request } from '../request';

/** get data source list */
export function fetchDataSourceList(params?: any) {
  const query: any = {
    from: params.from || 0,
    size: params.size || 10,
    sort: [
      {
        created: {
          order: 'desc'
        }
      }
    ]
  };
  if (params.query) {
    query.query = {
      bool: {
        must: [
          {
            query_string: {
              allow_leading_wildcard: false,
              fields: ['combined_fulltext'],
              fuzziness: 'AUTO',
              fuzzy_max_expansions: 10,
              fuzzy_prefix_length: 2,
              fuzzy_transpositions: true,
              query: params.query
            }
          }
        ]
      }
    };
  }
  return request<Api.Datasource.Datasource>({
    data: query,
    method: 'post',
    url: '/datasource/_search'
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
