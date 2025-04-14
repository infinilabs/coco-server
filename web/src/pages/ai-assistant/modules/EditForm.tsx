import {
  Button,
  Form,
  Input,
  message,
  Select,
  Switch,
  InputNumber,
  Space,
  Spin,
} from 'antd';
import type { FormProps } from 'antd';
import {getConnectorIcons} from '@/service/api/connector';
import { IconSelector } from "../../connector/new/icon_selector";
import {DeleteOutlined, PlusOutlined, UnorderedListOutlined} from "@ant-design/icons";
import { useRequest } from '@sa/hooks';
import { fetchDataSourceList, searchModelPovider,getEnabledModelProviders } from '@/service/api';
import { searchMCPServer } from '@/service/api/mcp-server';
import { useLoading } from '@sa/hooks';
import {AssistantMode} from "./AssistantMode";
import {DatasourceConfig} from "./DatasourceConfig";
import {DeepThink} from "./DeepThink";
import { formatESSearchResult } from '@/service/request/es';
import ModelSelect from './ModelSelect';

const PARAMETERS = [
  {
      key: 'temperature',
      input: <InputNumber min={0} step={0.1} max={1.0}/>
  },
  {
      key: 'top_p',
      input: <InputNumber min={0} step={0.1}  max={1.0}/>
  },
  {
    key: 'presence_penalty',
    input: <InputNumber min={-2.0} step={0.1} max={2.0} />
  },
  {
      key: 'frequency_penalty',
      input: <InputNumber min={-2.0} step={0.1} max={2.0} />
  },
  {
      key: 'max_tokens',
      input: <InputNumber min={1} step={1} precision={0} max={16385}/>
  },
  
]

interface AssistantFormProps  {
  initialValues: any;
  onSubmit: (values: any, startLoading:()=>void, endLoading: ()=>void)=>void;
  mode: string;
  loading: boolean;
}

export const EditForm = memo((props: AssistantFormProps)=> {
  const { initialValues, onSubmit, mode } = props;
  const [form] = Form.useForm();
  useEffect(()=>{
    if(initialValues){
      form.setFieldsValue({
        ...initialValues,
        model_settings: initialValues.answering_model?.settings || {},
      })
    }
  } , [initialValues])
  const { t } = useTranslation();
  const { endLoading, loading, startLoading } = useLoading();

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    onSubmit?.(values, startLoading, endLoading);
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

  const [showAdvanced, setShowAdvanced] = useState(false);
  const { data: result, run, loading: dataSourceLoading } = useRequest(fetchDataSourceList, {
    manual: true,
  });

  useEffect(() => {
    run({
      from: 0, 
      size: 10000,
    })
  }, [])

  const dataSource = useMemo(() => {
    return result?.hits?.hits?.map((item) => ({...item._source})) || []
  }, [JSON.stringify(result)])

  const { data: modelsResult, run: fetchModelProviders} = useRequest(getEnabledModelProviders, {
    manual: true,
  });
  const modelProviders = useMemo(() => {
    if(!modelsResult) return [];
    const res = formatESSearchResult(modelsResult);
    return res.data;
  }, [JSON.stringify(modelsResult)]);
  useEffect(() => {
    fetchModelProviders(10000)
  }, [])

  const { data: mcpServerResult, run: fetchMCPServers } = useRequest(searchMCPServer, {
    manual: true,
  });

  useEffect(() => {
    fetchMCPServers({
      from: 0, 
      size: 10000,
    })
  }, [])

  const mcpServers = useMemo(() => {
    return mcpServerResult?.hits?.hits?.map((item) => ({...item._source})) || []
  }, [JSON.stringify(mcpServerResult)])

  const [assistantMode, setAssistantMode] = useState(initialValues?.mode || 'simple');
  useEffect(() => {
    if(initialValues?.type) {
      setAssistantMode(initialValues.type);
    }
  }, initialValues?.type)
  const handleAssistantModeChange = (value: string) => {
    setAssistantMode(value);
  }

  const [suggestedChatChecked, setSuggestedChatChecked] = useState(initialValues?.chat_settings?.suggested?.enabled || false);
  useEffect(() => {
    setSuggestedChatChecked(initialValues?.chat_settings?.suggested?.enabled || false);
  }, [initialValues?.chat_settings?.suggested?.enabled])

  return (
        <Spin spinning={props.loading || loading || false}>
         <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={initialValues}
            colon={false}
            form={form}
            autoComplete="off"
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item label={t('page.assistant.labels.name')} rules={[{ required: true}]} name="name">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.assistant.labels.description')} name="description">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.assistant.labels.icon')} name="icon" rules={[{ required: true}]}>
              <IconSelector type="connector" icons={iconsMeta} className='max-w-300px' />
            </Form.Item>
            <Form.Item label={t('page.assistant.labels.type')} name="type" rules={[{ required: true}]}>
              <AssistantMode onChange={handleAssistantModeChange} />
            </Form.Item>
            <Form.Item label={t('page.assistant.labels.answering_model')} rules={[{ required: true}]} name={[ "answering_model"]}>
              <ModelSelect
                width="600px"
                providers={modelProviders}
              />
            </Form.Item>
            { assistantMode === 'deep_think' && <Form.Item label={t('page.assistant.labels.deep_think_model')}><DeepThink className='max-w-600px' providers={modelProviders} /></Form.Item> }
            <Form.Item label={t('page.assistant.labels.datasource')} rules={[{ required: true}]} name={["datasource"]}>
              <DatasourceConfig
                options={[{label: "*", value: "*"}].concat(dataSource.map((item) => ({
                  label: item.name,
                  value: item.id,
                })))}
              />
            </Form.Item>
            <Form.Item label={t('page.assistant.labels.mcp_servers')} rules={[{ required: true}]} name="mcp_servers">
              <DatasourceConfig
                  options={[{label: "*", value: "*"}].concat(mcpServers.map((item) => ({
                    label: item.name,
                    value: item.id,
                  })))}
                />
            </Form.Item>
            <Form.Item
                name={'keepalive'}
                label={t('page.assistant.labels.keepalive')}
                rules={[defaultRequiredRule]}
            >
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item
                name='role_prompt'
                label={t('page.assistant.labels.role_prompt')}
            >
              <Input.TextArea placeholder='Please enter the role prompt instructions' className='w-600px' />
            </Form.Item>
            <Form.Item label={t('page.assistant.labels.enabled')} name="enabled">
              <Switch />
            </Form.Item>
            <Form.Item
                    label=" "
                >
                <Button type="link" className="p-0" onClick={() => setShowAdvanced(!showAdvanced)}>
                    {t('common.advanced')} <SvgIcon icon={`${showAdvanced ? "mdi:chevron-up" : "mdi:chevron-down"}`}/>
                </Button>
            </Form.Item>
            <Form.Item
              className={`${showAdvanced ? '' : 'h-0px m-0px overflow-hidden'}`}
              label={t('page.assistant.labels.chat_settings')}>
              <div className='max-w-600px'>
                <div>
                  <div className='text-gray-400 leading-6 mb-1'>
                    {t('page.assistant.labels.greeting_settings')}
                  </div>
                  <Form.Item name={["chat_settings", "greeting_message"]}>
                    <Input.TextArea  className='w-600px' />
                  </Form.Item>
                </div>
                <SuggestedChatForm checked={suggestedChatChecked}/>
                <div>
                  <p>{t('page.assistant.labels.input_preprocessing')}</p>
                  <div className='text-gray-400 leading-6 mb-1'>{t('page.assistant.labels.input_preprocessing_desc')}</div>
                  <Form.Item name={["chat_settings", "input_preprocess_tpl"]}>
                    <Input.TextArea  placeholder={t('page.assistant.labels.input_preprocessing_placeholder')} className='w-600px' /> 
                  </Form.Item>
                </div>
                <div className='flex justify-between items-center'>
                  <div>
                    <p>{t('page.assistant.labels.history_message_number')}</p>
                    <div className='text-gray-400 leading-6 mb-1'>{t('page.assistant.labels.history_message_number_desc')}</div>
                  </div>
                  <Form.Item name={["chat_settings", "history_message", "number"]}>
                    <InputNumber min={0} max={64} /> 
                  </Form.Item>
                </div>
                <div className='flex justify-between items-center'>
                  <div>
                    <p>{t('page.assistant.labels.history_message_compression_threshold')}</p>
                    <div className='text-gray-400 leading-6 mb-1'>{t('page.assistant.labels.history_message_compression_threshold_desc')}</div>
                  </div>
                  <Form.Item name={["chat_settings", "history_message", "compression_threshold"]}>
                    <InputNumber min={500} max={4000} /> 
                  </Form.Item>
                </div>
                <div className='flex justify-between items-center'>
                  <div>
                    <p>{t('page.assistant.labels.history_summary')}</p>
                    <div className='text-gray-400 leading-6 mb-1'>{t('page.assistant.labels.history_summary_desc')}</div>
                  </div>
                  <Form.Item name={["chat_settings", "history_message", "summary"]}>
                    <Switch /> 
                  </Form.Item>
                </div>
              </div>
            </Form.Item>
            {/* <Form.Item
                label={t('page.assistant.labels.model_settings')}
                className={`${showAdvanced ? '' : 'h-0px m-0px overflow-hidden'}`}
            >
                {
                    PARAMETERS.map((item) => (
                        <div key={item.key} className={`flex justify-between items-center max-w-600px`}>
                            <div className="[flex:1]">
                                <div className="color-#333">{t(`page.assistant.labels.${item.key}`)}</div>
                                <div className="text-gray-400 mb-10px">{t(`page.assistant.labels.${item.key}_desc`)}</div>
                            </div>
                            <div >
                                <Form.Item
                                    name={['model_settings', item.key]}
                                    label=""
                                >
                                    {item.input}
                                </Form.Item>
                            </div>
                        </div>
                    ))
                }
            </Form.Item> */}
            <Form.Item label=" ">
             <Button type='primary'  htmlType="submit">{t('common.save')}</Button>
            </Form.Item>
          </Form>
        </Spin>
  )
})

export const SuggestedChatForm = ({checked}: {checked: boolean})=>{
  const {t} = useTranslation();
  const [enabled, setEnabled] = useState(checked);
  useEffect(() => {
    setEnabled(checked);
  }, [checked]);
  const onEnabledChange = (checked: boolean) => {
    setEnabled(checked);
  }
  return (<div><div className='text-gray-400 leading-6 mb-1 flex gap-1 items-center'>
    {t('page.assistant.labels.suggested_chat')} <Form.Item name={["chat_settings", "suggested", "enabled"]} style={{margin:0}}><Switch size='small' onChange={onEnabledChange} defaultChecked /></Form.Item>
  </div>
  <Form.Item name={["chat_settings", "suggested", "questions"]}  className={`${enabled ? '' : 'h-0px m-0px overflow-hidden'}`}>
    <SuggestedChat />
  </Form.Item>
  </div>)
}

export const SuggestedChat = ({ value = [], onChange }: any) => {
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

  const onDeleteClick = (key: string) => {
    const newValues = innerValue.filter((v) => v.key !== key);
    setInnerValue(newValues.length ? newValues : [{ value: '', key: crypto.randomUUID() }]);
    const newValue = newValues.map((v) =>v.value)
    prevValueRef.current = newValue;
    onChange?.(newValue);
  };

  const onAddClick = () => {
    setInnerValue([...innerValue, { value: '', key: crypto.randomUUID() }]);
  };

  const onItemChange = (key: string, newValue: string) => {
    const updatedValues = innerValue.map((v) =>
      v.key === key ? { ...v, value: newValue } : v
    );
    setInnerValue(updatedValues);
    const filterValues = updatedValues.filter((v) => v.value != "").map((v)=>v.value);
    prevValueRef.current = filterValues;
    onChange?.(filterValues);
  };

  const {t} = useTranslation();

  return (
    <div>
      {innerValue.map((v) => (
        <div key={v.key} className="flex items-center mb-15px">
          <Input value={v.value} placeholder='eg: what is easysearch?' onChange={(e)=>{
            onItemChange(v.key, e.target.value);
          }}/>
          <div className="cursor-pointer ml-15px" onClick={() => onDeleteClick(v.key)}>
            <DeleteOutlined />
          </div>
        </div>
      ))}
      <Button type="primary" icon={<PlusOutlined/>} style={{width: 80}} onClick={onAddClick}>
      </Button>
    </div>
  );
};