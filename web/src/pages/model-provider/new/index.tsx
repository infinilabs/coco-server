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
import {DeleteOutlined} from "@ant-design/icons";
import { formatESSearchResult } from '@/service/request/es';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    const newValues = {
      ...values,
      models: values.models.map((item: any) => ({name: item})),
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

  return <div className="bg-white pt-15px pb-15px min-h-full">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>{t('route.model-provider_new')}</div>
          </div>
        </div>
        <div>
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
              <Select options={[{label:"OpenAI", value:"openai"}, {label:"Gemini", value:"gemini"},{label:"Anthropic", value:"anthropic"}]} className='max-w-150px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.api_key')} rules={[{ required: initialValues.id === "openai" || initialValues.id === "deepseek"}]} name="api_key">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.base_url')} rules={formRules.endpoint} name="base_url">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.models')} rules={[{ required: true}]} name="models">
              <ModelsComponent/>
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.description')} name="description">
              <Input.TextArea className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.enabled')} name="enabled">
              <Switch />
            </Form.Item>
            <Form.Item label=" ">
              <Button type='primary'  htmlType="submit">{t('common.save')}</Button>
            </Form.Item>
          </Form>

        </div>
      </div>
  </div>
}

export const ModelsComponent = ({ value = [], onChange }: any) => {
  const initialValue = useMemo(() => {
    const iv = (value || []).map((v: string) => ({
      value: v,
      key: crypto.randomUUID(),
    }));
    return iv.length ? iv : [{ value: '', key: crypto.randomUUID() }];
  }, [value]);

  const [innerValue, setInnerValue] = useState<{ value: string; key: string }[]>(initialValue);
  const prevValueRef = useRef<string[]>([]);

  // Prevent unnecessary updates
  useEffect(() => {
    if (JSON.stringify(prevValueRef.current) !== JSON.stringify(value)) {
      prevValueRef.current = value;
      const iv = (value || []).map((v: string) => ({
        value: v,
        key: crypto.randomUUID(),
      }));
      setInnerValue(iv.length ? iv : [{ value: '', key: crypto.randomUUID() }]);
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
    setInnerValue(newValues.length ? newValues : [{ value: '', key: crypto.randomUUID() }]);
    onChange?.(newValues.map((v) => Array.isArray(v.value) ? v.value[0]: v.value));
  };

  const onAddClick = () => {
    setInnerValue([...innerValue, { value: '', key: crypto.randomUUID() }]);
  };

  const onItemChange = (key: string, newValue: string) => {
    const updatedValues = innerValue.map((v) =>
      v.key === key ? { ...v, value: newValue } : v
    );
    setInnerValue(updatedValues);
    onChange?.(updatedValues.filter((v) => v.value).map((v) => Array.isArray(v.value) ? v.value[0]: v.value));
  };

  const {t} = useTranslation();

  return (
    <div>
      {innerValue.map((v) => (
        <div key={v.key} className="flex items-center mb-15px">
          <Select
            mode="tags"
            value={v.value || undefined}
            className="max-w-570px"
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
          <div className="cursor-pointer ml-15px" onClick={() => onDeleteClick(v.key)}>
            <DeleteOutlined />
          </div>
        </div>
      ))}
      <Button type="primary" onClick={onAddClick}>
        {t('common.add')}
      </Button>
    </div>
  );
};

