import { request } from '../request';

export function fetchBatchEntityLabels(data: any) {
  return request({
    method: 'post',
    url: `/entity/label/_batch_get`,
    data
  });
}

export function fetchEntityCard(params: any) {
  const { type, id } = params

  return request({
    method: 'post',
    url: `/entity/card/${type}/${id}`
  });
}