import { request } from '../request';

export function searchMCPServer(params?: any) {
  const query: any = {
    from: params.from || 0,
    size: params.size || 10,
    sort: [
      {
        "created": {
          "order": "desc"
        }
      }
    ]
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
  return request({
    method: 'post',
    data: query,
    url: '/mcp_server/_search'
  });
}

export function createMCPServer(body: any){
  return request({
    method: 'post',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: '/mcp_server/'
  });
}

export function updateMCPServer(id:string, body: any){
  return request({
    method: 'put',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: `/mcp_server/${id}`
  });
}

export function deleteMCPServer(serverID: string){
  return request({
    method: 'delete',
    url: `/mcp_server/${serverID}`
  });
}

export function getMCPServer(serverID: string){
  return request({
    method: 'get',
    url: `/mcp_server/${serverID}`
  });
}

export function getMCPCategory() {
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
  return request({
    data: query,
    method: 'post',
    url: '/mcp_server/_search'
  });
}