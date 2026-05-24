import { Button, Form, Select, Spin } from 'antd';
import '../index.scss';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading, useRequest } from '@sa/hooks';
import { searchModelPovider } from '@/service/api/model-provider';
import { formatESSearchResult } from '@/service/request/es';
import ModelSelect from './ModelSelect';
import { setDefaultModel } from '@/store/slice/server';

const SearchSettings = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    update: hasAuth('coco#system/update'),
  }

  const { endLoading, loading, startLoading } = useLoading();
  const [modelProviderList, setModelProviderList] = useState([]);

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
    const res = await searchModelPovider({ from: 0, size: 10000 })
    if (res?.data) {
      const newResult = formatESSearchResult(res?.data);
      setModelProviderList(newResult.data as any);
    }
    endLoading();
  }

  useEffect(() => {
    run();
    fetchModelProvider();
  }, []);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    startLoading();
    const default_model = Object.keys(params).reduce((acc: any, key: string) => {
      const v = params[key];
      if (v && typeof v === 'object' && (v.name !== undefined || v.provider_id !== undefined)) {
        acc[key] = {};
        if (v.name !== undefined) acc[key].id = v.name;
        if (v.provider_id !== undefined) acc[key].provider_id = v.provider_id;
      } else {
        acc[key] = v;
      }
      return acc;
    }, {});
    const result = await updateSettings({
      default_model
    });
    if (result?.data?.acknowledged) {
      dispatch(setDefaultModel(default_model))
      window.$message?.success(t('common.updateSuccess'));
    }
    endLoading();
  };

  useMount(() => {
    run();
  });

  useEffect(() => {
    if (data?.default_model) {
      const mapped = Object.keys(data.default_model).reduce((acc: any, key: string) => {
        const v = data.default_model[key];
        if (!v || typeof v !== 'object') {
          acc[key] = v;
          return acc;
        }

        // try find provider by provider_id
        let provider = modelProviderList.find((p: any) => p.id === v.provider_id || p.id === v.providerId || p.id === v.provider);
        let model = null;

        if (provider) {
          model = (provider.models || []).find((m: any) => m.name === v.name || m.id === v.id || `${provider.id}_${m.name}` === v.id);
        }

        // try parse id like 'provider_modelName' to find provider/model
        if (!provider && v.id && typeof v.id === 'string' && v.id.includes('_')) {
          const parts = v.id.split('_');
          const pId = parts[0];
          provider = modelProviderList.find((p: any) => p.id === pId);
          if (provider) {
            const modelName = parts.slice(1).join('_');
            model = (provider.models || []).find((m: any) => m.name === modelName || m.id === v.id || m.name === v.name);
          }
        }

        // global search fallback
        if (!provider || !model) {
          for (const p of modelProviderList) {
            const m = (p.models || []).find((mm: any) => mm.id === v.id || mm.name === v.name || mm.name === v.id);
            if (m) {
              provider = p;
              model = m;
              break;
            }
          }
        }

        if (provider && model) {
          acc[key] = {
            provider_id: provider.id,
            id: `${provider.id}_${model.name}`,
            name: model.name,
          };
        } else if (v.provider_id && v.id) {
          // best-effort: extract name from id if it contains '_'
          const name = typeof v.id === 'string' && v.id.includes('_') ? v.id.split('_').slice(1).join('_') : v.name || undefined;
          acc[key] = {
            provider_id: v.provider_id,
            id: v.id,
            ...(name ? { name } : {}),
          };
        } else {
          acc[key] = v;
        }

        return acc;
      }, {});

      form.setFieldsValue(mapped);
    }
  }, [JSON.stringify(data), modelProviderList]);

  return (
    <ListContainer>
      <Spin spinning={dataLoading || loading}>
        <Form
          className="settings-form py-24px"
          colon={false}
          form={form}
          labelAlign="left"
        >
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
          />
          <ModelSelectItem 
            label={t('page.guide.visionModel.title')}
            desc={t('page.guide.visionModel.desc')}
            name="vision_model"
            modelProviderList={modelProviderList}
            type="vision"
          />
          <ModelSelectItem 
            label={t('page.guide.embeddingModel.title')}
            desc={t('page.guide.embeddingModel.desc')}
            name="embedding_model"
            modelProviderList={modelProviderList}
            type="embedding"
          />
          <Form.Item
            label={<span className='color-[var(--ant-color-text-tertiary)]'>{t('page.settings.default_model.labels.ai_assistant')}</span>}
          >
            <div className='color-[var(--ant-color-text-tertiary)]'>{t('page.settings.default_model.labels.ai_assistant_desc')}</div>
          </Form.Item>
          <ModelSelectItem 
            label={t('page.settings.default_model.labels.intent_analysis_model')}
            name="intent_analysis_model"
            modelProviderList={modelProviderList}
          />
          <ModelSelectItem 
            label={t('page.settings.default_model.labels.picking_doc_model')}
            name="picking_doc_model"
            modelProviderList={modelProviderList}
          />
          <ModelSelectItem 
            label={t('page.settings.default_model.labels.picking_tool_model')}
            name="picking_tool_model"
            modelProviderList={modelProviderList}
          />
          <ModelSelectItem 
            label={t('page.settings.default_model.labels.answering_model')}
            name="answering_model"
            modelProviderList={modelProviderList}
          />
          {
            permissions.update && (
              <Form.Item label=" " >
                <Button
                  type="primary"
                  onClick={() => handleSubmit()}
                >
                  {t('common.update')}
                </Button>
              </Form.Item>
            )
          }
        </Form>
      </Spin>
    </ListContainer>
  );
});

export default SearchSettings;

const ModelSelectItem = ({ label, desc, name, modelProviderList = [], type }: { label: string, desc?: string, name: string, modelProviderList?: any[], type?: string }) => {
  
  const providers = useMemo(() => {
    if (!type) return modelProviderList;
    return modelProviderList.map(item => {
      return {
        ...item,
        models: item.models?.filter((model: any) => model.type === type)
      };
    });
  }, [modelProviderList, type]);

  return (
    <Form.Item label=" ">
      <div className='m-b-4px'>{label}</div>
      { desc && <div className='m-b-8px color-[var(--ant-color-text-tertiary)]'>{desc}</div> }
      <Form.Item noStyle name={name}>
        <ModelSelect
          providers={providers}
        />
      </Form.Item>
    </Form.Item>
  )
}