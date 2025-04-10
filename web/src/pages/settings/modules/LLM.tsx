import { useLoading } from '@sa/hooks';
import { Button, Form, Input, InputNumber, Spin, Switch } from 'antd';
import { ReactSVG } from 'react-svg';

import '../index.scss';
import DeepseekSvg from '@/assets/svg-icon/deepseek.svg';
import OllamaSvg from '@/assets/svg-icon/ollama.svg';
import OpenAISvg from '@/assets/svg-icon/openai.svg';
import { fetchSettings, updateSettings } from '@/service/api/server';

const PARAMETERS = [
  {
    input: (
      <InputNumber
        min={0}
        step={0.1}
      />
    ),
    key: 'temperature'
  },
  {
    input: (
      <InputNumber
        min={0}
        step={0.1}
      />
    ),
    key: 'top_p'
  },
  {
    input: (
      <InputNumber
        min={0}
        precision={0}
        step={1}
      />
    ),
    key: 'max_tokens'
  },
  {
    input: (
      <InputNumber
        min={0}
        step={0.1}
      />
    ),
    key: 'presence_penalty'
  },
  {
    input: (
      <InputNumber
        min={0}
        step={0.1}
      />
    ),
    key: 'frequency_penalty'
  },
  {
    hideDesc: true,
    input: (
      <Switch
        defaultChecked
        size="small"
      />
    ),
    key: 'enhanced_inference'
  }
  // {
  //     key: '7',
  //     label: '推理强度',
  //     desc: '值越大，推理能力越强，但可能会增加响应时间和 Token 消耗',
  //     input: (
  //         <Select
  //             defaultValue="低"
  //             style={{ width: 88 }}
  //             options={[
  //                 { value: '低', label: '低' },
  //                 { value: '中', label: '中' },
  //                 { value: '高', label: '高' },
  //             ]}
  //         />
  //     )
  // },
];

type ModelType = 'deepseek' | 'ollama' | 'openai';

const LLM = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const [type, setType] = useState<ModelType>();
  const [showAdvanced, setShowAdvanced] = useState(false);

  const { endLoading, loading, startLoading } = useLoading();
  const { defaultRequiredRule, formRules } = useFormRules();

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  const handleSubmit = async () => {
    const params = await form.validateFields();
    startLoading();
    const result = await updateSettings({
      llm: params
    });
    if (result.data.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
    }
    endLoading();
  };

  useMount(() => {
    run();
  });

  useEffect(() => {
    if (data?.data?.llm) {
      form.setFieldsValue(data.data.llm || { keepalive: '30m', type: 'deepseek' });
      setType(data?.data?.llm?.type || 'deepseek');
    }
  }, [JSON.stringify(data)]);

  return (
    <Spin spinning={dataLoading || loading}>
      <Form
        className="settings-form"
        colon={false}
        form={form}
        labelAlign="left"
      >
        <Form.Item
          label={t(`page.settings.llm.type`)}
          name="type"
          rules={[defaultRequiredRule]}
        >
          <ButtonRadio
            options={[
              {
                label: (
                  <span className="deepseek-icon flex items-center">
                    <ReactSVG
                      className="m-r-4px"
                      src={DeepseekSvg}
                    />
                    Deepseek
                  </span>
                ),
                value: 'deepseek'
              },
              {
                label: (
                  <span className="flex items-center">
                    <ReactSVG
                      className="m-r-4px"
                      src={OllamaSvg}
                    />
                    Ollama
                  </span>
                ),
                value: 'ollama'
              },
              {
                label: (
                  <span className="flex items-center">
                    <ReactSVG
                      className="m-r-4px"
                      src={OpenAISvg}
                    />
                    OpenAI
                  </span>
                ),
                value: 'openai'
              }
            ]}
            onChange={(value: ModelType) => {
              setType(value);
              form.setFieldsValue({ token: undefined });
            }}
          />
        </Form.Item>
        <Form.Item
          label={t(`page.settings.llm.endpoint`)}
          name="endpoint"
          rules={formRules.endpoint}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label={t(`page.settings.llm.defaultModel`)}
          name="default_model"
          rules={[defaultRequiredRule]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label={t(`page.settings.llm.keepalive`)}
          name="keepalive"
          rules={[defaultRequiredRule]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label={t(`page.settings.llm.intent_analysis_model`)}
          name="intent_analysis_model"
          rules={[defaultRequiredRule]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label={t(`page.settings.llm.picking_doc_model`)}
          name="picking_doc_model"
          rules={[defaultRequiredRule]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label={t(`page.settings.llm.answering_model`)}
          name="answering_model"
          rules={[defaultRequiredRule]}
        >
          <Input />
        </Form.Item>
        {(type === 'openai' || type === 'deepseek') && (
          <Form.Item
            label="Token"
            name="token"
            rules={[defaultRequiredRule]}
          >
            <Input.Password />
          </Form.Item>
        )}
        <Form.Item label=" ">
          <Button
            className="p-0"
            type="link"
            onClick={() => setShowAdvanced(!showAdvanced)}
          >
            {t('common.advanced')} <SvgIcon icon={`${showAdvanced ? 'mdi:chevron-down' : 'mdi:chevron-up'}`} />
          </Button>
        </Form.Item>
        <Form.Item
          className={`${showAdvanced ? '' : 'h-0px m-0px overflow-hidden'}`}
          label={t(`page.settings.llm.requestParams`)}
        >
          {PARAMETERS.map(item => (
            <div
              className="flex items-center justify-between"
              key={item.key}
            >
              <div className="[flex:1]">
                <div className="color-#333">{t(`page.settings.llm.${item.key}`)}</div>
                {!item.hideDesc && <div className="color-#999">{t(`page.settings.llm.${item.key}_desc`)}</div>}
              </div>
              <div>
                <Form.Item
                  label=""
                  name={['parameters', item.key]}
                >
                  {item.input}
                </Form.Item>
              </div>
            </div>
          ))}
        </Form.Item>
        <Form.Item label=" ">
          <Button
            type="primary"
            onClick={() => handleSubmit()}
          >
            {t('common.update')}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});

export default LLM;
