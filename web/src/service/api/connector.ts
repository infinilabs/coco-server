import { request } from '../request';

/** Get connector list */
export function searchConnector(params: any) {
  const query: any = {
    from: params.from || 0,
    size: params.size || 10
  };
  if (params.query) {
    query.query = {
      bool: {
        must: [
          {
            query_string: {
              allow_leading_wildcard: false,
              fields: ['combined_fulltext'],
              fuzziness: 'AUTO',
              fuzzy_max_expansions: 10,
              fuzzy_prefix_length: 2,
              fuzzy_transpositions: true,
              query: params.query
            }
          }
        ]
      }
    };
  }
  return request<Api.Datasource.Connector>({
    data: query,
    method: 'post',
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
  const query: any = {
    _source: ['id', 'name', 'icon'],
    size: connectorIDs.length
  };
  query.query = {
    terms: {
      id: connectorIDs
    }
  };
  return request<Api.Datasource.Connector>({
    data: query,
    method: 'post',
    url: '/connector/_search'
  });
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
