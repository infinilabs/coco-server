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
