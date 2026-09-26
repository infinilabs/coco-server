import { BookOpen, Boxes, Braces, Check, ChevronDown, ChevronUp, File, FileText, FlaskConical, Image, Paperclip, Pencil, Rocket, Sparkles, Trash, Video } from 'lucide-react';
import { Button, Card, Col, Collapse, Drawer, Form, Input, InputNumber, Popconfirm, Row, Select, Space, Switch, Tag, Typography, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { aiGeneratePipelineChain, createPipeline, getPipelineProcessors, testPipelineChain, updatePipeline } from '@/service/api/pipeline';
import { availableTemplates, templateEntryNames, type PipelineTemplate } from './pipelineTemplates';

export interface PipelineDoc {
  id?: string;
  name: string;
  description?: string;
  enabled?: boolean;
  singleton?: boolean;
  auto_start?: boolean;
  keep_running?: boolean;
  retry_delay_in_ms?: number;
  processor?: Record<string, any>[];
}

interface ProcessorEntry {
  name: string;
  category?: string;
  config?: Record<string, any>;
}

interface StepSnapshot {
  name: string;
  fields?: Record<string, any>;
  count?: number;
  error?: string;
}

export const DOCUMENT_TEMPLATE = [
  {
    title: 'Coco Server 简介与快速入门',
    content:
      'Coco Server 是 INFINI Labs 推出的智能搜索引擎。本文介绍如何安装、配置数据源并开始搜索。第一步:下载安装包;第二步:修改配置文件;第三步:启动服务。',
    url: 'https://docs.example.com/coco-server-quickstart',
    lang: '',
    type: 'article',
    source: { id: 'web', name: 'Web' }
  }
];

export const ATTACHMENT_TEMPLATE = [
  {
    name: 'quarterly-report.pdf',
    mime_type: 'application/pdf',
    size: 102400,
    text: 'Q3 revenue grew 18% quarter over quarter, driven by enterprise subscriptions...'
  }
];

// ── helpers ──────────────────────────────────────────────────────────

function entryName(entry: Record<string, any>): string {
  if ('if' in entry) return 'if / then / else';
  return Object.keys(entry)[0] ?? '?';
}

function entrySummary(entry: Record<string, any>): string {
  if ('if' in entry) return JSON.stringify(entry.if);
  const name = Object.keys(entry)[0];
  if (name === undefined) return '';
  const cfg = entry[name];
  if (cfg && typeof cfg === 'object') {
    const keys = Object.keys(cfg).slice(0, 4).join(', ');
    return keys || '{}';
  }
  return String(cfg ?? '');
}

interface DiffItem {
  key: string;
  val: any;
  kind: 'add' | 'chg' | 'del';
}

function fieldDiff(prev: Record<string, any> | undefined, curr: Record<string, any>): DiffItem[] {
  const out: DiffItem[] = [];
  const keys = new Set([...Object.keys(prev ?? {}), ...Object.keys(curr)]);
  for (const k of keys) {
    const inP = prev != null && k in prev;
    const inC = k in curr;
    if (inC && !inP) out.push({ key: k, val: curr[k], kind: 'add' });
    else if (!inC && inP) out.push({ key: k, val: undefined, kind: 'del' });
    else if (JSON.stringify(prev?.[k]) !== JSON.stringify(curr[k])) out.push({ key: k, val: curr[k], kind: 'chg' });
  }
  return out.sort((a, b) => a.key.localeCompare(b.key));
}

const DIFF_COLOR: Record<DiffItem['kind'], string> = {
  add: 'text-green-600',
  chg: 'text-blue-600',
  del: 'text-red-500 line-through'
};

// ── chain card editor ────────────────────────────────────────────────

function ChainEditor({
  chain,
  setChain,
  catalog
}: {
  chain: Record<string, any>[];
  setChain: (c: Record<string, any>[]) => void;
  catalog: ProcessorEntry[];
}) {
  const { t } = useTranslation();
  const [editing, setEditing] = useState<number | null>(null);
  const [editText, setEditText] = useState('');

  const startEdit = (i: number) => {
    setEditing(i);
    setEditText(JSON.stringify(chain[i], null, 2));
  };

  const commitEdit = () => {
    if (editing == null) return;
    try {
      const v = JSON.parse(editText);
      if (typeof v !== 'object' || v == null || Array.isArray(v)) throw new Error('object');
      const next = [...chain];
      next[editing] = v;
      setChain(next);
      setEditing(null);
    } catch {
      message.error(t('page.pipeline.editor.invalidEntryJSON'));
    }
  };

  const move = (i: number, d: -1 | 1) => {
    const j = i + d;
    if (j < 0 || j >= chain.length) return;
    const next = [...chain];
    [next[i], next[j]] = [next[j], next[i]];
    setChain(next);
  };

  const insertFromCatalog = (name: string) => {
    const preset = catalog.find((p) => p.name === name);
    const seed: Record<string, any> = {};
    // prefill scalar defaults so the freshly inserted card is immediately meaningful
    if (preset?.config) {
      for (const [k, v] of Object.entries(preset.config)) {
        if (typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean') seed[k] = v;
      }
    }
    setChain([...chain, { [name]: seed }]);
  };

  return (
    <div className="flex flex-col gap-8px">
      {chain.length === 0 && (
        <div className="border-dashed border border-[var(--ant-color-border)] rounded-8px py-24px text-center text-13px text-[var(--ant-color-text-tertiary)]">
          {t('page.pipeline.editor.emptyChain')}
        </div>
      )}
      {chain.map((entry, i) => (
        <div key={i} className="border border-solid border-[var(--ant-color-border-secondary)] rounded-8px overflow-hidden bg-[var(--ant-color-bg-container)]">
          <div className="flex items-center gap-8px px-12px py-8px">
            <span className="text-11px text-[var(--ant-color-text-tertiary)] w-16px text-right">{i + 1}</span>
            <Tag color={entryName(entry) === 'if / then / else' ? 'gold' : 'geekblue'} className="m-0 font-mono">
              {entryName(entry)}
            </Tag>
            <span className="flex-1 truncate font-mono text-12px text-[var(--ant-color-text-secondary)]" title={entrySummary(entry)}>
              {entrySummary(entry)}
            </span>
            <Button size="small" type="text" icon={<ChevronUp size={14} />} onClick={() => move(i, -1)} />
            <Button size="small" type="text" icon={<ChevronDown size={14} />} onClick={() => move(i, 1)} />
            <Button size="small" type="text" icon={<Pencil size={14} />} onClick={() => (editing === i ? setEditing(null) : startEdit(i))} />
            <Popconfirm title={t('page.pipeline.editor.deleteEntryConfirm')} onConfirm={() => setChain(chain.filter((_, x) => x !== i))}>
              <Button size="small" type="text" danger icon={<Trash size={14} />} />
            </Popconfirm>
          </div>
          {'if' in entry && (
            <div className="px-12px pb-8px pl-36px font-mono text-11px text-[var(--ant-color-text-tertiary)] break-all">
              <div>then: {JSON.stringify(entry.then ?? [])}</div>
              {'else' in entry && entry.else != null && <div>else: {JSON.stringify(entry.else)}</div>}
            </div>
          )}
          {editing === i && (
            <div className="border-t border-solid border-[var(--ant-color-border-secondary)] p-8px flex flex-col gap-8px">
              <Input.TextArea
                className="font-mono text-12px"
                rows={Math.min(12, editText.split('\n').length + 1)}
                value={editText}
                onChange={(e) => setEditText(e.target.value)}
              />
              <Space>
                <Button size="small" type="primary" icon={<Check size={14} />} onClick={commitEdit}>
                  {t('common.confirm')}
                </Button>
                <Button size="small" onClick={() => setEditing(null)}>
                  {t('common.cancel')}
                </Button>
              </Space>
            </div>
          )}
        </div>
      ))}
      <div className="flex items-center gap-8px pt-4px">
        <Select
          size="small"
          className="min-w-200px"
          value=""
          placeholder={t('page.pipeline.editor.insertProcessor')}
          showSearch
          optionFilterProp="label"
          options={catalog.map((p) => ({ value: p.name, label: p.name }))}
          onChange={(name) => {
            if (name) insertFromCatalog(name);
          }}
        />
        <Button
          size="small"
          onClick={() =>
            setChain([...chain, { if: { equals: { lang: '' } }, then: [] } as unknown as Record<string, any>])
          }
        >
          {t('page.pipeline.editor.addBranch')}
        </Button>
      </div>
    </div>
  );
}

// ── processor catalog drawer content ─────────────────────────────────

function ProcessorCatalog({
  catalog,
  onInsert
}: {
  catalog: ProcessorEntry[];
  onInsert: (name: string) => void;
}) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState('');

  const byCategory = useMemo(() => {
    const m = new Map<string, ProcessorEntry[]>();
    for (const p of catalog) {
      const c = p.category || 'general';
      if (!m.has(c)) m.set(c, []);
      m.get(c)!.push(p);
    }
    return m;
  }, [catalog]);

  const match = (name: string) => !filter || name.toLowerCase().includes(filter.toLowerCase());

  return (
    <div className="flex flex-col gap-12px">
      <Input.Search size="small" placeholder={t('page.pipeline.editor.filterProcessors')} value={filter} onChange={(e) => setFilter(e.target.value)} allowClear />
      {[...byCategory.entries()].map(([cat, list]) => {
        const filtered = list.filter((p) => match(p.name));
        if (filtered.length === 0) return null;
        return (
          <div key={cat}>
            <div className="text-12px font-medium text-[var(--ant-color-text-secondary)] mb-4px">
              {cat} ({filtered.length})
            </div>
            <div className="flex flex-col gap-4px">
              {filtered.map((p) => (
                <div key={p.name} className="border border-solid border-[var(--ant-color-border-secondary)] rounded-8px px-8px py-4px">
                  <div className="flex items-center justify-between gap-8px">
                    <span className="font-mono text-12px">{p.name}</span>
                    <Button size="small" onClick={() => onInsert(p.name)}>
                      {t('page.pipeline.editor.insert')}
                    </Button>
                  </div>
                  {p.config && Object.keys(p.config).length > 0 && (
                    <Typography.Paragraph className="mb-0 font-mono text-11px text-[var(--ant-color-text-tertiary)] whitespace-pre-wrap break-all" style={{ marginTop: 4 }}>
                      {JSON.stringify(p.config, null, 1)}
                    </Typography.Paragraph>
                  )}
                </div>
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ── step debugger ────────────────────────────────────────────────────

function StepDebugger({ steps }: { steps: StepSnapshot[][] }) {
  const { t } = useTranslation();
  const [sampleIdx, setSampleIdx] = useState(0);
  if (steps.length === 0) return null;
  const idx = Math.min(sampleIdx, steps.length - 1);
  const sampleSteps = steps[idx] ?? [];

  return (
    <div className="flex flex-col gap-8px">
      {steps.length > 1 && (
        <div className="flex items-center gap-4px flex-wrap">
          <span className="text-12px text-[var(--ant-color-text-secondary)]">{t('page.pipeline.editor.sample')}:</span>
          {steps.map((_, i) => (
            <Button key={i} size="small" type={i === idx ? 'primary' : 'default'} onClick={() => setSampleIdx(i)}>
              #{i + 1}
            </Button>
          ))}
        </div>
      )}
      <div className="flex flex-col gap-4px max-h-360px overflow-auto">
        {sampleSteps.map((s, si) => {
          const prev = si === 0 ? undefined : sampleSteps[si - 1]?.fields;
          const diff = s.fields ? fieldDiff(prev, s.fields) : [];
          return (
            <div
              key={si}
              className={`border border-solid rounded-8px px-8px py-4px ${
                s.error ? 'border-[var(--ant-color-error)] bg-[var(--ant-color-error-bg)]' : 'border-[var(--ant-color-border-secondary)]'
              }`}
            >
              <div className="flex items-center gap-8px">
                <span className="text-11px text-[var(--ant-color-text-tertiary)]">{si + 1}</span>
                <span className="font-mono text-12px font-medium">{s.name}</span>
                {s.count != null && s.count !== 1 && <Tag className="m-0">{t('page.pipeline.editor.messages', { count: s.count })}</Tag>}
                {s.error && <span className="text-11px text-[var(--ant-color-error)]">{s.error}</span>}
              </div>
              {diff.length > 0 ? (
                <div className="pl-24px flex flex-wrap gap-x-16px gap-y-2px">
                  {diff.map((d) => (
                    <span key={d.key} className={`font-mono text-11px ${DIFF_COLOR[d.kind]}`}>
                      {d.key}: {JSON.stringify(d.val)}
                    </span>
                  ))}
                </div>
              ) : (
                <span className="pl-24px text-11px text-[var(--ant-color-text-tertiary)]">{t('page.pipeline.editor.noChanges')}</span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

const TEMPLATE_ICONS: Record<PipelineTemplate['icon'], React.ReactNode> = {
  'file-text': <FileText size={18} />,
  file: <File size={18} />,
  image: <Image size={18} />,
  video: <Video size={18} />,
  paperclip: <Paperclip size={18} />
};

/** One-click starter chains for common document tasks; shown when creating. */
function TemplateGallery({ templates, onPick }: { templates: PipelineTemplate[]; onPick: (t: PipelineTemplate) => void }) {
  const { t } = useTranslation();
  if (templates.length === 0) return null;
  return (
    <div className="flex flex-wrap gap-12px">
      {templates.map(tpl => (
        <div
          key={tpl.id}
          className="w-220px rounded-8px border border-solid border-[var(--ant-color-border-secondary)] p-12px flex flex-col gap-8px hover:border-[var(--ant-color-primary)] transition-colors"
        >
          <div className="flex items-center gap-8px text-[var(--ant-color-primary)]">
            {TEMPLATE_ICONS[tpl.icon]}
            <span className="text-13px font-medium text-[var(--ant-color-text)]">{t(`page.pipeline.templates.${tpl.id}.title`)}</span>
          </div>
          <span className="text-12px text-[var(--ant-color-text-tertiary)] leading-snug min-h-32px">{t(`page.pipeline.templates.${tpl.id}.description`)}</span>
          <div className="flex flex-wrap gap-4px">
            {templateEntryNames(tpl).slice(0, 4).map(n => (
              <span key={n} className="font-mono text-10px px-4px py-1px rounded-4px bg-[var(--ant-color-fill-tertiary)]">
                {n}
              </span>
            ))}
            {templateEntryNames(tpl).length > 4 && <span className="text-10px text-[var(--ant-color-text-tertiary)]">+{templateEntryNames(tpl).length - 4}</span>}
          </div>
          <Button size="small" block onClick={() => onPick(tpl)}>
            {t('page.pipeline.templates.use')}
          </Button>
        </div>
      ))}
    </div>
  );
}

// ── main editor ──────────────────────────────────────────────────────

export function PipelineEditor({ initial }: { initial?: PipelineDoc }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [form] = Form.useForm();

  const [advanced, setAdvanced] = useState({
    enabled: initial?.enabled ?? true,
    singleton: initial?.singleton ?? false,
    auto_start: initial?.auto_start ?? false,
    keep_running: initial?.keep_running ?? false,
    retry_delay_in_ms: initial?.retry_delay_in_ms ?? 1000
  });

  const [chain, setChain] = useState<Record<string, any>[]>(() => (Array.isArray(initial?.processor) ? initial!.processor! : []));
  const [view, setView] = useState<'cards' | 'json'>('cards');
  const [chainJSON, setChainJSON] = useState(() => JSON.stringify(initial?.processor ?? [], null, 2));
  const [jsonError, setJsonError] = useState<string | null>(null);

  const [catalog, setCatalog] = useState<ProcessorEntry[]>([]);
  const templates = useMemo(() => {
    const installed = new Set(catalog.map(p => p.name));
    return availableTemplates(installed);
  }, [catalog]);
  const [catalogOpen, setCatalogOpen] = useState(false);

  const [documentsJSON, setDocumentsJSON] = useState(() => JSON.stringify(DOCUMENT_TEMPLATE, null, 2));
  const [testResult, setTestResult] = useState<{ steps?: StepSnapshot[][]; finals?: Record<string, any>[] } | null>(null);
  const [testing, setTesting] = useState(false);
  const [aiRequirement, setAiRequirement] = useState('');
  const [aiRunning, setAiRunning] = useState<'generate' | 'refine' | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    getPipelineProcessors()
      .then((res: any) => {
        const raw = res?.data ?? {};
        const items: ProcessorEntry[] = Object.entries(raw).map(([name, meta]: [string, any]) => ({
          name,
          category: meta?.category,
          config: meta?.properties ?? undefined
        }));
        items.sort((a, b) => a.name.localeCompare(b.name));
        setCatalog(items);
      })
      .catch(() => {});
  }, []);

  const switchView = (v: 'cards' | 'json') => {
    if (v === 'json') {
      setChainJSON(JSON.stringify(chain, null, 2));
      setJsonError(null);
    } else {
      try {
        const parsed = JSON.parse(chainJSON);
        if (!Array.isArray(parsed)) throw new Error('array');
        setChain(parsed);
        setJsonError(null);
      } catch (e) {
        setJsonError(t('page.pipeline.editor.invalidChainJSON'));
        return;
      }
    }
    setView(v);
  };

  const applyTemplate = (tpl: PipelineTemplate) => {
    setChain(tpl.processor);
    setChainJSON(JSON.stringify(tpl.processor, null, 2));
    form.setFieldsValue({ name: tpl.suggestedName, description: t(`page.pipeline.templates.${tpl.id}.description`) });
    setView('cards');
  };

  const parseDocuments = (): Record<string, any>[] | null => {
    try {
      const parsed = JSON.parse(documentsJSON);
      const arr = Array.isArray(parsed) ? parsed : [parsed];
      if (arr.length === 0 || arr.some((d: any) => typeof d !== 'object' || d == null)) throw new Error('objects');
      return arr;
    } catch {
      message.error(t('page.pipeline.editor.invalidDocuments'));
      return null;
    }
  };

  const applyTestResult = (result: any) => {
    setTestResult({ steps: result?.steps, finals: result?.finals });
  };

  const runTest = async () => {
    const docs = parseDocuments();
    if (!docs) return;
    if (view === 'json') switchView('cards');
    setTesting(true);
    try {
      const res = await testPipelineChain({ processor: chain, documents: docs });
      applyTestResult(res?.data);
      message.success(t('page.pipeline.editor.testDone'));
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.pipeline.editor.testFailed'));
    } finally {
      setTesting(false);
    }
  };

  const runAI = async (mode: 'generate' | 'refine') => {
    const docs = parseDocuments();
    if (!docs) return;
    if (mode === 'refine' && chain.length === 0) {
      message.warning(t('page.pipeline.editor.refineNeedsChain'));
      return;
    }
    setAiRunning(mode);
    setTestResult(null);
    try {
      const res = await aiGeneratePipelineChain({
        documents: docs.map((d) => JSON.stringify(d)),
        requirements: aiRequirement,
        mode,
        current_chain: mode === 'refine' ? chain : undefined,
        auto_test: true
      });
      const data = res?.data ?? {};
      if (data.validation_error) {
        message.warning(`${t('page.pipeline.editor.validationError')}: ${data.validation_error}`);
      }
      const nextChain = Array.isArray(data.chain) ? data.chain : [];
      setChain(nextChain);
      setChainJSON(JSON.stringify(nextChain, null, 2));
      if (data.test) applyTestResult(data.test);
      message.success(t('page.pipeline.editor.aiDone'));
    } catch (e: any) {
      const msg = e?.response?.data?.message ?? e?.message ?? t('page.pipeline.editor.aiFailed');
      message.error(msg);
    } finally {
      setAiRunning(null);
    }
  };

  const onSave = async () => {
    const values = await form.validateFields();
    if (view === 'json') switchView('cards');
    setSaving(true);
    try {
      const payload = {
        name: values.name,
        description: values.description ?? '',
        ...advanced,
        processor: chain
      };
      if (initial?.id) {
        await updatePipeline(initial.id, payload);
      } else {
        // pipeline name doubles as the document id — process_documents loads
        // pipelines by name, so the id must be stable and human-readable
        await createPipeline({ ...payload, id: values.name });
      }
      message.success(t('common.updateSuccess'));
      navigate('/pipeline/list');
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.pipeline.editor.saveFailed'));
    } finally {
      setSaving(false);
    }
  };

  const chainEmpty = view === 'cards' ? chain.length === 0 : !chainJSON.trim() || chainJSON.trim() === '[]';

  return (
    <div className="flex flex-col gap-16px pb-24px">
      <Card className="card-wrapper" title={initial?.id ? t('page.pipeline.editor.editTitle', { name: initial.id }) : t('page.pipeline.editor.newTitle')} extra={
        <Space>
          <Button onClick={() => navigate('/pipeline/list')}>{t('page.pipeline.editor.back')}</Button>
          <Button type="primary" loading={saving} icon={<Rocket size={14} />} onClick={onSave}>
            {t('common.save')}
          </Button>
        </Space>
      }>
        <Form form={form} labelAlign="left" colon={false} initialValues={{ name: initial?.name, description: initial?.description }}>
          <Row gutter={24}>
            <Col span={12}>
              <Form.Item name="name" label={t('page.pipeline.editor.name')} rules={[{ required: true }]} extra={initial?.id ? undefined : t('page.pipeline.editor.nameAsId')}>
                <Input disabled={!!initial?.id} placeholder="my-enrichment-pipeline" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="description" label={t('page.pipeline.editor.description')}>
                <Input />
              </Form.Item>
            </Col>
          </Row>
        </Form>
        <Collapse
          ghost
          items={[
            {
              key: 'advanced',
              label: <span className="text-13px">{t('page.pipeline.editor.advanced')}</span>,
              children: (
                <Row gutter={24}>
                  <Col span={4}>
                    <div className="flex items-center gap-8px">
                      <Switch size="small" checked={advanced.enabled} onChange={(v) => setAdvanced((a) => ({ ...a, enabled: v }))} />
                      <span className="text-13px">{t('page.pipeline.editor.enabled')}</span>
                    </div>
                  </Col>
                  <Col span={4}>
                    <div className="flex items-center gap-8px">
                      <Switch size="small" checked={advanced.singleton} onChange={(v) => setAdvanced((a) => ({ ...a, singleton: v }))} />
                      <span className="text-13px">{t('page.pipeline.editor.singleton')}</span>
                    </div>
                  </Col>
                  <Col span={4}>
                    <div className="flex items-center gap-8px">
                      <Switch size="small" checked={advanced.auto_start} onChange={(v) => setAdvanced((a) => ({ ...a, auto_start: v }))} />
                      <span className="text-13px">{t('page.pipeline.editor.autoStart')}</span>
                    </div>
                  </Col>
                  <Col span={4}>
                    <div className="flex items-center gap-8px">
                      <Switch size="small" checked={advanced.keep_running} onChange={(v) => setAdvanced((a) => ({ ...a, keep_running: v }))} />
                      <span className="text-13px">{t('page.pipeline.editor.keepRunning')}</span>
                    </div>
                  </Col>
                  <Col span={6}>
                    <div className="flex items-center gap-8px">
                      <span className="text-13px">{t('page.pipeline.editor.retryDelay')}</span>
                      <InputNumber size="small" min={0} value={advanced.retry_delay_in_ms} onChange={(v) => setAdvanced((a) => ({ ...a, retry_delay_in_ms: v ?? 1000 }))} />
                    </div>
                  </Col>
                </Row>
              )
            }
          ]}
        />
      </Card>

      {!initial?.id && (
        <Card
          className="card-wrapper"
          title={<span className="flex items-center gap-8px"><Sparkles size={16} />{t('page.pipeline.templates.title')}</span>}
        >
          <TemplateGallery templates={templates} onPick={applyTemplate} />
        </Card>
      )}

      <Card
        className="card-wrapper"
        title={
          <span className="flex items-center gap-8px">
            <Boxes size={16} />
            {t('page.pipeline.editor.chainTitle')}
          </span>
        }
        extra={
          <Space>
            <Button size="small" onClick={() => setCatalogOpen(true)} icon={<BookOpen size={14} />}>
              {t('page.pipeline.editor.catalog')} ({catalog.length})
            </Button>
            <div className="flex rounded-6px border border-solid border-[var(--ant-color-border)] overflow-hidden">
              <Button size="small" type={view === 'cards' ? 'primary' : 'text'} onClick={() => switchView('cards')}>
                {t('page.pipeline.editor.cardsView')}
              </Button>
              <Button size="small" type={view === 'json' ? 'primary' : 'text'} icon={<Braces size={14} />} onClick={() => switchView('json')}>
                JSON
              </Button>
            </div>
          </Space>
        }
      >
        {view === 'cards' ? (
          <ChainEditor
            chain={chain}
            setChain={(c) => {
              setChain(c);
              setChainJSON(JSON.stringify(c, null, 2));
            }}
            catalog={catalog}
          />
        ) : (
          <div className="flex flex-col gap-4px">
            {jsonError && <span className="text-12px text-[var(--ant-color-error)]">{jsonError}</span>}
            <Input.TextArea
              className="font-mono text-12px"
              rows={12}
              placeholder={'[{"summary": {"enabled": true}}, {"if": {"equals": {"lang": "zh"}}, "then": []}]'}
              value={chainJSON}
              onChange={(e) => setChainJSON(e.target.value)}
            />
          </div>
        )}
      </Card>

      <Card
        className="card-wrapper"
        title={
          <span className="flex items-center gap-8px">
            <FlaskConical size={16} />
            {t('page.pipeline.editor.studioTitle')}
          </span>
        }
      >
        <Row gutter={16}>
          <Col span={12} className="flex flex-col gap-8px">
            <div className="flex items-center justify-between">
              <span className="text-13px font-medium">{t('page.pipeline.editor.sampleDocuments')}</span>
              <Space size={4}>
                <Button size="small" type="text" onClick={() => setDocumentsJSON(JSON.stringify(DOCUMENT_TEMPLATE, null, 2))}>
                  {t('page.pipeline.editor.documentTemplate')}
                </Button>
                <Button size="small" type="text" onClick={() => setDocumentsJSON(JSON.stringify(ATTACHMENT_TEMPLATE, null, 2))}>
                  {t('page.pipeline.editor.attachmentTemplate')}
                </Button>
              </Space>
            </div>
            <Input.TextArea className="font-mono text-12px" rows={10} value={documentsJSON} onChange={(e) => setDocumentsJSON(e.target.value)} />
            <Input.TextArea
              rows={2}
              placeholder={t('page.pipeline.editor.aiRequirementPlaceholder')}
              value={aiRequirement}
              onChange={(e) => setAiRequirement(e.target.value)}
            />
            <div className="flex gap-8px">
              <Button
                className="flex-1"
                type="primary"
                icon={<Sparkles size={14} className={aiRunning === 'generate' ? 'animate-pulse' : ''} />}
                loading={aiRunning === 'generate'}
                onClick={() => runAI('generate')}
              >
                {t('page.pipeline.editor.aiGenerate')}
              </Button>
              <Button
                className="flex-1"
                icon={<Sparkles size={14} className={aiRunning === 'refine' ? 'animate-pulse' : ''} />}
                loading={aiRunning === 'refine'}
                disabled={chainEmpty}
                onClick={() => runAI('refine')}
              >
                {t('page.pipeline.editor.aiRefine')}
              </Button>
            </div>
            <Button icon={<FlaskConical size={14} />} loading={testing} disabled={chainEmpty} onClick={runTest}>
              {t('page.pipeline.editor.runTest')}
            </Button>
          </Col>
          <Col span={12}>
            <span className="text-13px font-medium">{t('page.pipeline.editor.stepDebug')}</span>
            <div className="mt-8px">
              {testResult?.steps ? (
                <StepDebugger steps={testResult.steps} />
              ) : (
                <div className="border-dashed border border-[var(--ant-color-border)] rounded-8px py-24px text-center text-13px text-[var(--ant-color-text-tertiary)]">
                  {t('page.pipeline.editor.noTestYet')}
                </div>
              )}
            </div>
          </Col>
        </Row>
      </Card>

      <Drawer
        title={
          <span className="flex items-center gap-8px">
            <Boxes size={16} />
            {t('page.pipeline.editor.catalog')} ({catalog.length})
          </span>
        }
        open={catalogOpen}
        onClose={() => setCatalogOpen(false)}
        width={480}
      >
        <ProcessorCatalog
          catalog={catalog}
          onInsert={(name) => {
            setChain((c) => [...c, { [name]: {} }]);
            setChainJSON(JSON.stringify([...chain, { [name]: {} }], null, 2));
            setView('cards');
          }}
        />
      </Drawer>
    </div>
  );
}
