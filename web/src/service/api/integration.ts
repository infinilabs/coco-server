import { request } from '../request';

export function fetchIntegrations(params) {
  const query: any = {
    from: params.from || 0,
    size: params.size || 10,
    sort: [{ created: 'desc' }]
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
  return request({
    data: query,
    method: 'post',
    url: '/integration/_search'
  });
}

export function fetchIntegration(id: string) {
  return request({
    method: 'get',
    url: `/integration/${id}`
  });
}

export function createIntegration(data) {
  return request({
    data,
    method: 'post',
    url: '/integration/'
  });
}

export function updateIntegration(data) {
  const { id, ...rest } = data;
  return request({
    data: rest,
    method: 'put',
    url: `/integration/${id}`
  });
}

export function deleteIntegration(id: string) {
  return request({
    method: 'delete',
    url: `/integration/${id}`
  });
}

export function renewAPIToken(id: string) {
  return request({
    method: 'post',
    url: `/integration/${id}/_renew_token`
  });
}