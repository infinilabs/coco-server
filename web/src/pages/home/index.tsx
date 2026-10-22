import type { RouteKey } from '@elegant-router/types';
import { Button, Card, Skeleton, Tag, Tooltip } from 'antd';
import {
  ArrowRight,
  BookOpen,
  Bot,
  CheckCircle2,
  Circle,
  Database,
  FileText,
  Inbox,
  MessagesSquare,
  RefreshCw,
  Search
} from 'lucide-react';

import { fetchChatSessionCount, searchAssistant } from '@/service/api/assistant';
import { fetchDataSourceList, fetchDatasourceDetail } from '@/service/api/data-source';
import { fetchIntegrations } from '@/service/api/integration';
import { searchMCPServer } from '@/service/api/mcp-server';
import { searchModelPovider } from '@/service/api/model-provider';
import { fetchPipelineList } from '@/service/api/pipeline';
import { fetchSearchOpsOverview, fetchSyncStatus } from '@/service/api/search-ops';
import { formatESSearchResult } from '@/service/request/es';
import { searchWikiArticles, searchWikiKbs } from '@/service/api/wiki';
import { selectUserInfo } from '@/store/slice/auth';

import './home.scss';

type Count = number | null;

interface WorkbenchStats {
  assistants: Count;
  datasources: Count;
  documents: Count;
  integrations: Count;
  kbs: Count;
  mcps: Count;
  models: Count;
  pipelines: Count;
  articles: Count;
  syncRunning: Count;
}

interface TrendSegment {
  count: number;
  name: string;
}

interface TrendPoint {
  count: number;
  day: string;
  segs: TrendSegment[];
}

interface FeedEvent {
  id: string;
  kind: 'assistant' | 'document' | 'sync' | 'wiki';
  link: string;
  status?: string;
  sub?: string;
  time: number;
  title: string;
}

interface DsDistItem {
  count: number;
  name: string;
}

interface WorkbenchActivity {
  chats7d: Count;
  chatsPrev: Count;
  dsDist: DsDistItem[] | null;
  feed: FeedEvent[] | null;
  searchHealth: { total: number; zeroRate: number; prev: number } | null;
  trend: TrendPoint[] | null;
}

/** null = the request failed or the user lacks the permission — the block is hidden/degraded */
const EMPTY_STATS: WorkbenchStats = {
  assistants: null,
  datasources: null,
  documents: null,
  integrations: null,
  kbs: null,
  mcps: null,
  models: null,
  pipelines: null,
  articles: null,
  syncRunning: null
};

const EMPTY_ACTIVITY: WorkbenchActivity = {
  chats7d: null,
  chatsPrev: null,
  dsDist: null,
  feed: null,
  searchHealth: null,
  trend: null
};

/** Segment colors for the stacked trend chart, assigned per datasource name. */
const TREND_PALETTE = ['#1677ff', '#389e0d', '#faad14', '#722ed1', '#13c2c2', '#eb2f96', '#fa8c16', '#2f54eb'];

/** Icon + color for each feed event kind. */
const KIND_META: Record<FeedEvent['kind'], { cls: string; icon: React.ReactNode }> = {
  assistant: { cls: 'bg-[var(--ant-color-warning-bg)] color-[var(--ant-color-warning)]', icon: <Bot size={13} /> },
  document: { cls: 'bg-[var(--ant-color-primary-bg)] color-[var(--ant-color-primary)]', icon: <FileText size={13} /> },
  sync: { cls: 'bg-[var(--ant-color-success-bg)] color-[var(--ant-color-success)]', icon: <RefreshCw size={13} /> },
  wiki: { cls: 'bg-[var(--ant-color-error-bg)] color-[var(--ant-color-error)]', icon: <BookOpen size={13} /> }
};

/** Extract the hit total from a framework ORM `_search` response. */
function readTotal(res: any): Count {
  if (!res) return null;
  const es = formatESSearchResult(res?.data ?? res);
  return typeof es?.total === 'number' ? es.total : 0;
}

function settledTotal(r: PromiseSettledResult<any>): Count {
  return r.status === 'fulfilled' ? readTotal(r.value) : null;
}

export function Component() {
  const userInfo = useAppSelector(selectUserInfo);
  const routerPush = useRouterPush();
  const { t } = useTranslation();
  const [stats, setStats] = useState<WorkbenchStats>(EMPTY_STATS);
  const [activity, setActivity] = useState<WorkbenchActivity>(EMPTY_ACTIVITY);
  const [loading, setLoading] = useState(true);

  const loadStats = useCallback(async () => {
    setLoading(true);
    // Every call is best-effort: a 403 (missing permission) just leaves the
    // counter as null and the corresponding block degrades gracefully.
    const [ds, docs, ast, kb, mp, pl, itg, mcp, arts, sync, ops, chats, chatsPrevReq, trend, recent] = await Promise.allSettled([
      fetchDataSourceList({ size: 0 }),
      fetchDatasourceDetail({ size: 0 }),
      searchAssistant({ size: 3, sort: 'created:desc' }),
      searchWikiKbs(),
      searchModelPovider({ size: 0 }),
      fetchPipelineList({ size: 0 }),
      fetchIntegrations({ size: 0 }),
      searchMCPServer({ size: 0 }),
      searchWikiArticles({ size: 3, sort: 'updated:desc' }),
      fetchSyncStatus(),
      fetchSearchOpsOverview({ days: 7 }),
      fetchChatSessionCount(['created>=now-7d']),
      fetchChatSessionCount(['created>=now-14d', 'created<now-7d']),
      // URL aggs pass straight through the framework query builder (same trick as the datasource stats tab)
      fetchDatasourceDetail({
        size: 0,
        'agg[by_day][date_histogram][field]': 'created',
        'agg[by_day][date_histogram][interval]': '1d',
        'agg[by_day][date_histogram][format]': 'yyyy-MM-dd',
        // Nested sub-agg lives at the OUTER level (agg[name][aggs][...]), not inside date_histogram
        'agg[by_day][aggs][by_ds][terms][field]': 'source.name',
        'agg[by_day][aggs][by_ds][terms][size]': '10',
        // Overall per-datasource document distribution (for the bottom bar-list card)
        'agg[by_ds_total][terms][field]': 'source.name',
        'agg[by_ds_total][terms][size]': '10'
      }),
      fetchDatasourceDetail({ size: 5, sort: 'created:desc' })
    ]);

    setStats({
      assistants: settledTotal(ast),
      datasources: settledTotal(ds),
      documents: settledTotal(docs),
      integrations: settledTotal(itg),
      kbs: kb.status === 'fulfilled' ? (typeof kb.value?.total === 'number' ? kb.value.total : 0) : null,
      mcps: settledTotal(mcp),
      models: settledTotal(mp),
      pipelines: settledTotal(pl),
      // searchWikiArticles returns an already-normalized {data, total} — read .total directly
      // (readTotal would re-parse .data as an ES response and yield 0)
      articles: arts.status === 'fulfilled' ? (typeof arts.value?.total === 'number' ? arts.value.total : 0) : null,
      syncRunning:
        sync.status === 'fulfilled'
          ? (sync.value?.data?.items ?? sync.value?.items ?? []).filter((item: any) => item?.run?.state === 'running')
              .length
          : null
    });

    let trendPoints: TrendPoint[] | null = null;
    let dsDist: DsDistItem[] | null = null;
    if (trend.status === 'fulfilled') {
      const res = trend.value?.data ?? trend.value;
      const buckets = res?.aggregations?.by_day?.buckets;
      if (Array.isArray(buckets)) {
        trendPoints = buckets.slice(-7).map((b: any) => ({
          count: b.doc_count ?? 0,
          day: String(b.key_as_string ?? b.key ?? '').slice(5),
          segs: (b.by_ds?.buckets ?? []).map((s: any) => ({
            count: s.doc_count ?? 0,
            name: String(s.key ?? '')
          }))
        }));
      }
      const distBuckets = res?.aggregations?.by_ds_total?.buckets;
      if (Array.isArray(distBuckets) && distBuckets.length) {
        dsDist = distBuckets
          .map((b: any) => ({ count: b.doc_count ?? 0, name: String(b.key ?? '') }))
          .sort((a: DsDistItem, b: DsDistItem) => b.count - a.count);
      }
    }
    const feed: FeedEvent[] = [];
    if (recent.status === 'fulfilled') {
      const es = formatESSearchResult(recent.value?.data ?? recent.value);
      (es.data ?? []).slice(0, 5).forEach((doc: any) => {
        feed.push({
          id: `doc-${doc.id}`,
          kind: 'document',
          link: `/data-source/doc/${doc.id}`,
          status: doc.status ?? '',
          sub: doc.source?.name ?? '',
          time: new Date(doc.created ?? 0).getTime() || 0,
          title: doc.title || doc.name || doc.id
        });
      });
    }
    if (arts.status === 'fulfilled') {
      // searchWikiArticles already returns normalized articles ({data, total}) — no ES re-parse
      (arts.value?.data ?? []).slice(0, 3).forEach((art: any) => {
        feed.push({
          id: `wiki-${art.id}`,
          kind: 'wiki',
          link: '/wiki',
          time: new Date(art.updated_at ?? art.updated ?? art.created_at ?? 0).getTime() || 0,
          title: art.title || art.name || art.id
        });
      });
    }
    if (ast.status === 'fulfilled') {
      const es = formatESSearchResult(ast.value?.data ?? ast.value);
      (es.data ?? []).slice(0, 3).forEach((a: any) => {
        feed.push({
          id: `ast-${a.id}`,
          kind: 'assistant',
          link: '/ai-assistant',
          time: new Date(a.created ?? 0).getTime() || 0,
          title: a.name || a.id
        });
      });
    }
    if (sync.status === 'fulfilled') {
      (((sync.value?.data?.items ?? sync.value?.items) ?? []) as any[])
        .filter(item => item?.run && item.run.state !== 'running')
        .slice(0, 3)
        .forEach(item => {
          feed.push({
            id: `sync-${item.id}-${item.run.run_id ?? ''}`,
            kind: 'sync',
            link: '/data-source',
            sub: t('page.home.workbench.activity.syncDone', { n: item.run.documents ?? 0 }),
            time: Number(item.run.updated_at) || 0,
            title: item.name ?? item.id
          });
        });
    }
    feed.sort((a, b) => b.time - a.time);
    let chats7d: Count = null;
    let chatsPrev: Count = null;
    const readHitsTotal = (v: any): Count => {
      const total = (v?.data ?? v)?.hits?.total;
      return typeof total === 'number' ? total : (total?.value ?? null);
    };
    if (chats.status === 'fulfilled') chats7d = readHitsTotal(chats.value);
    if (chatsPrevReq.status === 'fulfilled') chatsPrev = readHitsTotal(chatsPrevReq.value);
    let searchHealth: { total: number; zeroRate: number; prev: number } | null = null;
    if (ops.status === 'fulfilled') {
      const o = ops.value?.data ?? ops.value;
      if (o && typeof o.total_searches === 'number') {
        searchHealth = { total: o.total_searches, zeroRate: o.zero_hit_rate ?? 0, prev: o.total_searches_prev ?? 0 };
      }
    }
    // feed: null = every feed source failed (show retry); [] = loaded but no events yet
    const feedFailed = [recent, arts, ast, sync].every(source => source.status !== 'fulfilled');
    setActivity({ chats7d, chatsPrev, dsDist, feed: feedFailed ? null : feed, searchHealth, trend: trendPoints });
    setLoading(false);
  }, [t]);

  useMount(() => {
    loadStats();
  });

  const go = (link: string, query?: Record<string, string>) => {
    if (query) {
      routerPush.routerPushByKey(link.replace(/^\//, '') as RouteKey, { query });
    } else {
      routerPush.routerPush(link);
    }
  };

  /** Precise event time: just now / N min ago / HH:mm today / yesterday / MM-dd HH:mm */
  const fmtEventTime = (ms: number) => {
    if (!ms) return '';
    const minutes = Math.floor((Date.now() - ms) / 60000);
    if (minutes < 1) return t('page.home.workbench.activity.timeJustNow');
    if (minutes < 60) return t('page.home.workbench.activity.timeMinutesAgo', { n: minutes });
    const d = new Date(ms);
    const hhmm = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
    const now = new Date();
    if (d.toDateString() === now.toDateString()) return hhmm;
    const yesterday = new Date(now);
    yesterday.setDate(now.getDate() - 1);
    if (d.toDateString() === yesterday.toDateString()) {
      return `${t('page.home.workbench.activity.yesterday')} ${hhmm}`;
    }
    return `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${hhmm}`;
  };

  const kindLabel = (kind: FeedEvent['kind']) => t(`page.home.workbench.activity.kind.${kind}`);

  const checklist = useMemo(() => {
    const items: { desc: string; done: boolean; link: string; title: string }[] = [];
    if (stats.models !== null) {
      items.push({
        done: stats.models > 0,
        link: '/model-provider/list',
        title: t('page.home.workbench.checklist.model'),
        desc: t('page.home.workbench.checklist.modelDesc')
      });
    }
    if (stats.datasources !== null) {
      items.push({
        done: stats.datasources > 0,
        link: '/data-source',
        title: t('page.home.workbench.checklist.datasource'),
        desc: t('page.home.workbench.checklist.datasourceDesc')
      });
    }
    if (stats.pipelines !== null) {
      items.push({
        done: stats.pipelines > 0,
        link: '/pipeline',
        title: t('page.home.workbench.checklist.pipeline'),
        desc: t('page.home.workbench.checklist.pipelineDesc')
      });
    }
    if (stats.integrations !== null || stats.mcps !== null) {
      items.push({
        done: (stats.integrations ?? 0) + (stats.mcps ?? 0) > 0,
        link: '/integration',
        title: t('page.home.workbench.checklist.access'),
        desc: t('page.home.workbench.checklist.accessDesc')
      });
    }
    return items;
  }, [stats, t]);

  const allDone = checklist.length > 0 && checklist.every(item => item.done);

  const last7Added = useMemo(() => {
    if (!activity.trend) return null;
    return activity.trend.reduce((sum, p) => sum + p.count, 0);
  }, [activity.trend]);

  const metrics = [
    {
      count: stats.datasources,
      icon: <Database size={18} />,
      iconClass: 'bg-[var(--ant-color-primary-bg)] color-[var(--ant-color-primary)]',
      label: t('page.home.workbench.metrics.datasources'),
      link: '/data-source',
      sub:
        stats.syncRunning !== null && stats.syncRunning > 0
          ? t('page.home.workbench.activity.syncRunning', { n: stats.syncRunning })
          : ''
    },
    {
      count: stats.documents,
      icon: <FileText size={18} />,
      iconClass: 'bg-[var(--ant-color-success-bg)] color-[var(--ant-color-success)]',
      label: t('page.home.workbench.metrics.documents'),
      link: '/data-source',
      sub: last7Added !== null && last7Added > 0 ? t('page.home.workbench.activity.docsAdded', { n: last7Added }) : ''
    },
    {
      count: stats.assistants,
      icon: <Bot size={18} />,
      iconClass: 'bg-[var(--ant-color-warning-bg)] color-[var(--ant-color-warning)]',
      label: t('page.home.workbench.metrics.assistants'),
      link: '/ai-assistant',
      sub: ''
    },
    {
      count: stats.kbs,
      icon: <BookOpen size={18} />,
      iconClass: 'bg-[var(--ant-color-error-bg)] color-[var(--ant-color-error)]',
      label: t('page.home.workbench.metrics.kbs'),
      link: '/wiki',
      sub: stats.articles !== null && stats.articles > 0 ? t('page.home.workbench.activity.articles', { n: stats.articles }) : ''
    }
  ];

  /** Unified stat tag shown on each app card. Search and chat both show the
   *  7-day count with a period-over-period delta rendered AFTER the tag as a
   *  separate element (no shared background). For search, a red zero-hit tag
   *  joins the row when recall is bad (the card doubles as the health alert). */
  const statDelta = (total: number, prev: number) => {
    if (total === 0 && prev === 0) return null;
    const delta = total - prev;
    const tip = t('page.home.workbench.apps.statDeltaTip');
    if (delta === 0) {
      return (
        <Tooltip title={tip}>
          <span className="cursor-default text-12px" style={{ color: 'var(--ant-color-text-quaternary, #999)' }}>
            —
          </span>
        </Tooltip>
      );
    }
    const up = delta > 0;
    return (
      <Tooltip title={tip}>
        <span
          className="cursor-default text-12px font-semibold tabular-nums"
          style={{ color: up ? '#52c41a' : '#ff4d4f' }}
        >
          {up ? '↑' : '↓'}
          {up ? '+' : ''}
          {delta}
        </span>
      </Tooltip>
    );
  };

  const searchStat =
    activity.searchHealth !== null ? (
      <div className="flex items-center gap-4px self-start">
        <Tag className="cursor-pointer" onClick={() => go('/search-ops')}>
          {t('page.home.workbench.apps.searchStat7d', { n: activity.searchHealth.total })}
        </Tag>
        {statDelta(activity.searchHealth.total, activity.searchHealth.prev)}
        {activity.searchHealth.zeroRate >= 0.5 ? (
          <Tooltip
            title={t('page.home.workbench.activity.zeroHitTip', {
              n: Math.round(activity.searchHealth.total * activity.searchHealth.zeroRate),
              total: activity.searchHealth.total
            })}
          >
            <Tag className="cursor-pointer" color="error" onClick={() => go('/search-ops')}>
              {t('page.home.workbench.activity.zeroHit', { pct: Math.round(activity.searchHealth.zeroRate * 100) })}
            </Tag>
          </Tooltip>
        ) : null}
      </div>
    ) : null;

  const chatStat =
    activity.chats7d !== null ? (
      <div className="flex items-center gap-4px self-start">
        <Tag className="cursor-pointer" onClick={() => go('/chat?mode=chat')}>
          {t('page.home.workbench.apps.chatStat', { n: activity.chats7d })}
        </Tag>
        {activity.chatsPrev !== null ? statDelta(activity.chats7d, activity.chatsPrev) : null}
      </div>
    ) : null;

  const wikiStat =
    stats.kbs !== null || stats.articles !== null ? (
      <Tag className="cursor-pointer self-start" onClick={() => go('/wiki')}>
        {t('page.home.workbench.apps.wikiStat', { kbs: stats.kbs ?? 0, articles: stats.articles ?? 0 })}
      </Tag>
    ) : null;

  const apps = [
    {
      stat: searchStat,
      icon: <Search size={22} />,
      link: '/search',
      title: t('page.home.workbench.apps.search'),
      desc: t('page.home.workbench.apps.searchDesc')
    },
    {
      stat: chatStat,
      icon: <MessagesSquare size={22} />,
      link: '/chat?mode=chat',
      title: t('page.home.workbench.apps.chat'),
      desc: t('page.home.workbench.apps.chatDesc')
    },
    {
      stat: wikiStat,
      icon: <BookOpen size={22} />,
      link: '/wiki',
      title: t('page.home.workbench.apps.wiki'),
      desc: t('page.home.workbench.apps.wikiDesc')
    }
  ];

  const statusTag = (status: string) => {
    const colorMap: Record<string, string> = { completed: 'success', indexing: 'processing', failed: 'error' };
    return (
      <Tag color={colorMap[status] ?? 'default'} className="mr-0 shrink-0">
        {t(`page.home.workbench.activity.status.${status || 'legacy'}`)}
      </Tag>
    );
  };

  const dsColors = useMemo(() => {
    const map = new Map<string, string>();
    if (activity.trend) {
      for (const p of activity.trend) {
        for (const s of p.segs) {
          if (!map.has(s.name)) map.set(s.name, TREND_PALETTE[map.size % TREND_PALETTE.length]);
        }
      }
    }
    return map;
  }, [activity.trend]);

  /** Nice Y-axis scale (1/2/5 × 10^k) so bars and gridlines align. */
  const trendScale = useMemo(() => {
    if (!activity.trend?.length) return null;
    const raw = Math.max(...activity.trend.map(p => p.count), 1);
    const pow = 10 ** Math.floor(Math.log10(raw));
    const top = [1, 2, 5, 10].map(c => c * pow).find(c => c >= raw) ?? 10 * pow;
    return { half: top / 2, top };
  }, [activity.trend]);

  const fmtTick = (n: number) => (n >= 1000 ? `${n / 1000}k` : n);

  const trendBars = useMemo(() => {
    if (!activity.trend?.length) return null;
    const max = trendScale?.top ?? 1;
    return activity.trend.map((p, i) => {
      const segs = p.segs.length ? p.segs : (p.count > 0 ? [{ count: p.count, name: '' }] : []);
      return (
        <Tooltip
          key={`${p.day}-${i}`}
          title={
            <div className="text-12px leading-20px">
              <div className="m-b-2px font-medium tabular-nums">{p.day} · {p.count}</div>
              {segs.map(s => (
                <div className="flex items-center gap-6px" key={s.name || '_'}>
                  <span className="h-8px w-8px shrink-0 rounded-full" style={{ background: dsColors.get(s.name) ?? '#8c8c8c' }} />
                  <span className="opacity-85">{s.name || t('page.home.workbench.activity.uncategorized')}</span>
                  <span className="ml-auto tabular-nums">{s.count}</span>
                </div>
              ))}
            </div>
          }
        >
          <div className="flex h-full w-full min-w-0 flex-1 cursor-default flex-col justify-end rounded-t-2px">
            {segs.map(s => (
              <div
                key={s.name || '_'}
                style={{
                  background: dsColors.get(s.name) ?? '#8c8c8c',
                  height: `${Math.max((s.count / max) * 100, p.count > 0 ? 3 : 0)}%`
                }}
              />
            ))}
          </div>
        </Tooltip>
      );
    });
  }, [activity.trend, dsColors, trendScale, t]);

  const showActivity = activity.trend !== null || activity.feed !== null;

  return (
    // pb-16px: the layout's GlobalContent has a fixed h-full box, so on pages taller than
    // the viewport its bottom padding is swallowed by the overflow — compensate here so the
    // last card keeps a 32px gap to the window bottom (16 own margin + 16 padding).
    <div className="pb-16px">
      <div className="m-b-16px">
        <div className="text-22px font-medium color-[var(--ant-color-text-heading)]">
          {t('page.home.workbench.greeting', { user: userInfo?.name ?? '' })}
        </div>
        <div className="m-t-4px text-13px color-[var(--ant-color-text-secondary)]">{t('page.home.workbench.subtitle')}</div>
      </div>

      <div className="m-b-16px grid grid-cols-2 gap-16px lg:grid-cols-4">
        {metrics.map(item => (
          <Card
            key={item.label}
            className="cursor-pointer transition-all hover:border-[var(--ant-color-primary)] hover:shadow-[0_2px_12px_rgba(0,0,0,0.08)]"
            classNames={{ body: '!p-16px' }}
            onClick={() => go(item.link)}
          >
            <div className="flex items-start gap-12px">
              <div className={`flex h-40px w-40px shrink-0 items-center justify-center rounded-10px ${item.iconClass}`}>
                {loading ? <Skeleton.Node active style={{ width: 18, height: 18 }} /> : item.icon}
              </div>
              <div className="min-w-0">
                <div className="text-12px color-[var(--ant-color-text-secondary)]">{item.label}</div>
                <div className="m-t-2px text-24px font-medium leading-tight tabular-nums color-[var(--ant-color-text-heading)]">
                  {loading ? <Skeleton.Input active size="small" style={{ width: 56, height: 24 }} /> : (item.count ?? '—')}
                </div>
                {item.sub ? <div className="m-t-2px truncate text-12px color-[var(--ant-color-text-tertiary)]">{item.sub}</div> : null}
              </div>
            </div>
          </Card>
        ))}
      </div>

      {!loading && !allDone && (
      <Card className="m-b-16px" title={t('page.home.workbench.checklist.title')}>
        <div className="flex flex-col gap-12px">
            {checklist.map(item => (
              <div className="flex items-center gap-12px" key={item.title}>
                {item.done ? (
                  <CheckCircle2 className="shrink-0" color="#389e0d" size={18} />
                ) : (
                  <Circle className="shrink-0" color="var(--ant-color-text-quaternary)" size={18} />
                )}
                <div className="flex-1">
                  <div
                    className={`text-14px ${
                      item.done ? 'color-[var(--ant-color-text-tertiary)] line-through' : 'color-[var(--ant-color-text-heading)]'
                    }`}
                  >
                    {item.title}
                  </div>
                  {!item.done && <div className="text-12px color-[var(--ant-color-text-secondary)]">{item.desc}</div>}
                </div>
                {!item.done && (
                  <Button size="small" type="link" onClick={() => go(item.link)}>
                    {t('page.home.workbench.apps.enter')} <ArrowRight size={14} />
                  </Button>
                )}
              </div>
            ))}
          </div>
      </Card>
      )}

      <div className="m-b-16px grid grid-cols-1 gap-16px md:grid-cols-3">
        {apps.map(app => (
          <Card
            key={app.title}
            className="group transition-all hover:border-[var(--ant-color-primary)] hover:shadow-[0_2px_12px_rgba(0,0,0,0.08)]"
          >
            <div className="flex flex-col gap-8px">
              <div className="flex h-44px w-44px items-center justify-center rounded-12px bg-[var(--ant-color-primary-bg)] color-[var(--ant-color-primary)] transition-transform group-hover:scale-105">
                {app.icon}
              </div>
              <div className="text-16px font-medium color-[var(--ant-color-text-heading)]">{app.title}</div>
              <div className="min-h-40px flex-1 text-13px color-[var(--ant-color-text-secondary)]">{app.desc}</div>
              {app.stat}
              <Button aria-label={t('page.home.workbench.apps.enter')} className="self-start" type="primary" onClick={() => go(app.link)}>
                <ArrowRight className="transition-transform group-hover:translate-x-2px" size={14} />
              </Button>
            </div>
          </Card>
        ))}
      </div>

      {showActivity && (
        <div className="workbench-activity m-b-16px">
          <div className="activity-left">
          <Card className="activity-trend-card" title={t('page.home.workbench.activity.trend')}>
            {loading ? (
              <Skeleton active paragraph={{ rows: 3 }} title={false} />
            ) : trendBars ? (
              <>
                <div className="activity-chart-row">
                  <div className="activity-yaxis">
                    <span>{fmtTick(trendScale?.top ?? 0)}</span>
                    <span>{fmtTick(trendScale?.half ?? 0)}</span>
                    <span>0</span>
                  </div>
                  <div className="activity-plot">
                    <div className="pointer-events-none absolute inset-x-0 top-0 border-t-1 border-dashed border-[var(--ant-color-border-secondary)]" />
                    <div className="pointer-events-none absolute inset-x-0 top-50% border-t-1 border-dashed border-[var(--ant-color-border-secondary)]" />
                    <div className="absolute inset-x-0 bottom-0 border-t-1 border-[var(--ant-color-border-secondary)]" />
                    <div className="activity-bars">{trendBars}</div>
                  </div>
                </div>
                <div className="m-t-4px flex justify-between text-11px tabular-nums color-[var(--ant-color-text-tertiary)]">
                  <span>{activity.trend?.[0]?.day}</span>
                  <span>{activity.trend?.[activity.trend.length - 1]?.day}</span>
                </div>
                {dsColors.size > 1 ? (
                  <div className="m-t-8px flex flex-wrap gap-x-12px gap-y-2px text-11px color-[var(--ant-color-text-secondary)]">
                    {[...dsColors.entries()].map(([name, color]) => (
                      <span className="flex items-center gap-4px" key={name}>
                        <span className="h-8px w-8px shrink-0 rounded-full" style={{ background: color }} />
                        <span className="max-w-120px truncate">{name}</span>
                      </span>
                    ))}
                  </div>
                ) : null}
              </>
            ) : null}
          </Card>
          {activity.dsDist && activity.dsDist.length > 0 ? (
          <Card className="activity-dist-card" title={t('page.home.workbench.dist.title')}>
                    <div className="workbench-dist-list">
                      {(() => {
                        const max = Math.max(...activity.dsDist.map(d => d.count), 1);
                        const total = activity.dsDist.reduce((sum, d) => sum + d.count, 0);
                        return activity.dsDist.map(d => (
                          <div
                            className="-mx-6px flex cursor-pointer items-center gap-12px rounded-sm px-6px py-2px transition-colors hover:bg-[var(--ant-color-fill-tertiary)]"
                            key={d.name}
                            onClick={() => go('/data-source')}
                          >
                            <span className="w-140px shrink-0 truncate text-13px color-[var(--ant-color-text-heading)]" title={d.name}>
                              {d.name}
                            </span>
                            <div className="h-16px min-w-0 flex-1 overflow-hidden rounded-4px bg-[var(--ant-color-fill-tertiary)]">
                              <div
                                className="h-full rounded-4px bg-[var(--ant-color-primary)] opacity-80 transition-opacity hover:opacity-100"
                                style={{ width: `${Math.max((d.count / max) * 100, 2)}%` }}
                              />
                            </div>
                            <span className="w-88px shrink-0 text-right text-12px tabular-nums color-[var(--ant-color-text-secondary)]">
                              {d.count} · {total > 0 ? Math.round((d.count / total) * 100) : 0}%
                            </span>
                          </div>
                        ));
                      })()}
                    </div>
                  </Card>
          ) : null}
          </div>
          <Card
            className="activity-feed-card"
            title={t('page.home.workbench.activity.feed')}
          >
            {loading ? (
              <Skeleton active paragraph={{ rows: 3 }} title={false} />
            ) : (
              <>
                <div className="flex flex-col">
                  {(activity.feed ?? []).slice(0, 8).map(ev => {
                    const meta = KIND_META[ev.kind];
                    return (
                      <div
                        className="-mx-6px flex cursor-pointer gap-10px rounded-sm px-6px py-6px transition-colors hover:bg-[var(--ant-color-fill-tertiary)]"
                        key={ev.id}
                        onClick={() => routerPush.routerPush(ev.link)}
                      >
                        <span className={`mt-2px flex h-24px w-24px shrink-0 items-center justify-center rounded-6px ${meta.cls}`}>
                          {meta.icon}
                        </span>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-6px">
                            <span className="min-w-0 truncate text-13px font-medium color-[var(--ant-color-text-heading)]" title={ev.title}>
                              {ev.title}
                            </span>
                            {ev.status ? statusTag(ev.status) : null}
                          </div>
                          <div className="m-t-2px flex items-center gap-6px text-12px color-[var(--ant-color-text-tertiary)]">
                            <span className={`shrink-0 rounded-4px px-4px py-2px text-11px leading-none ${meta.cls}`}>
                              {kindLabel(ev.kind)}
                            </span>
                            {ev.sub ? <span className="min-w-0 truncate">{ev.sub}</span> : null}
                            <span className="ml-auto shrink-0 tabular-nums">{fmtEventTime(ev.time)}</span>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                  {activity.feed === null ? (
                    <div
                      className="-mx-6px flex cursor-pointer items-center justify-center gap-6px rounded-sm py-16px text-12px color-[var(--ant-color-text-tertiary)] transition-colors hover:bg-[var(--ant-color-fill-tertiary)]"
                      onClick={loadStats}
                    >
                      <RefreshCw size={13} />
                      {t('page.home.workbench.activity.feedFailed')}
                    </div>
                  ) : !activity.feed.length ? (
                    <div className="flex flex-col items-center gap-6px py-24px">
                      <Inbox size={26} className="color-[var(--ant-color-text-quaternary)]" />
                      <div className="text-13px color-[var(--ant-color-text-secondary)]">
                        {t('page.home.workbench.activity.feedEmpty')}
                      </div>
                      <div className="text-12px color-[var(--ant-color-text-tertiary)]">
                        {t('page.home.workbench.activity.feedEmptyHint')}
                      </div>
                    </div>
                  ) : null}
                </div>
              </>
            )}
          </Card>
        </div>
      )}


    </div>
  );
}
