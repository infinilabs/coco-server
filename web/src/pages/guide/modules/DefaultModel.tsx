import { Button, Col, Form, Input, Modal, Row, Select, Steps, Switch, Tooltip } from 'antd';

import { useLoading } from '@sa/hooks';
import { setupModel } from '@/service/api/guide';
import { FormInstance } from 'antd/lib';
import { searchModelPovider } from '@/service/api/model-provider';
import { formatESSearchResult } from '@/service/request/es';
import { getUUID } from '@/utils/common';
import headBg from '@/assets/imgs/model-guide.png';
import { InfoCircleOutlined } from '@ant-design/icons';
import { localStg } from '@/utils/storage';
import { fetchSettings } from '@/service/api/server';
import { setDefaultModel } from '@/store/slice/server';

const CUSTOM_PROVIDER_ID = getUUID()
const CUSTOM_MODEL_ID = getUUID()

const ModelForm = memo(({ form, name, modelProviderList, apiTokenCache }: { form: FormInstance, name: string, modelProviderList: any[], apiTokenCache: React.MutableRefObject<Record<string, string>> }) => {
  const formItemClassNames = '[&_.ant-input]:!leading-24px [&_.ant-input]:!text-14px [&_.ant-select-content]:!text-14px m-b-24px [&_label]:!color-[var(--ant-color-text-secondary)]';
  const { defaultRequiredRule } = useFormRules();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    createModelProvider: hasAuth('coco#model_provider/create'),
  }

  const modelProviderID = Form.useWatch([name, 'model_provider', 'id'], form);
  const modelID = Form.useWatch([name, 'model_id'], form);

  const modelProvider = useMemo(() => {
    return modelProviderList.find((item) => item.id === modelProviderID)
  }, [modelProviderList, modelProviderID])

  const handleProviderChange = (value: string) => {
    // Reset model fields when provider changes
    form.resetFields([[name, 'model_id'], [name, 'model', 'id']]);
    // Handle api_token: use cache or clear
    if (value !== CUSTOM_PROVIDER_ID && apiTokenCache.current[value]) {
      form.setFieldValue([name, 'api_token'], apiTokenCache.current[value]);
    } else {
      form.setFieldValue([name, 'api_token'], undefined);
    }
  };

  return (
    <Form
      form={form}
      layout='vertical'
      name={name}
    >
      <Row gutter={16}>
        <Col span={12}>
          <Form.Item
            className={formItemClassNames}
            label={t(`page.guide.labels.modelProvider`)}
            name={[name, 'model_provider', 'id']}
            rules={[defaultRequiredRule]}
          >
            <Select
              size="large"
              onChange={handleProviderChange}
              options={modelProviderList
                .map(provider => ({ label: provider.name, value: provider.id }))
                .concat(permissions.createModelProvider ? [{ label: t(`page.guide.labels.custom`), value: CUSTOM_PROVIDER_ID }] : [])
              }
            />
          </Form.Item>
        </Col>
        {
          modelProviderID && !modelProvider ? (
            <>
              <Col span={12}>
                <Form.Item
                  className={formItemClassNames}
                  label={t(`page.guide.labels.modelProviderName`)}
                  name={[name, 'model_provider', 'name']}
                  rules={[defaultRequiredRule]}
                >
                  <Input size="large" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  className={formItemClassNames}
                  label={t(`page.guide.labels.apiType`)}
                  name={[name, 'model_provider', 'api_type']}
                  rules={[defaultRequiredRule]}
                >
                  <Select
                    size="large"
                    options={[
                      { label: 'OpenAI', value: 'openai' },
                      { label: 'Ollama', value: 'ollama' },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  className={formItemClassNames}
                  label={t(`page.guide.labels.baseUrl`)}
                  name={[name, 'model_provider', 'base_url']}
                  rules={[defaultRequiredRule]}
                >
                  <Input size="large" />
                </Form.Item>
              </Col>
            </>
          ) : (
            <Col span={12}>
              <Form.Item
                className={formItemClassNames}
                label={t(`page.guide.labels.model`)}
                name={[name, 'model_id']}
                rules={[defaultRequiredRule]}
              >
                <Select
                  size="large"
                  options={(modelProvider?.models || [])
                    .filter((item: any) => item.type === name)
                    .map((item: any) => ({ label: item.name, value: item.name }))
                    .concat([{ label: t(`page.guide.labels.custom`), value: CUSTOM_MODEL_ID }])
                  }
                />
              </Form.Item>
            </Col>
          )
        }
      </Row>
      {
        modelProviderID === CUSTOM_PROVIDER_ID || modelID === CUSTOM_MODEL_ID ? (
          <>
            <Form.Item
              required={true}
              className={formItemClassNames}
              label={t(`page.guide.labels.modelID`)}
              extra={
                name === 'language' ? (
                  <div className='flex items-center gap-8px pt-8px'>
                    <span>
                      {t('page.modelprovider.labels.inferenceMode')}
                      <Tooltip title={t('page.modelprovider.hints.inferenceMode')}>
                        <InfoCircleOutlined className='ml-4px cursor-pointer' />
                      </Tooltip>
                    </span>
                    <Form.Item
                      noStyle
                      name={[name, 'model', 'support_reasoning']}
                    >
                      <Switch size="small" />
                    </Form.Item>
                  </div>
                ) : null
              }
            >
              <Form.Item
                name={[name, 'model', 'id']}
                noStyle
                rules={[defaultRequiredRule]}
              >
                <Input size="large" />
              </Form.Item>
            </Form.Item>
          </>
        ) : null
      }
      <div className='h-0 w-0 overflow-hidden'>
        <input type="text" />
      </div>
      <Form.Item
        className={formItemClassNames}
        label={t(`page.guide.labels.apiSecret`)}
        name={[name, 'api_token']}
        rules={[defaultRequiredRule]}
      >
        <Input.Password size="large" />
      </Form.Item>
    </Form>
  )
})

const DefaultModel = memo(({ }: {}) => {
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { endLoading, loading, startLoading } = useLoading();
  const [open, setOpen] = useState(true);
  const [modelProviderList, setModelProviderList] = useState([]);
  const [step, setStep] = useState(0);
  const [isSuccess, setIsSuccess] = useState(false);
  const apiTokenCache = useRef<Record<string, string>>({});
  const dispatch = useAppDispatch();

  const { hasAuth } = useAuth()
  const permissions = {
    searchModelPovider: hasAuth('coco#model_provider/search'),
  }

  const stepList = useMemo(() => {
    return [
      {
        key: 'language',
        title: t('page.guide.languageModel.title'),
        desc: t('page.guide.languageModel.desc'),
      },
      {
        key: 'vision',
        title: t('page.guide.visionModel.title'),
        desc: t('page.guide.visionModel.desc')
      },
      {
        key: 'embedding',
        title: t('page.guide.embeddingModel.title'),
        desc: t('page.guide.embeddingModel.desc')
      },
    ]
  }, [step])

  const fetchModelProvider = async () => {
    startLoading();
    const res = await searchModelPovider({ from: 0, size: 10000 })
    if (res?.data) {
      const newResult = formatESSearchResult(res?.data);
      setModelProviderList(newResult.data as any);
    }
    endLoading();
  }

  const onStepChange = async (current: number) => {
    if (current > step) {
      const values = await form.validateFields();
      const currentKey = stepList[step].key;
      const providerId = values[currentKey]?.model_provider?.id;
      const apiToken = values[currentKey]?.api_token;
      if (providerId && providerId !== CUSTOM_PROVIDER_ID && apiToken) {
        apiTokenCache.current[providerId] = apiToken;
      }
      const res = await handleSubmit(values)
      if (res) setStep(current);
    } else {
      setStep(current);
    }
  }

  const formatModelValues = (values: any) => {
    if (!values) return values;
    const { model_id, model, model_provider, ...rest } = values;
    const { id, ...restProvider } = model_provider || {};
    return {
      ...rest,
      model_provider: id === CUSTOM_PROVIDER_ID ? restProvider : model_provider,
      ...(!model_id || model_id === CUSTOM_MODEL_ID ? { model } : { model_id })
    };
  }

  const handleSubmit = async (values: any) => {
    const body = {} as any;
    Object.keys(values).forEach((key) => {
      body[`${key}_model`] = formatModelValues(values[key])
    })
    startLoading();
    const { error } = await setupModel(body);
    endLoading();
    if (!error) {
      updateDefaultModel();
      return true;
    }
    return false;
  };

  const updateDefaultModel = async () => {
    const res = await fetchSettings()
    dispatch(setDefaultModel(res.data.default_model || {}))
  }

  const onClose = () => {
    setOpen(false);
    localStg.set('defaultModelGuide', 'false')
  }

  useEffect(() => {
    if (permissions.searchModelPovider) {
      fetchModelProvider();
    }
  }, []);

  return (
    <Modal
      closable
      open={open}
      onCancel={onClose}
      footer={null}
      width={660}
      classNames={{
        container: '!p-0px [&_.ant-modal-close-x]:color-#fff',
      }}
      destroyOnHidden
    >
      <div style={{ background: `url(${headBg}) no-repeat center/cover` }} className='h-120px px-32px flex flex-col justify-center'>
        <div className='m-b-8px text-24px color-#fff'>{t('page.guide.llm.title')}</div>
        <div className='break-all text-12px color-#fff'>{t('page.guide.llm.desc')}</div>
      </div>

      {
        isSuccess ? (
          <div className='h-280px flex flex-col items-center p-t-40px'>
            <div className='text-48px'>🎉</div>
            <div className='m-b-30px'>{t('page.guide.labels.tipsSuccess')}</div>
            <Button onClick={onClose} type='primary'>{t('page.guide.labels.tipsSuccessButton')}</Button>
          </div>
        ) : (
          <div className='p-32px'>
            <div className='m-b-24px'>
              <Steps
                current={step}
                size='small'
                onChange={onStepChange}
                items={stepList.map(s => ({ title: s.title }))}
              />
            </div>

            <div className='break-all m-b-24px text-14px'>
              {stepList[step].desc}
            </div>

            <ModelForm
              form={form}
              name={stepList[step].key}
              modelProviderList={modelProviderList}
              apiTokenCache={apiTokenCache}
            />

            <div className='flex justify-between m-t-32px'>
              <Button
                color="primary"
                variant="outlined"
                onClick={onClose}
              >
                {t('page.guide.setupLater')}
              </Button>

              <div className='flex gap-8px'>
                {
                  step > 0 ? (
                    <Button
                      className='w-80px'
                      disabled={loading}
                      variant="solid"
                      onClick={() => {
                        setStep(step - 1)
                      }}
                    >
                      {t('page.guide.previous')}
                    </Button>
                  ) : null
                }
                {
                  step < 2 ? (
                    <Button
                      className='w-80px'
                      loading={loading}
                      color="primary"
                      variant="solid"
                      onClick={() => {
                        onStepChange(step + 1)
                      }}
                    >
                      {t('page.guide.next')}
                    </Button>
                  ) : (
                    <Button
                      className='w-80px'
                      loading={loading}
                      color="primary"
                      variant="solid"
                      onClick={async () => {
                        const values = await form.getFieldsValue();
                        const res = await handleSubmit(values);
                        if (res) setIsSuccess(true);
                      }}
                    >
                      {t('common.ok')}
                    </Button>
                  )
                }
              </div>
            </div>
          </div>
        )
      }
    </Modal>
  );
}
);

export default DefaultModel;
