import { request } from '../request';

export function searchMCPServer(params?: any) {
  return request({
    method: 'get',
    params,
    url: '/mcp_server/_search'
  });
}

export function createMCPServer(body: any){
  return request({
    method: 'post',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: '/mcp_server/'
  });
}

export function updateMCPServer(id:string, body: any){
  return request({
    method: 'put',
    headers: {
      "Content-Type": "application/json",
    },
    data: body,
    url: `/mcp_server/${id}`
  });
}

export function deleteMCPServer(serverID: string){
  return request({
    method: 'delete',
    url: `/mcp_server/${serverID}`
  });
}

export function getMCPServer(serverID: string){
  return request({
    method: 'get',
    url: `/mcp_server/${serverID}`
  });
}

export function getMCPCategory() {
  const query = {
    aggs: {
      categories: {
        terms: {
          field: 'category',
          size: 100
        }
      }
    },
    size: 0
  };
  return request({
    data: query,
    method: 'post',
    url: '/mcp_server/_search'
  });
}