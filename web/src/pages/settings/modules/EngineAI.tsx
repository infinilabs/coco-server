import { CloudUploadOutlined, SyncOutlined } from '@ant-design/icons';
import { Badge, Button, Collapse, Form, Input, InputNumber, Spin, Switch, Tag } from 'antd';
import { memo, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import '../index.scss';
import { fetchEngineAIStatus, fetchSettings, syncEngineAI, updateSettings } from '@/service/api/server';
import { searchModelPovider } from '@/service/api/model-provider';
import { formatESSearchResult } from '@/service/request/es';
import { useLoading, useRequest } from '@sa/hooks';
import { useAuth } from '@/hooks/business/auth';
import ModelSelect from '@/pages/ai-assistant/modules/ModelSelect';
import { ModelSelectItem } from './DefaultModel';

interface EngineAIStatus {
  enabled: boolean;
  warnings?: string[];
  engine_errors?: string[];
  in_sync?: boolean;
  model?: {
    provider_id: string;
    id: string;
    vendor: string;
    url: string;
    text_field: string;
    vector_field: string;
  };
  ingest_pipeline?: { name: string; desired: any; actual: any; in_sync: boolean };
  search_pipeline?: { name: string; desired: any; actual: any; in_sync: boolean };
  document_index?: { name: string; default_pipeline: string; expected: string; in_sync: boolean };
}

const prettyJSON = (v: any) => (v == null ? '' : JSON.stringify(v, null, 2));

const PipelineStatus = ({ title, pipeline, t }: { title: string; pipeline: any; t: any }) => {
  if (!pipeline) return null;
  return (
    <div className="m-b-8px">
      <div className="m-b-4px flex items-center gap-8px">
        <span>{title}</span>
        <Tag color={pipeline.in_sync ? 'success' : 'warning'}>{pipeline.name}</Tag>
        <span className="color-[var(--ant-color-text-tertiary)]">
          {pipeline.in_sync ? t('page.settings.engine_ai.status.in_sync') : t('page.settings.engine_ai.status.out_of_sync')}
        </span>
      </div>
      <Collapse
        size="small"
        items={[
          {
            key: 'desired',
            label: t('page.settings.engine_ai.status.desired'),
            children: <pre className="m-0 max-h-240px overflow-auto text-12px">{prettyJSON(pipeline.desired)}</pre>
          },
          ...(pipeline.actual
            ? [
                {
                  key: 'actual',
                  label: t('page.settings.engine_ai.status.actual'),
                  children: <pre className="m-0 max-h-240px overflow-auto text-12px">{prettyJSON(pipeline.actual)}</pre>
                }
              ]
            : [])
        ]}
      />
    </div>
  );
};

const EngineAI = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    update: hasAuth('coco#system/update'),
    fetchModelProviders: hasAuth('coco#model_provider/search'),
    viewStatus: hasAuth('coco#search/studio')
  };

  const { endLoading, loading, startLoading } = useLoading();
  const [modelProviderList, setModelProviderList] = useState<any[]>([]);

  const { data: settingsData, loading: settingsLoading, run: runSettings } = useRequest(fetchSettings, { manual: true });
  const { data: statusData, loading: statusLoading, run: runStatus } = useRequest(fetchEngineAIStatus, { manual: true });

  const status: EngineAIStatus | undefined = useMemo(() => statusData as any, [statusData]);

  const fetchModelProvider = async () => {
    startLoading();
    const res = await searchModelPovider({ from: 0, size: 10000, filter: { enabled: [true] } });
    if (res?.data) {
      const newResult = formatESSearchResult(res?.data);
      setModelProviderList(newResult.data as any);
    }
    endLoading();
  };

  useEffect(() => {
    runSettings();
    runStatus();
    if (permissions.fetchModelProviders) {
      fetchModelProvider();
    }
  }, []);

  useEffect(() => {
    const cfg = (settingsData as any)?.engine_ai;
    if (!cfg) return;
    let embeddingModel: any = undefined;
    if (cfg.embedding_model?.provider_id && cfg.embedding_model?.id) {
      const provider = modelProviderList.find((p: any) => p.id === cfg.embedding_model.provider_id);
      const model = provider ? (provider.models || []).find((m: any) => m.name === cfg.embedding_model.id) : null;
      if (provider && model) {
        embeddingModel = { provider_id: provider.id, id: `${provider.id}_${model.name}`, name: model.name };
      } else {
        embeddingModel = { provider_id: cfg.embedding_model.provider_id, id: cfg.embedding_model.id, name: cfg.embedding_model.id };
      }
    }
    form.setFieldsValue({
      enabled: !!cfg.enabled,
      embedding_model: embeddingModel,
      batch_size: cfg.batch_size,
      rank_constant: cfg.rank_constant,
      text_field: cfg.text_field,
      vector_field: cfg.vector_field
    });
  }, [JSON.stringify(settingsData), modelProviderList]);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    startLoading();
    const engineAI: any = {
      enabled: !!params.enabled,
      batch_size: params.batch_size,
      rank_constant: params.rank_constant,
      text_field: params.text_field,
      vector_field: params.vector_field
    };
    if (params.embedding_model?.provider_id && params.embedding_model?.name) {
      engineAI.embedding_model = { provider_id: params.embedding_model.provider_id, id: params.embedding_model.name };
    }
    const result = await updateSettings({ engine_ai: engineAI });
    if (result?.data?.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
      // saving already triggers a best-effort background sync on the server;
      // pull the status so the drift panel reflects it
      runStatus();
    }
    endLoading();
  };

  const handleSync = async () => {
    startLoading();
    const res = (await syncEngineAI()) as any;
    if (res?.data && !res?.error) {
      window.$message?.success(t('page.settings.engine_ai.status.synced'));
      runStatus();
    }
    endLoading();
  };

  const onModelRefresh = useMemo(() => {
    if (!permissions.fetchModelProviders) return undefined;
    return () => fetchModelProvider();
  }, [permissions.fetchModelProviders]);

  return (
    <div className="settings-form py-24px">
      <Spin spinning={settingsLoading || statusLoading || loading}>
        <Form colon={false} form={form} labelAlign="left">
          <Form.Item
            label={(
              <span className="color-[var(--ant-color-text-tertiary)]">
                {t('page.settings.engine_ai.labels.engine_ai')}
              </span>
            )}
          >
            <div className="color-[var(--ant-color-text-tertiary)]">{t('page.settings.engine_ai.labels.engine_ai_desc')}</div>
          </Form.Item>
          <Form.Item label={t('page.settings.engine_ai.labels.enabled')} name="enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
          <ModelSelectItem
            label={t('page.settings.engine_ai.labels.embedding_model')}
            desc={t('page.settings.engine_ai.labels.embedding_model_desc')}
            name="embedding_model"
            modelProviderList={modelProviderList}
            type="embedding"
            onRefresh={onModelRefresh}
          />
          <Form.Item
            label={t('page.settings.engine_ai.labels.batch_size')}
            name="batch_size"
            tooltip={t('page.settings.engine_ai.labels.batch_size_desc')}
          >
            <InputNumber min={1} max={100} className="w-160px" placeholder="10" />
          </Form.Item>
          <Form.Item
            label={t('page.settings.engine_ai.labels.rank_constant')}
            name="rank_constant"
            tooltip={t('page.settings.engine_ai.labels.rank_constant_desc')}
          >
            <InputNumber min={1} max={1000} className="w-160px" placeholder="60" />
          </Form.Item>
          <Form.Item label={t('page.settings.engine_ai.labels.text_field')} name="text_field">
            <Input className="w-360px" placeholder="ai_insights.text" />
          </Form.Item>
          <Form.Item label={t('page.settings.engine_ai.labels.vector_field')} name="vector_field">
            <Input className="w-360px" placeholder="ai_insights.embedding.embedding1024" />
          </Form.Item>
          {permissions.update && (
            <Form.Item label=" ">
              <Button type="primary" onClick={() => handleSubmit()}>
                {t('common.update')}
              </Button>
            </Form.Item>
          )}
        </Form>
      </Spin>

      {permissions.viewStatus && status && (
        <div className="border-t border-t-solid border-[var(--ant-color-border-secondary)] pt-16px">
          <div className="m-b-12px flex items-center gap-8px">
            <Badge status={status.in_sync ? 'success' : status.enabled ? 'warning' : 'default'} />
            <span className="font-500">{t('page.settings.engine_ai.status.title')}</span>
            {status.model && (
              <Tag>
                {status.model.provider_id}/{status.model.id}
              </Tag>
            )}
            <Button icon={<CloudUploadOutlined />} loading={loading} onClick={() => handleSync()} size="small" type="link">
              {t('page.settings.engine_ai.status.sync_now')}
            </Button>
          </div>
          {(status.warnings || []).map((w: string) => (
            <div key={w} className="color-[var(--ant-color-text-warning)]">
              {w}
            </div>
          ))}
          {(status.engine_errors || []).map((e: string) => (
            <div key={e} className="color-[var(--ant-color-text-danger)]">
              {e}
            </div>
          ))}
          {status.model && (
            <div className="m-b-8px color-[var(--ant-color-text-tertiary)] text-12px">
              {t('page.settings.engine_ai.status.callout')}: {status.model.url} · {status.model.text_field} → {status.model.vector_field}
            </div>
          )}
          <PipelineStatus pipeline={status.ingest_pipeline} t={t} title={t('page.settings.engine_ai.status.ingest_pipeline')} />
          <PipelineStatus pipeline={status.search_pipeline} t={t} title={t('page.settings.engine_ai.status.search_pipeline')} />
          {status.document_index && (
            <div className="m-b-8px flex items-center gap-8px">
              <span>{t('page.settings.engine_ai.status.document_index')}</span>
              <Tag>{status.document_index.name}</Tag>
              <span className="color-[var(--ant-color-text-tertiary)]">
                default_pipeline: {status.document_index.default_pipeline || '-'} (expected: {status.document_index.expected})
              </span>
              {!status.document_index.in_sync && (
                <Tag color="warning">{t('page.settings.engine_ai.status.out_of_sync')}</Tag>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
});

export default EngineAI;
