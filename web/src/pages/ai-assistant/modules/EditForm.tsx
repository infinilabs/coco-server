import { Button, Collapse, Form, Input, InputNumber, Select, Spin, Switch, message } from 'antd';
import type { FormProps } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useLoading, useRequest } from '@sa/hooks';

import { getConnectorIcons } from '@/service/api/connector';
import { IconSelector } from '../../connector/new/icon_selector';
import { fetchDataSourceList, getEnabledModelProviders } from '@/service/api';
import { searchMCPServer } from '@/service/api/mcp-server';
import { AssistantMode } from './AssistantMode';
import { DatasourceConfig } from './DatasourceConfig';
import { MCPConfig } from './MCPConfig';
import { DeepThink } from './DeepThink';
import { formatESSearchResult } from '@/service/request/es';
import ModelSelect, { DefaultPromptTemplates } from './ModelSelect';
import { ToolsConfig } from './ToolsConfig';
import { getUUID } from '@/utils/common';
import { Tags } from '@/components/common/tags';
import { getAssistantCategory } from '@/service/api/assistant';
import { UploadConfig } from './UploadConfig';
import classNames from 'classnames';
import AvailableVariable from './AvailableVariable';

interface AssistantFormProps {
  initialValues: any;
  onSubmit: (values: any, startLoading: () => void, endLoading: () => void) => void;
  mode: string;
  loading: boolean;
}

export const EditForm = memo((props: AssistantFormProps) => {
  const { initialValues = {}, onSubmit, mode } = props;
  const [form] = Form.useForm();

  const { hasAuth } = useAuth();

  const permissions = {
    fetchModelProviders: hasAuth('coco#model_provider/search'),
    fetchMCPServers: hasAuth('coco#mcp_server/search'),
    fetchDataSources: hasAuth('coco#datasource/search')
  };

  useEffect(() => {
    if (initialValues) {
      if (initialValues.datasource?.filter) {
        initialValues.datasource.filter = JSON.stringify(initialValues.datasource.filter);
      }
      form.setFieldsValue({
        ...initialValues,
        icon: initialValues.icon || 'font_coco'
      });
    }
  }, [initialValues]);
  const { t } = useTranslation();
  const { endLoading, loading, startLoading } = useLoading();

  const onFinish: FormProps<any>['onFinish'] = values => {
    if (values.datasource?.filter) {
      try {
        values.datasource.filter = JSON.parse(values.datasource.filter);
      } catch (e) {
        message.error('Datasource filter is not valid JSON');
        return;
      }
    } else {
      if (!values.datasource) values.datasource = {};
      values.datasource.filter = null;
    }
    if (values.upload?.allowed_file_extensions) {
      values.upload.allowed_file_extensions = values.upload.allowed_file_extensions.filter(item => Boolean(item));
    }

    onSubmit?.(
      {
        ...values,
        category: values?.category?.[0] || ''
      },
      startLoading,
      endLoading
    );
  };

  const onFinishFailed: FormProps<any>['onFinishFailed'] = errorInfo => {
    console.log('Failed:', errorInfo);
  };
  const [iconsMeta, setIconsMeta] = useState([]);
  useEffect(() => {
    getConnectorIcons().then(res => {
      if (res.data?.length > 0) {
        setIconsMeta(res.data);
      }
    });
  }, []);
  const { defaultRequiredRule, formRules } = useFormRules();

  const [showAdvanced, setShowAdvanced] = useState(false);
  const {
    data: result,
    run,
    loading: dataSourceLoading
  } = useRequest(fetchDataSourceList, {
    manual: true
  });

  useEffect(() => {
    if (permissions.fetchDataSources) {
      run({
        from: 0,
        size: 10000
      });
    }
  }, [permissions.fetchDataSources]);

  const dataSource = useMemo(() => {
    return result?.hits?.hits?.map(item => ({ ...item._source })) || [];
  }, [JSON.stringify(result)]);

  const { data: modelsResult, run: fetchModelProviders } = useRequest(getEnabledModelProviders, {
    manual: true
  });
  const modelProviders = useMemo(() => {
    if (!modelsResult) return [];
    const res = formatESSearchResult(modelsResult);
    return res.data;
  }, [JSON.stringify(modelsResult)]);
  useEffect(() => {
    if (permissions.fetchModelProviders) {
      fetchModelProviders(10000);
    }
  }, [permissions.fetchModelProviders]);

  const { data: mcpServerResult, run: fetchMCPServers } = useRequest(searchMCPServer, {
    manual: true
  });

  useEffect(() => {
    if (permissions.fetchMCPServers) {
      fetchMCPServers({
        from: 0,
        size: 10000
      });
    }
  }, [permissions.fetchMCPServers]);

  const mcpServers = useMemo(() => {
    return mcpServerResult?.hits?.hits?.map(item => ({ ...item._source })) || [];
  }, [JSON.stringify(mcpServerResult)]);

  const [assistantMode, setAssistantMode] = useState(initialValues?.mode || 'simple');

  useEffect(() => {
    if (initialValues?.type) {
      setAssistantMode(initialValues.type);
    }
  }, [initialValues?.type]);

  const [suggestedChatChecked, setSuggestedChatChecked] = useState(
    initialValues?.chat_settings?.suggested?.enabled || false
  );
  useEffect(() => {
    setSuggestedChatChecked(initialValues?.chat_settings?.suggested?.enabled || false);
  }, [initialValues?.chat_settings?.suggested?.enabled]);

  const [categories, setCategories] = useState([]);
  useEffect(() => {
    getAssistantCategory().then(({ data }) => {
      if (!data?.error) {
        const newData = formatESSearchResult(data);
        const cates = newData?.aggregations?.categories?.buckets
          ? newData?.aggregations?.categories?.buckets.map((item: any) => {
              return item.key;
            })
          : [];
        setCategories(cates);
      }
    });
  }, []);

  const commonFormItemsClassName = `${showAdvanced || assistantMode === 'deep_think' ? '' : 'h-0px m-0px overflow-hidden'}`;

  const configPickTools = Form.useWatch(['config', 'pick_tools'], form);
  const configPickDatasource = Form.useWatch(['config', 'pick_datasource'], form);

  const renderIntentRecognitionCollapse = () => {
    if (assistantMode !== 'deep_think') return;

    return (
      <Collapse
        className='mb-4 w-150'
        defaultActiveKey='intent-recognition'
        items={[
          {
            key: 'intent-recognition',
            label: t('page.assistant.labels.intent_recognition'),
            forceRender: true,
            children: (
              <Form.Item className='mb-0!'>
                <DeepThink providers={modelProviders} />
              </Form.Item>
            )
          }
        ]}
      />
    );
  };

  const renderInternetSearchCollapse = () => {
    return (
      <Collapse
        className='mb-4 w-150'
        items={[
          {
            key: 'internet-search',
            label: t('page.assistant.labels.internet_search'),
            forceRender: true,
            children: (
              <>
                {assistantMode === 'deep_think' && (
                  <>
                    <Form.Item
                      className='mb-4! [&_.ant-form-item-control]:flex-[unset]!'
                      initialValue={false}
                      label={t('page.assistant.labels.executionStrategy')}
                      layout='vertical'
                      name={['config', 'pick_datasource']}
                      extra={
                        configPickDatasource
                          ? t('page.assistant.hints.alwaysExecute')
                          : t('page.assistant.hints.intelligentDecisionMaking')
                      }
                    >
                      <Select
                        options={[
                          {
                            label: t('page.assistant.options.alwaysExecute'),
                            value: true
                          },
                          {
                            label: t('page.assistant.options.intelligentDecisionMaking'),
                            value: false
                          }
                        ]}
                      />
                    </Form.Item>

                    <Form.Item
                      className='relative [&_.ant-form-item-explain-error]:(absolute top-8) [&_.ant-form-item-control]:flex-[unset]!'
                      label={t('page.settings.llm.picking_doc_model')}
                      layout='vertical'
                      name={['config', 'picking_doc_model']}
                      rules={[
                        {
                          required: true,
                          validator: (_, value) => {
                            if (!value || !value.id) {
                              return Promise.reject(new Error(t('page.assistant.hints.selectModel')));
                            }

                            return Promise.resolve();
                          }
                        }
                      ]}
                    >
                      <ModelSelect
                        modelType='picking_doc_model'
                        namePrefix={['config', 'picking_doc_model']}
                        providers={modelProviders}
                      />
                    </Form.Item>
                  </>
                )}

                <Form.Item
                  className='mb-0'
                  name='datasource'
                  rules={[{ required: true }]}
                >
                  <DatasourceConfig
                    loading={dataSourceLoading}
                    options={[{ label: '*', value: '*' }].concat(
                      dataSource.map(item => ({
                        label: item.name,
                        value: item.id
                      }))
                    )}
                  />
                </Form.Item>
              </>
            )
          }
        ]}
      />
    );
  };

  const renderLargeModelToolsCollapse = () => {
    return (
      <Collapse
        className='mb-4 w-150'
        items={[
          {
            key: 'large-model-tools',
            label: t('page.assistant.labels.large_model_tool'),
            forceRender: true,
            extra: (
              <div
                onClick={event => {
                  event.stopPropagation();
                }}
              >
                <Form.Item
                  className='mb-0! [&_*]:(min-h-[unset]!)'
                  name={['mcp_servers', 'enabled']}
                >
                  <Switch size='small' />
                </Form.Item>
              </div>
            ),
            children: (
              <>
                {assistantMode === 'deep_think' && (
                  <Form.Item
                    className='mb-4! [&_.ant-form-item-control]:flex-[unset]!'
                    initialValue={false}
                    label={t('page.assistant.labels.executionStrategy')}
                    layout='vertical'
                    name={['config', 'pick_tools']}
                    extra={
                      configPickTools
                        ? t('page.assistant.hints.alwaysExecute')
                        : t('page.assistant.hints.intelligentDecisionMaking')
                    }
                  >
                    <Select
                      options={[
                        {
                          label: t('page.assistant.options.alwaysExecute'),
                          value: true
                        },
                        {
                          label: t('page.assistant.options.intelligentDecisionMaking'),
                          value: false
                        }
                      ]}
                    />
                  </Form.Item>
                )}

                <Form.Item
                  className='mb-0 [&_.ant-form-item-control]:flex-[unset]!'
                  name='mcp_servers'
                >
                  <MCPConfig
                    modelProviders={modelProviders}
                    options={[{ label: '*', value: '*' }].concat(
                      mcpServers.map(item => ({
                        label: item.name,
                        value: item.id
                      }))
                    )}
                  >
                    <ToolsConfig />
                  </MCPConfig>
                </Form.Item>
              </>
            )
          }
        ]}
      />
    );
  };

  const renderGenerateAnswersCollapse = () => {
    if (assistantMode !== 'deep_think') return;

    return (
      <Collapse
        className='w-150'
        items={[
          {
            key: 'generate-answers',
            label: t('page.assistant.labels.generate_response'),
            forceRender: true,
            children: (
              <Form.Item
                className='relative [&_.ant-form-item-explain-error]:(absolute top-8) mb-0! [&_.ant-form-item-control]:flex-[unset]!'
                label={t('page.assistant.labels.answering_model')}
                layout='vertical'
                name={['answering_model']}
                rules={[
                  {
                    required: true,
                    validator: (_, value) => {
                      if (!value || !value.id) {
                        return Promise.reject(new Error(t('page.assistant.hints.selectModel')));
                      }
                      return Promise.resolve();
                    }
                  }
                ]}
              >
                <ModelSelect
                  modelType='answering_model'
                  namePrefix={['answering_model']}
                  providers={modelProviders}
                />
              </Form.Item>
            )
          }
        ]}
      />
    );
  };

  const renderUploadConfig = () => {
    return (
      <Form.Item
        className={commonFormItemsClassName}
        label={t('page.assistant.labels.upload')}
        name='upload'
        rules={[{ required: true }]}
      >
        <UploadConfig />
      </Form.Item>
    );
  };

  const commonFormItems = (
    <div
      className={classNames({
        hidden: !showAdvanced
      })}
    >
      <Form.Item
        className={commonFormItemsClassName}
        label={t('page.assistant.labels.system_prompt')}
        name='role_prompt'
      >
        <Input.TextArea
          className='w-150 h-80!'
          placeholder={t('page.assistant.hints.system_prompt')}
        />
      </Form.Item>

      <Form.Item
        className={commonFormItemsClassName}
        label={t('page.assistant.labels.keepalive')}
        name='keepalive'
        rules={[defaultRequiredRule]}
      >
        <Input className='max-w-600px' />
      </Form.Item>
    </div>
  );

  return (
    <Spin spinning={props.loading || loading || false}>
      <Form
        scrollToFirstError
        autoComplete='off'
        colon={false}
        form={form}
        initialValues={initialValues}
        labelCol={{ span: 4 }}
        layout='horizontal'
        wrapperCol={{ span: 18 }}
        onFinish={onFinish}
        onFinishFailed={onFinishFailed}
      >
        <Form.Item
          label={t('page.assistant.labels.name')}
          name='name'
          rules={[{ required: true }]}
        >
          <Input className='max-w-600px' />
        </Form.Item>
        <Form.Item
          label={t('page.assistant.labels.description')}
          name='description'
        >
          <Input className='max-w-600px' />
        </Form.Item>
        <Form.Item
          label={t('page.assistant.labels.icon')}
          name='icon'
          rules={[{ required: true }]}
        >
          <IconSelector
            className='max-w-600px'
            icons={iconsMeta}
            type='connector'
          />
        </Form.Item>
        <Form.Item
          label={t('page.assistant.labels.category')}
          name='category'
        >
          <Select
            className='max-w-600px'
            maxCount={1}
            mode='tags'
            placeholder='Select or input a category'
            options={categories.map(cate => {
              return { value: cate };
            })}
          />
        </Form.Item>
        <Form.Item
          label={t('page.assistant.labels.tags')}
          name='tags'
        >
          <Tags />
        </Form.Item>
        <Form.Item
          label={t('page.assistant.labels.type')}
          name='type'
          rules={[{ required: true }]}
        >
          <AssistantMode
            value={assistantMode}
            onChange={setAssistantMode}
          />
        </Form.Item>

        {assistantMode === 'deep_think' && (
          <>
            <Form.Item
              className={commonFormItemsClassName}
              label={t('page.assistant.labels.workflow_configuration')}
            >
              {renderIntentRecognitionCollapse()}

              {renderInternetSearchCollapse()}

              {renderLargeModelToolsCollapse()}

              {renderGenerateAnswersCollapse()}
            </Form.Item>

            {renderUploadConfig()}
          </>
        )}

        {assistantMode === 'simple' && (
          <Form.Item
            label={t('page.assistant.labels.answering_model')}
            name={['answering_model']}
            rules={[
              {
                required: true,
                validator: (_, value) => {
                  if (!value || !value.id) {
                    return Promise.reject(new Error(t('page.assistant.hints.selectModel')));
                  }
                  return Promise.resolve();
                }
              }
            ]}
          >
            <ModelSelect
              modelType='answering_model'
              namePrefix={['answering_model']}
              providers={modelProviders}
              showTemplate={false}
              width='600px'
            />
          </Form.Item>
        )}

        {assistantMode === 'simple' && (
          <Form.Item
            extra={<AvailableVariable type='answering_model' />}
            initialValue={DefaultPromptTemplates.answering_model}
            label={t('page.assistant.labels.role_prompt')}
            name={['answering_model', 'prompt', 'template']}
          >
            <Input.TextArea
              className='w-600px'
              placeholder='Please enter the role prompt instructions'
              style={{ height: 320 }}
            />
          </Form.Item>
        )}

        <Form.Item
          label={t('page.assistant.labels.greeting_settings')}
          name={['chat_settings', 'greeting_message']}
        >
          <Input.TextArea className='w-600px' />
        </Form.Item>
        <Form.Item
          label={t('page.assistant.labels.enabled')}
          name='enabled'
        >
          <Switch size='small' />
        </Form.Item>
        <Form.Item label=' '>
          <Button
            className='p-0'
            type='link'
            onClick={() => setShowAdvanced(!showAdvanced)}
          >
            {t('common.advanced')} <SvgIcon icon={`${showAdvanced ? 'mdi:chevron-up' : 'mdi:chevron-down'}`} />
          </Button>
        </Form.Item>

        {assistantMode === 'simple' && (
          <Form.Item
            className={commonFormItemsClassName}
            label={t('page.assistant.labels.capability_extension')}
          >
            {renderInternetSearchCollapse()}

            {renderLargeModelToolsCollapse()}
          </Form.Item>
        )}

        {assistantMode === 'simple' && renderUploadConfig()}

        <Form.Item
          className={`${showAdvanced ? '' : 'h-0px m-0px overflow-hidden'}`}
          label={t('page.assistant.labels.chat_settings')}
        >
          <div className='max-w-600px'>
            <SuggestedChatForm checked={suggestedChatChecked} />
            {/* <div>
              <p>{t("page.assistant.labels.input_preprocessing")}</p>
              <div className="text-gray-400 leading-6 mb-1">
                {t("page.assistant.labels.input_preprocessing_desc")}
              </div>
              <Form.Item name={["chat_settings", "input_preprocess_tpl"]}>
                <Input.TextArea
                  placeholder={t(
                    "page.assistant.labels.input_preprocessing_placeholder",
                  )}
                  className="w-600px"
                />
              </Form.Item>
            </div> */}
            <div>
              <p className='mb-1'>{t('page.assistant.labels.input_placeholder')}</p>
              <Form.Item name={['chat_settings', 'placeholder']}>
                <Input.TextArea className='w-600px' />
              </Form.Item>
            </div>
            <div className='flex items-center justify-between'>
              <div>
                <p>{t('page.assistant.labels.history_message_number')}</p>
                <div className='mb-1 text-gray-400 leading-6'>
                  {t('page.assistant.labels.history_message_number_desc')}
                </div>
              </div>
              <Form.Item name={['chat_settings', 'history_message', 'number']}>
                <InputNumber
                  max={64}
                  min={0}
                />
              </Form.Item>
            </div>
            <div className='flex items-center justify-between'>
              <div>
                <p>{t('page.assistant.labels.history_message_compression_threshold')}</p>
                <div className='mb-1 text-gray-400 leading-6'>
                  {t('page.assistant.labels.history_message_compression_threshold_desc')}
                </div>
              </div>
              <Form.Item name={['chat_settings', 'history_message', 'compression_threshold']}>
                <InputNumber
                  max={4000}
                  min={500}
                />
              </Form.Item>
            </div>
            <div className='flex items-center justify-between'>
              <div>
                <p>{t('page.assistant.labels.history_summary')}</p>
                <div className='mb-1 text-gray-400 leading-6'>{t('page.assistant.labels.history_summary_desc')}</div>
              </div>
              <Form.Item name={['chat_settings', 'history_message', 'summary']}>
                <Switch size='small' />
              </Form.Item>
            </div>
          </div>
        </Form.Item>

        {commonFormItems}

        <Form.Item label=' '>
          <Button
            htmlType='submit'
            type='primary'
          >
            {t('common.save')}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});

export const SuggestedChatForm = ({ checked }: { readonly checked: boolean }) => {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(checked);
  useEffect(() => {
    setEnabled(checked);
  }, [checked]);
  const onEnabledChange = (checked: boolean) => {
    setEnabled(checked);
  };
  return (
    <div>
      <div className='mb-1 flex items-center gap-1 text-gray-400 leading-6'>
        {t('page.assistant.labels.suggested_chat')}{' '}
        <Form.Item
          name={['chat_settings', 'suggested', 'enabled']}
          style={{ margin: 0 }}
        >
          <Switch
            defaultChecked
            size='small'
            onChange={onEnabledChange}
          />
        </Form.Item>
      </div>
      <Form.Item
        className={`${enabled ? '' : 'h-0px m-0px overflow-hidden'}`}
        name={['chat_settings', 'suggested', 'questions']}
      >
        <SuggestedChat />
      </Form.Item>
    </div>
  );
};

export const SuggestedChat = ({ value = [], onChange }: any) => {
  const initialValue = useMemo(() => {
    const iv = (value || []).map((v: string) => ({
      value: v,
      key: getUUID()
    }));
    return iv.length ? iv : [{ value: '', key: getUUID() }];
  }, [value]);

  const [innerValue, setInnerValue] = useState<{ value: string; key: string }[]>(initialValue);
  const prevValueRef = useRef<string[]>([]);

  // Prevent unnecessary updates
  useEffect(() => {
    if (JSON.stringify(prevValueRef.current) !== JSON.stringify(value)) {
      prevValueRef.current = value;
      const iv = (value || []).map((v: string) => ({
        value: v,
        key: getUUID()
      }));
      setInnerValue(iv.length ? iv : [{ value: '', key: getUUID() }]);
    }
  }, [value]);

  const onDeleteClick = (key: string) => {
    const newValues = innerValue.filter(v => v.key !== key);
    setInnerValue(newValues.length ? newValues : [{ value: '', key: getUUID() }]);
    const newValue = newValues.map(v => v.value);
    prevValueRef.current = newValue;
    onChange?.(newValue);
  };

  const onAddClick = () => {
    setInnerValue([...innerValue, { value: '', key: getUUID() }]);
  };

  const onItemChange = (key: string, newValue: string) => {
    const updatedValues = innerValue.map(v => (v.key === key ? { ...v, value: newValue } : v));
    setInnerValue(updatedValues);
    const filterValues = updatedValues.filter(v => v.value != '').map(v => v.value);
    prevValueRef.current = filterValues;
    onChange?.(filterValues);
  };

  const { t } = useTranslation();

  return (
    <div>
      {innerValue.map(v => (
        <div
          className='mb-15px flex items-center'
          key={v.key}
        >
          <Input
            placeholder='eg: what is easysearch?'
            value={v.value}
            onChange={e => {
              onItemChange(v.key, e.target.value);
            }}
          />
          <div
            className='ml-15px cursor-pointer'
            onClick={() => onDeleteClick(v.key)}
          >
            <DeleteOutlined />
          </div>
        </div>
      ))}
      <Button
        icon={<PlusOutlined />}
        style={{ width: 80 }}
        type='primary'
        onClick={onAddClick}
      />
    </div>
  );
};
