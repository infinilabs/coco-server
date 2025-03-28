import { request } from '../request';


export function fetchIntegrations(params) {
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
              "fields": ["combined_fulltext"],
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
  return request({
    method: 'post',
    data: query,
    url: '/integration/_search',
  });
}

export function fetchIntegration(id: string) {
    return request({
      method: 'get',
      url: `/integration/${id}`,
    });
}

export function createIntegration(data) {
  return request({
    method: 'post',
    data,
    url: '/integration/',
  });
}

export function updateIntegration(data) {
    const { id, ...rest } = data;
    return request({
      method: 'put',
      data: rest,
      url: `/integration/${id}`,
    });
}

export function deleteIntegration(id: string) {
  return request({
    method: 'delete',
    url: `/integration/${id}`,
  });
}