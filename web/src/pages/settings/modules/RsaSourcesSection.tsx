import { BugOutlined, DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, SendOutlined } from '@ant-design/icons';
import { Button, Drawer, Empty, Form, Input, InputNumber, Popconfirm, Select, Space, Switch, Table, Tag, Tooltip } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  createRemoteSource,
  debugRemoteSource,
  deleteRemoteSource,
  fetchRemoteSources,
  fetchRsaStats,
  type RemoteSource,
  type RsaDebugResult,
  type RsaSourceStats,
  type RsaTestResult,
  testRemoteSource,
  updateRemoteSource
} from '@/service/api';

/**
 * RSA 联查数据源管理 (DESIGN-RSA.md)：外部 ES/数据库在查询期直接联查、
 * 结果映射为标准文档参与 RRF 融合，不复制数据到本地。本区是即存即用的
 * CRUD（不走页面底部的保存条）；全局 rsa_enabled 开关与扇出/缓存参数在
 * 同一 tab 的检索表单里。凭据读回为 ******，原样提交即保持不变。
 */

const ADAPTER_OPTIONS = ['es_compatible'];

// the drawer's blank source — every control mirrors the backend defaults so
// the operator only overrides what they care about
const blankSource = (): RemoteSource => ({
  name: '',
  description: '',
  adapter: 'es_compatible',
  enabled: true,
  priority: 0,
  connection: { endpoints: [], auth: '', verify_tls: true },
  query: { indices: '', query_template: '' },
  mapping: {},
  controls: { timeout_ms: 3000, max_concurrent: 2, qps: 2, cache_ttl_seconds: 60, max_window: 100, retry_max_attempts: 2, retry_backoff_ms: 150 }
});

// the drawer reads one flat form and re-nests on submit
const toFormValues = (src: RemoteSource) => ({
  name: src.name,
  description: src.description || '',
  adapter: src.adapter,
  enabled: !!src.enabled,
  priority: src.priority ?? 0,
  endpoints: src.connection?.endpoints || [],
  auth: src.connection?.auth || '',
  username: src.connection?.username || '',
  password: src.connection?.password || '',
  api_key: src.connection?.api_key || '',
  verify_tls: src.connection?.verify_tls ?? true,
  indices: src.query?.indices || '',
  query_template: src.query?.query_template || '',
  title: src.mapping?.title || '',
  content: src.mapping?.content || '',
  url: src.mapping?.url || '',
  created: src.mapping?.created || '',
  tags: src.mapping?.tags || '',
  id_path: src.mapping?.id_path || '',
  timeout_ms: src.controls?.timeout_ms || 0,
  max_concurrent: src.controls?.max_concurrent || 0,
  qps: src.controls?.qps ?? 0,
  cache_ttl_seconds: src.controls?.cache_ttl_seconds ?? 0,
  max_window: src.controls?.max_window || 0,
  retry_max_attempts: src.controls?.retry_max_attempts || 0,
  retry_backoff_ms: src.controls?.retry_backoff_ms || 0,
  retry_on_timeout: !!src.controls?.retry_on_timeout
});

const fromFormValues = (v: any): RemoteSource => ({
  name: (v.name || '').trim(),
  description: (v.description || '').trim(),
  adapter: v.adapter,
  enabled: !!v.enabled,
  priority: Number(v.priority) || 0,
  connection: {
    endpoints: (v.endpoints || []).map((s: string) => s.trim()).filter(Boolean),
    auth: v.auth || '',
    username: v.auth === 'basic' ? (v.username || '').trim() : '',
    // "******" is the masked read-back — passing it through keeps the stored secret
    password: v.auth === 'basic' ? (v.password || '') : '',
    api_key: v.auth === 'api_key' ? (v.api_key || '') : '',
    verify_tls: v.verify_tls ?? true
  },
  query: { indices: (v.indices || '').trim(), query_template: (v.query_template || '').trim() },
  mapping: {
    title: (v.title || '').trim(),
    content: (v.content || '').trim(),
    url: (v.url || '').trim(),
    created: (v.created || '').trim(),
    tags: (v.tags || '').trim(),
    id_path: (v.id_path || '').trim()
  },
  controls: {
    timeout_ms: Number(v.timeout_ms) || 0,
    max_concurrent: Number(v.max_concurrent) || 0,
    qps: Number(v.qps) || 0,
    cache_ttl_seconds: Number(v.cache_ttl_seconds) || 0,
    max_window: Number(v.max_window) || 0,
    retry_max_attempts: Number(v.retry_max_attempts) || 0,
    retry_backoff_ms: Number(v.retry_backoff_ms) || 0,
    retry_on_timeout: !!v.retry_on_timeout
  }
});

const statusColor: Record<string, string> = { ok: 'green', cache: 'blue', timeout: 'orange', error: 'red' };

export default function RsaSourcesSection() {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();
  const [form] = Form.useForm();
  const authWatch = Form.useWatch('auth', form);

  const [sources, setSources] = useState<RemoteSource[]>([]);
  const [stats, setStats] = useState<RsaSourceStats[]>([]);
  const [loading, setLoading] = useState(false);
  const [statsById, setStatsById] = useState<Record<string, RsaSourceStats>>({});

  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<RemoteSource | null>(null);
  const [saving, setSaving] = useState(false);

  const [testOpen, setTestOpen] = useState(false);
  const [testTarget, setTestTarget] = useState<RemoteSource | null>(null);
  const [testQuery, setTestQuery] = useState('');
  const [testResult, setTestResult] = useState<RsaTestResult | null>(null);
  const [testing, setTesting] = useState(false);

  const [debugOpen, setDebugOpen] = useState(false);
  const [debugTarget, setDebugTarget] = useState<RemoteSource | null>(null);
  const [debugQuery, setDebugQuery] = useState('');
  const [debugResult, setDebugResult] = useState<RsaDebugResult | null>(null);
  const [debugging, setDebugging] = useState(false);

  const canManage = hasAuth('coco#rsa/create') || hasAuth('coco#rsa/update');
  const canDelete = hasAuth('coco#rsa/delete');
  const canDebug = hasAuth('coco#rsa/debug');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [srcRes, statRows] = await Promise.all([fetchRemoteSources(), fetchRsaStats().catch(() => [])]);
      setSources(srcRes?.hits || []);
      setStats(statRows || []);
      const byId: Record<string, RsaSourceStats> = {};
      for (const row of statRows || []) byId[row.source_id] = row;
      setStatsById(byId);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue(toFormValues(blankSource()));
    setDrawerOpen(true);
  };

  const openEdit = (src: RemoteSource) => {
    setEditing(src);
    form.setFieldsValue(toFormValues(src));
    setDrawerOpen(true);
  };

  const handleSave = async () => {
    let values;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }
    setSaving(true);
    try {
      const payload = fromFormValues(values);
      if (editing?.id) {
        await updateRemoteSource(editing.id, payload);
      } else {
        await createRemoteSource(payload);
      }
      window.$message?.success(t(editing?.id ? 'common.updateSuccess' : 'common.addSuccess'));
      setDrawerOpen(false);
      load();
    } catch (e: any) {
      window.$message?.error(e?.response?.data?.message ?? e?.message ?? t('common.operateFailed'));
    } finally {
      setSaving(false);
    }
  };

  // the enable toggle is immediate — it gates the search leg, not this form
  const toggleEnabled = async (src: RemoteSource, enabled: boolean) => {
    const prior = sources;
    setSources(rows => rows.map(r => (r.id === src.id ? { ...r, enabled } : r)));
    try {
      await updateRemoteSource(src.id!, { ...src, enabled });
    } catch (e: any) {
      setSources(prior);
      window.$message?.error(e?.response?.data?.message ?? e?.message ?? t('common.operateFailed'));
    }
  };

  const handleDelete = async (src: RemoteSource) => {
    try {
      await deleteRemoteSource(src.id!);
      window.$message?.success(t('common.deleteSuccess'));
      load();
    } catch (e: any) {
      window.$message?.error(e?.response?.data?.message ?? e?.message ?? t('common.operateFailed'));
    }
  };

  const openTest = (src: RemoteSource) => {
    setTestTarget(src);
    setTestQuery('');
    setTestResult(null);
    setTestOpen(true);
  };

  const runTest = async () => {
    if (!testTarget?.id) return;
    setTesting(true);
    try {
      setTestResult(await testRemoteSource(testTarget.id, testQuery.trim()));
      fetchRsaStats().then(rows => {
        const byId: Record<string, RsaSourceStats> = {};
        for (const row of rows || []) byId[row.source_id] = row;
        setStatsById(byId);
      }).catch(() => {});
    } catch (e: any) {
      setTestResult({ status: 'error', took_ms: 0, hits: 0, diag: { source_id: '', source_name: '', adapter: '', status: 'error', hits: 0, took_ms: 0, cache_hit: false }, error: e?.response?.data?.message ?? e?.message });
    } finally {
      setTesting(false);
    }
  };

  const openDebug = (src: RemoteSource) => {
    setDebugTarget(src);
    setDebugQuery('');
    setDebugResult(null);
    setDebugOpen(true);
  };

  const runDebug = async () => {
    if (!debugTarget?.id || !debugQuery.trim()) return;
    setDebugging(true);
    try {
      setDebugResult(await debugRemoteSource({ source_id: debugTarget.id, query: debugQuery.trim(), size: 10 }));
    } catch (e: any) {
      window.$message?.error(e?.response?.data?.message ?? e?.message ?? t('common.operateFailed'));
    } finally {
      setDebugging(false);
    }
  };

  const columns = useMemo(
    () => [
      {
        title: t('page.settings.search_settings.labels.rsa_col_name'),
        dataIndex: 'name',
        ellipsis: true,
        render: (v: string, r: RemoteSource) => (
          <Space size={6}>
            <span>{v}</span>
            {r.enabled ? <Tag color="green">{t('page.settings.search_settings.labels.rsa_enabled_badge')}</Tag> : null}
          </Space>
        )
      },
      {
        title: t('page.settings.search_settings.labels.rsa_col_adapter'),
        dataIndex: 'adapter',
        width: 130,
        render: (v: string) => <Tag>{v}</Tag>
      },
      {
        title: t('page.settings.search_settings.labels.rsa_col_enabled'),
        dataIndex: 'enabled',
        width: 80,
        render: (v: boolean, r: RemoteSource) => (
          <Switch checked={v} disabled={!canManage} onChange={checked => toggleEnabled(r, checked)} size="small" />
        )
      },
      {
        title: t('page.settings.search_settings.labels.rsa_col_endpoints'),
        key: 'endpoints',
        width: 230,
        ellipsis: true,
        render: (_: unknown, r: RemoteSource) => (r.connection?.endpoints || []).join(' , ')
      },
      {
        title: t('page.settings.search_settings.labels.rsa_col_stats'),
        key: 'stats',
        width: 240,
        render: (_: unknown, r: RemoteSource) => {
          const s = r.id ? statsById[r.id] : undefined;
          if (!s) return '-';
          const bad = s.errors > 0 || s.timeouts > 0;
          return (
            <Tooltip title={s.last_error || t('page.settings.search_settings.labels.rsa_stats_hint', { p50: s.latency_p50_ms, p95: s.latency_p95_ms, cache: s.cache_hits })}>
              <span className={bad ? 'color-[var(--ant-color-error)]' : ''}>
                {t('page.settings.search_settings.labels.rsa_stats_line', { req: s.requests, hits: s.hits, err: s.errors + s.timeouts, ms: s.latency_p95_ms })}
              </span>
            </Tooltip>
          );
        }
      },
      {
        title: t('page.settings.search_settings.labels.rsa_col_actions'),
        key: 'actions',
        width: 250,
        render: (_: unknown, r: RemoteSource) => (
          <Space size={4}>
            <Button onClick={() => openTest(r)} size="small">
              {t('page.settings.search_settings.labels.rsa_test')}
            </Button>
            {canDebug ? (
              <Button icon={<BugOutlined />} onClick={() => openDebug(r)} size="small">
                {t('page.settings.search_settings.labels.rsa_debug')}
              </Button>
            ) : null}
            {canManage ? (
              <>
                <Button icon={<EditOutlined />} onClick={() => openEdit(r)} size="small" type="text" />
                {canDelete ? (
                  <Popconfirm title={t('page.settings.search_settings.labels.rsa_delete_confirm')} onConfirm={() => handleDelete(r)}>
                    <Button danger icon={<DeleteOutlined />} size="small" type="text" />
                  </Popconfirm>
                ) : null}
              </>
            ) : null}
          </Space>
        )
      }
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, statsById, canManage, canDelete, canDebug, sources]
  );

  const L = (key: string, params?: Record<string, string | number>) =>
    t(`page.settings.search_settings.labels.${key}`, params);

  return (
    <div className="flex flex-col gap-12px">
      <div className="flex items-center justify-between">
        <div className="color-[var(--ant-color-text)] font-medium">{L('rsa_sources')}</div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} size="small">
            {L('rsa_refresh')}
          </Button>
          {canManage ? (
            <Button icon={<PlusOutlined />} onClick={openCreate} size="small" type="primary">
              {L('rsa_add')}
            </Button>
          ) : null}
        </Space>
      </div>

      <Table<RemoteSource>
        rowKey="id"
        size="small"
        loading={loading}
        columns={columns}
        dataSource={sources}
        pagination={false}
        locale={{ emptyText: <Empty description={L('rsa_empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
      />
      {stats.length > 0 ? <div className="settings-form-help">{L('rsa_stats_note')}</div> : null}

      <Drawer
        destroyOnClose
        onClose={() => setDrawerOpen(false)}
        open={drawerOpen}
        title={editing ? L('rsa_edit_title') : L('rsa_add_title')}
        width={640}
        extra={
          <Space>
            <Button onClick={() => setDrawerOpen(false)} size="small">
              {t('common.cancel')}
            </Button>
            <Button loading={saving} onClick={handleSave} size="small" type="primary">
              {t('common.confirm')}
            </Button>
          </Space>
        }
      >
        <Form className="settings-form" colon={false} form={form} labelAlign="left">
          <div className="color-[var(--ant-color-text)] font-medium mb-8px">{L('rsa_f_basic')}</div>
          <Form.Item label={L('rsa_f_name')} name="name" rules={[{ required: true, message: t('page.settings.search_settings.labels.rsa_required') }]}>
            <Input placeholder="Easysearch 生产集群" />
          </Form.Item>
          <Form.Item label={L('rsa_f_description')} name="description">
            <Input />
          </Form.Item>
          <Form.Item label={L('rsa_f_adapter')} name="adapter" rules={[{ required: true }]} tooltip={L('rsa_f_adapter_desc')}>
            <Select
              className="!w-full"
              options={ADAPTER_OPTIONS.map(v => ({ value: v, label: L(`rsa_adapter_${v}`) }))}
            />
          </Form.Item>
          <Form.Item label={L('rsa_f_priority')} name="priority" tooltip={L('rsa_f_priority_desc')}>
            <InputNumber className="w-full" min={0} precision={0} />
          </Form.Item>
          <Form.Item label={L('rsa_f_enabled')} name="enabled" tooltip={L('rsa_f_enabled_desc')} valuePropName="checked">
            <Switch size="small" />
          </Form.Item>

          <div className="color-[var(--ant-color-text)] font-medium mb-8px mt-16px">{L('rsa_f_connection')}</div>
          <Form.Item label={L('rsa_f_endpoints')} name="endpoints" rules={[{ required: true, message: t('page.settings.search_settings.labels.rsa_required') }]} tooltip={L('rsa_f_endpoints_desc')}>
            <Select className="!w-full" mode="tags" open={false} placeholder="http://es1:9200" tokenSeparators={[',', ' ']} />
          </Form.Item>
          <Form.Item label={L('rsa_f_auth')} name="auth">
            <Select
              className="!w-full"
              options={[
                { value: '', label: L('rsa_auth_none') },
                { value: 'basic', label: 'Basic' },
                { value: 'api_key', label: 'API Key' }
              ]}
            />
          </Form.Item>
          {authWatch === 'basic' ? (
            <>
              <Form.Item label={L('rsa_f_username')} name="username">
                <Input autoComplete="off" />
              </Form.Item>
              <Form.Item label={L('rsa_f_password')} name="password" tooltip={L('rsa_f_secret_hint')}>
                <Input autoComplete="new-password" type="password" />
              </Form.Item>
            </>
          ) : null}
          {authWatch === 'api_key' ? (
            <Form.Item label="API Key" name="api_key" tooltip={L('rsa_f_secret_hint')}>
              <Input autoComplete="off" />
            </Form.Item>
          ) : null}
          <Form.Item label={L('rsa_f_verify_tls')} name="verify_tls" tooltip={L('rsa_f_verify_tls_desc')} valuePropName="checked">
            <Switch size="small" />
          </Form.Item>

          <div className="color-[var(--ant-color-text)] font-medium mb-8px mt-16px">{L('rsa_f_query')}</div>
          <Form.Item label={L('rsa_f_indices')} name="indices" rules={[{ required: true, message: t('page.settings.search_settings.labels.rsa_required') }]} tooltip={L('rsa_f_indices_desc')}>
            <Input placeholder="logs-*,wiki*" />
          </Form.Item>
          <Form.Item label={L('rsa_f_template')} name="query_template" tooltip={L('rsa_f_template_desc')}>
            <Input.TextArea placeholder={'{"query":{"match":{"title":"{{.Text}}"}},"from":{{.From}},"size":{{.Size}}}'} rows={4} />
          </Form.Item>

          <div className="color-[var(--ant-color-text)] font-medium mb-8px mt-16px">{L('rsa_f_mapping')}</div>
          <div className="mb-8px settings-form-help">{L('rsa_f_mapping_hint')}</div>
          {[
            ['title', 'title'],
            ['content', 'content'],
            ['url', 'url'],
            ['created', 'created'],
            ['tags', 'tags'],
            ['id_path', '_id']
          ].map(([field, dft]) => (
            <Form.Item key={field} label={L(`rsa_m_${field}`)} name={field} tooltip={t(`page.settings.search_settings.labels.rsa_m_${field}_desc`)}>
              <Input placeholder={dft} />
            </Form.Item>
          ))}

          <div className="color-[var(--ant-color-text)] font-medium mb-8px mt-16px">{L('rsa_f_controls')}</div>
          <div className="mb-8px settings-form-help">{L('rsa_f_controls_hint')}</div>
          {[
            ['timeout_ms', 3000, 10000, 100],
            ['max_concurrent', 2, 16, 1],
            ['qps', 2, 1000, 1],
            ['cache_ttl_seconds', 60, 3600, 1],
            ['max_window', 100, 1000, 10],
            ['retry_max_attempts', 2, 3, 1],
            ['retry_backoff_ms', 150, 5000, 10]
          ].map(([field, dft, max, step]) => (
            <Form.Item key={field} label={L(`rsa_c_${field}`)} name={field} tooltip={t(`page.settings.search_settings.labels.rsa_c_${field}_desc`)}>
              <InputNumber className="w-full" max={max as number} min={0} placeholder={String(dft)} step={step as number} />
            </Form.Item>
          ))}
          <Form.Item label={L('rsa_c_retry_on_timeout')} name="retry_on_timeout" tooltip={L('rsa_c_retry_on_timeout_desc')} valuePropName="checked">
            <Switch size="small" />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        destroyOnClose
        onClose={() => setTestOpen(false)}
        open={testOpen}
        title={`${L('rsa_test')}: ${testTarget?.name ?? ''}`}
        width={560}
      >
        <div className="flex gap-8px mb-16px">
          <Input
            onChange={e => setTestQuery(e.target.value)}
            placeholder={L('rsa_test_query_ph')}
            value={testQuery}
          />
          <Button icon={<SendOutlined />} loading={testing} onClick={runTest} type="primary">
            {L('rsa_test_run')}
          </Button>
        </div>
        {testResult ? (
          <div className="flex flex-col gap-8px">
            <Space size={12}>
              <Tag color={statusColor[testResult.status] || 'default'}>{testResult.status}</Tag>
              <span>{L('rsa_test_took', { ms: testResult.took_ms })}</span>
              <span>{L('rsa_test_hits', { n: testResult.hits })}</span>
              {testResult.diag?.cache_hit ? <Tag color="blue">cache</Tag> : null}
            </Space>
            {testResult.error ? <div className="color-[var(--ant-color-error)]">{testResult.error}</div> : null}
            {testResult.diag?.attempts?.length ? (
              <div className="flex flex-col gap-4px">
                {(testResult.diag.attempts || []).map((a, i) => (
                  <div key={i} className="text-12px">
                    #{i + 1} {a.endpoint} · {a.took_ms}ms {a.final ? <Tag color="green">OK</Tag> : a.err ? <span className="color-[var(--ant-color-error)]">{a.err}</span> : null}
                  </div>
                ))}
              </div>
            ) : null}
            {testResult.sample ? (
              <div className="rounded-8px p-12px" style={{ border: '1px solid var(--ant-color-border-secondary)' }}>
                <div className="font-medium mb-4px">{testResult.sample.title || '-'}</div>
                {testResult.sample.url ? <div className="text-12px mb-4px">{testResult.sample.url}</div> : null}
                <div className="text-12px color-[var(--ant-color-text-tertiary)]">{testResult.sample.snippet}</div>
              </div>
            ) : null}
          </div>
        ) : (
          <div className="settings-form-help">{L('rsa_test_hint')}</div>
        )}
      </Drawer>

      <Drawer
        destroyOnClose
        onClose={() => setDebugOpen(false)}
        open={debugOpen}
        title={`${L('rsa_debug')}: ${debugTarget?.name ?? ''}`}
        width={720}
      >
        <div className="flex gap-8px mb-16px">
          <Input
            onChange={e => setDebugQuery(e.target.value)}
            placeholder={L('rsa_debug_query_ph')}
            onPressEnter={runDebug}
            value={debugQuery}
          />
          <Button icon={<BugOutlined />} disabled={!debugQuery.trim()} loading={debugging} onClick={runDebug} type="primary">
            {L('rsa_debug_run')}
          </Button>
        </div>
        {debugResult ? (
          <div className="flex flex-col gap-12px">
            <Space size={12}>
              <Tag color={statusColor[debugResult.status] || 'default'}>{debugResult.status}</Tag>
              <span>{L('rsa_test_took', { ms: debugResult.took_ms })}</span>
              {typeof debugResult.total === 'number' ? <span>{L('rsa_debug_total', { n: debugResult.total })}</span> : null}
            </Space>
            {debugResult.error ? <div className="color-[var(--ant-color-error)]">{debugResult.error}</div> : null}
            {debugResult.native?.es_search_url ? <div className="text-12px">{debugResult.native.es_search_url}</div> : null}
            {debugResult.native?.es_search_body ? (
              <pre className="rounded-8px p-12px text-12px overflow-auto max-h-280px" style={{ border: '1px solid var(--ant-color-border-secondary)' }}>
                {JSON.stringify(debugResult.native.es_search_body, null, 2)}
              </pre>
            ) : null}
            {debugResult.native?.render_error ? <div className="color-[var(--ant-color-error)] text-12px">{debugResult.native.render_error}</div> : null}
            {debugResult.mapped_hits?.length ? (
              <Table
                rowKey="doc_id"
                size="small"
                columns={[
                  { title: '#', dataIndex: 'rank', width: 44 },
                  { title: 'title', dataIndex: 'title', ellipsis: true },
                  { title: 'raw_id', dataIndex: 'raw_id', ellipsis: true, width: 160 },
                  { title: 'index', dataIndex: 'index', ellipsis: true, width: 140 }
                ]}
                dataSource={debugResult.mapped_hits}
                pagination={false}
              />
            ) : null}
          </div>
        ) : (
          <div className="settings-form-help">{L('rsa_debug_hint')}</div>
        )}
      </Drawer>
    </div>
  );
}
