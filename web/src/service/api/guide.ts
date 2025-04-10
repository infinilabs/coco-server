import { request } from '../request';

/** setup */
export function setup(data: { email: string; llm: any; name: string; password: string }) {
  return request({
    data,
    method: 'post',
    url: '/setup/_initialize'
  });
}
