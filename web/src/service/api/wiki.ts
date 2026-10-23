/* Wiki (knowledge hub) API layer.
 *
 * Backed by the /wiki/* endpoints (WS10 knowledge-hub design doc §4:
 * crud.RegisterCRUD five-piece + custom endpoints in modules/wiki). The
 * server objects are normalized into the UI types (Api.Wiki.*) — derived
 * display fields (updated_at, members/datasources summaries) get defaults
 * so ported components render unchanged. Generation/edit run over SSE
 * with the progress/article|chunk/done event contract (design doc §5.1).
 */
import { getApiBaseUrl } from '../request';
import { request } from '../request';
import { formatESSearchResult } from '../request/es';
import { localStg } from '@/utils/storage';

/* server envelopes (framework crud conventions):
 *   search  -> ES-shaped {hits:{hits:[{_id,_source}],total}} -> formatESSearchResult
 *   get     -> {found,_id,_source}
 *   create  -> {_id,result:"created"}   update -> {_id,result:"updated"}
 *   toc get -> {found,_id,_source:{kb_id,nodes}}
 *   versions-> {data,total} (already normalized server-side)
 */

const unwrapSource = <T>(res: any): T | null => res?.data?._source ?? null;

const fmtDate = (iso?: string): string => {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toISOString().slice(0, 16).replace('T', ' ');
};

/* ---------------- normalization (server _source -> UI types) ---------------- */

const normalizeKb = (src: any): Api.Wiki.Kb => ({
  _system: src?._system,
  id: src?.id ?? '',
  name: src?.name ?? '',
  description: src?.description ?? '',
  icon: src?.icon || '📚',
  visibility: src?.visibility ?? 'team',
  workspace_id: src?.workspace_id ?? '',
  datasource_ids: src?.datasource_ids ?? [],
  assistant_id: src?.assistant_id,
  sync_strategy: src?.sync_strategy,
  article_count: src?.article_count ?? 0,
  last_updated: fmtDate(src?.updated),
  // summaries are not modeled server-side yet; the UI tolerates empty
  members: [],
  datasources: [],
  ai_status: src?.ai_status,
  auto_approve: !!src?.auto_approve,
  // without this mapping the KB settings switch resets to off after every
  // save+refetch (the payload persists, the normalized view just drops it)
  experience_capture: !!src?.experience_capture,
  toc_layout: src?.toc_layout ?? []
});

const normalizeArticle = (src: any): Api.Wiki.Article => ({
  id: src?.id ?? '',
  kb_id: src?.kb_id ?? '',
  toc_node_id: src?.toc_node_id,
  title: src?.title ?? '',
  summary: src?.summary ?? '',
  content: src?.content ?? '',
  page_type: src?.page_type,
  subtype: src?.subtype,
  aliases: src?.aliases ?? [],
  tags: src?.tags ?? [],
  status: src?.status ?? 'draft',
  ai_generated: Boolean(src?.ai_generated),
  confidence: src?.confidence,
  sources: src?.sources ?? [],
  entity_id: src?.entity_id,
  linked_pages: src?.linked_pages ?? [],
  outline: src?.outline ?? [],
  key_entities: src?.key_entities ?? [],
  maturity: src?.maturity ?? undefined,
  created_by: { id: '', name: src?.created_by ?? '', avatar: 'U', role: 'owner' },
  contributors: [],
  created_at: fmtDate(src?.created),
  updated_at: fmtDate(src?.updated)
});

const normalizeVersion = (src: any): Api.Wiki.Version => ({
  id: src?.id ?? '',
  article_id: src?.article_id ?? '',
  version: src?.version ?? 0,
  change_type: src?.change_type ?? 'human-edited',
  change_summary: src?.change_summary,
  content: src?.content ?? '',
  created_by: src?.created_by ?? '',
  created_at: fmtDate(src?.created)
});

/* ---------------- KBs ---------------- */

export function searchWikiKbs(params?: { query?: string }) {
  const searchParams = new URLSearchParams();
  if (params?.query) searchParams.set('query', params.query);
  const qs = searchParams.toString();
  return request<{ hits: any }>({ method: 'get', url: `/wiki/kb/_search${qs ? `?${qs}` : ''}` }).then(res => {
    const es = formatESSearchResult(res?.data);
    return { data: (es.data || []).map((kb: any) => normalizeKb(kb)), total: es.total };
  });
}

export function getWikiKb(id: string) {
  return request({ method: 'get', url: `/wiki/kb/${id}` }).then(res => {
    const src = unwrapSource<any>(res);
    return src ? normalizeKb(src) : null;
  });
}

export function createWikiKb(body: Partial<Api.Wiki.Kb>) {
  return request<{ _id: string }>({ method: 'post', data: body, url: '/wiki/kb/' }).then(res => res?.data);
}

export function updateWikiKb(id: string, body: Partial<Api.Wiki.Kb>) {
  return request({ method: 'put', data: body, url: `/wiki/kb/${id}` }).then(res => res?.data);
}

export function deleteWikiKb(id: string) {
  return request({ method: 'delete', url: `/wiki/kb/${id}` });
}

/* Knowledge execution: land a chat answer into a KB as a draft article
 * (server keeps the D1 gate — the result is always a draft). */
export function createWikiArticleFromChat(body: {
  kb_id: string;
  title: string;
  content: string;
  summary?: string;
  sources?: { doc_id: string; title?: string; url?: string; excerpt?: string }[];
  message_id?: string;
  session_id?: string;
}) {
  return request<{ _id: string }>({ method: 'post', data: body, url: '/wiki/article/_from_chat' }).then(res => res?.data);
}

/* ---------------- Articles ---------------- */

export function searchWikiArticles(params: { kbId?: string; query?: string; size?: number; sort?: string }) {
  const searchParams = new URLSearchParams();
  if (params.kbId) searchParams.set('filter', `kb_id:${params.kbId}`);
  if (params.query) searchParams.set('query', params.query);
  // crud default page size is 10 — the KB pages render the full set;
  // explicit size (home workbench) overrides the default
  searchParams.set('size', String(params.size ?? 1000));
  if (params.sort) searchParams.set('sort', params.sort);
  return request<{ hits: any }>({
    method: 'get',
    url: `/wiki/article/_search?${searchParams.toString()}`
  }).then(res => {
    const es = formatESSearchResult(res?.data);
    return { data: (es.data || []).map((article: any) => normalizeArticle(article)), total: es.total };
  });
}

export function getWikiArticle(id: string) {
  // a dead article id renders the page's own not-found state — the global
  // error toast on top of it is noise (same for the two sub-fetches below)
  return request({ ignoreError: true, method: 'get', url: `/wiki/article/${id}` }).then(res => {
    const src = unwrapSource<any>(res);
    return src ? normalizeArticle(src) : null;
  });
}

export function createWikiArticle(body: Partial<Api.Wiki.Article>) {
  return request<{ _id: string }>({ method: 'post', data: body, url: '/wiki/article/' }).then(res => res?.data);
}

export function updateWikiArticle(id: string, body: Partial<Api.Wiki.Article>) {
  return request({ method: 'put', data: body, url: `/wiki/article/${id}` }).then(res => res?.data);
}

/** status machine transition: draft → reviewed → published (→ archived) */
export function deleteWikiArticle(id: string) {
  return request({ method: 'delete', url: `/wiki/article/${id}` }).then(res => res?.data);
}

export function updateWikiArticleStatus(id: string, status: Api.Wiki.ArticleStatus) {
  return request({ method: 'put', data: { status }, url: `/wiki/article/${id}/status` }).then(res => res?.data);
}

/* ---------------- Versions ---------------- */

export function getWikiArticleVersions(articleId: string) {
  return request<{ data: any[] }>({
    method: 'get',
    url: `/wiki/article/${articleId}/versions`
  }).then(res => (res?.data?.data || []).map(normalizeVersion));
}

/* ---------------- TOC ---------------- */

/** toc nodes plus the server-derived article_id→page_type map used for tree icons */
export function getWikiToc(kbId: string) {
  return request<{ _source?: { nodes?: Api.Wiki.TocNode[]; page_types?: Record<string, string> } }>({
    method: 'get',
    url: `/wiki/kb/${kbId}/toc`
  }).then(res => {
    // a failed/aborted fetch must NOT wipe the current tree — report null
    // nodes so the caller keeps what it already has
    const nodes = res?.data?._source?.nodes;
    // an EMPTY node list is either a genuinely new KB or a lookup miss —
    // report null either way so the caller never trades a populated tree
    // for a blank one
    const usable = Array.isArray(nodes) && nodes.length > 0;
    return {
      nodes: (usable ? nodes : null) as Api.Wiki.TocNode[] | null,
      pageTypes: res?.data?._source?.page_types ?? {}
    };
  });
}

export function updateWikiToc(kbId: string, toc: Api.Wiki.TocNode[]) {
  return request({ method: 'put', data: toc, url: `/wiki/kb/${kbId}/toc` }).then(() => toc);
}

/* ---------------- bookmarks & notifications ---------------- */

export function searchWikiBookmarks() {
  return request<{ hits: any }>({ method: 'get', url: '/wiki/bookmark/_search' }).then(res => {
    const es = formatESSearchResult(res?.data);
    return { data: es.data || [], total: es.total };
  });
}

export function createWikiBookmark(articleId: string) {
  return request<{ _id: string }>({ method: 'post', data: { article_id: articleId }, url: '/wiki/bookmark/' }).then(
    res => res?.data
  );
}

export function deleteWikiBookmark(id: string) {
  return request({ method: 'delete', url: `/wiki/bookmark/${id}` });
}

/* ---------------- likes ---------------- */

export function searchWikiLikes(articleId?: string) {
  const qs = articleId ? `?filter=article_id:${articleId}` : '';
  return request<{ hits: any }>({ method: 'get', url: `/wiki/like/_search${qs}` }).then(res => {
    const es = formatESSearchResult(res?.data);
    return { data: es.data || [], total: es.total };
  });
}

export function createWikiLike(articleId: string) {
  return request<{ _id: string }>({ method: 'post', data: { article_id: articleId }, url: '/wiki/like/' }).then(
    res => res?.data
  );
}

export function deleteWikiLike(id: string) {
  return request({ method: 'delete', url: `/wiki/like/${id}` });
}

/* ---------------- governance queue ---------------- */

export interface GovernanceProposal {
  id: string;
  kb_id: string;
  article_id: string;
  article_title: string;
  type: string;
  status: string;
  reason: string;
  evidence?: Record<string, any>;
  resolved_by?: string;
  created_at?: string;
}

export function searchWikiGovernance(params?: { status?: string; type?: string; kbId?: string }) {
  const searchParams = new URLSearchParams();
  const filters: string[] = [];
  if (params?.status) filters.push(`status:${params.status}`);
  if (params?.type) filters.push(`type:${params.type}`);
  if (params?.kbId) filters.push(`kb_id:${params.kbId}`);
  if (filters.length) searchParams.set('filter', filters.join(' AND '));
  // the crud default page size is 10 — the review queue paginates
  // client-side, so pull a full window (the badge may say 99+ while the
  // list would otherwise show one page and no pager)
  searchParams.set('size', '1000');
  const qs = searchParams.toString();
  return request<{ hits: any }>({ method: 'get', url: `/wiki/governance/_search${qs ? `?${qs}` : ''}` }).then(res => {
    const es = formatESSearchResult(res?.data);
    return {
      data: (es.data || []).map((p: any) => ({
        id: p.id ?? '',
        kb_id: p.kb_id ?? '',
        article_id: p.article_id ?? '',
        article_title: p.article_title ?? '',
        type: p.type ?? '',
        status: p.status ?? '',
        reason: p.reason ?? '',
        evidence: p.evidence,
        resolved_by: p.resolved_by,
        created_at: fmtDate(p.created)
      })) as GovernanceProposal[],
      total: es.total
    };
  });
}

export function updateWikiGovernanceStatus(id: string, status: string) {
  return request({ method: 'put', data: { status }, url: `/wiki/governance/${id}/status` }).then(res => res?.data);
}

// the queue's closed loop: run the fix the proposal implies (tree the
// orphan, merge the duplicate, rebuild the stale page, ...) — the backend
// retires the proposal as resolved once the fix lands
export function applyWikiGovernanceProposal(id: string) {
  return request({ method: 'post', url: `/wiki/governance/${id}/_apply` }).then(res => res?.data);
}

/* ---------------- ontology vocabulary (phase O1) ---------------- */

export interface OntologyPropertyDef {
  key: string;
  label?: string;
  type: string;
  required?: boolean;
  enum?: string[];
}

export interface OntologyRelationDef {
  name: string;
  label?: string;
  target_type: string;
  cardinality?: string;
  inverse?: string;
}

export interface OntologyPageSection {
  key: string;
  title: string;
}

export interface OntologyEntityTypeDef {
  name: string;
  label?: string;
  icon?: string;
  properties?: OntologyPropertyDef[];
  relations?: OntologyRelationDef[];
  /** per-type wiki page layout: sources/aliases/properties/relations:<name>/custom */
  page_sections?: OntologyPageSection[];
}

/** knowledge compile rules (D6): concept source floor, conflict strategy, source priority */
export interface OntologyCompileRules {
  concept_min_sources?: number;
  conflict_strategy?: string;
  source_priority?: string[];
}

export function getOntologySchema(kbId?: string) {
  const qs = kbId ? `?kb=${kbId}` : '';
  return request<{ _source: any }>({ method: 'get', url: `/wiki/ontology/schema${qs}` }).then(res => {
    const src = res?.data?._source || {};
    return {
      kb_id: src.kb_id || '',
      entity_types: (src.entity_types || []) as OntologyEntityTypeDef[],
      rules: (src.rules || {}) as OntologyCompileRules
    };
  });
}

export function putOntologySchema(entityTypes: OntologyEntityTypeDef[], kbId?: string, rules?: OntologyCompileRules) {
  return request({
    method: 'put',
    data: { kb_id: kbId || '', schema: { entity_types: entityTypes, ...(rules ? { rules } : {}) } },
    url: '/wiki/ontology/schema'
  }).then(res => res?.data);
}

export function relinkWikiArticle(id: string) {
  return request<{ _source: any }>({ method: 'post', url: `/wiki/article/${id}/_relink` }).then(res => {
    const src = res?.data?._source || {};
    return (src.linked_pages || []) as Api.Wiki.LinkedPage[];
  });
}

/** thumbs-up/down capture for the wiki experience layer (LLM-wiki P1):
 * targets resolve through the assistant↔KB binding; always 200 with a
 * count, so callers fire-and-forget */
export function voteAnswerExperience(body: {
  assistant_id: string;
  session_id?: string;
  message_id?: string;
  question?: string;
  answer?: string;
  feedback: 'positive' | 'negative';
}) {
  return request({ data: body, method: 'post', url: '/wiki/experience/_vote' });
}

/** report a wrong/outdated answer into the governance queue (D7) */
export function reportWikiCorrection(body: {
  query?: string;
  answer?: string;
  message_id?: string;
  cited_doc_ids?: string[];
  route_hint: string;
  comment: string;
  kb_id?: string;
}) {
  return request({ data: body, method: 'post', url: '/wiki/governance/_correction' });
}

export function createWikiEntity(body: { name: string; type?: string; subtype?: string; aliases?: string[]; status?: string; properties?: Record<string, unknown>; relations?: { relation: string; target_id: string }[]; kb_id?: string; lenient?: boolean }) {
  return request<{ _id: string }>({ method: 'post', data: body, url: '/wiki/entity/' }).then(res => res?.data);
}

export interface WikiEntityBacklink {
  article_id: string;
  title: string;
  kb_id: string;
  status?: string;
  updated?: string;
  link_type?: string;
  link_name?: string;
  snippet?: string;
}

export function fetchWikiEntityBacklinks(id: string) {
  return request<{ items?: WikiEntityBacklink[]; total?: number }>({ method: 'get', url: `/wiki/entity/${id}/backlinks` }).then(
    res => (res?.data?.items ?? []) as WikiEntityBacklink[]
  );
}

export interface WikiEntityNeighbor {
  entity: Api.Wiki.EntityInfo | null;
  relations_detailed?: Array<{
    target_id: string;
    target_name: string;
    target_type?: string;
    relation: string;
    label?: string;
    inverse?: string;
    inverse_label?: string;
  }>;
  incoming?: Array<{ source_id?: string; source_name?: string; source_type?: string; relation?: string; label?: string; inverse?: string; inverse_label?: string }>;
  neighbors?: Api.Wiki.EntityInfo[];
}

export function fetchWikiEntityNeighbors(id: string) {
  return request<{ entity?: Api.Wiki.EntityInfo; relations_detailed?: WikiEntityNeighbor['relations_detailed']; incoming?: WikiEntityNeighbor['incoming']; neighbors?: Api.Wiki.EntityInfo[] }>({
    method: 'get',
    url: `/wiki/entity/${id}/neighbors`
  }).then(res => (res?.data ?? {}) as WikiEntityNeighbor);
}

export function searchWikiEntityPage(params: { query?: string; type?: string; status?: string; kbId?: string; from?: number; size?: number }) {
  const searchParams = new URLSearchParams();
  if (params.query) searchParams.set('query', params.query);
  const filters: string[] = [];
  if (params.kbId) filters.push(`kb_id:${params.kbId}`);
  if (params.type) filters.push(`type:${params.type}`);
  if (params.status) filters.push(`status:${params.status}`);
  if (filters.length > 0) searchParams.set('filter', filters.join(','));
  searchParams.set('from', String(params.from ?? 0));
  searchParams.set('size', String(params.size ?? 20));
  searchParams.set('sort', 'updated:desc');
  return request<{ hits: any }>({ method: 'get', url: `/wiki/entity/_search?${searchParams.toString()}` }).then(res => {
    const es = formatESSearchResult(res?.data);
    return { data: (es.data || []) as Api.Wiki.EntityInfo[], total: es.total ?? 0 };
  });
}

export function updateWikiEntity(id: string, body: Partial<Api.Wiki.EntityInfo> & { kb_id?: string; properties?: Record<string, unknown> }) {
  return request({ method: 'put', data: body, url: `/wiki/entity/${id}` }).then(res => res?.data);
}

export function getWikiEntity(id: string) {
  return request<{ _id: string; _source: Api.Wiki.EntityInfo | null }>({ method: 'get', url: `/wiki/entity/${id}` }).then(
    res => (res?.data?._source ?? null) as Api.Wiki.EntityInfo | null
  );
}

export function searchWikiNotifications() {
  return request<{ hits: any }>({ method: 'get', url: '/wiki/notification/_search' }).then(res => {
    const es = formatESSearchResult(res?.data);
    return {
      data: (es.data || []).map((n: any) => ({
        id: n?.id ?? '',
        user_id: n?.user_id ?? '',
        target_type: n?.target_type ?? '',
        target_id: n?.target_id ?? '',
        action: n?.action ?? '',
        message: n?.message ?? '',
        read: Boolean(n?.read),
        created_at: fmtDate(n?.created)
      })),
      total: es.total
    };
  });
}

export function markWikiNotificationRead(id: string) {
  return request({ method: 'put', data: { read: true }, url: `/wiki/notification/${id}` }).then(res => res?.data);
}

/* ---------------- entities (knowledge graph) ---------------- */

/** Whole-KB knowledge graph: article/entity nodes, wikilink and typed-relation edges with schema labels (O3). */
export function fetchWikiKbGraph(kbId: string) {
  return request<{ hits: any }>({ method: 'get', url: `/wiki/kb/${kbId}/_graph` }).then(res => {
    const d = (res as any)?.data ?? res ?? {};
    return {
      nodes: ((d as any).nodes || []) as Api.Wiki.GraphNode[],
      edges: ((d as any).edges || []) as Api.Wiki.GraphEdge[],
      entity_types: ((d as any).entity_types || {}) as Record<string, { label?: string; icon?: string }>
    };
  });
}

export function searchWikiEntities(ids?: string[], kbId?: string) {
  const searchParams = new URLSearchParams();
  if (ids?.length) searchParams.set('filter', `id:${ids.join(',')}`);
  else if (kbId) searchParams.set('filter', `kb_id:${kbId}`);
  // crud default page size is 10 — the wikilink autocomplete wants the KB's
  // full entity pool (names + aliases), not the first page
  searchParams.set('size', '1000');
  return request<{ hits: any }>({ method: 'get', url: `/wiki/entity/_search?${searchParams.toString()}` }).then(res => {
    const es = formatESSearchResult(res?.data);
    return ((es.data || []) as any[]).map(e => ({
      id: e?.id ?? '',
      type: e?.type ?? '',
      subtype: e?.subtype,
      name: e?.name ?? '',
      aliases: e?.aliases ?? [],
      status: e?.status,
      article_id: e?.article_id,
      relations: (e?.relations || []) as { target_id: string; relation: string }[]
    })) as Api.Wiki.EntityInfo[];
  });
}

/* ---------------- comments ---------------- */

export function searchWikiComments(articleId: string) {
  const searchParams = new URLSearchParams();
  searchParams.set('filter', `article_id:${articleId}`);
  return request<{ hits: any }>({ method: 'get', url: `/wiki/comment/_search?${searchParams.toString()}` }).then(res => {
    const es = formatESSearchResult(res?.data);
    return ((es.data || []) as any[]).map(c => ({
      id: c?.id ?? '',
      article_id: c?.article_id ?? '',
      user_id: c?.user_id ?? '',
      user_name: c?.user_name ?? '',
      content: c?.content ?? '',
      created_at: fmtDate(c?.created)
    })) as Api.Wiki.Comment[];
  });
}

export function createWikiComment(body: { article_id: string; user_id: string; user_name: string; content: string }) {
  return request({ method: 'post', data: body, url: '/wiki/comment/' }).then(res => res?.data);
}

export function deleteWikiComment(id: string) {
  return request({ method: 'delete', url: `/wiki/comment/${id}` }).then(res => res?.data);
}

/* ---------------- workspaces ---------------- */

export function searchWikiWorkspaces() {
  return request<{ hits: any }>({ method: 'get', url: '/wiki/workspace/_search' }).then(res => {
    const es = formatESSearchResult(res?.data);
    return ((es.data || []) as any[]).map(w => ({ id: w?.id ?? '', name: w?.name ?? '' }));
  });
}

export function createWikiWorkspace(name: string) {
  return request({ method: 'post', data: { name }, url: '/wiki/workspace/' }).then(res => res?.data);
}

/* ---------------- misc ---------------- */

// assistant type & grouping helpers live in the dependency-free
// util (pages in the router's static import graph reuse them without
// dragging the request → store chain into their module init order)
export {
  BUILTIN_KM_ASSISTANT_ID,
  type KbAssistantGroup,
  type KbAssistantOption,
  assistantGroupOf,
  collectSearchBoundAssistantIds,
  groupKbAssistantOptions
} from '@/utils/assistant-groups';
import { type KbAssistantOption } from '@/utils/assistant-groups';

export function listWikiAssistants() {
  return request<{ hits: any }>({ method: 'get', url: '/assistant/_search' }).then(res => {
    const es = formatESSearchResult(res?.data);
    return ((es.data || []) as any[]).map(
      (a: any): KbAssistantOption => ({
        id: a?.id ?? '',
        name: a?.name ?? '',
        type: a?.type
      })
    );
  });
}

export function listWikiDatasources() {
  return request<{ hits: any }>({ method: 'get', url: '/datasource/_search' }).then(res => {
    const es = formatESSearchResult(res?.data);
    return (es.data || []).map(
      (ds: any): Api.Wiki.DatasourceInfo => ({
        id: ds?.id ?? '',
        name: ds?.name ?? '',
        type: ds?.type ?? '',
        status: ds?.enabled === false ? 'disconnected' : 'connected',
        last_synced: fmtDate(ds?.updated),
        // stamped by the backend: live per-datasource document aggregation
        document_count: ds?.document_count ?? 0
      })
    );
  });
}

/* ---------------- KM agent SSE (design doc §5.1 event contract) ---------------- */

export interface WikiGenerateEvent {
  progress?: { phase: string; phaseCurrent: number; phaseTotal: number; progress: number; eta: number };
  article?: Api.Wiki.Article;
  done?: { generated: number; updated?: number; failed?: string[]; reason?: string; usage?: any };
}

export interface WikiEditEvent {
  progress?: { phase: string };
  chunk?: { text: string };
  done?: { _id?: string; version?: number; reason?: string };
}

/** ssePost streams a text/event-stream POST; blocks are reassembled into
 * {event, data} pairs and handed to onEvent. Non-2xx replies reject with
 * the response body so callers can surface server errors. */
async function ssePost(
  url: string,
  body: any,
  onEvent: (event: string, data: any) => void,
  signal?: AbortSignal
): Promise<void> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = localStg.get('token');
  if (token) headers.Authorization = `Bearer ${token}`;
  if (import.meta.env.VITE_SERVICE_TOKEN) headers['X-API-TOKEN'] = import.meta.env.VITE_SERVICE_TOKEN;

  const res = await fetch(`${getApiBaseUrl()}${url}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body ?? {}),
    signal
  });
  if (!res.ok || !res.body) {
    const text = await res.text().catch(() => '');
    let message = `${res.status} ${text}`;
    try {
      // framework error envelope: {message: "..."} — prefer it when present
      const parsed = JSON.parse(text);
      if (parsed?.message) message = parsed.message;
    } catch {
      /* keep raw text */
    }
    throw new Error(message);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder('utf-8');
  let buffer = '';
  const dispatch = (block: string) => {
    let event = '';
    let data = '';
    for (const line of block.split('\n')) {
      if (line.startsWith('event: ')) event = line.slice(7).trim();
      else if (line.startsWith('data: ')) data += line.slice(6);
    }
    if (!event || !data) return;
    try {
      onEvent(event, JSON.parse(data));
    } catch {
      onEvent(event, data);
    }
  };
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    let sep;
    while ((sep = buffer.indexOf('\n\n')) >= 0) {
      dispatch(buffer.slice(0, sep));
      buffer = buffer.slice(sep + 2);
    }
  }
  if (buffer.trim()) dispatch(buffer);
}

/** KM agent generation: POST /wiki/kb/:id/ai/generate (SSE). */
export function generateWikiKb(
  kbId: string,
  body: { hint?: string; max_pages?: number },
  onEvent: (event: WikiGenerateEvent) => void,
  signal?: AbortSignal
) {
  return ssePost(
    `/wiki/kb/${kbId}/ai/generate`,
    body,
    (event, data) => {
      if (event === 'progress') onEvent({ progress: data });
      else if (event === 'article') onEvent({ article: normalizeArticle(data) });
      else if (event === 'done') onEvent({ done: data });
    },
    signal
  );
}

/** instruction-based AI edit: POST /wiki/article/:id/ai/edit (SSE). */
export function editWikiArticleAI(
  articleId: string,
  body: { instruction: string; selection?: string },
  onEvent: (event: WikiEditEvent) => void,
  signal?: AbortSignal
) {
  return ssePost(
    `/wiki/article/${articleId}/ai/edit`,
    body,
    (event, data) => {
      if (event === 'progress') onEvent({ progress: data });
      else if (event === 'chunk') onEvent({ chunk: data });
      else if (event === 'done') onEvent({ done: data });
    },
    signal
  );
}

export function getWikiArticleMaterials(id: string) {
  return request<{ sources: Api.Wiki.ArticleMaterial[] }>({
    ignoreError: true,
    method: 'get',
    url: `/wiki/article/${id}/_materials`
  }).then(res => (res as any)?.data?.sources ?? []);
}

/* ---------------- page-level attachments ---------------- */

export function getWikiArticleAttachments(id: string) {
  return request<{ attachments: Api.Wiki.WikiArticleAttachment[] }>({
    ignoreError: true,
    method: 'get',
    url: `/wiki/article/${id}/_attachments`
  }).then(res => (res as any)?.data?.attachments ?? []);
}

/** replaces the page's attachment id list wholesale (upload-append and
 * row-remove both render as a new array client-side) */
export function updateWikiArticleAttachments(id: string, attachmentIds: string[]) {
  return request<{ attachments: Api.Wiki.WikiArticleAttachment[]; dropped: number }>({
    method: 'put',
    data: { attachment_ids: attachmentIds },
    url: `/wiki/article/${id}/_attachments`
  }).then(res => (res as any)?.data);
}

/** batch review: resolve/dismiss N proposals in one call; the backend
 * applies the same transition rules per id and counts refusals as skipped */
export function batchUpdateWikiGovernanceStatus(ids: string[], status: string) {
  return request<{ updated: number; unchanged: number; skipped: number }>({
    method: 'put',
    data: { ids, status },
    url: '/wiki/governance/_batch_status'
  }).then(res => (res as any)?.data);
}

/** batch apply: kicks the background fix worker and returns immediately;
 * poll getBatchApplyProgress for {status,total,done,failed,skipped,current} */
export function startBatchApply(ids: string[]) {
  return request<{ started: boolean; total: number }>({
    method: 'post',
    data: { ids },
    url: '/wiki/governance/_batch_apply'
  }).then(res => (res as any)?.data);
}

export interface BatchApplyProgress {
  status: 'idle' | 'running' | 'done';
  total: number;
  done: number;
  failed: number;
  skipped: number;
  current?: string;
  errors?: string[];
}

export function getBatchApplyProgress() {
  return request<BatchApplyProgress>({
    method: 'get',
    url: '/wiki/governance/_batch_apply'
  }).then(res => (res as any)?.data);
}

/** batch entity review: set status for N entities in one call (reviewed/
 * published/retired — retired being the design's "delete" for entities) */
export function batchUpdateWikiEntityStatus(ids: string[], status: string) {
  return request<{ updated: number; skipped: number }>({
    method: 'put',
    data: { ids, status },
    url: '/wiki/entity/_batch_status'
  }).then(res => (res as any)?.data);
}

/* ---------------- KB-level files (raw material) ---------------- */

export function getWikiKbFiles(kbId: string) {
  return request<{ attachments: any[]; upload_source_id: string }>({
    method: 'get',
    url: `/wiki/kb/${kbId}/_files`
  }).then(res => (res as any)?.data);
}

/** manual uploads: files land as documents in the KB's upload datasource
 * and enter the standard enrichment pipeline */
export function uploadWikiKbFiles(kbId: string, files: any[]) {
  const fd = new FormData();
  for (const f of files) fd.append('files', f, f.name);
  return request<{ created: number; failed: { file: string; error: string }[] }>({
    method: 'post',
    data: fd,
    url: `/wiki/kb/${kbId}/_files/upload`
  }).then(res => (res as any)?.data);
}

/* ---------------- compile console ---------------- */

export interface WikiCompileJob {
  status: 'idle' | 'running' | 'done' | 'failed' | 'cancelled';
  stage?: string;
  docs?: number;
  batches?: number;
  batches_done?: number;
  batches_cached?: number;
  batches_failed?: number;
  extracts?: number;
  reduced?: number;
  topics?: number;
  topics_done?: number;
  proposals?: number;
  current?: string;
  logs?: string[];
  started_at?: string;
}

export function startWikiCompile(kbId: string) {
  return request<{ result: string }>({
    method: 'post',
    url: `/wiki/kb/${kbId}/_compile`
  }).then(res => (res as any)?.data);
}

export function getWikiCompileJob(kbId: string) {
  return request<WikiCompileJob>({
    method: 'get',
    url: `/wiki/kb/${kbId}/_compile`
  }).then(res => (res as any)?.data);
}

export function cancelWikiCompile(kbId: string) {
  return request<{ cancelled: boolean }>({
    method: 'post',
    url: `/wiki/kb/${kbId}/_compile/_cancel`
  }).then(res => (res as any)?.data);
}

/** merge N same-concept entities into one canonical row: absorbed names
 * become aliases, sources/relations union, references retarget */
export function mergeWikiEntities(ids: string[]) {
  return request<{ merged: number; canonical: string; name: string }>({
    method: 'post',
    data: { ids },
    url: '/wiki/entity/_merge'
  }).then(res => (res as any)?.data);
}

/* ---- LLM-wiki P1 loop surfaces (experience → distill → governance → evolution log) ---- */

/** append-only KB decision log: applied/dismissed proposals, distill runs,
 * compile-agent iterations — humans and auto-approve alike */
export function getWikiEvolutionLog(kbId: string, from = 0, size = 50) {
  return request<{ total: number; entries: any[] }>({
    method: 'get',
    url: `/wiki/kb/${kbId}/_evolution_log?from=${from}&size=${size}`
  }).then(res => {
    const body = (res as any)?.data;
    return { total: body?.total ?? 0, entries: (body?.entries ?? []) as Api.Wiki.EvolutionEntry[] };
  });
}

/** raw work signals captured for this KB (chat votes, corrections, gaps) */
export function listWikiExperience(kbId: string, from = 0, size = 50) {
  return request<{ total: number; entries: any[] }>({
    method: 'get',
    url: `/wiki/kb/${kbId}/_experience?from=${from}&size=${size}`
  }).then(res => {
    const body = (res as any)?.data;
    return { total: body?.total ?? 0, entries: (body?.entries ?? []) as Api.Wiki.Experience[] };
  });
}

/** mine this KB's experiences into pattern proposals (governance queue) */
export function distillWikiKb(kbId: string) {
  return request<{ sampled: number; proposals: number; expired: number }>({
    method: 'post',
    url: `/wiki/kb/${kbId}/_distill`
  }).then(res => (res as any)?.data);
}

/** run one on-demand pass of the EINO compile agent (files at most one
 * compile_patch proposal per iteration); synchronous, LLM-slow */
export function runWikiCompileAgent(kbId: string) {
  return request<{ iterations: number; patches: number; no_action: boolean; notes?: string[] }>({
    method: 'post',
    url: `/wiki/kb/${kbId}/_agent_compile`
  }).then(res => (res as any)?.data);
}

/** rewrite an article's prose from its own source claims (REFINE pass);
 * keeps a version snapshot, returns rewritten:false when no LM is set */
export function rewriteWikiArticle(id: string) {
  return request<{ rewritten: boolean; reason?: string }>({
    method: 'post',
    url: `/wiki/article/${id}/_rewrite`
  }).then(res => (res as any)?.data);
}

/** one-off projection of published wiki articles into the document index —
 * the preparation step for the wiki_projection search branch */
export function backfillWikiProjection() {
  return request<{ total: number; projected: number; skipped: number }>({
    method: 'post',
    url: '/wiki/_backfill_projection'
  }).then(res => (res as any)?.data);
}

/* ---- SKILL.md interoperation (export / import / manual proposer) ---- */

/** export a skill page as an Agent Skills SKILL.md document (markdown text) */
export function exportWikiSkillMd(id: string) {
  return request<string>({
    method: 'get',
    url: `/wiki/article/${id}/_skill_md`
  }).then(res => ((res as any)?.data ?? '') as string);
}

/** import a pasted SKILL.md as a draft skill page; a same-title page
 * reports { result: 'exists' } instead of stacking a twin */
export function importWikiSkill(kbId: string, content: string) {
  return request<{ _id?: string; id?: string; result: string }>({
    method: 'post',
    url: `/wiki/kb/${kbId}/_import_skill`,
    data: { content }
  }).then(res => (res as any)?.data);
}

/** manually run the skill proposer over the KB's recent experiences;
 * outcomes land in the evolution log (skill_proposed entries) */
export function proposeWikiSkills(kbId: string) {
  return request<Api.Wiki.SkillRun>({
    method: 'post',
    url: `/wiki/kb/${kbId}/_propose`
  }).then(res => (res as any)?.data);
}

/** the KB's company map — the published page_type=map entry page (what we
 * do, priorities, source priority, navigation); found:false when none */
export function getWikiCompanyMap(kbId: string) {
  return request<{
    found: boolean;
    id?: string;
    map?: { kb_id: string; title: string; summary?: string; content?: string; updated?: string; url?: string };
  }>({
    method: 'get',
    url: `/wiki/company-map?kb=${kbId}`
  }).then(res => (res as any)?.data ?? { found: false });
}

/** re-organize draft pages against whole-KB context: fold thin pages into
 * their dimension overview, merge near-duplicate titles; published pages
 * are never touched */
export function consolidateWikiKb(kbId: string) {
  return request<{
    result: string;
    stats: { scanned: number; merged: number; deduped: number; kept: number; merged_into?: string[]; removed_ids?: string[] };
  }>({
    method: 'post',
    url: `/wiki/kb/${kbId}/_consolidate`
  }).then(res => (res as any)?.data);
}
