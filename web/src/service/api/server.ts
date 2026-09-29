import { request } from '../request';

/** Get server's info */
export function fetchApplicationSetting() {
  return request<Api.Server.Info>({
    method: 'get',
    url: '/setting/application'
  });
}

export function fetchProviderInfo() {
  return request<Api.Server.Info>({
    method: 'get',
    url: '/provider/_info'
  });
}

/** Get settings */
/** Read back the engine AI pipelines Coco manages: desired vs deployed + drift */
export function fetchEngineAIStatus() {
  return request({
    method: 'get',
    url: '/search/engine-ai'
  });
}

/** Push the Engine AI settings to the engine and return the fresh status */
export function syncEngineAI() {
  return request({
    method: 'post',
    url: '/search/engine-ai/sync'
  });
}

export function fetchSettings() {
  return request({
    method: 'get',
    url: '/settings'
  });
}

/** Update server's settings */
export function updateSettings(data: any) {
  return request({
    data,
    method: 'put',
    url: '/settings'
  });
}
