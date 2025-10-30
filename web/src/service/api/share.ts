import { request } from '../request';
import { formatSearchFilter } from '../request/es';

export function fetchShares(params: any) {
  const { type, id } = params

  return request({
    method: 'get',
    url: `/resources/${type}/${id}/shares`
  });
}

export function fetchBatchShares(data: any) {
  return request({
    method: 'post',
    url: `/resources/shares/_batch_get`,
    data
  });
}

export function fetchCurrentUserPermission(params: any) {
  const { type, id } = params

  return request({
    method: 'get',
    url: `/resources/${type}/${id}/access`
  });
}

export function addShares(data: any) {
  return request({
      method: 'post',
      url: `/resources/shares/_batch_set`,
      data: data
  });
}

export function updateShares(data: any) {
  const { type, id, ...rest } = data

    return request({
        method: 'post',
        url: `/resources/${type}/${id}/share`,
        data: rest
    });
}

export function deleteShares(data: any) {
  const { type, id, fileID, ...rest } = data

    return request({
        method: 'delete',
        url: `/resources/${type}/${id}/share/${fileID}`,
        data: rest
    });
}

export function fetchPrincipals(params) {
  const { filter = {}, ...rest } = params || {}
  return request({
    method: 'get',
    params: rest,
    url: `/security/principal/_search?${formatSearchFilter(filter)}`
  })
}