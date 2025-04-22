import {
  Button,
  Form,
  Input,
  message,
  Select,
  Switch,
} from 'antd';
import type { FormProps } from 'antd';
import {createModelProvider, getLLMModels} from '@/service/api/model-provider';
import {getConnectorIcons} from '@/service/api/connector';
import { IconSelector } from "../../connector/new/icon_selector";
import {MinusCircleOutlined} from "@ant-design/icons";
import { formatESSearchResult } from '@/service/request/es';
import ModelSettings from '@/pages/ai-assistant/modules/ModelSettings';
import { settings } from 'nprogress';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    const newValues = {
      ...values,
    }
    createModelProvider(newValues).then((res)=>{
      if(res.data?.result == "created"){
        message.success(t('common.addSuccess'))
        nav('/model-provider/list', {});
      }
    })
  };
  
  const onFinishFailed: FormProps<any>['onFinishFailed'] = (errorInfo) => {
    console.log('Failed:', errorInfo);
  };
  const [iconsMeta, setIconsMeta] = useState([]);
  useEffect(() => {
    getConnectorIcons().then((res)=>{
      if(res.data?.length > 0){
        setIconsMeta(res.data);
      }
    });
  }, []);
  const { defaultRequiredRule, formRules } = useFormRules();
  const initialValues = {
    enabled: true,
  };

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{t('route.model-provider_new')}</div>
        </div>
        <div className="px-30px">
        <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={initialValues}
            colon={false}
            autoComplete="off"
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item label={t('page.modelprovider.labels.name')} rules={[{ required: true}]} name="name">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.icon')} name="icon" rules={[{ required: true}]}>
              <IconSelector type="connector" icons={iconsMeta} className='max-w-150px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.api_type')} name="api_type" rules={[{ required: true}]}>
              <Select options={[{label:"OpenAI", value:"openai"}, {label:"Ollama", value:"ollama"}, {label:"Deepseek", value:"deepseek"}]} className='max-w-150px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.api_key')} name="api_key">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.base_url')} rules={formRules.endpoint} name="base_url">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.models')} rules={[{ required: true}]} name="models">
              <ModelsComponent/>
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.description')} name="description">
              <Input.TextArea className='w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.enabled')} name="enabled">
              <Switch size="small" />
            </Form.Item>
            <Form.Item label=" ">
              <Button type='primary'  htmlType="submit">{t('common.save')}</Button>
            </Form.Item>
          </Form>
        </div>
      </ACard>
    </div>
  )

}

const defaultModelSettings = {
  temperature: 0.7,
  top_p: 0.9,
  presence_penalty: 0,
  frequency_penalty: 0,
  max_tokens: 4000,
}
export const ModelsComponent = ({ value = [], onChange }: any) => {
  const initialValue = useMemo(() => {
    const iv = (value || []).map((v: any) => ({
      value: v,
      key: crypto.randomUUID(),
    }));
    return iv.length ? iv : [{ value: {
      settings: defaultModelSettings,
    }, key: crypto.randomUUID() }];
  }, [value]);

  const [innerValue, setInnerValue] = useState<{ value: any; key: string }[]>(initialValue);
  const prevValueRef = useRef<any[]>([]);

  // Prevent unnecessary updates
  useEffect(() => {
    if (JSON.stringify(prevValueRef.current) !== JSON.stringify(value)) {
      prevValueRef.current = value;
      const iv = (value || []).map((v: any) => ({
        value: v,
        key: crypto.randomUUID(),
      }));
      setInnerValue(iv.length ? iv : [{ value: {settings: defaultModelSettings,}, key: crypto.randomUUID() }]);
    }
  }, [value]);

  const [models, setModels] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    getLLMModels().then(({data})=>{
      if(!data?.error){
        const newData = formatESSearchResult(data);
        const models = newData.aggregations.models.buckets.map((item: any)=>{
          return item.key;
        });
        setModels(models);
      }
    });
  }, []);

  const onDeleteClick = (key: string) => {
    const newValues = innerValue.filter((v) => v.key !== key);
    setInnerValue(newValues.length ? newValues : [{ value: {settings: defaultModelSettings,}, key: crypto.randomUUID() }]);
    onChange?.(newValues.map((v) => Array.isArray(v.value) ? v.value[0]: v.value));
  };

  const onAddClick = () => {
    setInnerValue([...innerValue, { value: {}, key: crypto.randomUUID() }]);
  };

  const onItemChange = (key: string, newValue: any) => {
    const newName = newValue?.[0] || newValue;
    const updatedValues = innerValue.map((v) =>
      v.key === key ? { ...v, value: {
        ...(v.value || {}),
        name: newName,
      } } : v
    );
    setInnerValue(updatedValues);
    onChange?.(updatedValues.filter((v) => v.value?.name).map(v => v.value));
  };

  const {t} = useTranslation();

  const onSettingsChange = (key: string, settings: any) => {
    const updatedValues = innerValue.map((v) =>
      v.key === key ? { ...v, value: {
        ...(v.value || {}),
        settings: settings,
      } } : v
    );
    setInnerValue(updatedValues);
    const filterValues = updatedValues.filter((v) => v.value?.name).map(v => v.value);
    onChange?.(filterValues);
  }

  return (
    <div>
      {innerValue.map((v) => (
        <div key={v.key} className="flex items-center mb-15px">
          <Select
            mode="tags"
            value={v.value?.name || undefined}
            className="max-w-548px"
            onChange={(newV) => onItemChange(v.key, newV)}
            placeholder="Select or input a model"
            maxCount={1}
            loading={loading}
          >
            {models.map((model) => (
              <Select.Option key={model} value={model}>
                {model}
              </Select.Option>
            ))}
          </Select>
          <div className="ml-10px">
            <ModelSettings value={v.value?.settings} onChange={(settings) => onSettingsChange(v.key, settings)}/>
          </div>
          <div className="cursor-pointer ml-15px" onClick={() => onDeleteClick(v.key)}>
            <MinusCircleOutlined className='text-[#999]' />
          </div>
        </div>
      ))}
      <Button type="primary" onClick={onAddClick}>
        {t('common.add')}
      </Button>
    </div>
  );
};

