import { Api } from '@/types/api';
import { request } from '../request';
import { formatSearchFilter } from '../request/es';

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
  const params: any = {
    size: size,
    filter: {
      enabled: [true]
    },
    sort: 'enabled:desc,created:desc',
  }
  const { filter = {}, ...rest } = params || {}
  return request<Api.LLM.ModelProvider>({
    method: 'get',
    params: rest,
    url: `/model_provider/_search?${formatSearchFilter(filter)}`
  });
}

