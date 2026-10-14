import { request } from '../request';

export function getEnablePipelines(params?: any) {
  return request({
    method: 'post',
    params,
    url: '/pipelines/_search?size=1000&filter=enabled:any(true)'
  });
}

export function fetchPipelineList(params?: any) {
  return request({
    method: 'post',
    params,
    url: '/pipelines/_search'
  });
}

export function getPipeline(id: string) {
  return request({
    method: 'get',
    url: `/pipelines/${id}`
  });
}

export function createPipeline(data?: any) {
  return request({
    method: 'post',
    data,
    url: '/pipelines/'
  });
}

export function updatePipeline(id: string, data?: any) {
  return request({
    method: 'put',
    data,
    url: `/pipelines/${id}`
  });
}

export function deletePipeline(id: string) {
  return request({
    method: 'delete',
    url: `/pipelines/${id}`
  });
}

/** flat registry: {name: {properties, category}} */
export function getPipelineProcessors() {
  return request({
    method: 'get',
    url: '/pipeline/processors'
  });
}

/** step-by-step dry run of a processor chain over sample documents */
export function testPipelineChain(data: { processor: Record<string, any>[]; documents: Record<string, any>[] }) {
  return request({
    method: 'post',
    data,
    url: '/pipeline/studio/test'
  });
}

export function aiGeneratePipelineChain(data: {
  documents: string[];
  requirements?: string;
  mode: 'generate' | 'refine';
  current_chain?: Record<string, any>[];
  auto_test?: boolean;
}) {
  return request({
    method: 'post',
    data,
    url: '/pipeline/studio/ai-generate'
  });
}
