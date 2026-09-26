import { request } from '../request';

export interface SearchStudioRRFConfig {
  k: number;
  text_weight: number;
  semantic_weight: number;
}

export interface SearchStudioRouteHit {
  id: string;
  title: string;
  datasource: string;
  rank: number;
  score: number;
}

export interface SearchStudioRouteResult {
  took_ms: number;
  total: number;
  hits: SearchStudioRouteHit[];
  error?: string;
}

export interface SearchStudioFusedHit {
  id: string;
  title: string;
  datasource: string;
  text_rank: number;
  semantic_rank: number;
  text_score: number;
  semantic_score: number;
  text_contribution: number;
  semantic_contribution: number;
  score: number;
}

export interface SearchStudioResult {
  query: string;
  size: number;
  fuzziness: number;
  rrf: SearchStudioRRFConfig;
  text: SearchStudioRouteResult;
  semantic: SearchStudioRouteResult;
  fused: {
    took_ms: number;
    total: number;
    hits: SearchStudioFusedHit[];
  };
}

/** run the BM25 and kNN recall routes and return the RRF fusion breakdown */
export function testSearchStudio(data: {
  query: string;
  datasource?: string;
  category?: string;
  subcategory?: string;
  rich_category?: string;
  fuzziness?: number;
  size?: number;
  rrf?: Partial<SearchStudioRRFConfig>;
}) {
  return request({
    data,
    method: 'post',
    url: '/search/studio/test'
  });
}
