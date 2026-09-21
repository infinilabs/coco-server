/* Wiki (knowledge hub) API layer.
 *
 * The backend /wiki/* endpoints come with WS10 (knowledge-hub design doc §4:
 * crud.RegisterCRUD five-piece + custom endpoints). Until they land, every
 * function serves the ported prototype fixtures behind USE_WIKI_MOCK so the
 * migrated UI is demo-able end to end. Flip the flag and delete the fixture
 * import when wiring the real server.
 */
import { request } from '../request';
import {
  wikiArticles,
  wikiAssistants,
  wikiKbs,
  wikiMembers,
  wikiTocs,
  wikiVersions
} from '@/pages/wiki/shared/fixture';

// TODO(wiki-backend): set to false when /wiki/* endpoints ship (WS10)
const USE_WIKI_MOCK = true;

const delay = <T>(value: T, ms = 120): Promise<T> =>
  new Promise(resolve => {
    setTimeout(() => resolve(value), ms);
  });

/* ---------------- KBs ---------------- */

export function searchWikiKbs(params?: { query?: string }) {
  if (USE_WIKI_MOCK) {
    const q = (params?.query || '').toLowerCase();
    const data = q
      ? wikiKbs.filter(kb => `${kb.name} ${kb.description}`.toLowerCase().includes(q))
      : wikiKbs;
    return delay({ data, total: { value: data.length } });
  }
  return request({ method: 'get', url: '/wiki/kb/_search' });
}

export function getWikiKb(id: string) {
  if (USE_WIKI_MOCK) {
    return delay(wikiKbs.find(kb => kb.id === id) ?? null);
  }
  return request({ method: 'get', url: `/wiki/kb/${id}` });
}

export function createWikiKb(body: Partial<Api.Wiki.Kb>) {
  if (USE_WIKI_MOCK) {
    const kb: Api.Wiki.Kb = {
      id: `kb-${Date.now()}`,
      name: body.name || 'Untitled',
      description: body.description || '',
      icon: body.icon || '📚',
      visibility: body.visibility || 'team',
      workspace_id: body.workspace_id || 'ws-default',
      datasource_ids: body.datasource_ids || [],
      assistant_id: body.assistant_id,
      sync_strategy: body.sync_strategy || 'manual',
      article_count: 0,
      last_updated: '刚刚',
      members: [wikiMembers[0]],
      datasources: [],
      ai_status: 'queued'
    };
    wikiKbs.push(kb);
    wikiTocs[kb.id] = [];
    return delay(kb);
  }
  return request({ method: 'post', data: body, url: '/wiki/kb/' });
}

export function updateWikiKb(id: string, body: Partial<Api.Wiki.Kb>) {
  if (USE_WIKI_MOCK) {
    const kb = wikiKbs.find(k => k.id === id);
    if (kb) Object.assign(kb, body);
    return delay(kb ?? null);
  }
  return request({ method: 'put', data: body, url: `/wiki/kb/${id}` });
}

export function deleteWikiKb(id: string) {
  if (USE_WIKI_MOCK) {
    const i = wikiKbs.findIndex(k => k.id === id);
    if (i >= 0) wikiKbs.splice(i, 1);
    return delay({ result: 'deleted' });
  }
  return request({ method: 'delete', url: `/wiki/kb/${id}` });
}

/* ---------------- Articles ---------------- */

export function searchWikiArticles(params: { kbId?: string; query?: string }) {
  if (USE_WIKI_MOCK) {
    const q = (params.query || '').toLowerCase();
    let data = params.kbId ? wikiArticles.filter(a => a.kb_id === params.kbId) : [...wikiArticles];
    if (q) {
      data = data.filter(a =>
        `${a.title} ${a.summary} ${a.tags.join(' ')}`.toLowerCase().includes(q)
      );
    }
    return delay({ data, total: { value: data.length } });
  }
  const searchParams = new URLSearchParams();
  if (params.kbId) searchParams.set('kb_id', params.kbId);
  if (params.query) searchParams.set('q', params.query);
  return request({ method: 'get', url: `/wiki/article/_search?${searchParams.toString()}` });
}

export function getWikiArticle(id: string) {
  if (USE_WIKI_MOCK) {
    return delay(wikiArticles.find(a => a.id === id) ?? null);
  }
  return request({ method: 'get', url: `/wiki/article/${id}` });
}

export function createWikiArticle(body: Partial<Api.Wiki.Article>) {
  if (USE_WIKI_MOCK) {
    const article: Api.Wiki.Article = {
      id: `art-${Date.now()}`,
      kb_id: body.kb_id || '',
      title: body.title || 'Untitled',
      summary: body.summary || '',
      content: body.content || '## 定义\n\n## 正文内容\n',
      page_type: body.page_type,
      subtype: body.subtype,
      aliases: body.aliases,
      tags: body.tags || [],
      status: 'draft',
      ai_generated: false,
      confidence: undefined,
      sources: [],
      created_by: wikiMembers[0],
      contributors: [wikiMembers[0]],
      created_at: new Date().toISOString().slice(0, 16).replace('T', ' '),
      updated_at: new Date().toISOString().slice(0, 16).replace('T', ' ')
    };
    wikiArticles.push(article);
    const kb = wikiKbs.find(k => k.id === article.kb_id);
    if (kb) kb.article_count += 1;
    return delay(article);
  }
  return request({ method: 'post', data: body, url: '/wiki/article/' });
}

export function updateWikiArticle(id: string, body: Partial<Api.Wiki.Article>) {
  if (USE_WIKI_MOCK) {
    const article = wikiArticles.find(a => a.id === id);
    if (article) {
      Object.assign(article, body);
      article.updated_at = new Date().toISOString().slice(0, 16).replace('T', ' ');
    }
    return delay(article ?? null);
  }
  return request({ method: 'put', data: body, url: `/wiki/article/${id}` });
}

/** status machine transition: draft → reviewed → published (→ archived) */
export function updateWikiArticleStatus(id: string, status: Api.Wiki.ArticleStatus) {
  return updateWikiArticle(id, { status });
}

/* ---------------- Versions ---------------- */

export function getWikiArticleVersions(articleId: string) {
  if (USE_WIKI_MOCK) {
    return delay(wikiVersions.filter(v => v.article_id === articleId));
  }
  return request({ method: 'get', url: `/wiki/article/${articleId}/versions` });
}

/* ---------------- TOC ---------------- */

export function getWikiToc(kbId: string) {
  if (USE_WIKI_MOCK) {
    return delay(wikiTocs[kbId] || []);
  }
  return request({ method: 'get', url: `/wiki/kb/${kbId}/toc` });
}

export function updateWikiToc(kbId: string, toc: Api.Wiki.TocNode[]) {
  if (USE_WIKI_MOCK) {
    wikiTocs[kbId] = toc;
    return delay(toc);
  }
  return request({ method: 'put', data: toc, url: `/wiki/kb/${kbId}/toc` });
}

/* ---------------- misc ---------------- */

export function listWikiAssistants() {
  if (USE_WIKI_MOCK) return delay(wikiAssistants);
  return request({ method: 'get', url: '/assistant/_search' });
}

export function listWikiDatasources() {
  if (USE_WIKI_MOCK) {
    const all = wikiKbs.flatMap(kb => kb.datasources);
    const seen = new Set<string>();
    return delay(all.filter(ds => !seen.has(ds.id) && seen.add(ds.id)));
  }
  return request({ method: 'get', url: '/datasource/_search' });
}
