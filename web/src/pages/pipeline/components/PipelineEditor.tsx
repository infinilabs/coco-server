import { BookOpen, Boxes, Braces, Check, ChevronDown, ChevronUp, File, FileText, FlaskConical, Image, Paperclip, Pencil, Rocket, SlidersHorizontal, Sparkles, Trash, Video } from 'lucide-react';
import { RightOutlined } from '@ant-design/icons';
import { Button, Card, Collapse, Drawer, Form, Input, InputNumber, Popconfirm, Select, Space, Switch, Tag, Typography, message } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { aiGeneratePipelineChain, createPipeline, getPipelineProcessors, testPipelineChain, updatePipeline } from '@/service/api/pipeline';
// shared studio components (ui-common package); the prebuilt stylesheet
// carries the tailwind utilities they use — coco has no tailwind toolchain
import { AiGeneratePanel, DryRunDrawer, identityTranslate, type DryRunReport } from '@infinilabs/pipeline-ui';
import '@infinilabs/pipeline-ui/styles.css';
// reuses the settings pages' .settings-form layout (fixed left label column,
// capped control width) for the name/description + advanced settings section
import '@/pages/settings/index.scss';
import '@/pages/pipeline/index.scss';
import { availableTemplates, type PipelineTemplate } from './pipelineTemplates';
import { GraphicalChainEditor } from './GraphicalChainEditor';

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
  labels?: Record<string, any>;
}

// 创建模式持久化：编辑时按此标签还原各自的编辑模式（模板创建→图形化，
// 高级模式/AI 生成→自由编排）。无标签的存量管道按链形态推断。
const EDITOR_MODE_LABEL = 'editor_mode';
type EditorMode = 'graphical' | 'free';

interface ProcessorEntry {
  name: string;
  category?: string;
  config?: Record<string, any>;
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

/** 链上全部是单键处理器条目（无 if 分支、无空条目）才能渲染为图形化行。 */
function isGraphableChain(entries: Record<string, any>[]): boolean {
  return entries.length > 0 && entries.every(e => !('if' in e) && Object.keys(e).length === 1);
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

const TEMPLATE_ICONS: Record<PipelineTemplate['icon'], React.ReactNode> = {
  'file-text': <FileText size={18} />,
  file: <File size={18} />,
  image: <Image size={18} />,
  video: <Video size={18} />,
  paperclip: <Paperclip size={18} />,
  sliders: <SlidersHorizontal size={18} />
};

/** One-click starter chains for common document tasks; shown when creating. */
function TemplateGallery({
  templates,
  selectedId,
  onPick
}: {
  templates: PipelineTemplate[];
  selectedId: string | null;
  onPick: (t: PipelineTemplate) => void;
}) {
  const { t } = useTranslation();
  if (templates.length === 0) return null;
  return (
    <div className="flex flex-wrap gap-12px">
      {templates.map(tpl => (
        <div
          key={tpl.id}
          className={`w-220px rounded-8px border border-solid p-12px flex flex-col gap-8px transition-colors ${
            selectedId === tpl.id
              ? 'border-[var(--ant-color-primary)] bg-[var(--ant-color-primary-bg)]'
              : 'border-[var(--ant-color-border-secondary)] hover:border-[var(--ant-color-primary)]'
          }`}
        >
          <div className="flex items-center gap-8px text-[var(--ant-color-primary)]">
            {TEMPLATE_ICONS[tpl.icon]}
            <span className="text-13px font-medium text-[var(--ant-color-text)]">{t(`page.pipeline.templates.${tpl.id}.title`)}</span>
          </div>
          <span className="text-12px text-[var(--ant-color-text-tertiary)] leading-snug">{t(`page.pipeline.templates.${tpl.id}.description`)}</span>
          <Button size="small" block className="mt-auto!" onClick={() => onPick(tpl)}>
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

  const [advancedOpen, setAdvancedOpen] = useState(false);
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

  // 处理器链的编辑模式由创建来源决定、不互切，编辑时按 labels.editor_mode
  // 还原各自的模式：
  // - 内置模板创建 → 图形化配置（行来源=模板链）
  // - 高级模式模板创建 / AI 生成 → 自由编排（ChainEditor，完全自定义）
  // - 编辑既有管道 → 按 labels.editor_mode；无标签的存量管道按链形态推断
  //   （可图形化→图形化，含 if 分支/空链→自由编排）
  const [graphicalSource, setGraphicalSource] = useState<Record<string, any>[] | null>(() => {
    const labeled = initial?.labels?.[EDITOR_MODE_LABEL];
    if (labeled === 'free') return null;
    if (labeled === 'graphical' && isGraphableChain(initial?.processor ?? [])) return initial!.processor!;
    // 无标签：按链形态推断
    if (initial?.id && isGraphableChain(initial.processor ?? [])) return initial.processor!;
    return null;
  });
  const [graphicalKey, setGraphicalKey] = useState(0);

  const [catalog, setCatalog] = useState<ProcessorEntry[]>([]);
  const templates = useMemo(() => {
    const installed = new Set(catalog.map(p => p.name));
    return availableTemplates(installed);
  }, [catalog]);
  // 当前选中的模板（新建页默认选中知识加工全链路）
  const [selectedTplId, setSelectedTplId] = useState<string | null>(null);
  const defaultAppliedRef = useRef(false);
  // 新建页：处理器目录加载完成后自动应用默认模板（知识加工全链路 8 节）
  useEffect(() => {
    if (initial || defaultAppliedRef.current) return;
    const tpl = templates.find(t => t.id === 'general-enrich');
    if (tpl) {
      defaultAppliedRef.current = true;
      applyTemplate(tpl);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [templates]);
  // GraphicalChainEditor 需要按名索引的注册表形态
  const registryMeta = useMemo(() => Object.fromEntries(catalog.map(p => [p.name, { properties: p.config || {} }])), [catalog]);
  const [catalogOpen, setCatalogOpen] = useState(false);

  const [documentsJSON, setDocumentsJSON] = useState(() => JSON.stringify(DOCUMENT_TEMPLATE, null, 2));
  const [testReport, setTestReport] = useState<DryRunReport | null>(null);
  const [aiMode, setAiMode] = useState<'generate' | 'refine'>('generate');
  const [aiError, setAiError] = useState<string | null>(null);
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
    setSelectedTplId(tpl.id);
    if (tpl.id === 'advanced') {
      // 高级模式：不预设处理器，保留自由编排（完全自定义）
      setGraphicalSource(null);
    } else {
      // 内置模板：处理器链以常规设置项 UI 呈现
      setGraphicalSource(tpl.processor);
      setGraphicalKey(k => k + 1);
    }
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

  const runTest = async (chainOverride?: Record<string, any>[]) => {
    const docs = parseDocuments();
    if (!docs) return;
    if (view === 'json') switchView('cards');
    setTesting(true);
    try {
      const res = await testPipelineChain({ processor: chainOverride ?? chain, documents: docs });
      setTestReport(res?.data?.report ?? null);
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
    setTestReport(null);
    setAiError(null);
    try {
      const res = await aiGeneratePipelineChain({
        documents: docs.map((d) => JSON.stringify(d)),
        requirements: aiRequirement,
        mode,
        current_chain: mode === 'refine' ? chain : undefined
      });
      const data = res?.data ?? {};
      const nextChain = Array.isArray(data.chain) ? data.chain : [];
      setChain(nextChain);
      setChainJSON(JSON.stringify(nextChain, null, 2));
      // AI 整链替换后切回自由编排
      setGraphicalSource(null);
      message.success(t('page.pipeline.editor.aiDone'));
      // generated chains are not trusted until the real engine replays
      // them — run the unified dry-run on the new chain right away
      if (nextChain.length > 0) await runTest(nextChain);
    } catch (e: any) {
      const msg = e?.response?.data?.message ?? e?.message ?? t('page.pipeline.editor.aiFailed');
      setAiError(msg);
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
      // 持久化当前编辑模式：编辑时按此进入各自的模式
      const mode: EditorMode = graphicalSource ? 'graphical' : 'free';
      const payload = {
        name: values.name,
        description: values.description ?? '',
        ...advanced,
        labels: { ...(initial?.labels || {}), [EDITOR_MODE_LABEL]: mode },
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
        <Form
          form={form}
          className="settings-form pt-8px pb-24px"
          colon={false}
          labelAlign="left"
          initialValues={{ name: initial?.name, description: initial?.description }}
        >
          <Form.Item name="name" label={t('page.pipeline.editor.name')} rules={[{ required: true }]} extra={initial?.id ? undefined : t('page.pipeline.editor.nameAsId')}>
            <Input disabled={!!initial?.id} placeholder="my-enrichment-pipeline" style={{ width: 500 }} />
          </Form.Item>
          <Form.Item name="description" label={t('page.pipeline.editor.description')}>
            <Input style={{ width: 500 }} />
          </Form.Item>
          <Collapse
          ghost
          activeKey={advancedOpen ? ['advanced'] : []}
          onChange={(keys) => setAdvancedOpen(keys.includes('advanced'))}
          expandIcon={() => null}
          items={[
            {
              key: 'advanced',
              label: (
                <span className="text-13px inline-flex items-center gap-6px">
                  {t('page.pipeline.editor.advanced')}
                  <RightOutlined
                    rotate={advancedOpen ? 90 : 0}
                    style={{ fontSize: 11, color: 'var(--ant-color-text-tertiary)' }}
                  />
                </span>
              ),
              children: (
                <div className="pipeline-advanced-pane flex flex-col pt-4px">
                  <Form.Item label={t('page.pipeline.editor.enabled')}>
                    <Switch size="small" checked={advanced.enabled} onChange={(v) => setAdvanced((a) => ({ ...a, enabled: v }))} />
                  </Form.Item>
                  <Form.Item label={t('page.pipeline.editor.singleton')} tooltip={t('page.pipeline.editor.singletonTip')}>
                    <Switch size="small" checked={advanced.singleton} onChange={(v) => setAdvanced((a) => ({ ...a, singleton: v }))} />
                  </Form.Item>
                  <Form.Item label={t('page.pipeline.editor.autoStart')} tooltip={t('page.pipeline.editor.autoStartTip')}>
                    <Switch size="small" checked={advanced.auto_start} onChange={(v) => setAdvanced((a) => ({ ...a, auto_start: v }))} />
                  </Form.Item>
                  <Form.Item label={t('page.pipeline.editor.keepRunning')} tooltip={t('page.pipeline.editor.keepRunningTip')}>
                    <Switch size="small" checked={advanced.keep_running} onChange={(v) => setAdvanced((a) => ({ ...a, keep_running: v }))} />
                  </Form.Item>
                  <Form.Item label={t('page.pipeline.editor.retryDelay')} tooltip={t('page.pipeline.editor.retryDelayTip')}>
                    <InputNumber min={0} style={{ width: 500 }} value={advanced.retry_delay_in_ms} onChange={(v) => setAdvanced((a) => ({ ...a, retry_delay_in_ms: v ?? 1000 }))} />
                  </Form.Item>
                </div>
              )
            }
          ]}
        />
        </Form>
      </Card>

      {!initial?.id && (
        <Card
          className="card-wrapper"
          title={<span className="flex items-center gap-8px"><Sparkles size={16} />{t('page.pipeline.templates.title')}</span>}
        >
          <TemplateGallery templates={templates} selectedId={selectedTplId} onPick={applyTemplate} />
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
          graphicalSource ? undefined : (
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
          )
        }
      >
        {graphicalSource ? (
          <GraphicalChainEditor
            key={graphicalKey}
            templateChain={graphicalSource}
            registry={registryMeta}
            onChange={(c) => {
              setChain(c);
              setChainJSON(JSON.stringify(c, null, 2));
            }}
          />
        ) : view === 'cards' ? (
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
        <div className="grid grid-cols-2 gap-16px items-start">
          <div className="flex flex-col gap-8px min-w-0">
            <div className="flex items-center justify-between h-24px">
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
            {/* AI 生成/改进是自由编排能力：结果整链替换并切回自由编排，图形化模式下不提供（方案：图形化只留测试）。
                共享 AiGeneratePanel (#110)：哑交互壳，产品绑定统一 ai-generate 端点并自行串联 dry-run。 */}
            {!graphicalSource && (
              <AiGeneratePanel
                value={aiRequirement}
                onChange={setAiRequirement}
                mode={aiMode}
                onModeChange={setAiMode}
                onRun={() => runAI(aiMode)}
                busy={aiRunning != null}
                error={aiError}
                canRefine={!chainEmpty}
                t={identityTranslate}
              />
            )}
            <Button icon={<FlaskConical size={14} />} loading={testing} disabled={chainEmpty} onClick={() => runTest()}>
              {t('page.pipeline.editor.runTest')}
            </Button>
          </div>
          <div className="min-w-0">
            <div className="flex items-center h-24px">
              <span className="text-13px font-medium">{t('page.pipeline.editor.stepDebug')}</span>
            </div>
            <div className="mt-8px">
              {/* 统一 dry-run 报告单源渲染 (#110)：编译判定/样本切换/逐步
                  outcome+耗时+字段 diff — 与 logpilot 画布试跑同款。 */}
              {testReport ? (
                <DryRunDrawer report={testReport} t={identityTranslate} />
              ) : (
                <div className="border-dashed border border-[var(--ant-color-border)] rounded-8px py-24px text-center text-13px text-[var(--ant-color-text-tertiary)]">
                  {t('page.pipeline.editor.noTestYet')}
                </div>
              )}
            </div>
          </div>
        </div>
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
