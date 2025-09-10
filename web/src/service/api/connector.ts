import { request } from '../request';
import { formatSearchFilter } from '../request/es';

/** Get connector list */
export function searchConnector(params: any) {
  return request<Api.Datasource.Connector>({
    method: 'get',
    params,
    url: '/connector/_search'
  });
}

export function createConnector(body: any) {
  return request<Api.Datasource.Connector>({
    data: body,
    method: 'post',
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
    data: body,
    method: 'put',
    url: `/connector/${connectorID}`
  });
}

export function getConnectorIcons() {
  return request({
    method: 'get',
    url: `/icons/list`
  });
}

export function getConnector(ID: string) {
  return request<Api.Datasource.Connector>({
    method: 'get',
    url: `/connector/${ID}`
  });
}

export function getConnectorByIDs(connectorIDs: string[]) {
  return request<Api.Datasource.Connector>({
    method: 'get',
    params: {
      _source_includes: ['id', 'name', 'icon'].join(','),
      size: connectorIDs.length
    },
    url: `/connector/_search?${formatSearchFilter({ id: connectorIDs })}`
  })
}

export function getConnectorCategory() {
  const query = {
    aggs: {
      categories: {
        terms: {
          field: 'category',
          size: 100
        }
      }
    },
    size: 0
  };
  return request<Api.Datasource.Connector>({
    data: query,
    method: 'post',
    url: '/connector/_search'
  });
}
