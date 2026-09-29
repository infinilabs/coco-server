import { Activity, AlertTriangle, Search, Timer } from 'lucide-react';
import { Button, Card, Col, Row, Space, Statistic, Table, Tag, Tooltip, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  fetchIndexHealth,
  fetchSearchOpsOverview,
  IndexHealthEntry,
  SearchOpsLowMiss,
  SearchOpsOverview,
  SearchOpsRow
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
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
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

  useEffect(() => {
    if (permissions.view) load();
  }, []);

  const strategyColumns = [
    { title: t('page.searchOps.strategy'), dataIndex: 'type', width: 140 },
    { title: t('page.searchOps.searches'), dataIndex: 'count', width: 100 },
    { title: t('page.searchOps.zeroHits'), dataIndex: 'zero_hits', width: 110 },
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
    </div>
  );
}
