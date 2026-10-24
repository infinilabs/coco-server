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

/**
 * Step-by-step dry run of a processor chain over sample documents via the
 * unified framework endpoint. Documents travel through the coco_document
 * record codec (the record convention the pipelines run under); the report
 * is mapped back into the studio's steps/finals shape.
 */
export async function testPipelineChain(data: { processor: Record<string, any>[]; documents: Record<string, any>[] }) {
  const res = await request({
    method: 'post',
    data: {
      processor: data.processor,
      samples: data.documents.map((doc) => ({ raw: JSON.stringify(doc), codec: 'coco_document' }))
    },
    url: '/pipeline/dry-run'
  });
  return { data: mapDryRunReport(res?.data) };
}

/** map the unified dry-run report onto the studio's steps/finals shape */
function mapDryRunReport(report: any) {
  if (!report) return { steps: undefined, finals: undefined };
  const badCompile = (report.compiled || []).find((c: any) => c && c.ok === false);
  if (badCompile) {
    throw new Error(`chain invalid: ${badCompile.note ?? badCompile.processor}`);
  }
  const steps = (report.steps || []).map((perDoc: any[]) =>
    (perDoc || []).map((step) => ({
      name: step?.processor,
      fields: step?.after,
      count: step?.outcome === 'dropped' ? 0 : 1,
      error: step?.err || (step?.outcome === 'error' ? 'error' : undefined)
    }))
  );
  const finals = (report.steps || []).map((perDoc: any[]) => {
    const last = (perDoc || [])[perDoc?.length - 1];
    return last?.after;
  });
  return { steps, finals };
}

export function aiGeneratePipelineChain(data: {
  documents: string[];
  requirements?: string;
  mode: 'generate' | 'refine';
  current_chain?: Record<string, any>[];
}) {
  return request({
    method: 'post',
    data,
    url: '/pipeline/ai-generate'
  });
}
