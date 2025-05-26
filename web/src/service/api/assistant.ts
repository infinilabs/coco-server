import { request } from '../request';

export function searchAssistant(params?: any) {
  const query: any = {
    from: params.from || 0,
    size: params.size || 10,
    sort: params?.sort ? params?.sort : [
      {
        "created": {
          "order": "desc"
        }
      }
    ],
    query: {
      bool: {
        must: [
          ...(params?.filters || [])
        ]
      }
    }
  }
  if (params.query) {
    query.query.bool.must.push({
      "query_string": {
        "fields": ["combined_fulltext"],
        "query": params.query,
        "fuzziness": "AUTO",
        "fuzzy_prefix_length": 2,
        "fuzzy_max_expansions": 10,
        "fuzzy_transpositions": true,
        "allow_leading_wildcard": false
      }
    })
  }
  return request({
    method: 'post',
    data: query,
    url: '/assistant/_search'
  });
}

export function createAssistant(body: any){
  return request({
    method: 'post',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: '/assistant/'
  });
}

export function updateAssistant(id:string, body: any){
  return request({
    method: 'put',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: `/assistant/${id}`
  });
}

export function deleteAssistant(assistantID: string){
  return request({
    method: 'delete',
    url: `/assistant/${assistantID}`
  });
}

export function getAssistant(assistantID: string){
  return request({
    method: 'get',
    url: `/assistant/${assistantID}`
  });
}

export function cloneAssistant(assistantID: string){
  return request({
    method: 'post',
    url: `/assistant/${assistantID}/_clone`
  });
}

export function getAssistantCategory() {
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
    url: '/assistant/_search'
  });
}