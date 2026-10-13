import { CheckCircleFilled, CloseCircleFilled, MinusCircleFilled, RightOutlined } from '@ant-design/icons';
import { Alert, Empty, Spin, Tag, Typography } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { request } from '@/service/request';

interface PipelineRunStep {
  name?: string;
  took_ms?: number;
  error?: string;
}

interface PipelineRun {
  pipeline?: string;
  passthrough?: string;
  took_ms?: number;
  success?: boolean;
  at?: string;
  error?: string;
  steps?: PipelineRunStep[];
}

interface TimelineData {
  status?: string;
  processed?: boolean;
  error_message?: string;
  embedding_model?: string;
  pipeline_runs?: PipelineRun[];
}

// 处理器名 → 产物页签映射（与 pipelineTemplates / processors 注册名对齐）。
// 组件内用它决定步骤名的可点击样式（link 色 + ↗）；文档详情页消费同表做实际跳转。
export const STEP_TAB_MAP: Record<string, string> = {
  document_text_attachment_extraction: 'attachments',
  attachment_text_extraction: 'attachments',
  process_attachments: 'attachments',
  generate_document_cover: 'attachments',
  generate_attachment_cover: 'attachments',
  face_extraction: 'attachments',
  file_metadata: 'attachments',
  document_summarization: 'ai',
  extract_tags: 'ai',
  extract_entities: 'ai',
  document_embedding: 'chunks'
};

// 耗时统一人类可读：<1s 用 ms，≥1s 用一位小数的 s（run 级与 step 级同规则）
const formatDuration = (ms?: number) => {
  if (ms == null) return null;
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
};

/**
 * 文档智能解析时间线：生命周期状态 + 运行记录（pipeline/passthrough、
 * 时间、耗时、错误）。数据源：GET /document/:id/_timeline（W1/W13a）。
 * 运行记录卡片化：左侧状态色条 + 头部(管道名/耗时) + 步骤 mini 表；
 * 有映射的处理器步骤名以 link 色呈现，点击由调用方映射到产物页签。
 * maxRuns 用于窄容器（列表页状态浮窗）只显示最近 N 次运行。
 */
export default function ProcessingTimeline({
  docId,
  headers,
  onStepClick,
  maxRuns
}: {
  docId?: string;
  headers?: Record<string, string>;
  onStepClick?: (stepName?: string) => void;
  maxRuns?: number;
}) {
  const { t } = useTranslation();
  const [timeline, setTimeline] = useState<TimelineData | null>(null);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    if (!docId) return;
    setLoading(true);
    try {
      const res: any = await request({ method: 'get', url: `/document/${docId}/_timeline`, headers }).catch(() => null);
      setTimeline((res?.data as TimelineData) ?? null);
    } finally {
      setLoading(false);
    }
  }, [docId, headers]);

  useEffect(() => {
    load();
  }, [load]);

  const statusTag = (status?: string) => {
    switch (status) {
      case 'indexing':
        return <Tag color='processing'>{t('page.preview.processing.statusIndexing')}</Tag>;
      case 'completed':
        return <Tag color='success'>{t('page.preview.processing.statusCompleted')}</Tag>;
      case 'failed':
        return <Tag color='error'>{t('page.preview.processing.statusFailed')}</Tag>;
      default:
        return <Tag>{t('page.preview.processing.statusLegacy')}</Tag>;
    }
  };

  // passthrough records carry an internal reason code instead of a pipeline
  // name — map the known codes to human-readable labels, unknown codes pass
  // through unchanged
  const passthroughLabel = (code?: string) => {
    switch (code) {
      case 'no-datasource':
        return t('page.preview.processing.ptNoDatasource');
      case 'datasource-not-found':
        return t('page.preview.processing.ptDatasourceNotFound');
      case 'no-pipeline-configured':
        return t('page.preview.processing.ptNoPipeline');
      case 'pipeline-not-found':
        return t('page.preview.processing.ptPipelineNotFound');
      case 'pipeline-config-invalid':
        return t('page.preview.processing.ptPipelineInvalid');
      case 'pipeline-init-failed':
        return t('page.preview.processing.ptPipelineInitFailed');
      default:
        return code;
    }
  };

  // 运行卡片：左侧 2px 状态色条（成功绿 / 失败红 / passthrough 灰）
  const runAccent = (run: PipelineRun) => {
    if (run.success === false) return 'var(--ant-color-error)';
    if (run.pipeline) return 'var(--ant-color-success)';
    return 'var(--ant-color-text-quaternary)';
  };

  // per-processor steps mini table: dot + name + duration aligned in rows;
  // names with a STEP_TAB_MAP entry get link styling (+↗) when onStepClick is
  // provided — unmapped ones stay plain text since clicking does nothing
  const stepsList = (steps?: PipelineRunStep[]) => {
    if (!steps?.length) return null;
    return (
      <div className='mt-8px rounded-md bg-[var(--ant-color-fill-quaternary)] px-10px py-6px'>
        {steps.map((step, j) => {
          const linked = Boolean(onStepClick && step.name && STEP_TAB_MAP[step.name]);
          return (
            <div key={String(j)} className='flex flex-col'>
              <div
                className={`flex items-center gap-8px rounded-4px py-3px pr-4px pl-4px ${linked ? 'hover:bg-[var(--ant-color-fill-tertiary)]' : ''}`}
                {...(linked
                  ? { onClick: () => onStepClick?.(step.name), style: { cursor: 'pointer' } }
                  : {})}
              >
                {step.error ? (
                  <CloseCircleFilled className='shrink-0 text-12px text-[var(--ant-color-error)]' />
                ) : (
                  <span
                    className='h-6px w-6px shrink-0 rounded-full'
                    style={{ background: 'var(--ant-color-success)' }}
                  />
                )}
                <Typography.Text
                  className={`font-mono text-12px ${linked ? 'color-[var(--ant-color-link)]!' : ''}`}
                >
                  {step.name || '-'}
                </Typography.Text>
                {linked ? <RightOutlined className='text-10px color-[var(--ant-color-link)]' /> : null}
                {step.took_ms != null ? (
                  <Typography.Text type='secondary' className='ml-auto font-mono text-11px'>
                    {formatDuration(step.took_ms)}
                  </Typography.Text>
                ) : null}
              </div>
              {step.error ? (
                <Typography.Text className='mb-2px ml-18px text-11px break-all text-[var(--ant-color-error)]'>
                  {step.error}
                </Typography.Text>
              ) : null}
            </div>
          );
        })}
      </div>
    );
  };

  const runs = timeline?.pipeline_runs ?? [];
  const visibleRuns = maxRuns != null ? runs.slice(0, maxRuns) : runs;
  const truncated = maxRuns != null && runs.length > maxRuns;

  // 空态 1：完全无解析记录（timeline 接口无数据，如存量文档）——只给缺省说明，
  // 不渲染 Legacy 状态标签误导用户
  if (!loading && !timeline) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.preview.processing.noRuns')} />;
  }

  return (
    <Spin spinning={loading}>
      <div className='flex flex-col gap-12px'>
        {/* 概览行：生命周期状态 + 运行次数 + 向量模型 */}
        <div className='flex flex-wrap items-center gap-8px'>
          {statusTag(timeline?.status)}
          <Typography.Text type='secondary' className='text-12px'>
            {t('page.preview.processing.runsCount', { count: String(runs.length) })}
          </Typography.Text>
          {timeline?.embedding_model ? (
            <Typography.Text type='secondary' className='ml-auto font-mono text-12px'>
              {timeline.embedding_model}
            </Typography.Text>
          ) : null}
        </div>
        {timeline?.error_message ? <Alert type='error' showIcon message={timeline.error_message} /> : null}

        {runs.length === 0 ? (
          // 空态 2：有生命周期状态但无运行记录
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.preview.processing.noRuns')} />
        ) : (
          visibleRuns.map((run, i) => (
            <div
              key={String(i)}
              className='rounded-md border border-[var(--ant-color-border-secondary)] py-10px pr-12px pl-12px'
              style={{ borderLeft: `2px solid ${runAccent(run)}` }}
            >
              <div className='flex flex-wrap items-center gap-8px'>
                {run.pipeline ? (
                  <Typography.Text strong className='font-mono text-13px'>
                    {run.pipeline}
                  </Typography.Text>
                ) : (
                  <Typography.Text type='secondary' className='text-13px'>
                    {passthroughLabel(run.passthrough) || '-'}
                  </Typography.Text>
                )}
                {run.success === false ? (
                  <Tag color='error' className='mr-0'>
                    {t('page.preview.processing.statusFailed')}
                  </Tag>
                ) : null}
                {run.took_ms != null ? (
                  <Typography.Text type='secondary' className='ml-auto font-mono text-12px'>
                    {formatDuration(run.took_ms)}
                  </Typography.Text>
                ) : null}
              </div>
              {run.at ? (
                <Typography.Text type='secondary' className='mt-2px block text-12px'>
                  {new Date(run.at).toLocaleString()}
                </Typography.Text>
              ) : null}
              {run.error ? (
                <Typography.Text className='mt-6px block text-12px break-all text-[var(--ant-color-error)]'>
                  {run.error}
                </Typography.Text>
              ) : null}
              {!run.pipeline && run.passthrough ? (
                <Typography.Text type='secondary' className='mt-4px block text-12px'>
                  {t('page.preview.processing.passthroughHint')}
                </Typography.Text>
              ) : null}
              {run.pipeline ? stepsList(run.steps) : null}
            </div>
          ))
        )}

        {truncated ? (
          <Typography.Text type='secondary' className='text-12px'>
            {t('page.preview.processing.recentRuns', { count: String(visibleRuns.length) })}
          </Typography.Text>
        ) : null}
      </div>
    </Spin>
  );
}
