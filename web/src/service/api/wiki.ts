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
  ai_status: src?.ai_status
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

export function searchWikiArticles(params: { kbId?: string; query?: string }) {
  const searchParams = new URLSearchParams();
  if (params.kbId) searchParams.set('filter', `kb_id:${params.kbId}`);
  if (params.query) searchParams.set('query', params.query);
  return request<{ hits: any }>({
    method: 'get',
    url: `/wiki/article/_search?${searchParams.toString()}`
  }).then(res => {
    const es = formatESSearchResult(res?.data);
    return { data: (es.data || []).map((article: any) => normalizeArticle(article)), total: es.total };
  });
}

export function getWikiArticle(id: string) {
  return request({ method: 'get', url: `/wiki/article/${id}` }).then(res => {
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

export function getWikiToc(kbId: string) {
  return request<{ _source?: { nodes?: Api.Wiki.TocNode[] } }>({
    method: 'get',
    url: `/wiki/kb/${kbId}/toc`
  }).then(res => res?.data?._source?.nodes ?? []);
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

export interface OntologyEntityTypeDef {
  name: string;
  label?: string;
  icon?: string;
  properties?: OntologyPropertyDef[];
  relations?: OntologyRelationDef[];
}

export function getOntologySchema(kbId?: string) {
  const qs = kbId ? `?kb=${kbId}` : '';
  return request<{ _source: any }>({ method: 'get', url: `/wiki/ontology/schema${qs}` }).then(res => {
    const src = res?.data?._source || {};
    return { kb_id: src.kb_id || '', entity_types: (src.entity_types || []) as OntologyEntityTypeDef[] };
  });
}

export function putOntologySchema(entityTypes: OntologyEntityTypeDef[], kbId?: string) {
  return request({ method: 'put', data: { kb_id: kbId || '', schema: { entity_types: entityTypes } }, url: '/wiki/ontology/schema' }).then(
    res => res?.data
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

export function searchWikiEntities(ids?: string[]) {
  const searchParams = new URLSearchParams();
  if (ids?.length) searchParams.set('filter', `id:${ids.join(',')}`);
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

export function listWikiAssistants() {
  return request<{ hits: any }>({ method: 'get', url: '/assistant/_search' }).then(res => {
    const es = formatESSearchResult(res?.data);
    return es.data || [];
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
        status: 'connected',
        last_synced: fmtDate(ds?.updated),
        document_count: 0
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
