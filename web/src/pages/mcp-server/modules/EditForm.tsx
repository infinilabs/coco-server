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
import {DeleteOutlined, PlusOutlined, UnorderedListOutlined} from "@ant-design/icons";
import { useLoading } from '@sa/hooks';

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
  const { defaultRequiredRule, formRules } = useFormRules();
  const [type, setType] = useState(initialValues.type || 'sse');
  const onTypeChange = (e: any) => {
    setType(e.target.value);
  }

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
            <Form.Item
                name={'type'}
                label={t('page.mcpserver.labels.type')}
                rules={[defaultRequiredRule]}
            >
               <Radio.Group onChange={onTypeChange}>
                  <Radio value="sse">SSE</Radio>
                  <Radio value="stdio">Stdio</Radio>
                  <Radio value="streamable_http">Streamable HTTP</Radio>
              </Radio.Group>
            </Form.Item>

            {type === "sse" &&<Form.Item
                rules={[{ required: true}]}
                name={["config", "url"]}
                label={"URL"}
            >
              <Input className='w-600px' />
            </Form.Item>
            }
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
              <Input.TextArea className='w-600px' />
            </Form.Item>
            <Form.Item
                name={["config", "env"]}
                label={t('page.mcpserver.labels.config.env')}
            >
              <Input.TextArea className='w-600px' />
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