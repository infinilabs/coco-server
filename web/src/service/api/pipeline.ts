import { request } from '../request';

export function getEnablePipelines(params?: any) {
  return request({
    method: 'post',
    params,
    url: '/pipelines/_search?size=1000&filter=enabled:any(true)'
  });
}

export function createPipeline(data?: any) {
  return request({
    method: 'post',
    data,
    url: '/pipelines/'
  });
}