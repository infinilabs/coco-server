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
import {createDatasource} from '@/service/api/data-source'

//gogole_drive
// credential:
//     client_id
//     client_secret
//     redirect_uri
//     endpoint

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    const sValues = {
      name: values.name,
      type: "connector",
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
    createDatasource(sValues).then((res)=>{
      if(res.data?.result == "created"){
        message.success("submitted successfully!")
        nav('/data-source/list', {});
      }
    })
  };
  
  const onFinishFailed: FormProps<any>['onFinishFailed'] = (errorInfo) => {
    console.log('Failed:', errorInfo);
  };
  return <div className="bg-white pt-15px pb-15px">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>{t('page.datasource.new.title')}</div>
          </div>
        </div>
        <div>
         <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={{}}
            onValuesChange={()=>{}}
            colon={false}
            autoComplete="off"
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item label={t('page.datasource.new.labels.name')} rules={[{ required: true, message: 'Please input datasource name!' }]} name="name">
              <Input className='max-w-600px' placeholder='Please input datasource name'/>
            </Form.Item>
            <Form.Item rules={[{ required: true, message: 'Please select datasource type!' }]} label={t('page.datasource.new.labels.type')} name="connector">
              <TypeList/>
            </Form.Item>
            <Form.Item initialValue={{sync_type: "interval", interval: "60s"}} label={t('page.datasource.new.labels.data_sync')} name="sync_config">
             <DataSync/>
            </Form.Item>
            <Form.Item initialValue={true} label={t('page.datasource.new.labels.sync_enabled')} name="sync_enabled">
              <Switch />
            </Form.Item>
            <Form.Item label=" ">
              <Button type='primary'  htmlType="submit">{t('common.save')}</Button>
              {/* <div className='mt-10px'>
                <Checkbox className='mr-5px' />{t('page.datasource.new.labels.immediate_sync')}
              </div> */}
            </Form.Item>
          </Form>

        </div>
      </div>
  </div>
}