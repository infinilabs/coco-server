import { request } from '../request';

export interface GatewayInstance {
  id: string;
  name: string;
  alias?: string;
  type: string;
  status: 'pending' | 'approved' | string;
  online: boolean;
  version?: string;
  endpoint?: string;
  ip?: string;
  hostname?: string;
  os?: string;
  labels?: Record<string, string>;
  tags?: string[];
  groups?: string[];
  note?: string;
  last_sync?: number;
  reverse_connected?: boolean;
  created?: string;
}

export interface AssignedConfig {
  name: string;
  version: number;
  size: number;
  target?: string;
}

export interface GatewayInstanceDetail extends GatewayInstance {
  services?: { name?: string; endpoint?: string }[];
  assigned_configs?: AssignedConfig[];
}

export function fetchGatewayInstances(type?: string) {
  return request<{ total: number; items: GatewayInstance[] }>({
    method: 'get',
    url: `/gateway/instances${type ? `?type=${type}` : ''}`
  });
}

export function fetchGatewayInstanceDetail(id: string) {
  return request<GatewayInstanceDetail>({
    method: 'get',
    url: `/gateway/instances/${id}/detail`
  });
}

export function approveGatewayInstance(id: string) {
  return request<{ id: string; status: string; manager_token?: string }>({
    method: 'post',
    url: `/gateway/instances/${id}/_approve`
  });
}

export function updateGatewayInstanceMeta(id: string, data: { alias?: string; note?: string; groups?: string[] }) {
  return request<{ id: string; alias: string; note: string; groups: string[] }>({
    method: 'put',
    data,
    url: `/gateway/instances/${id}/_meta`
  });
}

export function forgetGatewayInstance(id: string) {
  return request<{ forgotten: boolean; id: string }>({
    method: 'delete',
    url: `/gateway/instances/${id}`
  });
}

export function proxyGatewayInstance(id: string, path: string) {
  return request<{ status: number; data: any }>({
    method: 'get',
    url: `/gateway/instances/${id}/proxy?path=${encodeURIComponent(path)}`
  });
}

// ─── pipeline deployments (framework Fleet API) ───────────────────────────

export interface PipelineDeployment {
  id: string;
  name: string;
  pipeline_id: string;
  enabled: boolean;
  version: number;
  placement: {
    selector?: Record<string, string>;
    instances?: string[];
    strategy: string; // all | singleton
    rollout?: { canary?: number; batch?: number; max_unavailable?: number };
  };
  created?: string;
  updated?: string;
}

export interface DeploymentInstanceReport {
  instance_id: string;
  state: string; // RUNNING | PENDING | stopped | failed | unknown
  version: number;
  last_seen: string;
  reason?: string;
  stale?: boolean;
}

export interface DeploymentStatus {
  deployment_id: string;
  version: number;
  instances: DeploymentInstanceReport[];
  desired_count: number;
  ready_count: number;
}

export function fetchPipelineDeployments() {
  return request<{ total: number; items: PipelineDeployment[] }>({
    method: 'get',
    url: '/pipeline/deployments'
  });
}

export function createPipelineDeployment(data: Partial<PipelineDeployment>) {
  return request<PipelineDeployment>({
    method: 'post',
    data,
    url: '/pipeline/deployments'
  });
}

export function updatePipelineDeployment(id: string, data: Partial<PipelineDeployment>) {
  return request<PipelineDeployment>({
    method: 'put',
    data,
    url: `/pipeline/deployments/${id}`
  });
}

export function deletePipelineDeployment(id: string) {
  return request<{ deleted: boolean; id: string }>({
    method: 'delete',
    url: `/pipeline/deployments/${id}`
  });
}

export function fetchPipelineDeploymentStatus(id: string) {
  return request<DeploymentStatus>({
    method: 'get',
    url: `/pipeline/deployments/${id}/status`
  });
}
