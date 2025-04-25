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
  Radio,
} from 'antd';
import type { FormProps } from 'antd';
import { useLoading } from '@sa/hooks';
import { IconSelector } from "@/pages/connector/new/icon_selector";
import {getConnectorIcons} from '@/service/api/connector';
import {getMCPCategory} from '@/service/api/mcp-server';
import { formatESSearchResult } from '@/service/request/es';

interface MCPServerFormProps  {
  initialValues: any;
  onSubmit: (values: any, startLoading:()=>void, endLoading: ()=>void)=>void;
  mode: string;
  loading: boolean;
}

export const EditForm = memo((props: MCPServerFormProps)=> {
  const { initialValues, onSubmit, mode } = props;
  const [form] = Form.useForm();
  useEffect(()=>{
    if(initialValues){
      if(initialValues.type === "stdio"){
        if (!initialValues.config) initialValues.config = {}
        if (Array.isArray(initialValues.config.args)) {
          initialValues.config.args = initialValues.config.args?.join("\n");
        }
        const env: any = {}
        if (initialValues.config.env) {
          for (const key in initialValues.config.env) {
            env[key] = `${key}=${initialValues.config.env[key]}`
          }
        }
        initialValues.config.env = Object.values(env).join("\n");
      }
      form.setFieldsValue({
        ...initialValues,
        model_settings: initialValues.answering_model?.settings || {},
      })
    }
  } , [initialValues])
  const { t } = useTranslation();
  const { endLoading, loading, startLoading } = useLoading();

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    Array.isArray(values.category) && (values.category = values.category[0]);
    if(values.type === "stdio"){
      if(values.config?.args){
        values.config.args = values.config.args?.split("\n");
      }else{
        values.config.args = [];
      }
      if(values.config?.env){
        const env: any = {}
        values.config.env?.split("\n").forEach((item: string) => {
          const [key, value] = item.split('=');
          env[key] = value;
        })
        values.config.env = env;
      }else{
        values.config.env = {};
      }
    }
    onSubmit?.(values, startLoading, endLoading);
  };
  
  const onFinishFailed: FormProps<any>['onFinishFailed'] = (errorInfo) => {
    console.log('Failed:', errorInfo);
  };
  const { defaultRequiredRule, formRules } = useFormRules();
  const [type, setType] = useState(initialValues.type || 'sse');
  useEffect(() => {
    if(initialValues.type){
      setType(initialValues.type);
    }
  }, [initialValues.type]);
  const onTypeChange = (e: any) => {
    setType(e.target.value);
  }
  const [iconsMeta, setIconsMeta] = useState([]);
  useEffect(() => {
    getConnectorIcons().then((res)=>{
      if(res.data?.length > 0){
        setIconsMeta(res.data);
      }
    });
  }, []);

  const [categories, setCategories] = useState([]);
  useEffect(() => {
    getMCPCategory().then(({ data }) => {
      if (!data?.error) {
        const newData = formatESSearchResult(data);
        const cates = newData.aggregations.categories.buckets.map((item: any) => {
          return item.key;
        });
        setCategories(cates);
      }
    });
  }, []);
  const argsPlaceholder = `arg1
arg2`;
  const envPlaceholder = `Key1=a
Key2=b`;

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
            <Form.Item label={t('page.mcpserver.labels.name')} rules={[{ required: true}]} name="name">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.mcpserver.labels.description')} name="description">
              <Input.TextArea className='w-600px' />
            </Form.Item>
            <Form.Item label={t('page.mcpserver.labels.icon')} name="icon" rules={[{ required: true}]}>
              <IconSelector type="connector" icons={iconsMeta} className='max-w-300px' />
            </Form.Item>
            <Form.Item
            label={t('page.mcpserver.labels.category')}
            name="category"
            rules={[{ required: true }]}
          >
            <Select
              className="max-w-600px"
              maxCount={1}
              mode="tags"
              placeholder="Select or input a category"
              options={categories.map(cate => {
                return { value: cate };
              })}
            />
          </Form.Item>
            <Form.Item
                name={'type'}
                label={t('page.mcpserver.labels.type')}
                rules={[defaultRequiredRule]}
            >
               <Radio.Group onChange={onTypeChange}>
                  <Radio value="stdio">Stdio</Radio>
                  <Radio value="streamable_http">Streamable HTTP</Radio>
              </Radio.Group>
            </Form.Item>
            {type === "streamable_http" &&<Form.Item
                rules={[{ required: true}]}
                name={["config", "url"]}
                label={"URL"}
            >
              <Input className='w-600px' />
            </Form.Item>
            }
            {type === "stdio" && <><Form.Item
                name={["config", "command"]}
                rules={[{ required: true}]}
                label={t('page.mcpserver.labels.config.command')}
            >
              <Input className='w-600px' />
            </Form.Item>
            <Form.Item
                name={["config", "args"]}
                label={t('page.mcpserver.labels.config.args')}
            >
              <Input.TextArea placeholder={argsPlaceholder} className='w-600px' />
            </Form.Item>
            <Form.Item
                name={["config", "env"]}
                label={t('page.mcpserver.labels.config.env')}
            >
              <Input.TextArea placeholder={envPlaceholder} className='w-600px' />
            </Form.Item>
            </>}
            <Form.Item label={t('page.mcpserver.labels.enabled')} name="enabled">
              <Switch size="small"/>
            </Form.Item>
            <Form.Item label=" ">
             <Button type='primary'  htmlType="submit">{t('common.save')}</Button>
            </Form.Item>
          </Form>
        </Spin>
  )
})