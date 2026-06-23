import { request } from '../request';

/** setup */
export function setup(data: any) {
  return request({
    data,
    method: 'post',
    url: '/setup/_initialize'
  });
}

export function setupModel(data: any) {
  return request({
    data,
    method: 'post',
    url: '/setup/_initialize/default_model'
  });
}