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

// POST /security/authorization/
export function fetchAuthorization(data: any) {
  return request({
    data,
    method: 'post',
    url: '/security/authorization/'
  });
}

// GET /security/authorization/_search
export function fetchAuthorizationSearch(params: any) {
  const { filter = {}, ...rest } = params || {};
  return request({
    method: 'get',
    params: rest,
    url: `/security/authorization/_search?${formatSearchFilter(filter)}`
  });
}

// PUT /security/authorization/:id
export function updateAuthorization(data: any) {
  const { id, ...rest } = data;
  return request({
    data: rest,
    method: 'put',
    url: `/security/authorization/${id}`
  });
}

// DELETE /security/authorization/:id
export function deleteAuthorization(id: string) {
  return request({
    method: 'delete',
    url: `/security/authorization/${id}`
  });
}

// GET /security/authorization/:id
export function fetchAuthorizationDetail(id: string) {
  return request({
    method: 'get',
    url: `/security/authorization/${id}`
  });
}

// GET /security/principal/_search?from=0&size=10&type=user type=user/team
export function fetchUserSearch(params: any) {
  const { filter = {}, ...rest } = params || {};
  return request({
    method: 'get',
    params: rest,
    url: `/security/principal/_search?${formatSearchFilter(filter)}`
  });
}
