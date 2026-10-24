import { DownOutlined, QuestionCircleOutlined, UpOutlined } from '@ant-design/icons';
import { AutoComplete, Input, InputNumber, Select, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import ModelSelect from '@/pages/ai-assistant/modules/ModelSelect';
import { searchModelPovider } from '@/service/api/model-provider';
import { formatESSearchResult } from '@/service/request/es';

/**
 * 处理器「常规设置项」形态的共享 UI：一行开关（启/停）+ 可展开的参数表单。
 *
 * 消费方：
 * - 设置后台 ProcessingSettings（固定管道的处理器链设置项）
 * - 新建管道 GraphicalChainEditor（模板选入后的图形化处理器链）
 *
 * 字段元数据来自 GET /pipeline/processors 的处理器注册表；显式
 * PROCESSOR_FIELD_DEFS 声明类型与 i18n label，注册表里出现但未声明的
 * 标量字段按 string 兜底，管道接线字段（消息键/队列）不对外暴露。
 */

export const PROCESSOR_WIRING_FIELDS = new Set(['message_field', 'output_queue', 'input_queue', 'queues']);

export type ProcessorFieldType = 'string' | 'number' | 'bool' | 'array';

export const PROCESSOR_FIELD_DEFS: Record<string, { type: ProcessorFieldType; labelKey: string }> = {
  tika_endpoint: { type: 'string', labelKey: 'tika_endpoint' },
  tika_timeout_in_seconds: { type: 'number', labelKey: 'tika_timeout_in_seconds' },
  chunk_size: { type: 'number', labelKey: 'chunk_size' },
  extract_attachments: { type: 'bool', labelKey: 'extract_attachments' },
  xlsx_first_row_as_header: { type: 'bool', labelKey: 'xlsx_first_row_as_header' },
  image_content_format: { type: 'string', labelKey: 'image_content_format' },
  remote_parse_url: { type: 'string', labelKey: 'remote_parse_url' },
  remote_parse_exts: { type: 'array', labelKey: 'remote_parse_exts' },
  remote_parse_timeout_in_seconds: { type: 'number', labelKey: 'remote_parse_timeout_in_seconds' },
  ai_insights_max_length: { type: 'number', labelKey: 'ai_insights_max_length' },
  max_input_document_length: { type: 'number', labelKey: 'max_input_document_length' },
  min_input_document_length: { type: 'number', labelKey: 'min_input_document_length' },
  llm_generation_lang: { type: 'string', labelKey: 'llm_generation_lang' },
  model: { type: 'string', labelKey: 'model' },
  model_provider: { type: 'string', labelKey: 'model_provider' },
  model_context_length: { type: 'number', labelKey: 'model_context_length' },
  vision_model: { type: 'string', labelKey: 'vision_model' },
  vision_model_provider: { type: 'string', labelKey: 'vision_model_provider' },
  pigo_facefinder_path: { type: 'string', labelKey: 'pigo_facefinder_path' }
};

/** i18n 缺键时 t() 返回键原文——退化为 fallback。 */
const safeT = (t: (k: string) => string, key: string, fallback: string) => {
  const v = t(key);
  return !v || v === key || v.startsWith('page.') ? fallback : v;
};

/** 字段描述（hover 提示）：i18n 无该字段的描述键时返回空串，即不加问号 icon。 */
const fieldDesc = (t: (k: string) => string, key: string) =>
  safeT(t, `page.settings.processing.field_descs.${key}`, '');

/** 字段标签：有描述时在标签后追加问号 icon，hover 显示说明。 */
const FieldLabel = ({ t, keyName, fallback }: { t: (k: string) => string; keyName: string; fallback: string }) => {
  const desc = fieldDesc(t, keyName);
  return (
    <span className='text-13px inline-flex items-center gap-4px' style={{ color: 'var(--ant-color-text-secondary)' }}>
      {safeT(t, `page.settings.processing.fields.${keyName}`, fallback)}
      {desc && (
        <Tooltip title={desc}>
          <QuestionCircleOutlined style={{ color: 'var(--ant-color-text-quaternary)', fontSize: 13 }} />
        </Tooltip>
      )}
    </span>
  );
};

// ── 模型选择 ─────────────────────────────────────────────────────────
// 处理器配置里的模型字段是扁平标量对（model+model_provider /
// vision_model+vision_model_provider），UI 侧合并为现有的模型选择组件：
// 选一个模型同时写回两个键。

// (model key, provider key)：model 对供 LLM/embedding 处理器、vision 对供
// 视觉抽取（OCR/人脸/图片描述），均不过滤模型类型（embedding 与 llm 同用
// model 键，前端无法从注册表区分类型）。
const MODEL_FIELD_PAIRS: Array<{ model: string; provider: string }> = [
  { model: 'model', provider: 'model_provider' },
  { model: 'vision_model', provider: 'vision_model_provider' }
];

const isModelField = (k: string) => MODEL_FIELD_PAIRS.some(p => p.model === k || p.provider === k);

/** 拉取已启用的模型提供商列表（权限不足/失败时返回空，UI 退化为文本输入）。 */
export function useModelProviders(): { loaded: boolean; providers: any[]; reload: () => void } {
  const [providers, setProviders] = useState<any[]>([]);
  const [loaded, setLoaded] = useState(false);
  const fetch = () => {
    setLoaded(false);
    searchModelPovider({ from: 0, size: 10000, filter: { enabled: [true] } })
      .then(res => {
        if (res?.data) setProviders((formatESSearchResult(res.data).data as any[]) || []);
      })
      .catch(() => {})
      .finally(() => setLoaded(true));
  };
  useEffect(() => {
    fetch();
  }, []);
  return { loaded, providers, reload: fetch };
}

// 模型类型口径与设置-默认参数（DefaultParams.ModelSelectItem）一致：
// vision 对 → vision；model 对按处理器语义分型（embedding 类处理器吃 embedding 模型，其余吃 language 模型）。
const modelFieldType = (processorName: string, isVision: boolean): string => {
  if (isVision) return 'vision';
  return /embedding/.test(processorName) ? 'embedding' : 'language';
};

/** 与 DefaultParams.ModelSelectItem 同款：按 model.type 过滤每个提供商下的模型列表。
 *  语言模型额外按名称排除 rerank/embedding——部分提供商数据里同一模型有两条
 *  不同 type 的记录（如 reranker 同时挂 language），单靠 type 会漏网。 */
const filterProvidersByType = (providers: any[], type: string) =>
  providers.map(item => ({
    ...item,
    models: (item.models || []).filter((m: any) => {
      if (m.type !== type) return false;
      if (type === 'language' && /rerank|embedding/i.test(m.name || '')) return false;
      return true;
    })
  }));

/** 模型对字段标签：按类型显示具体模型类型（Embedding 模型 / 语言模型 / 视觉模型）。 */
const modelFieldLabelKey = (type: string) =>
  type === 'embedding' ? 'fields.embedding_model' : type === 'vision' ? 'fields.vision_model' : 'fields.model';

/** 单个模型对的选择控件：值 {name, provider_id} ↔ cfg.model / cfg.model_provider。 */
const ModelPairControl = ({
  label,
  value,
  providers,
  onRefresh,
  onChange
}: {
  label: string;
  value: { name?: string; provider_id?: string };
  providers: any[];
  onRefresh?: () => void;
  onChange: (v: { name?: string; provider_id?: string } | null) => void;
}) => {
  const { t } = useTranslation();
  return (
    <div className='flex flex-col gap-2px py-4px'>
      <span className='text-13px' style={{ color: 'var(--ant-color-text-secondary)' }}>
        {label}
      </span>
      <ModelSelect
        providers={providers}
        showSettings={false}
        showTemplate={false}
        allowClear
        onRefresh={onRefresh}
        width={500}
        // 未选择时提示走产品默认模型；清空选择即回落默认
        placeholder={t('page.settings.processing.placeholders.default_model')}
        value={value?.name || value?.provider_id ? { name: value.name, provider_id: value.provider_id } : undefined}
        onChange={(v: any) => onChange(v?.name || v?.provider_id ? { name: v?.name, provider_id: v?.provider_id } : null)}
      />
    </div>
  );
};

/** 表单字段 = 显式定义 ∪ 注册表标量字段（去掉接线字段），显式定义在前、保持稳定顺序。
 *  excludeModelFields：模型字段改为独立的模型选择控件渲染时，从平铺字段中剔除。 */
export function processorFieldKeys(
  name: string,
  config: Record<string, any> | undefined,
  registry: Record<string, any>,
  excludeModelFields = false
): string[] {
  const registryProps = (registry?.[name]?.properties || {}) as Record<string, any>;
  const dynamicKeys = Object.keys(registryProps).filter(
    k =>
      !PROCESSOR_WIRING_FIELDS.has(k) &&
      !PROCESSOR_FIELD_DEFS[k] &&
      ['string', 'uint32', 'int', 'float64', 'bool'].includes(registryProps[k]?.type)
  );
  const pick = (keys: string[]) => (excludeModelFields ? keys.filter(k => !isModelField(k)) : keys);
  return [
    ...pick(Object.keys(PROCESSOR_FIELD_DEFS).filter(k => k in registryProps || k in (config || {}))),
    ...pick(dynamicKeys)
  ];
}

/** 单个参数控件：按字段类型渲染，受控值直接写回调用方的 paramsMap。 */
export const ProcessorFieldControl = ({
  fieldKey,
  def,
  value,
  onChange
}: {
  fieldKey: string;
  def: { type: ProcessorFieldType };
  value: any;
  onChange: (v: any) => void;
}) => {
  const { t } = useTranslation();
  // placeholder 缺 i18n key 时退化为无 placeholder。
  const ph = (k: string) => {
    const v = t(`page.settings.processing.placeholders.${k}`);
    return v.startsWith('page.settings') ? undefined : v;
  };
  switch (def.type) {
    case 'bool':
      return (
        <div className='w-fit'>
          <Switch checked={!!value} onChange={onChange} size='small' />
        </div>
      );
    case 'number':
      return <InputNumber className='w-500px!' min={0} value={value} onChange={v => onChange(v)} placeholder={ph(fieldKey)} />;
    case 'array':
      return (
        <Select className='w-500px!' mode='tags' open={false} value={value || []} onChange={(v: string[]) => onChange(v)} placeholder={ph(fieldKey)} />
      );
    default:
      // LLM 生成语言：BCP 47 标签（后端空/非法回落 en-US），预设常用项 + 可自由输入自定义标签
      if (fieldKey === 'llm_generation_lang') {
        return (
          <AutoComplete
            className='w-500px!'
            value={value}
            onChange={onChange}
            options={[
              { value: 'en-US', label: 'English' },
              { value: 'zh-CN', label: '中文' }
            ]}
            placeholder={ph(fieldKey)}
          />
        );
      }
      return <Input className='w-500px!' value={value} onChange={e => onChange(e.target.value)} placeholder={ph(fieldKey)} />;
  }
};

/** 一个处理器的设置行：开关 + 名称/描述 + 可展开参数表。 */
export const ProcessorRow = ({
  name,
  config,
  enabled,
  expanded,
  registry,
  modelProviders = [],
  onModelRefresh,
  onToggleEnabled,
  onFieldChange,
  onPatch,
  onToggleExpand
}: {
  name: string;
  config: Record<string, any> | undefined;
  enabled: boolean;
  expanded: boolean;
  registry: Record<string, any>;
  modelProviders?: any[];
  onModelRefresh?: () => void;
  onToggleEnabled: (name: string, on: boolean) => void;
  onFieldChange: (name: string, key: string, value: any) => void;
  /** 一次写多键（模型对 model+provider 必须原子更新，两次单键调用会互相覆盖）。 */
  onPatch: (name: string, patch: Record<string, any>) => void;
  onToggleExpand: () => void;
}) => {
  const { t } = useTranslation();
  const cfg = config || {};
  // 模型字段在提供商列表可用时合并为模型选择控件，否则按普通文本输入兜底
  const registryProps = (registry?.[name]?.properties || {}) as Record<string, any>;
  const presentPairs = modelProviders.length
    ? MODEL_FIELD_PAIRS.filter(p => p.model in registryProps || p.model in cfg || p.provider in registryProps || p.provider in cfg)
    : [];
  const fieldKeys = processorFieldKeys(name, cfg, registry, presentPairs.length > 0);
  const hasParams = fieldKeys.length > 0 || presentPairs.length > 0;

  return (
    <div
      className='rounded-8px'
      style={{ border: '1px solid var(--ant-color-border-secondary)', background: 'var(--ant-color-bg-container)' }}
    >
      {/* 整个卡片头部都是展开/收起的点击热区；开关阻止冒泡，避免切换两次 */}
      <div
        className={`flex items-center gap-12px px-16px py-12px ${hasParams ? 'cursor-pointer select-none' : ''}`}
        onClick={hasParams ? onToggleExpand : undefined}
      >
        <span className='flex items-center' onClick={e => e.stopPropagation()}>
          <Switch checked={enabled} onChange={on => onToggleEnabled(name, on)} size='small' />
        </span>
        <div className='min-w-0 flex-1'>
          <div className='flex items-center gap-8px text-14px font-500'>
            {safeT(t, `page.settings.processing.processors.${name}.name`, name)}
            {hasParams && (
              <span className='flex items-center text-12px' style={{ color: 'var(--ant-color-text-tertiary)' }}>
                {expanded ? <UpOutlined /> : <DownOutlined />}
              </span>
            )}
          </div>
          <div className='text-12px' style={{ color: 'var(--ant-color-text-tertiary)' }}>
            {safeT(t, `page.settings.processing.processors.${name}.desc`, '')}
          </div>
        </div>
      </div>
      {expanded && enabled && hasParams && (
        <div className='flex flex-col px-16px pb-16px'>
          {presentPairs.map(p => {
            const mType = modelFieldType(name, p.model === 'vision_model');
            const typedProviders = filterProvidersByType(modelProviders, mType);
            return (
              <ModelPairControl
                key={p.model}
                label={safeT(t, `page.settings.processing.${modelFieldLabelKey(mType)}`, p.model)}
                providers={typedProviders}
                onRefresh={onModelRefresh}
                value={{ name: cfg[p.model], provider_id: cfg[p.provider] }}
                onChange={v => onPatch(name, { [p.model]: v?.name ?? '', [p.provider]: v?.provider_id ?? '' })}
              />
            );
          })}
          {fieldKeys.map(k => {
            const def = PROCESSOR_FIELD_DEFS[k] || { type: 'string' as ProcessorFieldType, labelKey: k };
            return (
              <div key={k} className='flex flex-col gap-2px py-4px'>
                <FieldLabel t={t} keyName={def.labelKey} fallback={k} />
                <ProcessorFieldControl
                  fieldKey={k}
                  def={def}
                  value={cfg[k]}
                  onChange={(v: any) => onFieldChange(name, k, v)}
                />
              </div>
            );
          })}
        </div>
      )}
      {expanded && !enabled && hasParams && (
        <div className='px-16px pb-12px text-12px' style={{ color: 'var(--ant-color-text-quaternary)' }}>
          {t('page.settings.processing.disabled_hint')}
        </div>
      )}
    </div>
  );
};
