import { request } from '../request';

/**
 * RSA (Remote Source Adapter) — federated query-time sources
 * (DESIGN-RSA.md). A source is queried directly on every hybrid search
 * (as one extra RRF leg) and hits are mapped into standard documents;
 * nothing is copied into the local index. Credentials read back masked
 * ("******") and round-trip unchanged — leave the masked value in place
 * when saving.
 */

export interface RsaConnection {
  endpoints: string[];
  auth?: string; // '' | basic | api_key
  username?: string;
  password?: string;
  api_key?: string;
  verify_tls?: boolean;
}

export interface RsaQueryConfig {
  indices?: string;
  query_template?: string;
}

export interface RsaMapping {
  title?: string;
  content?: string;
  url?: string;
  created?: string;
  tags?: string;
  id_path?: string;
}

export interface RsaControls {
  timeout_ms?: number;
  max_concurrent?: number;
  qps?: number;
  cache_ttl_seconds?: number;
  max_window?: number;
  retry_max_attempts?: number;
  retry_backoff_ms?: number;
  retry_on_timeout?: boolean;
}

export interface RemoteSource {
  id?: string;
  name: string;
  description?: string;
  adapter: string;
  enabled?: boolean;
  priority?: number;
  connection: RsaConnection;
  query?: RsaQueryConfig;
  mapping?: RsaMapping;
  controls?: RsaControls;
}

export interface RsaAttempt {
  endpoint: string;
  err?: string;
  final?: boolean;
  took_ms: number;
}

export interface RsaSourceDiag {
  source_id: string;
  source_name: string;
  adapter: string;
  status: string; // ok | timeout | error | cache
  hits: number;
  took_ms: number;
  cache_hit: boolean;
  attempts?: RsaAttempt[];
  err?: string;
}

export interface RsaTestResult {
  status: string;
  took_ms: number;
  hits: number;
  diag: RsaSourceDiag;
  error?: string;
  sample?: { title?: string; url?: string; snippet?: string };
}

export interface RsaDebugResult {
  status: string;
  took_ms: number;
  cache_hit: boolean;
  native: { adapter: string; es_search_body?: unknown; es_search_url?: string; render_error?: string };
  diag: RsaSourceDiag;
  error?: string;
  total?: number;
  mapped_hits?: Array<{ rank: number; doc_id: string; title?: string; url?: string; content?: string; raw_id?: string; index?: string }>;
}

export interface RsaSourceStats {
  source_id: string;
  requests: number;
  hits: number;
  timeouts: number;
  errors: number;
  cache_hits: number;
  attempts: number;
  failovers: number;
  latency_p50_ms: number;
  latency_p95_ms: number;
  latency_max_ms: number;
  last_error?: string;
  last_at?: string;
}

export function fetchRemoteSources() {
  return request<{ total: number; hits: RemoteSource[] }>({
    method: 'get',
    url: '/rsa/_search?from=0&size=100'
  }).then(res => (res as any)?.data ?? { total: 0, hits: [] });
}

export function fetchRemoteSource(id: string) {
  return request<{ _id: string; found: boolean; _source?: RemoteSource }>({
    method: 'get',
    url: `/rsa/${id}`
  }).then(res => (res as any)?.data);
}

export function createRemoteSource(data: RemoteSource) {
  return request<{ _id: string; result: string }>({
    method: 'post',
    url: '/rsa/',
    data
  }).then(res => (res as any)?.data);
}

export function updateRemoteSource(id: string, data: RemoteSource) {
  return request<{ _id: string; result: string }>({
    method: 'put',
    url: `/rsa/${id}`,
    data
  }).then(res => (res as any)?.data);
}

export function deleteRemoteSource(id: string) {
  return request<{ _id: string; result: string }>({
    method: 'delete',
    url: `/rsa/${id}`
  }).then(res => (res as any)?.data);
}

/** one tiny live query — proves connectivity, auth and mapping in a single
 * round trip; empty query runs a match-anything "*" probe */
export function testRemoteSource(id: string, query: string) {
  return request<RsaTestResult>({
    method: 'post',
    url: `/rsa/${id}/_test`,
    data: { query }
  }).then(res => (res as any)?.data);
}

/** the dry-run: renders the exact native DSL, runs it once (cache
 * bypassed), and returns the raw→document mapping side by side */
export function debugRemoteSource(data: { source_id: string; query: string; from?: number; size?: number }) {
  return request<RsaDebugResult>({
    method: 'post',
    url: '/rsa/_debug',
    data
  }).then(res => (res as any)?.data);
}

/** live per-source counters since process start */
export function fetchRsaStats() {
  return request<{ stats: RsaSourceStats[] }>({
    method: 'get',
    url: '/rsa/_stats'
  }).then(res => (res as any)?.data?.stats ?? []);
}
