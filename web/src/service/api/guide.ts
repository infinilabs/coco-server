import { request } from '../request';

/**
 * setup
 *
 */
export function setup(data: { name: string; email: string; password: string; llm: any }) {
  return request({
    data: data,
    method: 'post',
    url: '/setup/_initialize'
  });
}