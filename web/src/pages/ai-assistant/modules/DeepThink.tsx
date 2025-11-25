import { Form, Select, Space, Switch } from 'antd';
import ModelSelect from './ModelSelect';

interface DeepThinkProps {
  readonly providers: any[];
  readonly className?: string;
}

export const DeepThink = (props: DeepThinkProps) => {
  const { providers = [], className } = props;
  const { t } = useTranslation();
  return (
    <div className={className}>
      <Form.Item
        className='[&_.ant-form-item-control]:flex-[unset]!'
        label={t('page.assistant.labels.intent_recognition_model')}
        layout='vertical'
        name={['config', 'intent_analysis_model']}
      >
        <ModelSelect
          modelType='intent_analysis_model'
          providers={providers}
        />
      </Form.Item>

      <div>{t('page.assistant.labels.feature_visibility_deep_thought')}</div>

      <Form.Item
        className='mb-0!'
        label={t('page.assistant.labels.show_in_chat')}
        name={['config', 'visible']}
      >
        <Switch size='small' />
      </Form.Item>

      {/* 
      <div>
        <Space>
          <span>{t('page.assistant.labels.pick_datasource')}</span>
          <Form.Item
            className='my-[0px]'
            name={['config', 'pick_datasource']}
          >
            <Switch size='small' />
          </Form.Item>
        </Space>
      </div>
      <div>
        <Space>
          <span>{t('page.assistant.labels.pick_tools')}</span>
          <Form.Item
            className='my-[0px]'
            name={['config', 'pick_tools']}
          >
            <Switch size='small' />
          </Form.Item>
        </Space>
      </div> */}
    </div>
  );
};
