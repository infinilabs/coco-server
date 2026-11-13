import { Database, RefreshCw } from 'lucide-react';
import { Button, Card, Collapse, Modal, Table, Tag, Tooltip, message } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import InfiniIcon from '@/components/common/icon';
import { formatESSearchResult } from '@/service/request/es';
import {
  fetchProcessingOverview,
  fetchSyncStatus,
  retryFailedDocs,
  ProcessingFailedDoc,
  ProcessingOverview,
  SyncStatusItem
} from '@/service/api/search-ops';
import {
  fetchDatasourceCapabilities,
  getDatasource,
  reprocessDocument
} from '@/service/api/data-source';
import { getConnectorByIDs } from '@/service/api/connector';

/**
 * 数据源详情顶部卡片 = 基本信息（左）+ 解析统计（右）+ 失败清单（折叠）：
 * - 基本信息：icon/名称/描述/启用状态 + 连接器类型/同步间隔/创建时间（GET /datasource/:id + connector 补全）
 * - 解析统计：生命周期堆叠条（/search/ops/processing?datasource=）+ 文档总数（capabilities）+ 采集 run 状态
 * ops 接口需要 search:ops 权限：403 时统计半边静默降级，基本信息不受影响
 */

interface Props {
  id: string;
}

interface DsInfo {
  name: string;
  description?: string;
  icon?: string;
  tags?: string[];
  enabled: boolean;
  sync?: { enabled?: boolean; interval?: string };
  connector?: { id?: string };
  created?: string;
}

export default function StatsOverview({ id }: Props) {
  const { t } = useTranslation();

  const [ds, setDs] = useState<DsInfo | null>(null);
  const [connectorName, setConnectorName] = useState<string>('');
  const [processing, setProcessing] = useState<ProcessingOverview | null>(null);
  const [caps, setCaps] = useState<{ documents: number; vector: boolean; faq: boolean; wiki: boolean } | null>(null);
  const [syncItem, setSyncItem] = useState<SyncStatusItem | null>(null);
  const [loading, setLoading] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const loadingRef = useRef(false);

  const load = useCallback(async () => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setLoading(true);
    try {
      // 基本信息：只在首次拉取，不参与 20s 轮询；GET /datasource/:id 为 ES 形状（found/_source）
      if (!ds) {
        const dsRes: any = await getDatasource(id).catch(() => null);
        const info = ((dsRes?.data?._source as DsInfo) ?? dsRes?.data ?? null) as DsInfo | null;
        setDs(info);
        if (info?.connector?.id) {
          const connRes: any = await getConnectorByIDs([info.connector.id]).catch(() => null);
          const conn = (formatESSearchResult(connRes?.data)?.data ?? [])[0];
          setConnectorName(conn?.name ?? '');
        }
      }
      const procRes: any = await fetchProcessingOverview(id).catch(() => null);
      setProcessing((procRes?.data as ProcessingOverview) ?? null);
      const syncRes: any = await fetchSyncStatus().catch(() => null);
      const items = (syncRes?.data?.items as SyncStatusItem[]) ?? [];
      setSyncItem(items.find(item => item.id === id) ?? null);
      const capsRes: any = await fetchDatasourceCapabilities(id).catch(() => null);
      setCaps(capsRes?.data ?? null);
    } finally {
      loadingRef.current = false;
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  useEffect(() => {
    load();
    // 20s 轮询：解析进行中/采集运行中时保持进度新鲜；空闲时也仅 3 个轻量请求
    const timer = setInterval(load, 20000);
    return () => clearInterval(timer);
  }, [load]);

  // 生命周期堆叠条数据：比例即状态分布（分母口径 = capabilities 文档总数，缺省用状态合计）
  const failedCount = processing?.statuses?.failed ?? 0;
  const st = processing?.statuses;
  const parts = [
    { key: 'completed', count: st?.completed ?? 0, color: '#389e0d', label: t('page.searchOps.statusCompleted') },
    { key: 'indexing', count: st?.indexing ?? 0, color: '#1677ff', label: t('page.searchOps.statusIndexing') },
    { key: 'failed', count: failedCount, color: '#cf1322', label: t('page.searchOps.statusFailed') },
    { key: 'legacy', count: st?.legacy ?? 0, color: 'var(--ant-color-text-quaternary)', label: t('page.searchOps.statusLegacy') }
  ];
  const barTotal = Math.max(caps?.documents ?? 0, parts.reduce((s, p) => s + p.count, 0), 1);

  const retryFailed = useCallback(() => {
    Modal.confirm({
      title: t('page.searchOps.retryFailed'),
      content: t('page.searchOps.retryConfirm'),
      onOk: async () => {
        setRetrying(true);
        try {
          const res: any = await retryFailedDocs(id);
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
  }, [t, id, load]);

  const retryOne = useCallback(async (docId: string) => {
    try {
      await reprocessDocument(docId);
      message.success(t('page.datasource.stats.retryOneQueued'));
      await load();
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchOps.loadFailed'));
    }
  }, [t, load]);

  const runStateTag = () => {
    const run = syncItem?.run;
    if (!run) return <Tag>{t('page.searchOps.syncNever')}</Tag>;
    if (run.stale) return <Tooltip title={t('page.searchOps.syncStale')}><Tag color="red">{t('page.searchOps.syncStale')}</Tag></Tooltip>;
    if (run.state === 'running') return <Tag color="processing">{t('page.searchOps.syncRunning')}</Tag>;
    if (run.state === 'completed') return <Tag color="green">{t('page.searchOps.syncCompleted')}</Tag>;
    if (run.state === 'superseded') return <Tag color="orange">{t('page.searchOps.syncSuperseded')}</Tag>;
    return <Tag>{run.state}</Tag>;
  };

  const failedColumns = [
    { title: t('page.searchOps.failedDocTitle'), dataIndex: 'title', ellipsis: true },
    { title: t('page.searchOps.failedError'), dataIndex: 'error', ellipsis: true },
    {
      title: t('page.searchOps.failedUpdated'),
      dataIndex: 'updated',
      width: 170,
      render: (v: string) => (v ? new Date(v).toLocaleString() : '-')
    },
    {
      title: t('page.searchOps.syncRunState'),
      key: 'action',
      width: 90,
      render: (_: unknown, record: ProcessingFailedDoc) => (
        <Button size="small" type="link" className="px-0!" onClick={() => retryOne(record.id)}>
          {t('page.datasource.stats.retryOne')}
        </Button>
      )
    }
  ];

  const iconSrc = ds?.icon;
  const metaItems = [
    connectorName ? `${t('page.datasource.stats.connector')} ${connectorName}` : null,
    ds ? `${t('page.datasource.stats.syncLabel')} ${ds.sync?.enabled ? ds.sync?.interval || '-' : t('page.datasource.stats.syncOff')}` : null,
    ds?.created ? `${t('page.datasource.stats.createdAt')} ${new Date(ds.created).toLocaleDateString()}` : null
  ].filter(Boolean) as string[];

  return (
    <Card
      size="small"
      extra={
        failedCount > 0 && (
          <Button size="small" danger loading={retrying} onClick={retryFailed}>
            {t('page.searchOps.retryFailed')}
          </Button>
        )
      }
    >
      {/* 上下布局：数据源信息（上）/ 解析统计 + 采集状态（下） */}
      <div className="flex flex-col gap-12px">
        {/* 上：数据源基本信息 */}
        <div className="flex items-start gap-12px">
          <div className="flex h-44px w-44px shrink-0 items-center justify-center overflow-hidden rounded-lg bg-[var(--ant-color-fill-tertiary)] text-24px">
            {iconSrc ? (
              <InfiniIcon height="1em" src={iconSrc} width="1em" />
            ) : (
              <Database className="h-22px w-22px color-[var(--ant-color-text-tertiary)]" />
            )}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-8px">
              <span className="truncate text-16px font-medium" title={ds?.name}>{ds?.name ?? '-'}</span>
              {ds && (
                <Tag color={ds.enabled ? 'green' : 'default'} className="mr-0!">
                  {ds.enabled ? t('page.datasource.stats.enabled') : t('page.datasource.stats.disabled')}
                </Tag>
              )}
              {ds?.tags?.map(tag => (
                <Tag key={tag} className="mr-0!">{tag}</Tag>
              ))}
            </div>
            {ds?.description ? (
              <div className="mt-2px line-clamp-2 text-12px color-[var(--ant-color-text-secondary)]" title={ds.description}>
                {ds.description}
              </div>
            ) : null}
            {metaItems.length ? (
              <div className="mt-4px flex flex-wrap items-center gap-x-8px gap-y-2px text-12px color-[var(--ant-color-text-tertiary)]">
                {metaItems.map((item, i) => (
                  <span key={item} className="flex items-center gap-8px">
                    {i > 0 && <span className="color-[var(--ant-color-border)]">·</span>}
                    {item}
                  </span>
                ))}
              </div>
            ) : null}
          </div>
        </div>

        {/* 分隔线 */}
        <div className="border-t border-[var(--ant-color-split)]" />

        {/* 下：解析统计（总数锚点 + 全宽生命周期堆叠条 + 图例；解析中段呼吸动效） */}
        <div>
          <div className="flex items-end gap-16px">
            <div className="shrink-0">
              <div className="text-12px color-[var(--ant-color-text-secondary)]">{t('page.datasource.stats.totalDocs')}</div>
              <div className="text-20px font-medium leading-tight tabular-nums">{caps?.documents ?? 0}</div>
            </div>
            <div className="mb-2px min-w-0 flex-1">
              <div className="h-12px flex w-full overflow-hidden rounded-full bg-[var(--ant-color-fill-tertiary)]">
                {parts.map(
                  p =>
                    p.count > 0 && (
                      <div
                        key={p.key}
                        className={p.key === 'indexing' ? 'h-full animate-pulse' : 'h-full'}
                        style={{ width: `${(p.count / barTotal) * 100}%`, minWidth: 3, background: p.color }}
                        title={`${p.label} ${p.count}`}
                      />
                    )
                )}
              </div>
            </div>
          </div>
          <div className="mt-6px flex flex-wrap items-center gap-x-16px gap-y-4px text-12px">
            {parts.map(p => (
              <span key={p.key} className="flex items-center gap-4px">
                <span className="h-8px w-8px shrink-0 rounded-full" style={{ background: p.color }} />
                <span>{p.label}</span>
                <span
                  className='font-medium tabular-nums'
                  style={{ color: p.key === 'failed' && p.count > 0 ? p.color : undefined }}
                >
                  {p.count}
                </span>
              </span>
            ))}
          </div>
        </div>

        {/* 采集侧运行状态；刷新按钮贴着它刷新的数据（统计半边：ops/采集/capabilities） */}
        <div className="flex flex-wrap items-center gap-x-10px gap-y-2px text-12px">
          {syncItem ? (
            <>
              {runStateTag()}
              {syncItem.run ? (
                <span className="color-[var(--ant-color-text-secondary)]">
                  {t('page.searchOps.syncRunProgress', { batches: String(syncItem.run.batches), documents: String(syncItem.run.documents) })}
                </span>
              ) : null}
              <span className="color-[var(--ant-color-text-tertiary)]">
                {t('page.searchOps.syncLastDispatch')}
                {syncItem.last_dispatch ? new Date(syncItem.last_dispatch).toLocaleString() : '-'}
              </span>
            </>
          ) : (
            <span className="color-[var(--ant-color-text-tertiary)]">{t('page.searchOps.syncNever')}</span>
          )}
          <Tooltip title={t('common.refresh')}>
            <Button size="small" type="text" className="h-20px! w-20px! p-0!" icon={<RefreshCw className="h-12px w-12px" />} loading={loading} onClick={load} />
          </Tooltip>
        </div>
      </div>

      {processing?.failed?.length ? (
        <Collapse
          className="mt-12px"
          size="small"
          items={[
            {
              key: 'failed',
              label: `${t('page.searchOps.failedTitle')} (${processing.failed_top})`,
              children: (
                <Table<ProcessingFailedDoc>
                  rowKey="id"
                  size="small"
                  pagination={{ pageSize: 5, hideOnSinglePage: true }}
                  columns={failedColumns}
                  dataSource={processing.failed}
                  locale={{ emptyText: t('page.searchOps.noData') }}
                />
              )
            }
          ]}
        />
      ) : null}
    </Card>
  );
}
