import { request } from '../request';

export function querySearch(data: any, search: string, options?: any) {
  return request({
    ...(options || {}),
    data,
    method: 'post',
    url: `/query/_search?${search || ''}`
  });
}

export function assistantAsk(data: any, id: string, options?: any) {
  return request({
    ...(options || {}),
    data,
    method: 'post',
    url: `/assistant/${id}/_ask`
  });
}

export function fetchSuggestions(tag: string | undefined, params: any, options?: any) {
  return request({
    ...(options || {}),
    params,
    method: 'get',
    url: `/query/_suggest${tag ? `/${tag}` : ''}`
  });
}

export function fetchRecommends(tag: string | undefined, options?: any) {
  return request({
    ...(options || {}),
    method: 'get',
    url: `/query/_recommend${tag ? `/${tag}` : ''}`
  });
}

export function fetchHotSearches(options?: any) {
  return request({
    ...(options || {}),
    method: 'get',
    url: `/query/_hot`
  });
}

// 相关搜索 (方案 B · 词面关联): GET /query/_related?q= — search-log
// lexical neighbors for the result-page sidebar
export function fetchRelatedSearches(query: string, options?: any) {
  return request({
    ...(options || {}),
    method: 'get',
    url: `/query/_related?q=${encodeURIComponent(query)}`
  });
}

export function fetchFieldsMeta(fields: string[], options?: any) {
  return request({
    ...(options || {}),
    method: 'get',
    url: `/field_meta/${fields.join(',')}`
  });
}

export function uploadAttachment(files: any[], options?: any) {
  const formData = new FormData();
  for (const f of files) {
    formData.append('files', f, f.name);
  }
  return request({
    ...(options || {}),
    method: 'post',
    url: `/attachment/_upload`,
    data: formData
  });
}