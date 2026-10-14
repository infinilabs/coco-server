import { Activity, Cpu, Database, FolderTree, FileType2, HardDrive, Languages, Layers, Gauge, Tags, TrendingUp } from 'lucide-react';
import { Card, Col, Progress, Row, Space, Tag, Tooltip, Typography } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { LucideIcon } from 'lucide-react';

import { request } from '@/service/request';
import { fetchDatasourceCapabilities } from '@/service/api/data-source';

/**
 * 数据源深度统计 tab（数据通道全部走 GET /document/_search，framework
 * 查询构造器原生支持 URL 传聚合：agg[name][type][key]=value 深合并）。
 * 只需 document:search 权限（与文件列表同门控），不依赖 search:ops。
 * 维度：入库趋势(created date_histogram)、类型/生命周期/语言/向量模型/
 * 分类/高频标签(terms)、存储量(sum size)、向量化覆盖(_exists_:embedding_model 计数)。
 */

interface Props {
  id: string;
}

interface Bucket {
  key: string;
  key_as_string?: string;
  doc_count: number;
}

interface AggState {
  total: number;
  total_size: number;
  avg_size: number;
  by_type: Bucket[];
  by_status: Bucket[];
  by_lang: Bucket[];
  by_tag: Bucket[];
  by_cat: Bucket[];
  by_model: Bucket[];
  by_day: Bucket[];
}

const formatSize = (size?: number) => {
  if (size == null || size <= 0) return '-';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`;
  return `${(size / 1024 / 1024 / 1024).toFixed(2)} GB`;
};

// URLSearchParams 会把 [] 编码成 %5B%5D，framework 侧 url.Query() 解码后深合并 —— 与手工 curl 验证一致
const buildAggQuery = (id: string) => {
  const p = new URLSearchParams();
  p.set('size', '1');
  p.set('filter', `source.id:any(${id})`);
  const addAgg = (name: string, type: string, kv: Record<string, string>) => {
    Object.entries(kv).forEach(([k, v]) => p.set(`agg[${name}][${type}][${k}]`, v));
  };
  addAgg('by_type', 'terms', { field: 'type', size: '10' });
  addAgg('by_status', 'terms', { field: 'status', size: '10' });
  addAgg('by_lang', 'terms', { field: 'lang', size: '10' });
  addAgg('by_tag', 'terms', { field: 'tags', size: '10' });
  addAgg('by_cat', 'terms', { field: 'category', size: '8' });
  addAgg('by_model', 'terms', { field: 'embedding_model', size: '5' });
  addAgg('by_day', 'date_histogram', { field: 'created', interval: '1d', format: 'yyyy-MM-dd' });
  addAgg('total_size', 'sum', { field: 'size' });
  addAgg('avg_size', 'avg', { field: 'size' });
  return p.toString();
};

/** 卡片标题：主色小图标 + 文案 */
function cardTitle(Icon: LucideIcon, label: string) {
  return (
    <span className="flex items-center gap-6px text-13px">
      <Icon className="h-14px w-14px color-[var(--ant-color-primary)]" />
      {label}
    </span>
  );
}

/** 概览小卡：彩色图标块 + label/value（+ 右侧附加区如覆盖率圆环） */
function StatCard({ icon, label, value, extra }: { icon: React.ReactNode; label: string; value: string; extra?: React.ReactNode }) {
  return (
    <Card size="small" className="h-full transition-shadow hover:shadow-[0_2px_8px_rgba(0,0,0,0.08)]">
      <div className="flex items-center gap-12px">
        <div className="flex h-36px w-36px shrink-0 items-center justify-center rounded-lg bg-[var(--ant-color-primary-bg)] text-[var(--ant-color-primary)]">
          {icon}
        </div>
        <div className="min-w-0">
          <div className="text-12px color-[var(--ant-color-text-secondary)]">{label}</div>
          <div className="truncate text-20px font-medium leading-tight tabular-nums" title={value}>
            {value}
          </div>
        </div>
        {extra ? <div className="ml-auto shrink-0">{extra}</div> : null}
      </div>
    </Card>
  );
}

function BarList({ items, empty }: { items: Bucket[]; empty: string }) {
  if (!items.length) return <Typography.Text type="secondary">{empty}</Typography.Text>;
  const max = Math.max(...items.map(b => b.doc_count), 1);
  const total = items.reduce((s, b) => s + b.doc_count, 0);
  return (
    <div className="flex flex-col">
      {items.map(b => {
        const pct = total ? Math.round((b.doc_count / total) * 100) : 0;
        return (
          <div
            key={b.key}
            className="-mx-6px flex items-center gap-8px rounded-sm px-6px py-3px text-12px transition-colors hover:bg-[var(--ant-color-fill-tertiary)]"
            title={`${b.key || '-'}: ${b.doc_count} (${pct}%)`}
          >
            <span className="w-100px shrink-0 truncate text-right" title={b.key}>
              {b.key || '-'}
            </span>
            <div className="h-12px min-w-0 flex-1 overflow-hidden rounded-full bg-[var(--ant-color-fill-tertiary)]">
              <div
                className="h-full rounded-full bg-[var(--ant-color-primary)] opacity-80 transition-opacity hover:opacity-100"
                style={{ width: `${Math.max((b.doc_count / max) * 100, 3)}%` }}
              />
            </div>
            <span className="w-44px shrink-0 text-right font-medium tabular-nums">{b.doc_count}</span>
          </div>
        );
      })}
    </div>
  );
}

function TrendChart({ buckets, empty }: { buckets: Bucket[]; empty: string }) {
  const [hoverIdx, setHoverIdx] = useState<number | null>(null);
  if (buckets.length <= 1) {
    // 单日全量入库的源没有趋势可看
    return <Typography.Text type="secondary">{empty}</Typography.Text>;
  }
  const max = Math.max(...buckets.map(b => b.doc_count), 1);
  // Y 轴上限取「好看」的整数（1/2/5 × 10^k），让网格线刻度可读
  const nice = (v: number) => {
    if (v <= 5) return v;
    const p = Math.pow(10, Math.floor(Math.log10(v)));
    const n = v / p;
    const m = n <= 1 ? 1 : n <= 2 ? 2 : n <= 5 ? 5 : 10;
    return m * p;
  };
  const top = nice(max);
  const dayLabel = (b: Bucket) => (b.key_as_string || String(b.key)).slice(5); // MM-dd
  const first = buckets[0];
  const last = buckets[buckets.length - 1];
  const mid = buckets.length > 2 ? buckets[Math.floor(buckets.length / 2)] : null;
  // 首尾柱的浮层靠边收进图内，避免溢出卡片
  const bubbleAlign = (i: number) =>
    i === 0 ? 'left-0' : i === buckets.length - 1 ? 'right-0' : 'left-1/2 -translate-x-1/2';
  return (
    <div>
      <div className="flex gap-8px">
        {/* Y 轴刻度：top / top/2 / 0，与网格线位置对齐 */}
        <div className="flex h-120px w-44px shrink-0 flex-col items-end justify-between text-11px tabular-nums color-[var(--ant-color-text-tertiary)]">
          <span>{top}</span>
          <span>{Math.round(top / 2)}</span>
          <span>0</span>
        </div>
        <div className="relative min-w-0 flex-1">
          <div className="absolute inset-x-0 top-0 border-t border-dashed border-[var(--ant-color-border-secondary)]" />
          <div className="absolute inset-x-0 top-1/2 border-t border-dashed border-[var(--ant-color-border-secondary)]" />
          <div className="absolute inset-x-0 bottom-0 border-t border-[var(--ant-color-split)]" />
          <div className="relative flex h-120px items-end gap-2px">
            {buckets.map((b, i) => {
              const h = Math.max((b.doc_count / top) * 100, 2);
              return (
                <div
                  key={b.key}
                  className="relative flex h-full min-w-3px flex-1 items-end"
                  onMouseEnter={() => setHoverIdx(i)}
                  onMouseLeave={() => setHoverIdx(null)}
                >
                  {hoverIdx === i && (
                    <div
                      className={`pointer-events-none absolute -top-4px z-10 whitespace-nowrap rounded-sm bg-[var(--ant-color-bg-spotlight)] px-6px py-2px text-11px text-white shadow-sm ${bubbleAlign(i)}`}
                    >
                      {dayLabel(b)}: {b.doc_count}
                    </div>
                  )}
                  <div
                    className={`w-full rounded-t-sm bg-[var(--ant-color-primary)] transition-opacity ${hoverIdx === i ? 'opacity-100' : 'opacity-80'}`}
                    style={{ height: `${h}%` }}
                  />
                </div>
              );
            })}
          </div>
        </div>
      </div>
      {/* X 轴：首/中/尾日期 */}
      <div className="mt-6px flex gap-8px">
        <div className="w-44px shrink-0" />
        <div className="flex min-w-0 flex-1 items-center justify-between text-11px color-[var(--ant-color-text-tertiary)]">
          <span>{dayLabel(first)}</span>
          {mid ? <span>{dayLabel(mid)}</span> : null}
          <span>{dayLabel(last)}</span>
        </div>
      </div>
    </div>
  );
}

export default function StatsTab({ id }: Props) {
  const { t } = useTranslation();
  const [agg, setAgg] = useState<AggState | null>(null);
  const [vectorized, setVectorized] = useState(0);
  const [caps, setCaps] = useState<{ documents: number; vector: boolean; faq: boolean; wiki: boolean } | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const res: any = await request({ method: 'get', url: `/document/_search?${buildAggQuery(id)}` }).catch(() => null);
      const a = res?.data?.aggregations;
      if (a) {
        const pick = (name: string) => (a[name]?.buckets as Bucket[]) ?? [];
        setAgg({
          total: res?.data?.hits?.total?.value ?? 0,
          total_size: a.total_size?.value ?? 0,
          avg_size: a.avg_size?.value ?? 0,
          by_type: pick('by_type'),
          by_status: pick('by_status'),
          by_lang: pick('by_lang'),
          by_tag: pick('by_tag'),
          by_cat: pick('by_cat'),
          by_model: pick('by_model'),
          by_day: (pick('by_day') as Bucket[]).slice().sort((x, y) => (x.key < y.key ? -1 : 1))
        });
      }
      // 向量化覆盖：exists 计数单独一发（聚合族没有 exists 语义）
      const vecRes: any = await request({
        method: 'get',
        url: `/document/_search?size=1&filter=${encodeURIComponent(`source.id:any(${id})`)}&query=${encodeURIComponent('_exists_:embedding_model')}`
      }).catch(() => null);
      setVectorized(vecRes?.data?.hits?.total?.value ?? 0);
      // 能力标签（自顶部卡片移入）：vector/faq/wiki 属分析信息
      const capsRes: any = await fetchDatasourceCapabilities(id).catch(() => null);
      setCaps(capsRes?.data ?? null);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  const empty = t('page.searchOps.noData');
  const coverage = agg?.total ? Math.round((vectorized / agg.total) * 100) : 0;

  return (
    <div className="flex flex-col gap-16px p-16px">
      {/* 概览四格：图标小卡 */}
      <Row gutter={[16, 16]}>
        <Col span={6}>
          <StatCard icon={<Database className="h-18px w-18px" />} label={t('page.datasource.stats.totalDocs')} value={String(agg?.total ?? 0)} />
        </Col>
        <Col span={6}>
          <StatCard icon={<HardDrive className="h-18px w-18px" />} label={t('page.datasource.stats.storage')} value={formatSize(agg?.total_size)} />
        </Col>
        <Col span={6}>
          <StatCard icon={<Layers className="h-18px w-18px" />} label={t('page.datasource.stats.avgSize')} value={formatSize(agg?.avg_size)} />
        </Col>
        <Col span={6}>
          <StatCard
            icon={<Gauge className="h-18px w-18px" />}
            label={t('page.datasource.stats.coverage')}
            value={`${vectorized}/${agg?.total ?? 0}`}
            extra={<Progress type="circle" size={44} percent={coverage} />}
          />
        </Col>
      </Row>

      {caps && (
        <Space size={8} wrap className="-mt-4px">
          <Tooltip title={t('page.datasource.stats.vectorTip')}>
            <Tag color={caps.vector ? 'green' : 'default'}>
              {caps.vector ? t('page.datasource.stats.vectorized') : t('page.datasource.stats.notVectorized')}
            </Tag>
          </Tooltip>
          {caps.faq ? <Tag color="blue">{t('page.datasource.stats.hasFaq')}</Tag> : null}
          {caps.wiki ? <Tag color="purple">{t('page.datasource.stats.wikiBound')}</Tag> : null}
        </Space>
      )}

      <Card size="small" title={cardTitle(TrendingUp, t('page.datasource.stats.trend'))}>
        <TrendChart buckets={agg?.by_day ?? []} empty={t('page.datasource.stats.trendEmpty')} />
      </Card>

      <Row gutter={[16, 16]}>
        <Col span={8}>
          <Card size="small" title={cardTitle(FileType2, t('page.datasource.stats.byType'))}>
            <BarList items={agg?.by_type ?? []} empty={empty} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title={cardTitle(Activity, t('page.datasource.stats.byStatus'))}>
            <BarList items={agg?.by_status ?? []} empty={empty} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title={cardTitle(Languages, t('page.datasource.stats.byLang'))}>
            <BarList items={agg?.by_lang ?? []} empty={empty} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col span={8}>
          <Card size="small" title={cardTitle(FolderTree, t('page.datasource.stats.byCat'))}>
            <BarList items={agg?.by_cat ?? []} empty={empty} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title={cardTitle(Cpu, t('page.datasource.stats.byModel'))}>
            <BarList items={agg?.by_model ?? []} empty={empty} />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title={cardTitle(Tags, t('page.datasource.stats.byTag'))}>
            <BarList items={agg?.by_tag ?? []} empty={empty} />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
