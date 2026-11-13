import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { ProcessorRow, useModelProviders } from '@/components/common/ProcessorChainUI';

/**
 * 图形化处理器链编辑器：内置模板选入后，处理器链以「常规设置项」形态
 * 呈现（与设置后台 ProcessingSettings 同款 UI）——每行一个处理器：开关
 * （启/停）+ 可展开参数表单。链顺序由模板固定，不提供重排/插入/JSON
 * 编辑；需要完全自定义时切回自由编排（高级模式）。
 *
 * 停用的处理器参数保留在组件内部，重新开启后表单值不丢；对外只回调
 * 启用中的处理器（顺序即模板链顺序）。
 */

const entryName = (entry: Record<string, any>): string => Object.keys(entry)[0] ?? '?';
const entryConfig = (entry: Record<string, any>): Record<string, any> => {
  const cfg = entry[entryName(entry)];
  return cfg && typeof cfg === 'object' ? cfg : {};
};

export function GraphicalChainEditor({
  templateChain,
  onChange,
  registry
}: {
  /** 模板定义的完整处理器链（顺序即产品语义），作为行的唯一来源。 */
  templateChain: Record<string, any>[];
  /** 链变化回调：仅包含启用中的处理器，{{name}: config} 形态、模板顺序。 */
  onChange: (next: Record<string, any>[]) => void;
  /** GET /pipeline/processors 注册表（字段元数据）。 */
  registry: Record<string, any>;
}) {
  const { t } = useTranslation();
  const { providers: modelProviders, reload: reloadModelProviders } = useModelProviders();
  const [params, setParams] = useState<Record<string, Record<string, any>>>(() =>
    Object.fromEntries(templateChain.map(e => [entryName(e), entryConfig(e)]))
  );
  const [enabled, setEnabled] = useState<Set<string>>(() => new Set(templateChain.map(entryName)));
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const emit = (enabledSet: Set<string>, paramsMap: Record<string, Record<string, any>>) =>
    onChange(templateChain.filter(e => enabledSet.has(entryName(e))).map(e => ({ [entryName(e)]: paramsMap[entryName(e)] || {} })));

  // 空值剔除：避免把 placeholder 持久化成字面配置。
  const pruneEmpty = (cfg: Record<string, any>) => {
    Object.keys(cfg).forEach(k => {
      if (cfg[k] === '' || cfg[k] === null || cfg[k] === undefined) delete cfg[k];
    });
    return cfg;
  };

  // 参数更新全部走函数式 setState：同一次交互可能连续写多个键（如模型对
  // model+provider），基于旧 state 展开会让后一次覆盖前一次。
  const setField = (name: string, key: string, value: any) => {
    setParams(prev => {
      const merged = pruneEmpty({ ...(prev[name] || {}), [key]: value });
      return { ...prev, [name]: merged };
    });
  };

  const patchField = (name: string, patch: Record<string, any>) => {
    setParams(prev => {
      const merged = pruneEmpty({ ...(prev[name] || {}), ...patch });
      return { ...prev, [name]: merged };
    });
  };

  // 状态变化后统一对外 emit（函数式 setState 里做副作用会在 StrictMode 下双触发）。
  useEffect(() => {
    emit(enabled, params);
  }, [params, enabled]);

  const toggleEnabled = (name: string, on: boolean) => {
    setEnabled(prev => {
      const next = new Set(prev);
      if (on) next.add(name);
      else next.delete(name);
      return next;
    });
  };

  const toggleExpand = (name: string) =>
    setExpanded(prev => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });

  if (templateChain.length === 0) {
    return (
      <div className='border-dashed border border-[var(--ant-color-border)] rounded-8px py-24px text-center text-13px text-[var(--ant-color-text-tertiary)]'>
        {t('page.pipeline.editor.emptyChain')}
      </div>
    );
  }

  return (
    <div className='flex flex-col gap-8px'>
      {templateChain.map(entry => {
        const name = entryName(entry);
        return (
          <ProcessorRow
            key={name}
            name={name}
            config={params[name]}
            enabled={enabled.has(name)}
            expanded={expanded.has(name)}
            registry={registry}
            modelProviders={modelProviders}
            onModelRefresh={reloadModelProviders}
            onToggleEnabled={toggleEnabled}
            onFieldChange={setField}
            onPatch={patchField}
            onToggleExpand={() => toggleExpand(name)}
          />
        );
      })}
    </div>
  );
}
