import { Activity, AlertTriangle, DatabaseZap, Search, Timer } from 'lucide-react';
import PageHeaderTitle from '@/components/stateless/common/PageHeaderTitle';
import { MonitorOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Col, Drawer, Input, Modal, Row, Segmented, Space, Statistic, Switch, Table, Tabs, Tag, Tooltip, Typography, message } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';

import ProcessingPanel from '@/components/ProcessingPanel';
import '../settings/index.scss';
import {
  fetchIndexHealth,
  fetchProcessingOverview,
  fetchSearchLogs,
  fetchSearchOpsOverview,
  fetchSyncStatus,
  retryFailedDocs,
  IndexHealthEntry,
  ProcessingFailedDoc,
  ProcessingOverview,
  SearchLogEntry,
  SearchOpsLowMiss,
  SearchOpsOverview,
  SearchOpsRow,
  SyncStatusItem
} from '@/service/api/search-ops';

type TabKey = 'overview' | 'logs' | 'ingestion' | 'sync';

/**
 * Search operations (P2), organized as tabs — the same ops domain seen from
 * four angles (aggregate overview / per-query logs / ingestion health / sync
 * status), so the page no longer grows one long-scroll section per feature.
 * The active tab lives in the URL (?tab=) so a view is deep-linkable and the
 * browser back button works. Each tab loads its own data on activation.
 */
export function Component() {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    view: hasAuth('coco#search/ops')
  };

  const [searchParams, setSearchParams] = useSearchParams();
  const activeKey = (searchParams.get('tab') as TabKey) || 'overview';

  const [data, setData] = useState<SearchOpsOverview | null>(null);
  const [health, setHealth] = useState<IndexHealthEntry[] | null>(null);
  const [processing, setProcessing] = useState<ProcessingOverview | null>(null);
  const [syncStatus, setSyncStatus] = useState<SyncStatusItem[] | null>(null);
  const [retrying, setRetrying] = useState(false);
  const [loading, setLoading] = useState(false);
  const [detailDoc, setDetailDoc] = useState<ProcessingFailedDoc | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);
  // query-log drill-down: raw searches with filters — the per-query half of
  // query analysis (the overview tab is the aggregate)
  const [logs, setLogs] = useState<SearchLogEntry[] | null>(null);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logFilter, setLogFilter] = useState<'all' | 'zero' | 'slow'>('all');
  const [logQ, setLogQ] = useState('');
  // controlled page size: the fetch batch follows it so the pager never
  // disappears (uncontrolled + hideOnSinglePage hid the whole pager when
  // pageSize >= row count, leaving no way to switch back without a reload)
  const [logPageSize, setLogPageSize] = useState(10);

  const loadLogs = useCallback(async () => {
    setLogsLoading(true);
    try {
      const params: Record<string, unknown> = { size: Math.max(50, logPageSize) };
      if (logFilter === 'zero') params.zero_hit = true;
      if (logFilter === 'slow') params.min_took_ms = 1000;
      if (logQ.trim()) params.q = logQ.trim();
      const res: any = await fetchSearchLogs(params);
      setLogs((res?.data?.items as SearchLogEntry[]) ?? []);
    } catch {
      setLogs([]);
    } finally {
      setLogsLoading(false);
    }
  }, [logFilter, logQ, logPageSize]);

  const loadOverview = useCallback(async () => {
    setLoading(true);
    try {
      const res: any = await fetchSearchOpsOverview();
      setData((res?.data as SearchOpsOverview) ?? null);
      const healthRes: any = await fetchIndexHealth();
      setHealth((healthRes?.data?.indices as IndexHealthEntry[]) ?? null);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchOps.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  // ingestion tasks (W13a) — failures are the actionable half
  const loadProcessing = useCallback(async () => {
    setLoading(true);
    try {
      const procRes: any = await fetchProcessingOverview();
      setProcessing((procRes?.data as ProcessingOverview) ?? null);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchOps.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const loadSync = useCallback(async () => {
    setLoading(true);
    try {
      const syncRes: any = await fetchSyncStatus();
      setSyncStatus((syncRes?.data?.items as SyncStatusItem[]) ?? null);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchOps.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const retryFailed = useCallback(() => {
    Modal.confirm({
      title: t('page.searchOps.retryFailed'),
      content: t('page.searchOps.retryConfirm'),
      onOk: async () => {
        setRetrying(true);
        try {
          const res: any = await retryFailedDocs();
          const d = res?.data ?? {};
          message.success(t('page.searchOps.retryDone', { retried: String(d.retried ?? 0), still: String(d.still_failed ?? 0) }));
          await loadProcessing();
        } catch (e: any) {
          message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchOps.loadFailed'));
        } finally {
          setRetrying(false);
        }
      }
    });
  }, [t, loadProcessing]);

  // per-tab data loading: every activation (re)fetches, so switching tabs
  // doubles as a refresh and no tab shows stale numbers from mount time
  useEffect(() => {
    if (!permissions.view) return;
    if (activeKey === 'overview') loadOverview();
    else if (activeKey === 'logs') loadLogs();
    else if (activeKey === 'ingestion') loadProcessing();
    else if (activeKey === 'sync') loadSync();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeKey, permissions.view]);

  // ops polling: the queue depths and run states only mean something live;
  // off by default so the tab stays a quiet report until asked to watch
  useEffect(() => {
    if (!autoRefresh || !permissions.view || activeKey !== 'ingestion') return;
    const timer = setInterval(loadProcessing, 5000);
    return () => clearInterval(timer);
  }, [autoRefresh, loadProcessing, permissions.view, activeKey]);

  const strategyColumns = [
    { title: t('page.searchOps.strategy'), dataIndex: 'type', width: 140 },
    { title: t('page.searchOps.searches'), dataIndex: 'count', width: 100 },
    { title: t('page.searchOps.zeroHits'), dataIndex: 'zero_hits', width: 110 },
    { title: t('page.searchOps.rewritten'), dataIndex: 'rewritten', width: 100 },
    { title: t('page.searchOps.avgTook'), dataIndex: 'avg_took_ms', width: 120, render: (v: number) => `${v ?? 0} ms` }
  ];

  const lowRecallColumns = [
    {
      title: t('page.searchOps.query'),
      dataIndex: 'query',
      ellipsis: true,
      render: (v: string, r: SearchOpsLowMiss) =>
        r.proposal ? (
          <Space size={4}>
            <span>{v}</span>
            <Tag color="volcano">{t('page.searchOps.gapFiled')}</Tag>
          </Space>
        ) : (
          v
        )
    },
    { title: t('page.searchOps.misses'), dataIndex: 'count', width: 90 },
    {
      title: t('page.searchOps.lastSeen'),
      dataIndex: 'last_seen',
      width: 180,
      render: (v: string) => (v ? new Date(v).toLocaleString() : '-')
    }
  ];

  const failedColumns = [
    { title: t('page.searchOps.failedDocTitle'), dataIndex: 'title', ellipsis: true },
    { title: t('page.searchOps.failedDatasource'), dataIndex: 'datasource', width: 140, ellipsis: true },
    { title: t('page.searchOps.failedError'), dataIndex: 'error', ellipsis: true },
    {
      title: t('page.searchOps.failedUpdated'),
      dataIndex: 'updated',
      width: 170,
      render: (v: string) => (v ? new Date(v).toLocaleString() : '-')
    }
  ];

  const runStateTag = (item: SyncStatusItem) => {
    if (!item.run) return <Tag>{t('page.searchOps.syncNever')}</Tag>;
    if (item.run.stale) return <Tooltip title={t('page.searchOps.syncStale')}><Tag color="red">{t('page.searchOps.syncStale')}</Tag></Tooltip>;
    if (item.run.state === 'running') return <Tag color="processing">{t('page.searchOps.syncRunning')}</Tag>;
    if (item.run.state === 'completed') return <Tag color="green">{t('page.searchOps.syncCompleted')}</Tag>;
    if (item.run.state === 'superseded') return <Tag color="orange">{t('page.searchOps.syncSuperseded')}</Tag>;
    return <Tag>{item.run.state}</Tag>;
  };

  // age of the oldest in-flight document, in minutes — fresh means the
  // backlog drains, a growing number means a stuck pipeline
  const oldestIndexingAge = () => {
    const stamp = processing?.live?.oldest_indexing;
    if (!stamp) return null;
    const age = Date.now() - new Date(stamp).getTime();
    if (!(age >= 0)) return null;
    const minutes = Math.floor(age / 60000);
    if (minutes < 1) return t('page.searchOps.liveJustNow');
    if (minutes < 60) return t('page.searchOps.liveMinutesAgo', { minutes: String(minutes) });
    return t('page.searchOps.liveHoursAgo', { hours: String(Math.floor(minutes / 60)) });
  };

  const syncColumns = [
    { title: t('page.searchOps.syncName'), dataIndex: 'name', ellipsis: true },
    { title: t('page.searchOps.syncType'), dataIndex: 'type', width: 100 },
    {
      title: t('page.searchOps.syncEnabled'),
      dataIndex: 'enabled',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="green">on</Tag> : <Tag>off</Tag>)
    },
    { title: t('page.searchOps.syncInterval'), dataIndex: 'sync_interval', width: 100, render: (v: string) => v || '-' },
    {
      title: t('page.searchOps.syncLastDispatch'),
      dataIndex: 'last_dispatch',
      width: 170,
      render: (v: string) => (v ? new Date(v).toLocaleString() : '-')
    },
    { title: t('page.searchOps.syncCursor'), dataIndex: 'increment_cursor', width: 170, ellipsis: true, render: (v: string) => v || '-' },
    {
      title: t('page.searchOps.syncRunState'),
      key: 'run',
      width: 220,
      render: (_: unknown, item: SyncStatusItem) =>
        item.run ? (
          <Space size={4}>
            {runStateTag(item)}
            <span className="text-12px">
              {t('page.searchOps.syncRunProgress', { batches: String(item.run.batches), documents: String(item.run.documents) })}
            </span>
          </Space>
        ) : (
          runStateTag(item)
        )
    }
  ];

  const healthColumns = [
    { title: t('page.searchOps.indexName'), dataIndex: 'name', width: 180 },
    {
      title: t('page.searchOps.indexStatus'),
      dataIndex: 'status',
      width: 120,
      render: (v: string, r: IndexHealthEntry) =>
        v === 'ok' ? <Tag color="green">ok</Tag> : (
          <Tooltip title={r.note}>
            <Tag color="red">unavailable</Tag>
          </Tooltip>
        )
    },
    { title: t('page.searchOps.indexDocs'), dataIndex: 'docs', width: 120, render: (v: number) => (v < 0 ? '-' : v) }
  ];

  // merge consecutive executions of the same query into one row: one user
  // search fans out to several backend searches (aggregation buckets used to log, the
  // AI overview's RAG tool re-runs the query seconds later) — operators read
  // one search as one row and expand for the individual executions
  const logGroups = useMemo<SearchLogEntry[][]>(() => {
    const items = [...(logs ?? [])].sort((a, b) => (b.created ?? '').localeCompare(a.created ?? ''));
    const WINDOW_MS = 60_000;
    const groups: SearchLogEntry[][] = [];
    for (const it of items) {
      const last = groups[groups.length - 1];
      if (
        last &&
        (last[0].query ?? '') === (it.query ?? '') &&
        new Date(last[0].created ?? 0).getTime() - new Date(it.created ?? 0).getTime() <= WINDOW_MS
      ) {
        last.push(it);
      } else {
        groups.push([it]);
      }
    }
    return groups;
  }, [logs]);

  const logColumns = [
    {
      title: t('page.searchOps.logTime'),
      dataIndex: 'created',
      width: 175,
      render: (v?: string) => (v ? v.replace('T', ' ').slice(0, 19) : '—')
    },
    {
      title: t('page.searchOps.logQuery'),
      dataIndex: 'raw',
      ellipsis: true,
      render: (v: string | undefined, r: SearchLogEntry & { _group?: SearchLogEntry[] }) => {
        const text = v || r.query;
        // the aggregation key is the normalized form; when the original
        // input differs (case/spacing), surface it on hover
        const label = v && v !== r.query ? (
          <Tooltip title={`Query: ${r.query}`}>
            <span>{text}</span>
          </Tooltip>
        ) : (
          text
        );
        const group = r._group ?? [];
        return (
          <Space size={4}>
            {label}
            {group.length > 1 ? (
              <Tooltip title={t('page.searchOps.logExecutions')}>
                <Tag color="blue">×{group.length}</Tag>
              </Tooltip>
            ) : null}
            {group.some(x => x.error) ? <Tag color="red">{t('page.searchOps.logError')}</Tag> : null}
          </Space>
        );
      }
    },
    { title: t('page.searchOps.strategy'), dataIndex: 'search_type', width: 120, render: (v: string) => <Tag>{v}</Tag> },
    {
      title: t('page.searchOps.logHits'),
      dataIndex: 'total',
      width: 80,
      render: (v: number, r: SearchLogEntry) => {
        if (r.error) return <Typography.Text type="secondary">—</Typography.Text>;
        return r.zero_hit ? <Typography.Text type="danger">0</Typography.Text> : v;
      }
    },
    {
      title: t('page.searchOps.logTook'),
      dataIndex: 'took_ms',
      width: 90,
      render: (v: number, r: SearchLogEntry) => {
        const legs = [r.leg_keyword_ms, r.leg_semantic_ms, r.leg_rewrite_ms].some(x => !!x);
        const text = <Typography.Text type={v >= 1000 ? 'danger' : undefined}>{v} ms</Typography.Text>;
        if (!legs) return text;
        return (
          <Tooltip title={`${t('page.searchOps.logLegKeyword')} ${r.leg_keyword_ms ?? 0} / ${t('page.searchOps.logLegSemantic')} ${r.leg_semantic_ms ?? 0} / ${t('page.searchOps.logLegRewrite')} ${r.leg_rewrite_ms ?? 0} ms`}>
            <span>{text}</span>
          </Tooltip>
        );
      }
    },
    {
      title: t('page.searchOps.logRewritten'),
      dataIndex: 'rewritten',
      width: 100,
      render: (v: boolean) => (v ? <Tag color="purple">{t('page.searchOps.logRewritten')}</Tag> : null)
    },
    {
      title: t('page.searchOps.logUser'),
      dataIndex: 'user_id',
      width: 120,
      ellipsis: true,
      // show the resolved account name; the raw id is the fallback (legacy rows
      // whose user_id no longer matches an account)
      render: (v: string | undefined, r: SearchLogEntry) => r.user_name || v || '—'
    }
  ];

  const items: { component: any; key: string; label: string }[] = [
    {
      key: 'overview',
      label: t('page.searchOps.tabOverview'),
      component: (
        <div className="flex flex-col gap-16px">
          <Card
            title={
              <span className="inline-flex items-center gap-8px">
                <Activity className="w-18px h-18px" />
                {t('page.searchOps.tabOverview')}
              </span>
            }
            extra={
              <Button size="small" loading={loading} disabled={!permissions.view} onClick={loadOverview}>
                {t('page.searchOps.refresh')}
              </Button>
            }
          >
            <Row gutter={16} className="mt-4px">
              <Col span={6}>
                <Statistic title={t('page.searchOps.totalSearches')} value={data?.total_searches ?? 0} prefix={<Search className="w-14px h-14px mr-4px" />} />
              </Col>
              <Col span={6}>
                <Statistic
                  title={t('page.searchOps.zeroHitRate')}
                  value={data ? (data.zero_hit_rate * 100).toFixed(1) : '0'}
                  suffix="%"
                  prefix={<AlertTriangle className="w-14px h-14px mr-4px" />}
                  valueStyle={{ color: (data?.zero_hit_rate ?? 0) > 0.3 ? '#cf1322' : undefined }}
                />
              </Col>
              <Col span={6}>
                <Statistic title={t('page.searchOps.avgTook')} value={data?.avg_took_ms ?? 0} suffix="ms" prefix={<Timer className="w-14px h-14px mr-4px" />} />
              </Col>
              <Col span={6}>
                <Statistic title={t('page.searchOps.maxTook')} value={data?.max_took_ms ?? 0} suffix="ms" />
              </Col>
            </Row>
            <Typography.Text type="secondary" className="text-12px block mt-8px">
              {t('page.searchOps.rewriteHint', {
                total: String(data?.rewritten_searches ?? 0),
                missed: String(data?.rewritten_zero_hit_searches ?? 0)
              })}
            </Typography.Text>
            {data?.latency_p95_ms !== undefined ? (
              <Typography.Text type="secondary" className="text-12px block mt-4px">
                {t('page.searchOps.latencyPercentiles')}: {data.latency_p50_ms ?? 0} / {data.latency_p95_ms ?? 0} / {data.latency_p99_ms ?? 0} ms
              </Typography.Text>
            ) : null}
            {data?.signals_caveat ? (
              <Alert className="mt-8px" type="warning" showIcon message={t('page.searchOps.zeroHitCaveat')} />
            ) : null}
          </Card>

          <Row gutter={16}>
            <Col span={10}>
              {/* h-full: the two cards in this row share the taller one's height */}
              <Card size="small" className="h-full" title={t('page.searchOps.strategies')}>
                <Table<SearchOpsRow>
                  rowKey="type"
                  size="small"
                  pagination={false}
                  loading={loading}
                  columns={strategyColumns}
                  dataSource={data?.strategies ?? []}
                  locale={{ emptyText: t('page.searchOps.noData') }}
                />
              </Card>
            </Col>
            <Col span={14}>
              <Card size="small" className="h-full" title={t('page.searchOps.lowRecall')}>
                <Table<SearchOpsLowMiss>
                  rowKey="query"
                  size="small"
                  pagination={{ pageSize: 10, hideOnSinglePage: true }}
                  loading={loading}
                  columns={lowRecallColumns}
                  dataSource={data?.low_recall ?? []}
                  locale={{ emptyText: t('page.searchOps.noData') }}
                />
                <Typography.Text type="secondary" className="text-12px">
                  {t('page.searchOps.lowRecallHint')}
                </Typography.Text>
              </Card>
            </Col>
          </Row>

          <Card size="small" title={t('page.searchOps.indexHealth')}>
            <Table<IndexHealthEntry>
              rowKey="name"
              size="small"
              pagination={false}
              loading={loading}
              columns={healthColumns}
              dataSource={health ?? []}
              locale={{ emptyText: t('page.searchOps.noData') }}
            />
          </Card>
        </div>
      )
    },
    {
      key: 'logs',
      label: t('page.searchOps.queryLogs'),
      component: (
        <Card
          size="small"
          title={t('page.searchOps.queryLogs')}
          extra={
            <Space size={8} wrap>
              <Segmented
                size="small"
                value={logFilter}
                onChange={v => setLogFilter(v as 'all' | 'zero' | 'slow')}
                options={[
                  { label: t('page.searchOps.filterAll'), value: 'all' },
                  { label: t('page.searchOps.filterZeroHit'), value: 'zero' },
                  { label: t('page.searchOps.filterSlow'), value: 'slow' }
                ]}
              />
              <Input.Search
                size="small"
                allowClear
                placeholder={t('page.searchOps.logSearchPlaceholder')}
                style={{ width: 200 }}
                onSearch={v => setLogQ(v)}
              />
              <Button size="small" onClick={loadLogs}>
                {t('page.searchOps.refresh')}
              </Button>
            </Space>
          }
        >
          <Typography.Text type="secondary" className="text-12px">
            {t('page.searchOps.queryLogsHint')}
          </Typography.Text>
          <Table<SearchLogEntry>
            className="mt-8px"
            rowKey="id"
            size="small"
            loading={logsLoading}
            pagination={{
              pageSize: logPageSize,
              showSizeChanger: true,
              pageSizeOptions: [10, 20, 50, 100],
              onChange: (_page, size) => {
                if (size !== logPageSize) setLogPageSize(size);
              }
            }}
            columns={logColumns}
            dataSource={logGroups.map(g => ({ ...g[0], _group: g }))}
            locale={{ emptyText: t('page.searchOps.noData') }}
            expandable={{
              // the expanded area lists every execution in the group with its
              // own diagnostics: leg timings, engine took, top hit and failures
              rowExpandable: r => {
                const group = (r as SearchLogEntry & { _group?: SearchLogEntry[] })._group ?? [r];
                return group.length > 1 || group.some(x => !!x.error || !!x.top_hit || [x.leg_keyword_ms, x.leg_semantic_ms, x.leg_rewrite_ms, x.engine_took_ms].some(y => !!y));
              },
              expandedRowRender: r => {
                const group = (r as SearchLogEntry & { _group?: SearchLogEntry[] })._group ?? [r];
                return (
                  <div className="text-12px flex flex-col gap-6px">
                    {group.map(x => (
                      <div key={x.id} className="flex flex-col gap-2px border-l-2 border-solid border-blue-300 pl-8px">
                        <div className="text-gray-500">
                          {(x.created ?? '').replace('T', ' ').slice(0, 19)} · {x.search_type} ·{' '}
                          {x.error ? (
                            <Typography.Text type="danger">{t('page.searchOps.logError')}</Typography.Text>
                          ) : x.zero_hit ? (
                            <Typography.Text type="danger">0</Typography.Text>
                          ) : (
                            x.total
                          )}{' '}
                          · {x.took_ms} ms
                        </div>
                        {!!x.error && (
                          <div>
                            {t('page.searchOps.logError')}: <Typography.Text type="danger">{x.error}</Typography.Text>
                          </div>
                        )}
                        {[x.leg_keyword_ms, x.leg_semantic_ms, x.leg_rewrite_ms, x.engine_took_ms].some(y => !!y) ? (
                          <div>
                            {t('page.searchOps.logLegKeyword')}: {x.leg_keyword_ms ?? 0} ms · {t('page.searchOps.logLegSemantic')}: {x.leg_semantic_ms ?? 0} ms · {t('page.searchOps.logLegRewrite')}: {x.leg_rewrite_ms ?? 0} ms · {t('page.searchOps.logEngineTook')}: {x.engine_took_ms ?? 0} ms
                          </div>
                        ) : null}
                        {x.top_hit ? (
                          <div>
                            {t('page.searchOps.logTopHit')}: {x.top_hit}
                            {x.top_score ? ` (score ${x.top_score.toFixed(4)})` : ''}
                          </div>
                        ) : null}
                      </div>
                    ))}
                  </div>
                );
              }
            }}
          />
        </Card>
      )
    },
    {
      key: 'ingestion',
      label: t('page.searchOps.ingestionTitle'),
      component: (
        <>
          <Card
            size="small"
            title={
              <span className="inline-flex items-center gap-8px">
                <DatabaseZap className="w-16px h-16px" />
                {t('page.searchOps.ingestionTitle')}
              </span>
            }
            extra={
              <Space size={12}>
                <span className="inline-flex items-center gap-4px text-12px">
                  {t('page.searchOps.autoRefresh')}
                  <Switch checked={autoRefresh} size="small" onChange={setAutoRefresh} />
                </span>
                <Button size="small" danger loading={retrying} disabled={!processing?.statuses?.failed} onClick={retryFailed}>
                  {t('page.searchOps.retryFailed')}
                </Button>
              </Space>
            }
          >
            <Typography.Text type="secondary" className="text-12px">
              {t('page.searchOps.ingestionSubtitle')}
            </Typography.Text>
            <Row gutter={16} className="mt-12px">
              <Col flex={1}>
                <Statistic title={t('page.searchOps.statusIndexing')} value={processing?.statuses?.indexing ?? 0} />
              </Col>
              <Col flex={1}>
                <Statistic title={t('page.searchOps.statusCompleted')} value={processing?.statuses?.completed ?? 0} />
              </Col>
              <Col flex={1}>
                <Statistic
                  title={t('page.searchOps.statusFailed')}
                  value={processing?.statuses?.failed ?? 0}
                  valueStyle={{ color: (processing?.statuses?.failed ?? 0) > 0 ? '#cf1322' : undefined }}
                />
              </Col>
              <Col flex={1}>
                <Statistic title={t('page.searchOps.statusLegacy')} value={processing?.statuses?.legacy ?? 0} />
              </Col>
              {processing?.live ? (
                <Col flex={1}>
                  <Statistic
                    title={t('page.searchOps.oldestIndexing')}
                    value={oldestIndexingAge() ?? '—'}
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
              ) : null}
            </Row>
            <Typography.Text type="secondary" className="text-12px block mt-8px">
              {t('page.searchOps.failedTitle')}
            </Typography.Text>
            <Table<ProcessingFailedDoc>
              rowKey="id"
              size="small"
              pagination={{ pageSize: 5, hideOnSinglePage: true }}
              loading={loading}
              columns={failedColumns}
              dataSource={processing?.failed ?? []}
              locale={{ emptyText: t('page.searchOps.noData') }}
              onRow={record => ({
                onClick: () => setDetailDoc(record),
                style: { cursor: 'pointer' }
              })}
            />
          </Card>
        </>
      )
    },
    {
      key: 'sync',
      label: t('page.searchOps.syncTitle'),
      component: (
        <Card size="small" title={t('page.searchOps.syncTitle')}>
          <Table<SyncStatusItem>
            rowKey="id"
            size="small"
            pagination={{ pageSize: 10, hideOnSinglePage: true }}
            loading={loading}
            columns={syncColumns}
            dataSource={syncStatus ?? []}
            locale={{ emptyText: t('page.searchOps.noData') }}
          />
        </Card>
      )
    }
  ];

  const activeItem = items.find(item => item.key === activeKey);

  return (
    <div className="pb-16px">
    <Card
      styles={{ body: { padding: 0 } }}
      title={
        <PageHeaderTitle
          description={t('page.searchOps.subtitle')}
          icon={<MonitorOutlined />}
          title={t('page.searchOps.title')}
        />
      }
    >
      <Tabs
        className="settings-tabs"
        activeKey={activeKey}
        items={items}
        onChange={key => setSearchParams({ tab: key })}
      />
      <div className="settings-tabs-content">{activeItem?.component}</div>
      <Drawer
        open={!!detailDoc}
        title={detailDoc?.title || t('page.searchOps.processingDetail')}
        width={560}
        onClose={() => setDetailDoc(null)}
        destroyOnHidden
      >
        {detailDoc ? <ProcessingPanel docId={detailDoc.id} /> : null}
      </Drawer>
    </Card>
    </div>
  );
}
