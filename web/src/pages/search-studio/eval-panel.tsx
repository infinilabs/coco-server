import { AuditOutlined, DeleteOutlined, EditOutlined, PlayCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Form, Input, Modal, Popconfirm, Select, Space, Statistic, Table, Tag, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { querySearch } from '@/service/api/ai-search';
import {
  createEvalCase,
  deleteEvalCase,
  fetchEvalCases,
  fetchEvalRuns,
  runSearchEval,
  SearchEvalCase,
  SearchEvalRun
} from '@/service/api/search-studio';

/**
 * Golden-query evaluation set (D9): a human-annotated set of query →
 * expected-document pairs, run against the live pipeline. The rule it
 * enforces: no recall-stack change (RRF weights, rerank window, query
 * rewrite) ships without a before/after run — tuning without an evaluation
 * set is guessing with extra steps.
 */
export function EvalPanel({ datasourceOptions }: { datasourceOptions: { label: string; value: string }[] }) {
  const { t } = useTranslation();

  const [cases, setCases] = useState<SearchEvalCase[]>([]);
  const [runs, setRuns] = useState<SearchEvalRun[]>([]);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);

  // add/edit-case modal — createEvalCase upserts on the normalized query,
  // so editing an existing case reuses the same endpoint
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<SearchEvalCase | null>(null);
  const [form] = Form.useForm();
  const [docOptions, setDocOptions] = useState<{ label: string; value: string }[]>([]);
  const [picking, setPicking] = useState(false);
  const [saving, setSaving] = useState(false);

  const [lastRun, setLastRun] = useState<SearchEvalRun | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [caseRes, runRes]: any[] = await Promise.all([fetchEvalCases(), fetchEvalRuns()]);
      setCases((caseRes?.data?.data as SearchEvalCase[]) ?? []);
      setRuns((runRes?.data?.data as SearchEvalRun[]) ?? []);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchStudio.evalLoadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    load();
  }, [load]);

  // search documents to pick expected hits — the annotation surface is the
  // same search the users get
  const searchDocs = useCallback(async (q: string) => {
    if (!q.trim()) return;
    setPicking(true);
    try {
      const res: any = await querySearch({}, `query=${encodeURIComponent(q)}&size=10`);
      const hits = res?.data?.hits?.hits ?? [];
      setDocOptions(
        hits.map((hit: any) => ({
          label: `${hit._source?.title ?? hit._id}`,
          value: hit._id
        }))
      );
    } catch {
      // picker is best-effort; manual id entry still works via the select
    } finally {
      setPicking(false);
    }
  }, []);

  const openAdd = useCallback(() => {
    setEditing(null);
    setDocOptions([]);
    form.resetFields();
    setModalOpen(true);
  }, [form]);

  const openEdit = useCallback(
    (r: SearchEvalCase) => {
      setEditing(r);
      // prefill options so the select renders the stored titles as labels
      const ids = r.expected_ids ?? [];
      const titles = r.expected_titles ?? [];
      setDocOptions(ids.map((id, i) => ({ label: titles[i] || id, value: id })));
      form.setFieldsValue({ query: r.query, expected_ids: ids, datasource: r.datasource, note: r.note });
      setModalOpen(true);
    },
    [form]
  );

  const saveCase = useCallback(async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const titleByID = new Map(docOptions.map((o) => [o.value, o.label]));
      await createEvalCase({
        query: values.query.trim(),
        expected_ids: values.expected_ids,
        expected_titles: values.expected_ids.map((id: string) => titleByID.get(id) ?? ''),
        datasource: values.datasource,
        note: values.note?.trim() || undefined
      });
      message.success(t('page.searchStudio.evalSaved'));
      setModalOpen(false);
      load();
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchStudio.evalSaveFailed'));
    } finally {
      setSaving(false);
    }
  }, [docOptions, form, load, t]);

  const removeCase = useCallback(
    async (id: string) => {
      try {
        await deleteEvalCase(id);
        load();
      } catch (e: any) {
        message.error(e?.response?.data?.message ?? e?.message);
      }
    },
    [load]
  );

  const run = useCallback(async () => {
    setRunning(true);
    try {
      const res: any = await runSearchEval();
      setLastRun((res?.data?.run as SearchEvalRun) ?? null);
      const runRes: any = await fetchEvalRuns();
      setRuns((runRes?.data?.data as SearchEvalRun[]) ?? []);
      message.success(t('page.searchStudio.evalRunDone'));
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.searchStudio.evalRunFailed'));
    } finally {
      setRunning(false);
    }
  }, [t]);

  const caseColumns = [
    { title: t('page.searchStudio.evalQueryCol'), dataIndex: 'query', ellipsis: true },
    {
      title: t('page.searchStudio.evalExpectedCol'),
      dataIndex: 'expected_titles',
      ellipsis: true,
      render: (v: string[], r: SearchEvalCase) => (
        <Typography.Text type="secondary">{(v?.filter(Boolean).length ? v.filter(Boolean) : r.expected_ids).join(' · ')}</Typography.Text>
      )
    },
    {
      title: t('page.searchStudio.datasource'),
      dataIndex: 'datasource',
      width: 140,
      ellipsis: true,
      render: (v: string) => v || '-'
    },
    { title: t('page.searchStudio.evalNoteCol'), dataIndex: 'note', width: 160, ellipsis: true, render: (v: string) => v || '-' },
    {
      title: '',
      width: 100,
      render: (_: any, r: SearchEvalCase) => (
        <Space size={0}>
          <Button size="small" type="text" icon={<EditOutlined />} onClick={() => openEdit(r)} />
          <Popconfirm title={t('page.searchStudio.evalDeleteConfirm')} onConfirm={() => removeCase(r.id)}>
            <Button size="small" type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    }
  ];

  const runColumns = [
    { title: t('page.searchStudio.evalQueryCol'), dataIndex: 'query', ellipsis: true },
    {
      title: t('page.searchStudio.evalHitRank'),
      dataIndex: 'hit_rank',
      width: 110,
      render: (v: number) =>
        v > 0 ? (
          <Tag color={v <= 4 ? 'green' : 'orange'}>
            #{v}
            {v > 4 ? ` · ${t('page.searchStudio.evalOutsideTop4')}` : ''}
          </Tag>
        ) : (
          <Tag color="red">{t('page.searchStudio.evalMiss')}</Tag>
        )
    },
    {
      title: t('page.searchStudio.evalTopReturns'),
      dataIndex: 'top_titles',
      ellipsis: true,
      render: (v: string[]) => (v?.length ? v.join(' · ') : '-')
    },
    { title: t('page.searchStudio.evalTook'), dataIndex: 'took_ms', width: 100, render: (v: number) => `${v ?? 0} ms` },
    {
      title: t('page.searchStudio.evalError'),
      dataIndex: 'error',
      width: 180,
      ellipsis: true,
      render: (v: string) => (v ? <Typography.Text type="danger" ellipsis>{v}</Typography.Text> : '-')
    }
  ];

  const trendColumns = [
    {
      title: t('page.searchStudio.evalRunAt'),
      dataIndex: 'created',
      width: 170,
      render: (v: string) => (v ? new Date(v).toLocaleString() : '-')
    },
    { title: t('page.searchStudio.evalCasesCol'), dataIndex: 'total_cases', width: 90 },
    { title: t('page.searchStudio.evalTop4Col'), dataIndex: 'top4_hits', width: 100 },
    {
      title: t('page.searchStudio.evalTop4Rate'),
      dataIndex: 'top4_rate',
      width: 120,
      render: (v: number) => `${((v ?? 0) * 100).toFixed(1)}%`
    },
    { title: 'MRR', dataIndex: 'mrr', width: 100, render: (v: number) => (v ?? 0).toFixed(3) },
    { title: t('page.searchStudio.evalAvgTook'), dataIndex: 'avg_took_ms', width: 110, render: (v: number) => `${v ?? 0} ms` }
  ];

  return (
    <Card
      title={
        <Space size={8}>
          <AuditOutlined className="w-16px h-16px" />
          <span>{t('page.searchStudio.evalTitle')}</span>
          <Typography.Text type="secondary" className="text-12px">
            {t('page.searchStudio.evalSubtitle')}
          </Typography.Text>
        </Space>
      }
      extra={
        <Space>
          <Button size="small" loading={loading} onClick={load}>
            {t('page.searchOps.refresh')}
          </Button>
          <Button size="small" type="primary" icon={<PlayCircleOutlined />} loading={running} disabled={cases.length === 0} onClick={run}>
            {t('page.searchStudio.evalRun')}
          </Button>
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={openAdd}>
            {t('page.searchStudio.evalAdd')}
          </Button>
        </Space>
      }
    >
      {lastRun && (
        <div className="mb-16px p-12px rounded-4px bg-gray-50">
          <Space size={32} wrap>
            <Statistic title={t('page.searchStudio.evalTop4Rate')} value={((lastRun.top4_rate ?? 0) * 100).toFixed(1)} suffix="%" />
            <Statistic title="MRR" value={(lastRun.mrr ?? 0).toFixed(3)} />
            <Statistic title={t('page.searchStudio.evalCasesCol')} value={lastRun.total_cases} />
            <Statistic title={t('page.searchStudio.evalAvgTook')} value={lastRun.avg_took_ms ?? 0} suffix="ms" />
          </Space>
          <Table
            className="mt-8px"
            rowKey="query"
            size="small"
            pagination={false}
            scroll={{ y: 240 }}
            columns={runColumns}
            dataSource={lastRun.cases ?? []}
          />
          <Typography.Text type="secondary" className="text-12px">
            {t('page.searchStudio.evalGapHint')}
          </Typography.Text>
        </div>
      )}

      <Table
        rowKey="id"
        size="small"
        pagination={{ pageSize: 10, hideOnSinglePage: true }}
        loading={loading}
        columns={caseColumns}
        dataSource={cases}
        locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.searchStudio.evalEmpty')} /> }}
      />

      <Typography.Text type="secondary" className="text-12px block mb-8px mt-16px">
        {t('page.searchStudio.evalTrend')}
      </Typography.Text>
      <Table rowKey="id" size="small" pagination={false} columns={trendColumns} dataSource={runs} locale={{ emptyText: t('page.searchOps.noData') }} />

      <Modal
        open={modalOpen}
        title={editing ? t('page.searchStudio.evalEdit') : t('page.searchStudio.evalAdd')}
        confirmLoading={saving}
        onOk={saveCase}
        onCancel={() => setModalOpen(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item label={t('page.searchStudio.evalQueryCol')} name="query" rules={[{ required: true, message: t('page.searchStudio.evalQueryRequired') }]}>
            <Input allowClear placeholder={t('page.searchStudio.evalQueryPlaceholder')} />
          </Form.Item>
          <Form.Item label={t('page.searchStudio.evalExpectedCol')} name="expected_ids" rules={[{ required: true, message: t('page.searchStudio.evalExpectedRequired') }]}>
            <Select
              mode="multiple"
              allowClear
              showSearch
              filterOption={false}
              loading={picking}
              onSearch={searchDocs}
              notFoundContent={picking ? undefined : t('page.searchStudio.evalPickHint')}
              placeholder={t('page.searchStudio.evalPickPlaceholder')}
              options={docOptions}
            />
          </Form.Item>
          <Form.Item label={t('page.searchStudio.datasource')} name="datasource">
            <Select allowClear placeholder={t('page.searchStudio.datasourceAll')} options={datasourceOptions} />
          </Form.Item>
          <Form.Item label={t('page.searchStudio.evalNoteCol')} name="note">
            <Input allowClear placeholder={t('page.searchStudio.evalNotePlaceholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
