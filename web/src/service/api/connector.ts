import { request } from '../request';

/**
 * Get connector list
 *
 */
export function searchConnector(params: any) {
  return request<Api.Datasource.Connector>({
    method: 'get',
    params,
    url: '/connector/_search'
  });
}

export function createConnector(body: any) {
  return request<Api.Datasource.Connector>({
    method: 'post',
    data: body,
    url: '/connector/'
  });
}

export function deleteConnector(connectorID: string) {
  return request({
    method: 'delete',
    url: `/connector/${connectorID}`
  });
}

export function updateConnector(connectorID: string, body: any) {
  return request({
    method: 'put',
    data: body,
    url: `/connector/${connectorID}`
  });
}

export function getConnectorIcons() {
  return request({
    method: 'get',
    url: `/connector/icons/list`
  });
}