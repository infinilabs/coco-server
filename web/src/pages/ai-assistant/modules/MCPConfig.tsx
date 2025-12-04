import { Flex, Form, InputNumber, Select, Switch } from 'antd';
import ModelSelect from './ModelSelect';
import type { ReactNode } from 'react';

interface MCPConfigProps {
  readonly value?: any;
  readonly onChange?: (value: string) => void;
  readonly options: any[];
  readonly modelProviders: any[];
  readonly children?: ReactNode;
}

export const MCPConfig = (props: MCPConfigProps) => {
  const { t } = useTranslation();
  const { value = {}, onChange, children } = props;

  const onModelChange = (model: any) => {
    onChange?.({
      ...value,
      model
    });
  };

  useEffect(() => {
    if (value.visible) return;

    onChange?.({
      ...value,
      enabled_by_default: true
    });
  }, [value.visible]);

  return (
    <>
      <Form.Item
        className='relative [&_.ant-form-item-explain-error]:(absolute top-8)'
        label={t('page.assistant.labels.tool_invoked_model')}
        layout='vertical'
        name='mcp_servers'
        rules={[
          {
            required: value.enabled,
            validator: (_, value) => {
              if (!value.enabled) {
                return Promise.resolve();
              }

              if (!value?.model?.id) {
                return Promise.reject(new Error(t('page.assistant.hints.selectModel')));
              }

              return Promise.resolve();
            }
          }
        ]}
      >
        <div>
          <ModelSelect
            modelType='caller_model'
            namePrefix={['mcp_servers', 'model']}
            providers={props.modelProviders}
            value={value.model}
            width='100%'
            onChange={onModelChange}
          />
        </div>
      </Form.Item>

      {/* <div className='mb-2'>{t('page.assistant.labels.tool_invoked_model')}</div>

      <ModelSelect
        modelType='caller_model'
        namePrefix={['mcp_servers', 'model']}
        providers={props.modelProviders}
        value={value.model}
        width='100%'
        onChange={onModelChange}
      /> */}

      <Form.Item
        className='mt-4'
        label={t('page.assistant.labels.mcp_service')}
        layout='vertical'
        name={['mcp_servers', 'ids']}
      >
        <Select
          allowClear
          mode='multiple'
          options={props.options}
        />
      </Form.Item>

      {children}

      <Form.Item
        className='mt-4'
        initialValue={5}
        label={t('page.assistant.labels.max_iterations')}
        layout='vertical'
        name={['mcp_servers', 'max_iterations']}
      >
        <InputNumber
          className='w-full'
          max={100}
          min={1}
        />
      </Form.Item>

      <div className='mt-4 -mb-1'>{t('page.assistant.labels.feature_visibility')}</div>

      <Flex className='[&>div]:(flex-1 m-0!)'>
        <Form.Item
          label={t('page.assistant.labels.show_in_chat')}
          name={['mcp_servers', 'visible']}
        >
          <Switch size='small' />
        </Form.Item>

        <Form.Item
          label={t('page.assistant.labels.enabled_by_default')}
          name={['mcp_servers', 'enabled_by_default']}
        >
          <Switch
            disabled={!value.visible}
            size='small'
          />
        </Form.Item>
      </Flex>
    </>
  );
};
