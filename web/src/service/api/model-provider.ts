import { Api } from '@/types/api';
import { request } from '../request';
import { formatSearchFilter } from '../request/es';

/**
 * Get connector list
 *
 */
export function searchModelPovider(params: any) {
  const { filter = {}, sort, ...rest } = params || {};
  // Convert sort array to string format if needed
  let sortStr = sort;
  if (Array.isArray(sort)) {
    sortStr = sort.map(([field, order]: [string, string]) => `${field}:${order}`).join(',');
  }
  return request<Api.LLM.ModelProvider>({
    method: 'get',
    params: { ...rest, sort: sortStr },
    url: `/model_provider/_search?${formatSearchFilter(filter)}`
  });
}

export function createModelProvider(body: any) {
  return request<Api.LLM.ModelProvider>({
    method: 'post',
    data: body,
    url: '/model_provider/'
  });
}

export function fetchProviderModels(body: { api_type: string; base_url: string; api_key?: string }) {
  return request<{ ok: boolean; models?: string[]; status?: number; error?: string }>({
    method: 'post',
    data: body,
    url: '/model_provider/_models'
  });
}

/**
 * Probe model availability before the provider is saved: the backend issues
 * one minimal vendor call (chat/embeddings/rerank per model type) with the
 * given url + key + model id and reports ok/status/latency.
 */
export function testModelProvider(body: {
  api_type?: string;
  base_url: string;
  api_key?: string;
  model: string;
  type?: string;
}) {
  return request<{
    ok: boolean;
    status?: number;
    latency_ms?: number;
    detail?: string;
    error?: string;
  }>({
    method: 'post',
    data: body,
    url: '/model_provider/_test'
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

