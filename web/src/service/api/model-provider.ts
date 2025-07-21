import { Api } from '@/types/api';
import { request } from '../request';

/**
 * Get connector list
 *
 */
export function searchModelPovider(params: any) {
  return request<Api.LLM.ModelProvider>({
    method: 'get',
    params,
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

export function getLLMModels() {
  const query: any = {
    size: 0,
    aggs: {
      models: {
        terms: {
          field: 'models',
          size: 100
        }
      }
    }
  }
  return request<Api.LLM.ModelProvider>({
    method: 'post',
    data: query,
    url: '/model_provider/_search'
  });
}


export function getEnabledModelProviders(size: number = 100) {
  const query: any = {
    size: size,
    query: {
      term: {
        enabled: true
      }
    },
    sort: [
      {
        "enabled": {
          "order": "desc"
        }
      },
      {
        "created": {
          "order": "desc"
        }
      }
    ],
  }
  return request<Api.LLM.ModelProvider>({
    method: 'post',
    data: query,
    url: '/model_provider/_search'
  });
}

