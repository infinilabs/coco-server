import { request } from '../request';

export interface SearchStudioRRFConfig {
  k: number;
  weights: Record<string, number>;
}

export interface SearchStudioRouteHit {
  id: string;
  title: string;
  datasource: string;
  rank: number;
  score: number;
}

export interface SearchStudioRouteResult {
  name: string;
  took_ms: number;
  total: number;
  hits: SearchStudioRouteHit[];
  /** semantic only: engine | client | skipped */
  route?: string;
  note?: string;
  error?: string;
}

export interface SearchStudioFusedHit {
  id: string;
  title: string;
  datasource: string;
  ranks: Record<string, number>;
  route_scores: Record<string, number>;
  contributions: Record<string, number>;
  score: number;
}

export interface SearchStudioRerankRow {
  id: string;
  title: string;
  rrf_rank: number;
  rerank_rank: number;
  delta: number;
  relevance_score: number;
}

export interface SearchStudioRerank {
  applied: boolean;
  model?: string;
  note?: string;
  took_ms?: number;
  hits?: SearchStudioRerankRow[];
}

/** D8 query-rewrite verdict: what the model rewrote the query into (or why not) */
export interface SearchStudioRewrite {
  applied: boolean;
  query: string;
  cached?: boolean;
  took_ms?: number;
  note?: string;
}

export interface SearchStudioResult {
  query: string;
  size: number;
  fuzziness: number;
  rrf: SearchStudioRRFConfig;
  rewrite?: SearchStudioRewrite;
  routes: SearchStudioRouteResult[];
  rerank?: SearchStudioRerank;
  fused: {
    took_ms: number;
    total: number;
    hits: SearchStudioFusedHit[];
  };
}

/** run every recall route (BM25 / semantic / wiki / graph / rewrite) and return the RRF fusion breakdown */
export function testSearchStudio(data: {
  query: string;
  datasource?: string;
  category?: string;
  subcategory?: string;
  rich_category?: string;
  fuzziness?: number;
  size?: number;
  rrf?: Partial<{ k: number; text_weight: number; semantic_weight: number; wiki_weight: number; graph_weight: number; rewrite_weight: number }>;
}) {
  return request({
    data,
    method: 'post',
    url: '/search/studio/test'
  });
}

// --- golden-query evaluation set (D9) ---

export interface SearchEvalCase {
  id: string;
  query: string;
  expected_ids: string[];
  expected_titles?: string[];
  datasource?: string;
  note?: string;
  created?: string;
}

export interface SearchEvalCaseResult {
  query: string;
  hit_rank: number;
  total: number;
  took_ms: number;
  top_ids?: string[];
  top_titles?: string[];
  error?: string;
}

export interface SearchEvalRun {
  id: string;
  total_cases: number;
  top4_hits: number;
  top4_rate: number;
  mrr: number;
  avg_took_ms: number;
  cases?: SearchEvalCaseResult[];
  created?: string;
}

export function fetchEvalCases() {
  return request<{ total: number; data: SearchEvalCase[] }>({
    method: 'get',
    url: '/search/studio/eval/_cases'
  });
}

/** upsert keyed on the normalized query — re-annotating replaces the expectation */
export function createEvalCase(data: {
  query: string;
  expected_ids: string[];
  expected_titles?: string[];
  datasource?: string;
  note?: string;
}) {
  return request<SearchEvalCase>({
    data,
    method: 'post',
    url: '/search/studio/eval/_cases'
  });
}

export function deleteEvalCase(id: string) {
  return request({
    method: 'delete',
    url: `/search/studio/eval/_cases/${id}`
  });
}

/** run the whole set against the live pipeline; returns the stored run with per-case outcomes */
export function runSearchEval(data?: { top_n?: number }) {
  return request<{ run: SearchEvalRun; took_ms: number; top4_rate: number; mrr: number }>({
    data: data ?? {},
    method: 'post',
    url: '/search/studio/eval/_run'
  });
}

export function fetchEvalRuns() {
  return request<{ total: number; data: SearchEvalRun[] }>({
    method: 'get',
    url: '/search/studio/eval/_runs'
  });
}
