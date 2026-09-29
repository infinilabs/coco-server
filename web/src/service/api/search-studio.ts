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

export interface SearchStudioResult {
  query: string;
  size: number;
  fuzziness: number;
  rrf: SearchStudioRRFConfig;
  routes: SearchStudioRouteResult[];
  rerank?: SearchStudioRerank;
  fused: {
    took_ms: number;
    total: number;
    hits: SearchStudioFusedHit[];
  };
}

/** run every recall route (BM25 / semantic / wiki / graph) and return the RRF fusion breakdown */
export function testSearchStudio(data: {
  query: string;
  datasource?: string;
  category?: string;
  subcategory?: string;
  rich_category?: string;
  fuzziness?: number;
  size?: number;
  rrf?: Partial<{ k: number; text_weight: number; semantic_weight: number; wiki_weight: number; graph_weight: number }>;
}) {
  return request({
    data,
    method: 'post',
    url: '/search/studio/test'
  });
}
