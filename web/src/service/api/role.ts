import { request } from '../request';
import { formatSearchFilter } from '../request/es';

export function fetchRoles(params: any) {
  const { filter = {}, ...rest } = params || {};
  return request({
    method: 'get',
    params: rest,
    url: `/security/role/_search?${formatSearchFilter(filter)}`
  });
}

export function fetchRole(id: string) {
  return request({
    method: 'get',
    url: `/security/role/${id}`
  });
}

export function createRole(data: any) {
  return request({
    data,
    method: 'post',
    url: '/security/role/'
  });
}

export function updateRole(data: any) {
  const { id, ...rest } = data;
  return request({
    data: rest,
    method: 'put',
    url: `/security/role/${id}`
  });
}

export function deleteRole(id: string) {
  return request({
    method: 'delete',
    url: `/security/role/${id}`
  });
}

export function fetchPermissions() {
  return request({
    method: 'get',
    url: `/security/permission/`
  });
}
