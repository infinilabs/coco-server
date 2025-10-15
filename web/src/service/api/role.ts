import { request } from '../request';
import { formatSearchFilter } from '../request/es';

export function fetchRoles(params) {
  const { filter = {}, ...rest } = params || {}
  return request({
    method: 'get',
    params: rest,
    url: `/integration/_search?${formatSearchFilter(filter)}`
  })
}

export function fetchRole(id: string) {
  return request({
    method: 'get',
    url: `/integration/${id}`
  });
}

export function createRole(data) {
  return request({
    data,
    method: 'post',
    url: '/integration/'
  });
}

export function updateRole(data) {
  const { id, ...rest } = data;
  return request({
    data: rest,
    method: 'put',
    url: `/integration/${id}`
  });
}

export function deleteRole(id: string) {
  return request({
    method: 'delete',
    url: `/integration/${id}`
  });
}

export function fetchPermissions() {
  return request({
    method: 'get',
    url: `/integration/_search`
  })
}