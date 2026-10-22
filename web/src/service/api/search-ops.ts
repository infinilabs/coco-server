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
  total_searches_prev?: number;
  zero_hit_searches: number;
  zero_hit_rate: number;
  avg_took_ms: number;
  max_took_ms: number;
  /** latency percentiles over the overview scan window */
  latency_p50_ms?: number;
  latency_p95_ms?: number;
  latency_p99_ms?: number;
  /** D8: searches where the query-rewrite leg fired, and how many still missed */
  rewritten_searches: number;
  rewritten_zero_hit_searches: number;
  strategies: SearchOpsRow[];
  low_recall: SearchOpsLowMiss[];
  /** standing annotation qualifying the zero-hit family; empty when unqualified */
  signals_caveat?: string;
}

/** one completed search, the raw drill-down row behind the overview aggregates */
export interface SearchLogEntry {
  id: string;
  /** stored lowercased/trimmed/whitespace-collapsed — the aggregation/filter key */
  query: string;
  /** the user's input as typed (rows before this field existed have none) */
  raw?: string;
  search_type: string;
  total: number;
  took_ms: number;
  zero_hit: boolean;
  rewritten: boolean;
  user_id?: string;
  created?: string;
  /** diagnostics: per-leg timings, engine-side took, top hit and failure text */
  leg_keyword_ms?: number;
  leg_semantic_ms?: number;
  leg_rewrite_ms?: number;
  engine_took_ms?: number;
  top_hit?: string;
  top_score?: number;
  /** set when the search itself failed; failed rows have zero_hit=false by construction */
  error?: string;
}

/** raw query log with optional AND-combined filters */
export function fetchSearchLogs(params?: { q?: string; search_type?: string; zero_hit?: boolean; min_took_ms?: number; size?: number }) {
  return request<{ items: SearchLogEntry[]; total: number }>({
    method: 'get',
    url: '/search/ops/search-logs',
    params
  });
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
export function fetchSearchOpsOverview(params?: { days?: number }) {
  return request<SearchOpsOverview>({
    method: 'get',
    url: '/search/ops/overview',
    params
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

/** real-time ingestion signals: the oldest in-flight document.
 * NOTE: backend `live.queues` (disk_queue Depth) is deliberately not typed here —
 * in consumer mode it is a cumulative write counter, not a backlog (misleading) */
export interface ProcessingLive {
  /** `updated` stamp of the earliest status=indexing document; empty when idle. A stale timestamp means a stuck pipeline */
  oldest_indexing?: string;
}

export interface ProcessingOverview {
  statuses: Record<string, number>;
  failed: ProcessingFailedDoc[];
  failed_top: number;
  live?: ProcessingLive;
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
