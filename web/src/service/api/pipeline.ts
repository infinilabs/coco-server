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

/** fleet view: deployments × instances matrix (framework pipeline module) */
export function fetchClusterOverview() {
  return request({
    method: 'get',
    url: '/pipeline/cluster/overview'
  });
}

/** cluster-wide per-stage totals; optional pipeline filter */
export function fetchClusterStages(pipeline?: string) {
  return request({
    method: 'get',
    params: pipeline ? { pipeline } : undefined,
    url: '/pipeline/cluster/stages'
  });
}

// shared types + client from the ui-common package (source reference);
// the product binds its own axios-backed request instance
import { createPipelineClient, type DryRunReport, type StepResult } from '@infinilabs/pipeline-ui/api';

const pipelineClient = createPipelineClient({
  // the product adapter boundary: coco's request returns FlatResponseData
  // (data + response); the shared client only consumes .data
  post: <T,>(url: string, data: unknown) =>
    request({ method: 'post', data, url }) as unknown as Promise<{ data: T }>,
  get: <T,>(url: string) => request({ method: 'get', url }) as unknown as Promise<{ data: T }>
});

/**
 * Step-by-step dry run of a processor chain over sample documents via the
 * unified framework endpoint. Documents travel through the coco_document
 * record codec (the record convention the pipelines run under); the report
 * is mapped back into the studio's steps/finals shape.
 */
export async function testPipelineChain(data: { processor: Record<string, any>[]; documents: Record<string, any>[] }) {
  const report = await pipelineClient.dryRun(
    data.processor,
    data.documents.map((doc) => ({ raw: JSON.stringify(doc), codec: 'coco_document' }))
  );
  // report = the raw unified payload (the shared DryRunDrawer renders it
  // directly); steps/finals remain as the mapped legacy shape.
  return { data: { report, ...mapDryRunReport(report) } };
}

/** map the unified dry-run report onto the studio's steps/finals shape */
function mapDryRunReport(report: DryRunReport | undefined) {
  if (!report) return { steps: undefined, finals: undefined };
  const badCompile = (report.compiled || []).find((c) => c && c.ok === false);
  if (badCompile) {
    throw new Error(`chain invalid: ${badCompile.note ?? badCompile.processor}`);
  }
  const steps = (report.steps || []).map((perDoc: StepResult[]) =>
    (perDoc || []).map((step) => ({
      name: step?.processor,
      fields: step?.after,
      count: step?.outcome === 'dropped' ? 0 : 1,
      error: step?.err || (step?.outcome === 'error' ? 'error' : undefined)
    }))
  );
  const finals = (report.steps || []).map((perDoc: StepResult[]) => {
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
