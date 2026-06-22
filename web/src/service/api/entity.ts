import { request } from '../request';

export function fetchBatchEntityLabels(data: any, options?: any) {
  return request({
    method: 'post',
    url: `/entity/label/_batch_get`,
    data,
    ...(options || {})
  });
}

export function fetchEntityCard(params: any, options?: any) {
  const { type, id } = params

  return request({
    method: 'post',
    url: `/entity/card/${type}/${id}`,
    ...(options || {})
  });
}

export function fetchEntityUser(params: any, options?: any) {
  const { id } = params

  return request({
    method: 'post',
    url: `/entity/card/user/${id}`,
    ...(options || {})
  });
}