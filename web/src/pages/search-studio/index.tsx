import { ArrowRight, FlaskConical, RefreshCw, Search } from 'lucide-react';
import { Button, Card, Col, Input, InputNumber, Row, Select, Slider, Space, Spin, Table, Tag, Tooltip, Typography, message } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { fetchDataSourceList } from '@/service/api/data-source';
import {
  SearchStudioFusedHit,
  SearchStudioResult,
  SearchStudioRouteResult,
  testSearchStudio
} from '@/service/api/search-studio';

const fmtScore = (v: number | undefined) => {
  if (v === undefined || v === null) return '-';
  if (v === 0) return '0';
  if (Math.abs(v) >= 1000) return v.toFixed(0);
  return Number(v.toFixed(6)).toString();
};

const fmtMs = (v: number | undefined) => `${v ?? 0} ms`;

// canonical route presentation: name, color, weight key and label key
const ROUTE_META: { name: string; color: string; bg: string; weightKey: string; labelKey: string }[] = [
  { name: 'text', color: 'blue', bg: 'bg-[#1784FC]', weightKey: 'text_weight', labelKey: 'page.searchStudio.textRoute' },
  { name: 'semantic', color: 'purple', bg: 'bg-[#722ED1]', weightKey: 'semantic_weight', labelKey: 'page.searchStudio.semanticRoute' },
  { name: 'wiki', color: 'green', bg: 'bg-[#52C41A]', weightKey: 'wiki_weight', labelKey: 'page.searchStudio.wikiRoute' }
];

export function Component() {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    run: hasAuth('coco#search/studio')
  };

  const [query, setQuery] = useState('');
  const [datasource, setDatasource] = useState<string[]>([]);
  const [size, setSize] = useState(10);
  const [fuzziness, setFuzziness] = useState(3);
  const [rrfK, setRrfK] = useState(60);
  const [weights, setWeights] = useState<Record<string, number>>({ text: 1, semantic: 1, wiki: 1 });
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<SearchStudioResult | null>(null);

  const [datasourceOptions, setDatasourceOptions] = useState<{ label: string; value: string }[]>([]);

  useEffect(() => {
    fetchDataSourceList({ size: 200 }).then((res: any) => {
      const list = res?.data?.hits?.hits ?? [];
      setDatasourceOptions(
        list.map((hit: any) => ({
          label: hit._source?.name ?? hit._id,
          value: hit._id
        }))
      );
    });
  }, []);

  const runTest = useCallback(async () => {
    const trimmed = query.trim();
    if (!trimmed) {
      message.warning(t('page.searchStudio.queryRequired'));
      return;
    }
    setRunning(true);
    try {
      const res: any = await testSearchStudio({
        query: trimmed,
        datasource: datasource.join(','),
        size,
        fuzziness,
        rrf: {
          k: rrfK,
          text_weight: weights.text,
          semantic_weight: weights.semantic,
          wiki_weight: weights.wiki
        }
      });
      setResult((res?.data as SearchStudioResult) ?? null);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchStudio.runFailed'));
    } finally {
      setRunning(false);
    }
  }, [query, datasource, size, fuzziness, rrfK, weights, t]);

  const routeColumns = useMemo(
    () => [
      { title: '#', width: 48, render: (_: any, __: any, index: number) => index + 1 },
      { title: t('page.searchStudio.docTitle'), dataIndex: 'title', ellipsis: true, render: (v: string, r: any) => <Tooltip title={r.id}>{v || r.id}</Tooltip> },
      { title: t('page.searchStudio.docSource'), dataIndex: 'datasource', width: 140, ellipsis: true },
      { title: t('page.searchStudio.rawScore'), dataIndex: 'score', width: 110, render: fmtScore }
    ],
    [t]
  );

  const fusedColumns = useMemo(() => {
    const cols: any[] = [
      { title: '#', width: 44, render: (_: any, __: any, index: number) => index + 1 },
      { title: t('page.searchStudio.docTitle'), dataIndex: 'title', ellipsis: true, render: (v: string, r: SearchStudioFusedHit) => <Tooltip title={r.id}>{v || r.id}</Tooltip> }
    ];
    for (const meta of ROUTE_META) {
      cols.push({
        title: t(meta.labelKey),
        width: 130,
        render: (_: any, r: SearchStudioFusedHit) => (
          <Space size={4}>
            <Tag color={r.ranks?.[meta.name] ? meta.color : 'default'}>#{r.ranks?.[meta.name] || '—'}</Tag>
          </Space>
        )
      });
    }
    cols.push({
      title: t('page.searchStudio.contribution'),
      width: 240,
      render: (_: any, r: SearchStudioFusedHit) => {
        const total = r.score || 1;
        return (
          <Tooltip
            title={ROUTE_META.filter((m) => r.contributions?.[m.name]).map((m) => `${t(m.labelKey)}: ${fmtScore(r.contributions[m.name])}`).join(' · ')}
          >
            <div className="flex items-center gap-2px w-200px">
              {ROUTE_META.filter((m) => (r.contributions?.[m.name] ?? 0) > 0).map((m) => (
                <div key={m.name} className={`h-6px rounded-3px ${m.bg} flex-none`} style={{ width: `${((r.contributions[m.name] ?? 0) / total) * 100}%` }} />
              ))}
            </div>
          </Tooltip>
        );
      }
    });
    cols.push({ title: 'RRF', dataIndex: 'score', width: 120, render: (v: number) => <strong>{fmtScore(v)}</strong> });
    return cols;
  }, [t]);

  const routePanel = (meta: (typeof ROUTE_META)[number], route: SearchStudioRouteResult | undefined) => (
    <Card
      size="small"
      title={
        <Space size={6}>
          <span className={`inline-block w-8px h-8px rounded-4px ${meta.bg}`} />
          {t(meta.labelKey)}
          {route?.route && (
            <Tag color={route.route === 'engine' ? 'geekblue' : route.route === 'client' ? 'cyan' : 'default'}>{route.route}</Tag>
          )}
        </Space>
      }
      extra={route ? <span className="text-12px text-gray-400">{fmtMs(route.took_ms)} · {route.total}</span> : null}
    >
      {route?.error ? (
        <Typography.Text type="danger">{route.error}</Typography.Text>
      ) : route?.note ? (
        <div className="mb-8px">
          <Typography.Text type="secondary" className="text-12px">{route.note}</Typography.Text>
        </div>
      ) : null}
      <Table
        rowKey="id"
        size="small"
        pagination={false}
        scroll={{ y: 360 }}
        columns={routeColumns}
        dataSource={route?.hits ?? []}
        locale={{ emptyText: t('page.searchStudio.noResults') }}
      />
    </Card>
  );

  return (
    <div className="flex flex-col gap-16px">
      <Card
        title={
          <Space size={8}>
            <FlaskConical className="w-18px h-18px" />
            <span>{t('page.searchStudio.title')}</span>
          </Space>
        }
        extra={<Typography.Text type="secondary" className="text-12px">{t('page.searchStudio.subtitle')}</Typography.Text>}
      >
        <Space.Compact style={{ width: '100%' }} className="mb-16px">
          <Input
            size="large"
            allowClear
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onPressEnter={runTest}
            placeholder={t('page.searchStudio.queryPlaceholder')}
            prefix={<Search className="w-16px h-16px text-gray-400" />}
          />
          <Button size="large" type="primary" loading={running} disabled={!permissions.run} onClick={runTest}>
            {t('page.searchStudio.run')}
          </Button>
        </Space.Compact>

        <Row gutter={[16, 8]}>
          <Col span={10}>
            <div className="text-13px mb-4px">{t('page.searchStudio.datasource')}</div>
            <Select
              mode="multiple"
              allowClear
              maxTagCount="responsive"
              style={{ width: '100%' }}
              placeholder={t('page.searchStudio.datasourceAll')}
              value={datasource}
              onChange={setDatasource}
              options={datasourceOptions}
            />
          </Col>
          <Col span={4}>
            <div className="text-13px mb-4px">{t('page.searchStudio.size')}</div>
            <InputNumber min={1} max={50} style={{ width: '100%' }} value={size} onChange={(v) => setSize(v ?? 10)} />
          </Col>
          <Col span={5}>
            <div className="text-13px mb-4px">{t('page.searchStudio.fuzziness')}</div>
            <Slider min={0} max={5} value={fuzziness} onChange={(v) => setFuzziness(v)} />
          </Col>
          <Col span={5}>
            <div className="text-13px mb-4px">RRF k</div>
            <InputNumber min={1} max={1000} style={{ width: '100%' }} value={rrfK} onChange={(v) => setRrfK(v ?? 60)} />
          </Col>
          {ROUTE_META.map((meta) => (
            <Col span={8} key={meta.name}>
              <div className="text-13px mb-4px">
                {t('page.searchStudio.weightOf', { route: t(meta.labelKey) })} <span className={meta.bg.replace('bg-[', 'text-[')}>{weights[meta.name] ?? 1}</span>
              </div>
              <Slider min={0} max={5} step={0.1} value={weights[meta.name] ?? 1} onChange={(v) => setWeights({ ...weights, [meta.name]: v })} />
            </Col>
          ))}
        </Row>

        <Typography.Text type="secondary" className="text-12px">
          {t('page.searchStudio.formula')}
        </Typography.Text>
      </Card>

      <Spin spinning={running}>
        {result ? (
          <div className="flex flex-col gap-16px">
            <Row gutter={16}>
              {ROUTE_META.map((meta) => {
                const route = result.routes?.find((r) => r.name === meta.name);
                return <Col span={8} key={meta.name}>{routePanel(meta, route)}</Col>;
              })}
            </Row>
            <Card
              size="small"
              title={
                <Space size={6}>
                  <ArrowRight className="w-14px h-14px" />
                  {t('page.searchStudio.fusedTitle')}
                  <Tag color="gold">k={result.rrf.k}</Tag>
                  {ROUTE_META.map((meta) => (
                    <Tag key={meta.name} color={meta.color}>
                      w={result.rrf.weights?.[meta.name] ?? 1}
                    </Tag>
                  ))}
                </Space>
              }
              extra={
                <Button size="small" icon={<RefreshCw className="w-12px h-12px" />} onClick={runTest}>
                  {t('page.searchStudio.rerun')}
                </Button>
              }
            >
              <Table
                rowKey="id"
                size="small"
                pagination={false}
                columns={fusedColumns}
                dataSource={result.fused?.hits ?? []}
                locale={{ emptyText: t('page.searchStudio.noResults') }}
              />
            </Card>
          </div>
        ) : (
          <Card>
            <div className="py-48px text-center text-gray-400">{t('page.searchStudio.empty')}</div>
          </Card>
        )}
      </Spin>
    </div>
  );
}
