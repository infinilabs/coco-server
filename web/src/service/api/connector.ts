import { request } from '../request';

/**
 * Get connector list
 *
 */
export function searchConnector(params: any) {
  const query: any = {
    from: params.from || 0,
    size: params.size || 10,
  }
  if (params.query) {
    query['query'] = {
      bool: {
        must: [
          {
            "query_string": {
              "fields": ["combined_fulltext"],
              "query": params.query,
              "fuzziness": "AUTO",
              "fuzzy_prefix_length": 2,
              "fuzzy_max_expansions": 10,
              "fuzzy_transpositions": true,
              "allow_leading_wildcard": false
            }
          }
        ]
      }
    }
  }
  return request<Api.Datasource.Connector>({
    method: 'post',
    data: query,
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

export function getConnector(ID: string) {
  return request<Api.Datasource.Connector>({
    method: 'get',
    url: `/connector/${ID}`
  });
}

export function getConnectorByIDs(connectorIDs: string[]) {
  const query: any = {
    size: connectorIDs.length,
    _source: ["id", "name", "icon"],
  }
  query["query"] = {
    terms: {
      "id": connectorIDs,
    }
  }
  return request<Api.Datasource.Connector>({
    method: 'post',
    data: query,
    url: '/connector/_search'
  });
}