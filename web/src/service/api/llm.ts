import { Api } from '@/types/api';
import { request } from '../request';

/**
 * Get connector list
 *
 */
export function searchModelPovider(params: any) {
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
  return request<Api.LLM.ModelProvider>({
    method: 'post',
    data: query,
    url: '/model_provider/_search'
  });
}

export function createModelProvider(body: any) {
  return request<Api.LLM.ModelProvider>({
    method: 'post',
    data: body,
    url: '/model_provider/'
  });
}

export function deleteModelProvider(providerID: string) {
  return request({
    method: 'delete',
    url: `/model_provider/${providerID}`
  });
}

export function updateModelProvider(providerID: string, body: any) {
  return request({
    method: 'put',
    data: body,
    url: `/model_provider/${providerID}`
  });
}

export function getModelProvider(providerID: string) {
  return request({
    method: 'get',
    url: `/model_provider/${providerID}`
  });
}