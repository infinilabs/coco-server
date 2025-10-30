import { request } from '../request';

export function fetchEntity(params: any) {
  const { type, id } = params

  return request({
    method: 'get',
    url: `/entity/card/${type}/${id}`
  });
}

export function fetchBatchEntity(data: any) {
  return request({
    method: 'post',
    url: `/entity/label/_batch_get`,
    data
  });
}