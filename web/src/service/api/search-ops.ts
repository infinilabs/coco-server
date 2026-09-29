import { request } from '../request';

export interface SearchOpsRow {
  type: string;
  count: number;
  zero_hits: number;
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
  strategies: SearchOpsRow[];
  low_recall: SearchOpsLowMiss[];
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
