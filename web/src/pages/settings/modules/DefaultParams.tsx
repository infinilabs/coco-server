import { Button, Form, Select, Spin } from 'antd';
import '../index.scss';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading, useRequest } from '@sa/hooks';
import { searchModelPovider } from '@/service/api/model-provider';
import { formatESSearchResult } from '@/service/request/es';
import { setDefaultModel } from '@/store/slice/server';
import ModelSelect from '@/pages/ai-assistant/modules/ModelSelect';
import { createPipeline, getEnablePipelines } from '@/service/api/pipeline';
import { ChevronDown, ChevronUp } from 'lucide-react';
import SaveBar from './SaveBar';

// 默认参数：合并原「默认模型」与「文档处理」两个 tab。
// 两者都是全局兜底参数（数据源/Pipeline 未单独指定时生效），共用一张表单、
// 一个保存条；保存时按字段拆分，分别写 settings 的 default_model 与
// document_processing 两节，后端接口与存储结构不变。
const DOC_PROCESSING_KEYS = ['default_pipeline_for_attachment', 'default_pipeline_for_document', 'llm_generation_language'];

const SectionTitle = ({ title }: { title: string }) => (
  <div
    className='mb-16px mt-4px pb-8px text-15px font-500'
    style={{ borderBottom: '1px solid var(--ant-color-split)' }}
  >
    {title}
  </div>
);

const DefaultParams = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const navigate = useNavigate();

  const { hasAuth } = useAuth()

  const permissions = {
    update: hasAuth('coco#system/update'),
    fetchModelProviders: hasAuth('coco#model_provider/search'),
  }

  const { endLoading, loading, startLoading } = useLoading();
  const [modelProviderList, setModelProviderList] = useState<any[]>([]);
  const [pipelineList, setPipelineList] = useState([]);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [dirty, setDirty] = useState(false);

  const dispatch = useAppDispatch();

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  const fetchModelProvider = async () => {
    startLoading();
    const res = await searchModelPovider({ from: 0, size: 10000, filter: { enabled: [true] } });
    if (res?.data) {
      const newResult = formatESSearchResult(res?.data);
      setModelProviderList(newResult.data as any);
    }
    endLoading();
  }

  const fetchPipelines = async () => {
    startLoading();
    const res = await getEnablePipelines()
    if (res?.data) {
      const newResult = formatESSearchResult(res?.data);
      setPipelineList(newResult.data as any);
    }
    endLoading();
  }

  useEffect(() => {
    run();
    fetchPipelines();
    if (permissions.fetchModelProviders) {
      fetchModelProvider();
    }
  }, []);

  const onModelRefresh = useMemo(() => {
    if (!permissions.fetchModelProviders) return;
    return () => fetchModelProvider();
  }, [permissions.fetchModelProviders]);

  const handleSubmit = async () => {
    const values = await form.validateFields();

    // 按字段拆分回各自的 settings 节
    const document_processing: Record<string, any> = {};
    const default_model: Record<string, any> = {};
    Object.keys(values).forEach((key) => {
      if (DOC_PROCESSING_KEYS.includes(key)) {
        document_processing[key] = values[key];
      } else {
        default_model[key] = values[key];
      }
    });

    // 模型字段序列化：{ provider_id, id, name } → 后端存储形状
    const serializedModels = Object.keys(default_model).reduce((acc: any, key: string) => {
      const v = default_model[key];
      if (v && typeof v === 'object' && (v.name !== undefined || v.provider_id !== undefined)) {
        acc[key] = {};
        if (v.name !== undefined) acc[key].id = v.name;
        if (v.provider_id !== undefined) acc[key].provider_id = v.provider_id;
      } else {
        acc[key] = v;
      }
      return acc;
    }, {});

    startLoading();
    const r1 = await updateSettings({ default_model: serializedModels });
    const r2 = await updateSettings({ document_processing });
    if (r1?.data?.acknowledged && r2?.data?.acknowledged) {
      dispatch(setDefaultModel(serializedModels))
      window.$message?.success(t('common.updateSuccess'));
      setDirty(false);
    }
    endLoading();
  };

  useMount(() => {
    run();
  });

  useEffect(() => {
    if (!data) return;

    if (data.default_model) {
      const mapped = Object.keys(data.default_model).reduce((acc: any, key: string) => {
        const v = data.default_model[key];
        if (!v || typeof v !== 'object') {
          acc[key] = v;
          return acc;
        }

        if (!v.id && !v.name && !v.provider_id && !v.providerId && !v.provider) {
          acc[key] = v;
          return acc;
        }

        // Match provider by provider_id, then match model within that provider.
        // The backend stores the model name in `id`; model names are unique only
        // within a single provider's model list.
        const provider = modelProviderList.find((p: any) => p.id === v.provider_id || p.id === v.providerId || p.id === v.provider);
        const modelName = v.id;
        const model = provider
          ? (provider.models || []).find((m: any) => m.name === modelName || m.id === modelName)
          : null;

        if (provider && model) {
          acc[key] = {
            provider_id: provider.id,
            id: `${provider.id}_${model.name}`,
            name: model.name,
          };
        } else if (v.provider_id && v.id) {
          // Preserve the original value when the provider or model cannot be
          // resolved; do not fall back to a different provider.
          acc[key] = {
            provider_id: v.provider_id,
            id: v.id,
            ...(v.name ? { name: v.name } : {}),
          };
        } else {
          acc[key] = v;
        }

        return acc;
      }, {});
      form.setFieldsValue(mapped);
    }

    if (data.document_processing) {
      form.setFieldsValue({ ...data.document_processing });
    }
    setDirty(false);
  }, [JSON.stringify(data), modelProviderList]);

  // 放弃更改：重新拉取服务端配置回填
  const handleDiscard = () => {
    setDirty(false);
    run();
  };

  return (
    <ListContainer>
      <Spin spinning={dataLoading || loading}>
        <Form
          className="settings-form py-24px"
          colon={false}
          form={form}
          labelAlign="left"
          onValuesChange={() => setDirty(true)}
        >
          <SectionTitle title={t('page.settings.default_model.title')} />
          <Form.Item
            label={(
              <span className='color-[var(--ant-color-text-tertiary)]'>
                {t('page.settings.default_model.labels.default_model')}
              </span>
            )}
          >
            <div className='color-[var(--ant-color-text-tertiary)]'>{t('page.settings.default_model.labels.default_model_desc')}</div>
          </Form.Item>
          <ModelSelectItem
            label={t('page.guide.languageModel.title')}
            desc={t('page.guide.languageModel.desc')}
            name="language_model"
            modelProviderList={modelProviderList}
            type="language"
            onRefresh={onModelRefresh}
          />
          <ModelSelectItem
            label={t('page.guide.visionModel.title')}
            desc={t('page.guide.visionModel.desc')}
            name="vision_model"
            modelProviderList={modelProviderList}
            type="vision"
            onRefresh={onModelRefresh}
          />
          <ModelSelectItem
            label={t('page.guide.embeddingModel.title')}
            desc={t('page.guide.embeddingModel.desc')}
            name="embedding_model"
            modelProviderList={modelProviderList}
            type="embedding"
            onRefresh={onModelRefresh}
          />
          <ModelSelectItem
            label={t('page.guide.rerankModel.title')}
            desc={t('page.guide.rerankModel.desc')}
            name="rerank_model"
            modelProviderList={modelProviderList}
            type="rerank"
            onRefresh={onModelRefresh}
          />
          <Form.Item label=' '>
            <Button
              className='p-0'
              type='link'
              onClick={() => setShowAdvanced(!showAdvanced)}
            >
              {t('common.advanced')} {showAdvanced ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
            </Button>
          </Form.Item>
          {
            showAdvanced ? (
              <>
                <Form.Item
                  label={<span className='color-[var(--ant-color-text-tertiary)]'>{t('page.settings.default_model.labels.ai_assistant')}</span>}
                >
                  <div className='color-[var(--ant-color-text-tertiary)]'>{t('page.settings.default_model.labels.ai_assistant_desc')}</div>
                </Form.Item>
                <ModelSelectItem
                  label={t('page.settings.default_model.labels.intent_analysis_model')}
                  name="intent_analysis_model"
                  modelProviderList={modelProviderList}
                  type="language"
                  onRefresh={onModelRefresh}
                />
                <ModelSelectItem
                  label={t('page.settings.default_model.labels.picking_doc_model')}
                  name="picking_doc_model"
                  modelProviderList={modelProviderList}
                  type="language"
                  onRefresh={onModelRefresh}
                />
                <ModelSelectItem
                  label={t('page.settings.default_model.labels.picking_tool_model')}
                  name="picking_tool_model"
                  modelProviderList={modelProviderList}
                  type="language"
                  onRefresh={onModelRefresh}
                />
                <ModelSelectItem
                  label={t('page.settings.default_model.labels.answering_model')}
                  name="answering_model"
                  modelProviderList={modelProviderList}
                  type="language"
                  onRefresh={onModelRefresh}
                />
              </>
            ) : null
          }

          <SectionTitle title={t('page.settings.document_processing.title')} />
          <Form.Item
            label={(
              <span className='color-[var(--ant-color-text-tertiary)]'>
                {t('page.settings.document_processing.labels.default_pipeline_for_attachment')}
              </span>
            )}
          >
            <PipelineSelectItem
              label={t('page.settings.document_processing.labels.processing_pipeline')}
              name="default_pipeline_for_attachment"
              pipelineList={pipelineList}
            />
            <Button
              type="link"
              className='!px-0'
              onClick={() => navigate('/pipeline/list')}
            >
              {t('page.pipeline.manageLink')}
            </Button>
          </Form.Item>
          <Form.Item
            label={(
              <span className='color-[var(--ant-color-text-tertiary)]'>
                {t('page.settings.document_processing.labels.default_pipeline_for_document')}
              </span>
            )}
          >
            <PipelineSelectItem
              label={t('page.settings.document_processing.labels.processing_pipeline')}
              name="default_pipeline_for_document"
              pipelineList={pipelineList}
            />
          </Form.Item>
          <Form.Item
            label={(
              <span className='color-[var(--ant-color-text-tertiary)]'>
                {t('page.settings.document_processing.labels.output_language')}
              </span>
            )}
          >
            <div className='m-b-8px color-[var(--ant-color-text-tertiary)]'>
              {t('page.settings.document_processing.labels.output_language_desc')}
            </div>
            <Form.Item noStyle name={'llm_generation_language'}>
              <Select
                options={[
                  {
                    value: 'en-US',
                    label: 'English'
                  },
                  {
                    value: 'zh-CN',
                    label: '中文'
                  }
                ]}
              />
            </Form.Item>
          </Form.Item>
        </Form>
      </Spin>
      {/* sticky save bar, same pattern as the ontology page */}
      {permissions.update && (
        <SaveBar dirty={dirty} loading={loading} onDiscard={handleDiscard} onSave={() => handleSubmit()} />
      )}
    </ListContainer>
  );
});

export default DefaultParams;

export const ModelSelectItem = ({ label, desc, name, modelProviderList = [], type, onRefresh, width }: { label: string, desc?: string, name: string, modelProviderList?: any[], type?: string, onRefresh?: () => void, width?: string }) => {

  const providers = useMemo(() => {
    if (!type) return modelProviderList;
    return modelProviderList.map(item => {
      return {
        ...item,
        // vision slot: legacy type=vision plus language models flagged
        // support_vision; language slots likewise include vision-capable
        // models so they can be picked for text workloads
        models: item.models?.filter((model: any) =>
          type === 'vision'
            ? model.type === 'vision' || model.support_vision
            : model.type === type || (type === 'language' && (model.type === 'vision' || model.support_vision))
        )
      };
    });
  }, [modelProviderList, type]);

  return (
    <Form.Item label=" ">
      <div className='m-b-4px'>{label}</div>
      {desc && <div className='m-b-8px color-[var(--ant-color-text-tertiary)]'>{desc}</div>}
      <Form.Item noStyle name={name}>
        <ModelSelect
          providers={providers}
          showSettings={false}
          showTemplate={false}
          onRefresh={onRefresh}
          allowClear
          width={width}
        />
      </Form.Item>
    </Form.Item>
  )
}

export const PipelineSelectItem = ({ label, desc, name, pipelineList = [] }: { label: string, desc?: string, name: string, pipelineList?: any[] }) => {

  return (
    <>
      <div className='m-b-4px'>{label}</div>
      {desc && <div className='m-b-8px color-[var(--ant-color-text-tertiary)]'>{desc}</div>}
      <Form.Item noStyle name={name}>
        <Select
          options={pipelineList.map(item => ({
            label: item.name,
            value: item.id
          }))}
        />
      </Form.Item>
    </>
  )
}
