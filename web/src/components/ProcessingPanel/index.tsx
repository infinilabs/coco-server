import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Spin, Table, Tag, Typography } from 'antd';

import { request } from '@/service/request';

/**
 * Processing panel (W13): the artifact face of the ingestion pipeline —
 * the timeline (lifecycle status, last error, embedding model, capped run
 * history) and the chunk list (breadcrumb, page range, excerpt,
 * vectorized bit) for the document being previewed. Read-only: deep
 * actions (reprocess) live on the ops surface.
 */

interface PipelineRun {
  pipeline?: string;
  passthrough?: string;
  took_ms?: number;
  success?: boolean;
  at?: string;
  error?: string;
}

interface TimelineData {
  status?: string;
  processed?: boolean;
  error_message?: string;
  embedding_model?: string;
  pipeline_runs?: PipelineRun[];
}

interface ChunkItem {
  index: number;
  breadcrumb?: string;
  pages?: { start: number; end: number };
  excerpt?: string;
  vectorized?: boolean;
}

export default function ProcessingPanel({ docId, headers }: { docId?: string; headers?: Record<string, string> }) {
  const { t } = useTranslation();
  const [timeline, setTimeline] = useState<TimelineData | null>(null);
  const [chunks, setChunks] = useState<ChunkItem[]>([]);
  const [chunkTotal, setChunkTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    if (!docId) return;
    setLoading(true);
    try {
      const tl: any = await request({ method: 'get', url: `/document/${docId}/_timeline`, headers }).catch(() => null);
      setTimeline((tl?.data as TimelineData) ?? null);
      const ck: any = await request({ method: 'get', url: `/document/${docId}/_chunks?size=50`, headers }).catch(() => null);
      setChunks((ck?.data?.chunks as ChunkItem[]) ?? []);
      setChunkTotal(ck?.data?.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [docId, headers]);

  useEffect(() => {
    load();
  }, [load]);

  const statusTag = (status?: string) => {
    switch (status) {
      case 'indexing':
        return <Tag color="processing">{t('page.preview.processing.statusIndexing')}</Tag>;
      case 'completed':
        return <Tag color="success">{t('page.preview.processing.statusCompleted')}</Tag>;
      case 'failed':
        return <Tag color="error">{t('page.preview.processing.statusFailed')}</Tag>;
      default:
        return <Tag>{t('page.preview.processing.statusLegacy')}</Tag>;
    }
  };

  const chunkColumns = [
    { title: '#', dataIndex: 'index', width: 50 },
    {
      title: t('page.preview.processing.breadcrumb'),
      dataIndex: 'breadcrumb',
      width: 180,
      ellipsis: true,
      render: (v: string) => v || '-'
    },
    {
      title: t('page.preview.processing.pages'),
      key: 'pages',
      width: 90,
      render: (_: unknown, r: ChunkItem) => (r.pages ? `p${r.pages.start}-${r.pages.end}` : '-')
    },
    {
      title: t('page.preview.processing.excerpt'),
      dataIndex: 'excerpt',
      ellipsis: true
    },
    {
      title: t('page.preview.processing.vectorized'),
      dataIndex: 'vectorized',
      width: 90,
      render: (v: boolean) =>
        v ? <Tag color="green">✓</Tag> : <Tag color="default">—</Tag>
    }
  ];

  return (
    <Spin spinning={loading}>
      <div className="flex flex-col gap-12px">
        <div className="flex items-center gap-8px flex-wrap">
          {statusTag(timeline?.status)}
          {timeline?.embedding_model ? (
            <Typography.Text type="secondary" className="text-12px">
              {t('page.preview.processing.model')}: {timeline.embedding_model}
            </Typography.Text>
          ) : null}
        </div>
        {timeline?.error_message ? <Alert type="error" showIcon message={timeline.error_message} /> : null}

        <Typography.Text type="secondary" className="text-12px">
          {t('page.preview.processing.runsTitle')}
        </Typography.Text>
        <div className="flex flex-col gap-4px">
          {(timeline?.pipeline_runs ?? []).length === 0 ? (
            <Typography.Text type="secondary" className="text-12px">
              {t('page.preview.processing.noRuns')}
            </Typography.Text>
          ) : (
            (timeline?.pipeline_runs ?? []).map((run, i) => (
              <div key={i} className="text-12px flex items-center gap-8px flex-wrap">
                <Tag color={run.success ? 'green' : 'red'}>{run.success ? 'ok' : 'fail'}</Tag>
                <span>{run.pipeline || run.passthrough || '-'}</span>
                {run.took_ms ? <span>{run.took_ms}ms</span> : null}
                {run.at ? <span className="text-gray-400">{new Date(run.at).toLocaleString()}</span> : null}
                {run.error ? <span className="text-red-500" style={{ maxWidth: '100%', overflow: 'hidden', textOverflow: 'ellipsis' }}>{run.error}</span> : null}
              </div>
            ))
          )}
        </div>

        <Typography.Text type="secondary" className="text-12px">
          {t('page.preview.processing.chunksTitle', { total: String(chunkTotal) })}
        </Typography.Text>
        <Table<ChunkItem>
          rowKey="index"
          size="small"
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          columns={chunkColumns}
          dataSource={chunks}
          locale={{ emptyText: t('page.searchOps.noData') }}
        />
      </div>
    </Spin>
  );
}
