import { Spin } from 'antd';
import SaveBar from './SaveBar';
import { getPipeline, updatePipeline, getPipelineProcessors } from '@/service/api/pipeline';
import { useLoading } from '@sa/hooks';
import { useTranslation } from 'react-i18next';
import { ProcessorRow, useModelProviders } from '@/components/common/ProcessorChainUI';

/**
 * 解析管道（Processing Pipelines）设置模块。
 *
 * 产品只依赖两条固定管道（enrich_documents / enrich_attachments），处理器链
 * 的顺序由产品定义、不可编辑——这里把链上的每个处理器以「常规设置项」的
 * 形态呈现：一行开关（启/停）+ 可展开的参数表单（共享 UI 见
 * components/common/ProcessorChainUI.tsx，字段来自 GET /pipeline/processors
 * 的处理器元数据注册表，未知字段兜底为文本输入）。
 * 保存时按固定顺序重组 processor 数组整体 PUT，被关掉的处理器即从链中移除。
 *
 * 自由编排的 Pipeline Studio（/pipeline）保留但隐藏入口，供开发调试。
 */

// 两条固定管道与处理器链顺序 —— 顺序即产品语义（类型检测→元数据→封面→解析→人脸→摘要→标签→向量化）。
const DOC_CHAIN = [
  'file_type_detection',
  'file_metadata',
  'generate_document_cover',
  'document_text_attachment_extraction',
  'face_extraction',
  'document_summarization',
  'extract_tags',
  'document_embedding'
] as const;

const ATTACH_CHAIN = ['attachment_text_extraction'] as const;

const PIPELINES = [
  { id: 'enrich_documents', chain: DOC_CHAIN, titleKey: 'doc_section', descKey: 'doc_section_desc' },
  { id: 'enrich_attachments', chain: ATTACH_CHAIN, titleKey: 'attach_section', descKey: 'attach_section_desc' }
] as const;

type ChainState = {
  // 每个处理器的参数（config 对象）；即使停用也保留，重新开启时表单值不丢。
  params: Record<string, Record<string, any>>;
  enabled: Set<string>;
  // GET 到的整份 _source，PUT 时原样带回（只重组 processor 数组）。
  source: Record<string, any> | null;
};

const emptyChainState = (): ChainState => ({ params: {}, enabled: new Set(), source: null });

// 与 DefaultParams 的 SectionTitle 同款：细分割线小节标题。
const SectionTitle = ({ title }: { title: string }) => (
  <div
    className='mb-16px mt-4px pb-8px text-15px font-500'
    style={{ borderBottom: '1px solid var(--ant-color-split)' }}
  >
    {title}
  </div>
);

const ProcessingSettings = memo(() => {
  const { t } = useTranslation();
  const { endLoading, loading, startLoading } = useLoading();
  const { hasAuth } = useAuth();
  const { providers: modelProviders, reload: reloadModelProviders } = useModelProviders();
  const permissions = {
    read: hasAuth('generic#pipeline/search'),
    update: hasAuth('generic#pipeline/update')
  };

  const [registry, setRegistry] = useState<Record<string, any>>({});
  const [states, setStates] = useState<Record<string, ChainState>>({});
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [dirty, setDirty] = useState(false);

  const fetchAll = async () => {
    startLoading();
    try {
      const regRes = await getPipelineProcessors();
      setRegistry(regRes?.data || {});
      const next: Record<string, ChainState> = {};
      for (const p of PIPELINES) {
        const st = emptyChainState();
        try {
          const res = await getPipeline(p.id);
          const source = res?.data?._source || res?.data || null;
          if (source) {
            st.source = source;
            for (const entry of source.processor || []) {
              const [name, cfg] = Object.entries(entry || {})[0] || [];
              if (!name) continue;
              st.params[name] = cfg || {};
              st.enabled.add(name);
            }
          }
        } catch {
          // 管道不存在（首次初始化前）：整条链视为停用，保存时才创建。
        }
        next[p.id] = st;
      }
      setStates(next);
      setDirty(false);
    } finally {
      endLoading();
    }
  };

  useEffect(() => {
    if (permissions.read) fetchAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const mutate = (pipelineId: string, fn: (st: ChainState) => void) => {
    setStates(prev => {
      const st = { ...prev[pipelineId], params: { ...prev[pipelineId].params }, enabled: new Set(prev[pipelineId].enabled) };
      fn(st);
      return { ...prev, [pipelineId]: st };
    });
    setDirty(true);
  };

  const toggleEnabled = (pipelineId: string, name: string, on: boolean) =>
    mutate(pipelineId, st => (on ? st.enabled.add(name) : st.enabled.delete(name)));

  const setField = (pipelineId: string, name: string, key: string, value: any) =>
    mutate(pipelineId, st => {
      st.params[name] = { ...(st.params[name] || {}), [key]: value };
      // 空值不落库：避免把 placeholder 持久化成字面配置。
      if (value === '' || value === null || value === undefined) delete st.params[name][key];
    });

  // 一次写多键（模型对 model+provider 原子更新）。
  const setPatch = (pipelineId: string, name: string, patch: Record<string, any>) =>
    mutate(pipelineId, st => {
      st.params[name] = { ...(st.params[name] || {}), ...patch };
      Object.keys(patch).forEach(k => {
        if (patch[k] === '' || patch[k] === null || patch[k] === undefined) delete st.params[name][k];
      });
    });

  const toggleExpand = (name: string) =>
    setExpanded(prev => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });

  const save = async () => {
    startLoading();
    try {
      for (const p of PIPELINES) {
        const st = states[p.id];
        if (!st.source) continue;
        // PUT 对未提供字段按零值覆盖（已实测 keep_running 会被清掉），
        // 必须回传 source 的全部产品字段；仅排除服务端管理的系统元数据。
        const { _system, created, updated, ...payload } = st.source;
        const processor = p.chain.filter(name => st.enabled.has(name)).map(name => ({ [name]: st.params[name] || {} }));
        await updatePipeline(p.id, { ...payload, processor });
      }
      await fetchAll();
    } finally {
      endLoading();
    }
  };

  const discard = () => fetchAll();

  return (
    <>
      <div className='mb-16px text-12px' style={{ color: 'var(--ant-color-text-tertiary)' }}>
        {t('page.settings.processing.desc')}
      </div>
      <Spin spinning={loading}>
        {PIPELINES.map(p => {
          const st = states[p.id] || emptyChainState();
          return (
            <div key={p.id} className='mb-24px'>
              <SectionTitle title={t(`page.settings.processing.${p.titleKey}`)} />
              <div className='mb-8px text-12px' style={{ color: 'var(--ant-color-text-tertiary)' }}>
                {t(`page.settings.processing.${p.descKey}`)}
              </div>
              <div className='flex flex-col gap-8px'>
                {p.chain.map(name => (
                  <ProcessorRow
                    key={name}
                    registry={registry}
                    modelProviders={modelProviders}
                    onModelRefresh={reloadModelProviders}
                    expanded={expanded.has(name)}
                    name={name}
                    config={st.params[name]}
                    enabled={st.enabled.has(name)}
                    onFieldChange={(n: string, k: string, v: any) => setField(p.id, n, k, v)}
                    onPatch={(n: string, patch: Record<string, any>) => setPatch(p.id, n, patch)}
                    onToggleEnabled={(n: string, on: boolean) => toggleEnabled(p.id, n, on)}
                    onToggleExpand={() => toggleExpand(name)}
                  />
                ))}
              </div>
            </div>
          );
        })}
      </Spin>
      <SaveBar dirty={dirty} loading={loading} onDiscard={discard} onSave={save} />
    </>
  );
});

export default ProcessingSettings;
