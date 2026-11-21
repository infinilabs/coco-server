import { request } from '../request';

export function searchSkill(params?: any) {
  return request({
    method: 'get',
    params,
    url: '/skill/_search'
  });
}

export function getSkill(id: string) {
  return request({
    method: 'get',
    url: `/skill/${id}`
  });
}

export function createSkill(body: any) {
  return request({
    method: 'post',
    headers: {
      'Content-Type': 'application/json'
    },
    data: body,
    url: '/skill/'
  });
}

export function updateSkill(id: string, body: any) {
  return request({
    method: 'put',
    headers: {
      'Content-Type': 'application/json'
    },
    data: body,
    url: `/skill/${id}`
  });
}

export function deleteSkill(id: string) {
  return request({
    method: 'delete',
    url: `/skill/${id}`
  });
}

export function exportSkillsMD() {
  return request({
    method: 'get',
    responseType: 'text',
    url: '/skill/_export/skills_md'
  });
}

export function exportMCPJSON() {
  return request({
    method: 'get',
    url: '/skill/_export/mcp_json'
  });
}
