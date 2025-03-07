import {
  Button,
  Checkbox,
  Form,
  Input,
  message,
  Switch,
} from 'antd';
import type { FormProps } from 'antd';
import {TypeList} from '@/components/datasource/type';
import {DataSync} from '@/components/datasource/data_sync';
import {updateDatasource} from '@/service/api/data-source'

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const {state: initialDatasource} = useLocation();
  const datasourceID = initialDatasource?.id || '';
  const [loading, setLoading] = useState(false);

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    const sValues = {
      name: values.name,
      type: "connector",
      enabled: values.enabled,
      sync_enabled: values.sync_enabled,
      connector: {
        id: values.connector.id,
        config: {
          ...values.connector.config,
          interval: values.sync_config.interval,
          sync_type: values.sync_config.sync_type || '',
        }
      }
    }
    updateDatasource(datasourceID, sValues).then((res)=>{
      if(res.data?.result == "updated"){
        setLoading(false);
        message.success(t('common.modifySuccess'))
        nav('/data-source/list', {});
      }
    })
  };
  initialDatasource.sync_config = {
    interval: initialDatasource?.connector?.config?.interval,
    sync_type: initialDatasource?.connector?.config?.sync_type || ''
  } 
  const onFinishFailed: FormProps<any>['onFinishFailed'] = (errorInfo) => {
    console.log('Failed:', errorInfo);
    setLoading(false);
  };
  return <div className="bg-white pt-15px pb-15px">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            {t('page.datasource.edit.title')}
          </div>
        </div>
        <div>
         <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={initialDatasource || {}}
            colon={false}
            autoComplete="off"
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item label={t('page.datasource.new.labels.name')} rules={[{ required: true, message: 'Please input datasource name!' }]} name="name">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item rules={[{ required: true, message: 'Please select datasource type!' }]} label={t('page.datasource.new.labels.type')} name="connector">
              <TypeList/>
            </Form.Item>
            <Form.Item label={t('page.datasource.new.labels.data_sync')} name="sync_config">
             <DataSync/>
            </Form.Item>
            <Form.Item label={t('page.datasource.new.labels.sync_enabled')} name="sync_enabled">
              <Switch />
            </Form.Item>
            <Form.Item label=" ">
              <Button type='primary' loading={loading}  htmlType="submit">{t('common.save')}</Button>
              {/* <div className='mt-10px'>
                <Checkbox className='mr-5px' />{t('page.datasource.new.labels.immediate_sync')}
              </div> */}
            </Form.Item>
          </Form>

        </div>
      </div>
  </div>
}