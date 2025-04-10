import { request } from '../request';

/** Get server's info */
export function fetchServer() {
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
export function updateSettings(data: { llm?: any; server?: any }) {
  return request({
    data,
    method: 'put',
    url: '/settings'
  });
}
