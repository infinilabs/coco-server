import {
  Button,
  Form,
  Input,
  message,
  Switch,
} from 'antd';
import type { FormProps } from 'antd';
import {getConnectorIcons} from '@/service/api/connector';
import {getModelProvider, updateModelProvider, getLLMModels} from '@/service/api/llm';
import { IconSelector } from "../../connector/new/icon_selector";
import {ModelsComponent} from "../new/index";
import { LoaderFunctionArgs, useLoaderData } from 'react-router-dom';
import InfiniIcon from '@/components/common/icon';

export function Component() {
  const { t } = useTranslation();
  const {id}:any = useLoaderData();
  const initialValues = {};
  const [modelProvider, setModelProvider] = useState<any>(initialValues);
  const nav = useNavigate();
  const [form] = Form.useForm();
  useEffect(() => {
    if (!id) return;
    getModelProvider(id).then((res)=>{
      if(res.data?.found === true){
        setModelProvider(res.data._source || {});
        form.setFieldsValue(res.data._source || {});
      }
    });
  }, [id]);

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    updateModelProvider(id, values).then((res)=>{
      if(res.data?.result == "updated"){
        message.success(t('common.updateSuccess'));
        nav('/model-provider/list');
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

  return <div className="bg-white pt-15px pb-15px min-h-full">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>{t('route.model-provider_edit')}</div>
          </div>
        </div>
        <div>
         <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={modelProvider || {}}
            colon={false}
            form={form}
            autoComplete="off"
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item label={t('page.modelprovider.labels.name')} rules={[{ required: true}]} name="name">
              <Input className='max-w-600px' readOnly={modelProvider.builtin === true } />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.api_key')} rules={[{ required: modelProvider.id === "openai" || modelProvider.id === "deepseek"}]} name="api_key">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.icon')} name="icon" rules={[{ required: true}]}>
              {modelProvider.builtin === true ? <InfiniIcon src={modelProvider.icon} height="2em" width="2em"/>: <IconSelector type="connector" icons={iconsMeta} className='max-w-150px' />}
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.endpoint')} rules={formRules.endpoint} name="api_endpoint">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.modelprovider.labels.models')} rules={[{ required: true}]} name="models">
              <ModelsComponent/>
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

export async function loader({ params }: LoaderFunctionArgs) {
  return params;
 }