import { Activity, AlertTriangle, DatabaseZap, Search, Timer } from 'lucide-react';
import { Alert, Button, Card, Col, Modal, Row, Space, Statistic, Table, Tag, Tooltip, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  fetchIndexHealth,
  fetchProcessingOverview,
  fetchSearchOpsOverview,
  fetchSyncStatus,
  retryFailedDocs,
  IndexHealthEntry,
  ProcessingFailedDoc,
  ProcessingOverview,
  SearchOpsLowMiss,
  SearchOpsOverview,
  SearchOpsRow,
  SyncStatusItem
} from '@/service/api/search-ops';

/**
 * Search operations overview (P2): what people searched, which strategies
 * ran, where recall came up empty and what it cost in latency. The
 * low-recall board is the front half of the knowledge-gap loop — queries
 * that keep missing file knowledge-gap proposals in the governance queue.
 */
export function Component() {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    view: hasAuth('coco#search/ops')
  };

  const [data, setData] = useState<SearchOpsOverview | null>(null);
  const [health, setHealth] = useState<IndexHealthEntry[] | null>(null);
  const [processing, setProcessing] = useState<ProcessingOverview | null>(null);
  const [syncStatus, setSyncStatus] = useState<SyncStatusItem[] | null>(null);
  const [retrying, setRetrying] = useState(false);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res: any = await fetchSearchOpsOverview();
      setData((res?.data as SearchOpsOverview) ?? null);
      const healthRes: any = await fetchIndexHealth();
      setHealth((healthRes?.data?.indices as IndexHealthEntry[]) ?? null);
      // ingestion tasks (W13a) — failures are the actionable half
      const procRes: any = await fetchProcessingOverview();
      setProcessing((procRes?.data as ProcessingOverview) ?? null);
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
          await load();
        } catch (e: any) {
          message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchOps.loadFailed'));
        } finally {
          setRetrying(false);
        }
      }
    });
  }, [t, load]);

  useEffect(() => {
    if (permissions.view) load();
  }, []);

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
    if (item.run.state === 'superseded') return <Tag color="orange">{t('page.searchOps.syncSuperseded')}</Tag>;
    return <Tag>{item.run.state}</Tag>;
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

  return (
    <div className="flex flex-col gap-16px">
      <Card
        title={
          <span className="inline-flex items-center gap-8px">
            <Activity className="w-18px h-18px" />
            {t('page.searchOps.title')}
          </span>
        }
        extra={
          <Button size="small" loading={loading} disabled={!permissions.view} onClick={load}>
            {t('page.searchOps.refresh')}
          </Button>
        }
      >
        <Typography.Text type="secondary" className="text-12px">
          {t('page.searchOps.subtitle')}
        </Typography.Text>
        <Row gutter={16} className="mt-16px">
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
        {data?.signals_caveat ? (
          <Alert className="mt-8px" type="warning" showIcon message={t('page.searchOps.zeroHitCaveat')} />
        ) : null}
      </Card>

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

      <Row gutter={16}>
        <Col span={10}>
          <Card size="small" title={t('page.searchOps.strategies')}>
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
          <Card size="small" title={t('page.searchOps.lowRecall')}>
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

      <Card
        size="small"
        title={
          <span className="inline-flex items-center gap-8px">
            <DatabaseZap className="w-16px h-16px" />
            {t('page.searchOps.ingestionTitle')}
          </span>
        }
        extra={
          <Button size="small" danger loading={retrying} disabled={!processing?.statuses?.failed} onClick={retryFailed}>
            {t('page.searchOps.retryFailed')}
          </Button>
        }
      >
        <Typography.Text type="secondary" className="text-12px">
          {t('page.searchOps.ingestionSubtitle')}
        </Typography.Text>
        <Row gutter={16} className="mt-12px">
          <Col span={6}>
            <Statistic title={t('page.searchOps.statusIndexing')} value={processing?.statuses?.indexing ?? 0} />
          </Col>
          <Col span={6}>
            <Statistic title={t('page.searchOps.statusCompleted')} value={processing?.statuses?.completed ?? 0} />
          </Col>
          <Col span={6}>
            <Statistic
              title={t('page.searchOps.statusFailed')}
              value={processing?.statuses?.failed ?? 0}
              valueStyle={{ color: (processing?.statuses?.failed ?? 0) > 0 ? '#cf1322' : undefined }}
            />
          </Col>
          <Col span={6}>
            <Statistic title={t('page.searchOps.statusLegacy')} value={processing?.statuses?.legacy ?? 0} />
          </Col>
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
        />
      </Card>

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
    </div>
  );
}
