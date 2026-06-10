import { Form, Input, InputNumber, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import ModelSelect from './ModelSelect';

// ── Research Settings (研究设置) ──────────────────────────────────────────────
export const DeepResearchLimits = () => {
  const { t } = useTranslation();
  return (
    <div>
      <Form.Item
        className='[&_.ant-form-item-control]:flex-[unset]!'
        label={t('page.assistant.labels.research_depth')}
        layout='vertical'
        name={['config', 'research_depth']}
      >
        <Select
          className='max-w-600px'
          options={[
            { label: t('page.assistant.labels.research_depth_basic'), value: 'basic' },
            { label: t('page.assistant.labels.research_depth_comprehensive'), value: 'comprehensive' },
            { label: t('page.assistant.labels.research_depth_exhaustive'), value: 'exhaustive' }
          ]}
        />
      </Form.Item>
      <Form.Item
        className='[&_.ant-form-item-control]:flex-[unset]!'
        extra={t('page.assistant.labels.max_steps_desc')}
        label={t('page.assistant.labels.max_steps')}
        layout='vertical'
        name={['config', 'max_steps']}
      >
        <InputNumber className='max-w-600px w-full' min={1} max={100} />
      </Form.Item>
      <Form.Item
        className='[&_.ant-form-item-control]:flex-[unset]!'
        extra={t('page.assistant.labels.max_researcher_iterations_desc')}
        label={t('page.assistant.labels.max_researcher_iterations')}
        layout='vertical'
        name={['config', 'max_researcher_iterations']}
      >
        <InputNumber className='max-w-600px w-full' min={1} max={50} />
      </Form.Item>
      <Form.Item
        className='mb-0! [&_.ant-form-item-control]:flex-[unset]!'
        extra={t('page.assistant.labels.timeout_desc')}
        label={t('page.assistant.labels.timeout')}
        layout='vertical'
        name={['config', 'timeout']}
        rules={[
          {
            pattern: /^\d+(\.\d+)?(ns|us|µs|ms|s|m|h)(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))*$/,
            message: t('page.assistant.labels.timeout_invalid')
          }
        ]}
      >
        <Input className='max-w-600px' placeholder='e.g. 30m, 1h' />
      </Form.Item>
    </div>
  );
};

// ── Output Settings (输出设置) ────────────────────────────────────────────────
export const DeepResearchOutput = () => {
  const { t } = useTranslation();
  return (
    <div>
      <Form.Item
        className='[&_.ant-form-item-control]:flex-[unset]!'
        label={t('page.assistant.labels.report_lang')}
        layout='vertical'
        name={['config', 'report_lang']}
      >
        <Select
          className='max-w-600px'
          options={[
            { label: t('page.assistant.labels.report_lang_zh'), value: 'zh-CN' },
            { label: t('page.assistant.labels.report_lang_en'), value: 'en-US' }
          ]}
        />
      </Form.Item>
      <Form.Item
        className='mb-0! [&_.ant-form-item-control]:flex-[unset]!'
        label={t('page.assistant.labels.report_format')}
        layout='vertical'
        name={['config', 'report_format']}
      >
        <Select
          className='max-w-600px'
          options={[
            { label: 'Markdown', value: 'markdown' },
            { label: 'HTML', value: 'html' }
          ]}
        />
      </Form.Item>
    </div>
  );
};

// ── Search Settings (搜索设置) ────────────────────────────────────────────────
interface DeepResearchSearchProps {
  readonly datasourceOptions: { label: string; value: string }[];
}

export const DeepResearchSearch = ({ datasourceOptions }: DeepResearchSearchProps) => {
  const { t } = useTranslation();
  const externalEngine = Form.useWatch(['config', 'external_search', 'engine']);
  return (
    <div>
      <p className='text-sm mb-4'>{t('page.assistant.labels.datasource_search')}</p>
      <Form.Item
        className='[&_.ant-form-item-control]:flex-[unset]!'
        label={t('page.assistant.labels.internal_datasource_ids')}
        layout='vertical'
        name={['config', 'internal_search', 'datasource_ids']}
      >
        <Select
          className='max-w-600px'
          allowClear
          mode='multiple'
          placeholder={t('page.assistant.labels.datasource_ids_placeholder')}
          options={datasourceOptions}
        />
      </Form.Item>

      <p className='text-sm mb-4 mt-2'>{t('page.assistant.labels.internet_search')}</p>
      <Form.Item
        className='[&_.ant-form-item-control]:flex-[unset]!'
        label={t('page.assistant.labels.external_search_engine')}
        layout='vertical'
        name={['config', 'external_search', 'engine']}
      >
        <Select
          className='max-w-600px'
          options={[
            { label: 'DuckDuckGo', value: 'duckduckgo' },
            { label: t('page.assistant.labels.search_engine_wikipedia'), value: 'wikipedia' },
            { label: 'Tavily', value: 'tavily' }
          ]}
        />
      </Form.Item>
      {externalEngine === 'tavily' && (
        <Form.Item
          className='[&_.ant-form-item-control]:flex-[unset]!'
          label={t('page.assistant.labels.external_search_api_key')}
          layout='vertical'
          name={['config', 'external_search', 'api_key']}
          rules={[{ required: true }]}
        >
          <Input.Password className='max-w-600px' placeholder={t('page.assistant.labels.tavily_api_key_placeholder')} />
        </Form.Item>
      )}
      <Form.Item
        className='mb-0! [&_.ant-form-item-control]:flex-[unset]!'
        label={t('page.assistant.labels.max_results')}
        layout='vertical'
        name={['config', 'max_results']}
      >
        <InputNumber className='max-w-600px w-full' min={1} max={500} />
      </Form.Item>
    </div>
  );
};

// ── Model Settings (模型设置) ─────────────────────────────────────────────────
interface DeepResearchModelsProps {
  readonly providers: any[];
  readonly defaultModel?: any;
  readonly onModelRefresh?: () => void;
}

export const DeepResearchModels = ({ providers, defaultModel, onModelRefresh }: DeepResearchModelsProps) => {
  const { t } = useTranslation();
  const modelSelectProps = {
    providers,
    allowClear: true,
    showTemplate: false,
    width: '600px',
    placeholder: t('page.assistant.labels.modelSelectPlaceholder'),
    defaultModel,
    onRefresh: onModelRefresh
  };
  const fields: [string, string][] = [
    ['planning_model', 'planning_model'],
    ['research_model', 'research_model'],
    ['synthesis_model', 'synthesis_model'],
    ['report_model', 'report_model']
  ];
  return (
    <div>
      {fields.map(([field, modelType], idx) => (
        <Form.Item
          key={field}
          className={idx === fields.length - 1 ? 'mb-0! [&_.ant-form-item-control]:flex-[unset]!' : '[&_.ant-form-item-control]:flex-[unset]!'}
          label={t(`page.assistant.labels.${field}`)}
          layout='vertical'
          name={['config', field]}
        >
          <ModelSelect
            modelType={modelType}
            namePrefix={['config', field]}
            {...modelSelectProps}
          />
        </Form.Item>
      ))}
    </div>
  );
};


