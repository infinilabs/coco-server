import { ArrowRight, FlaskConical, RefreshCw, Search } from 'lucide-react';
import { Button, Card, Col, Input, InputNumber, Row, Select, Slider, Space, Spin, Table, Tag, Tooltip, Typography, message } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { fetchDataSourceList } from '@/service/api/data-source';
import {
  SearchStudioFusedHit,
  SearchStudioResult,
  testSearchStudio
} from '@/service/api/search-studio';

const fmtScore = (v: number | undefined) => {
  if (v === undefined || v === null) return '-';
  if (v === 0) return '0';
  if (Math.abs(v) >= 1000) return v.toFixed(0);
  return Number(v.toFixed(6)).toString();
};

const fmtMs = (v: number | undefined) => `${v ?? 0} ms`;

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
  const [textWeight, setTextWeight] = useState(1);
  const [semanticWeight, setSemanticWeight] = useState(1);
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
        rrf: { k: rrfK, text_weight: textWeight, semantic_weight: semanticWeight }
      });
      setResult((res?.data as SearchStudioResult) ?? null);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchStudio.runFailed'));
    } finally {
      setRunning(false);
    }
  }, [query, datasource, size, fuzziness, rrfK, textWeight, semanticWeight, t]);

  const routeColumns = useMemo(
    () => [
      { title: '#', width: 48, render: (_: any, __: any, index: number) => index + 1 },
      { title: t('page.searchStudio.docTitle'), dataIndex: 'title', ellipsis: true, render: (v: string, r: any) => <Tooltip title={r.id}>{v || r.id}</Tooltip> },
      { title: t('page.searchStudio.docSource'), dataIndex: 'datasource', width: 140, ellipsis: true },
      { title: t('page.searchStudio.rawScore'), dataIndex: 'score', width: 110, render: fmtScore }
    ],
    [t]
  );

  const fusedColumns = useMemo(
    () => [
      { title: '#', width: 44, render: (_: any, __: any, index: number) => index + 1 },
      { title: t('page.searchStudio.docTitle'), dataIndex: 'title', ellipsis: true, render: (v: string, r: SearchStudioFusedHit) => <Tooltip title={r.id}>{v || r.id}</Tooltip> },
      {
        title: 'BM25',
        width: 150,
        render: (_: any, r: SearchStudioFusedHit) => (
          <Space size={4}>
            <Tag color={r.text_rank ? 'blue' : 'default'}>#{r.text_rank || '—'}</Tag>
            <span className="text-12px">{fmtScore(r.text_score)}</span>
          </Space>
        )
      },
      {
        title: 'kNN',
        width: 150,
        render: (_: any, r: SearchStudioFusedHit) => (
          <Space size={4}>
            <Tag color={r.semantic_rank ? 'purple' : 'default'}>#{r.semantic_rank || '—'}</Tag>
            <span className="text-12px">{fmtScore(r.semantic_score)}</span>
          </Space>
        )
      },
      {
        title: t('page.searchStudio.contribution'),
        width: 220,
        render: (_: any, r: SearchStudioFusedHit) => (
          <Tooltip title={`${t('page.searchStudio.textRoute')}: ${fmtScore(r.text_contribution)} · ${t('page.searchStudio.semanticRoute')}: ${fmtScore(r.semantic_contribution)}`}>
            <div className="flex items-center gap-4px w-180px">
              <div className="h-6px rounded-3px bg-[#1784FC] flex-none" style={{ width: `${(r.text_contribution / (r.score || 1)) * 100}%` }} />
              <div className="h-6px rounded-3px bg-[#722ed1] flex-none" style={{ width: `${(r.semantic_contribution / (r.score || 1)) * 100}%` }} />
            </div>
          </Tooltip>
        )
      },
      { title: 'RRF', dataIndex: 'score', width: 120, render: (v: number) => <strong>{fmtScore(v)}</strong> }
    ],
    [t]
  );

  const routePanel = (title: string, color: string, route: SearchStudioResult['text'] | undefined) => (
    <Card
      size="small"
      title={
        <Space size={6}>
          <span className={`inline-block w-8px h-8px rounded-4px ${color}`} />
          {title}
        </Space>
      }
      extra={route ? <span className="text-12px text-gray-400">{fmtMs(route.took_ms)} · {route.total}</span> : null}
    >
      {route?.error ? (
        <Typography.Text type="danger">{route.error}</Typography.Text>
      ) : (
        <Table
          rowKey="id"
          size="small"
          pagination={false}
          scroll={{ y: 360 }}
          columns={routeColumns}
          dataSource={route?.hits ?? []}
          locale={{ emptyText: t('page.searchStudio.noResults') }}
        />
      )}
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
          <Col span={12}>
            <div className="text-13px mb-4px">
              {t('page.searchStudio.textWeight')} <span className="text-[#1784FC]">{textWeight}</span>
            </div>
            <Slider min={0} max={5} step={0.1} value={textWeight} onChange={(v) => setTextWeight(v)} />
          </Col>
          <Col span={12}>
            <div className="text-13px mb-4px">
              {t('page.searchStudio.semanticWeight')} <span className="text-[#722ed1]">{semanticWeight}</span>
            </div>
            <Slider min={0} max={5} step={0.1} value={semanticWeight} onChange={(v) => setSemanticWeight(v)} />
          </Col>
        </Row>

        <Typography.Text type="secondary" className="text-12px">
          {t('page.searchStudio.formula')}
        </Typography.Text>
      </Card>

      <Spin spinning={running}>
        {result ? (
          <div className="flex flex-col gap-16px">
            <Row gutter={16}>
              <Col span={12}>{routePanel(`BM25 · ${t('page.searchStudio.textRoute')}`, 'bg-[#1784FC]', result.text)}</Col>
              <Col span={12}>{routePanel(`kNN · ${t('page.searchStudio.semanticRoute')}`, 'bg-[#722ed1]', result.semantic)}</Col>
            </Row>
            <Card
              size="small"
              title={
                <Space size={6}>
                  <ArrowRight className="w-14px h-14px" />
                  {t('page.searchStudio.fusedTitle')}
                  <Tag color="gold">k={result.rrf.k}</Tag>
                  <Tag color="blue">w={result.rrf.text_weight}</Tag>
                  <Tag color="purple">w={result.rrf.semantic_weight}</Tag>
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
