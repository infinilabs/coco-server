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
/** Run (or fetch cached) the content dedup scan */
export function fetchDedupReport(refresh = false) {
  return request({
    method: 'get',
    url: '/document/dedup/report',
    params: refresh ? { refresh: 1 } : {}
  });
}

/** Mark a pair (or whole group) as NOT duplicates — never reported again */
export function dismissDedupGroup(ids: string[]) {
  return request({
    method: 'post',
    url: '/document/dedup/dismiss',
    data: { ids }
  });
}

/** Apply a reviewer decision to whole documents: exclude (disable) | delete */
/** confirm a near-duplicate group — its members fold in search results */
export function confirmDedupGroup(ids: string[], tier: string) {
  return request({
    method: 'post',
    url: '/document/dedup/confirm_group',
    data: { ids, tier }
  });
}

/** remove a confirmed group (un-folds its members) */
export function removeConfirmedGroup(key: string) {
  return request({
    method: 'delete',
    url: `/document/dedup/confirm_group/${key}`
  });
}

export function actDedupGroup(ids: string[], action: 'exclude' | 'delete') {
  return request({
    method: 'post',
    url: '/document/dedup/action',
    data: { ids, action }
  });
}

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

/** Probe the server's document-pipeline dependencies (tika, libreoffice, poppler, chrome) and engine health */
export function fetchEnvironmentCheck() {
  return request<Api.Environment.CheckResult>({
    method: 'get',
    url: '/environment/_check'
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
