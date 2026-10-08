import { request } from '../request';

export interface SearchOpsRow {
  type: string;
  count: number;
  zero_hits: number;
  rewritten: number;
  avg_took_ms: number;
}

export interface SearchOpsLowMiss {
  query: string;
  count: number;
  last_seen: string;
  proposal: boolean;
}

export interface SearchOpsOverview {
  total_searches: number;
  zero_hit_searches: number;
  zero_hit_rate: number;
  avg_took_ms: number;
  max_took_ms: number;
  /** D8: searches where the query-rewrite leg fired, and how many still missed */
  rewritten_searches: number;
  rewritten_zero_hit_searches: number;
  strategies: SearchOpsRow[];
  low_recall: SearchOpsLowMiss[];
  /** standing annotation qualifying the zero-hit family; empty when unqualified */
  signals_caveat?: string;
}

export interface IndexHealthEntry {
  name: string;
  model: string;
  status: string;
  docs: number;
  note?: string;
}

/** one size-0 count per knowledge-hub store: is every layer there and how much is in it */
export function fetchIndexHealth() {
  return request<{ healthy: number; total: number; indices: IndexHealthEntry[]; checked_at: number }>({
    method: 'get',
    url: '/search/ops/index-health'
  });
}

/** aggregated search telemetry: strategy distribution, low-recall board, latency */
export function fetchSearchOpsOverview() {
  return request<SearchOpsOverview>({
    method: 'get',
    url: '/search/ops/overview'
  });
}

/** W13a task-management: ingestion processing lifecycle distribution + the failed list */
export interface ProcessingFailedDoc {
  id: string;
  title: string;
  datasource?: string;
  error?: string;
  updated?: string;
  reprocessurl?: string;
}

export interface ProcessingOverview {
  statuses: Record<string, number>;
  failed: ProcessingFailedDoc[];
  failed_top: number;
}

export function fetchProcessingOverview(datasource?: string) {
  return request<ProcessingOverview>({
    method: 'get',
    url: '/search/ops/processing',
    params: datasource ? { datasource } : undefined
  });
}

/** retry every failed document (optionally scoped to one datasource) */
export function retryFailedDocs(datasource?: string) {
  return request<{ retried: number; still_failed: number; failed_seen: number; truncated: boolean }>({
    method: 'post',
    url: '/document/_retry_failed',
    data: datasource ? { datasource } : undefined
  });
}

/** W8: per-datasource ingestion state — sync config, dispatcher tick, incremental cursor, run journal */
export interface SyncRunState {
  run_id: string;
  state: string;
  started_at: number;
  updated_at: number;
  batches: number;
  documents: number;
  stale: boolean;
}

export interface SyncStatusItem {
  id: string;
  name: string;
  type?: string;
  enabled: boolean;
  sync_enabled?: boolean;
  sync_interval?: string;
  last_dispatch?: string;
  increment_cursor?: string;
  run?: SyncRunState;
}

export function fetchSyncStatus() {
  return request<{ items: SyncStatusItem[]; total: number }>({
    method: 'get',
    url: '/datasource/_sync_status'
  });
}
