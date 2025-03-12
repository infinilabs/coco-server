import {
  Button,
  Checkbox,
  Form,
  Input,
  message,
  Switch,
} from 'antd';
import type { FormProps } from 'antd';
import {TypeList, Types} from '@/components/datasource/type';
import {DataSync} from '@/components/datasource/data_sync';
import {createDatasource} from '@/service/api/data-source'
import GoogleDrive from './google_drive';
import Yuque from './yuque';
import Notion from './notion';
import HugoSite from './hugo_site';


export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const type = params.get('type')??Types.GoogleDrive;

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    let config: any = {};
    switch (type) {
      case Types.Yuque:
        config = {
          ...(values.indexing_scope || {}),
          token: values.token || '',
        }
        break;
      case Types.Notion:
        config = {
          token: values.token || '',
        };
        break;
      case Types.HugoSite:
        config = {
          urls: values.urls || [],
        };
        break;
    }
    const sValues = {
      name: values.name,
      type: "connector",
      sync_enabled: values.sync_enabled,
      connector: {
        id: type,
        config: {
          ...config,
          interval: values.sync_config.interval,
          sync_type: values.sync_config.sync_type || '',
        }
      }
    }
    createDatasource(sValues).then((res)=>{
      if(res.data?.result == "created"){
        message.success(t('common.addSuccess'))
        nav('/data-source/list', {});
      }
    })
  };
  
  const onFinishFailed: FormProps<any>['onFinishFailed'] = (errorInfo) => {
    console.log('Failed:', errorInfo);
  };
  let connector = 'Google Drive';
  switch (type) {
    case Types.Yuque:
      connector = 'Yuque';
      break;
    case Types.Notion:
      connector = 'Notion';
      break;
    case Types.HugoSite:
      connector = 'Hugo Site';
      break;
  }
  return <div className="bg-white pt-15px pb-15px min-h-full">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>{t('page.datasource.new.title', {
              connector: connector,
            })}</div>
          </div>
        </div>
        {type === Types.GoogleDrive ? <GoogleDrive />:
        <div>
         <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={{connector: {id: type, config: {}}, sync_config: {sync_type: "interval", interval: "60s"}, sync_enabled: true}}
            colon={false}
            autoComplete="off"
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item label={t('page.datasource.new.labels.name')} rules={[{ required: true, message: 'Please input datasource name!' }]} name="name">
              <Input className='max-w-600px' />
            </Form.Item>
            {type === Types.Yuque && <Yuque />}
            {type === Types.Notion && <Notion />}
            {type === Types.HugoSite && <HugoSite />}
            <Form.Item label={t('page.datasource.new.labels.data_sync')} name="sync_config">
             <DataSync/>
            </Form.Item>
            <Form.Item label={t('page.datasource.new.labels.sync_enabled')} name="sync_enabled">
              <Switch />
            </Form.Item>
            <Form.Item label=" ">
              <Button type='primary'  htmlType="submit">{t('common.save')}</Button>
              {/* <div className='mt-10px'>
                <Checkbox className='mr-5px' />{t('page.datasource.new.labels.immediate_sync')}
              </div> */}
            </Form.Item>
          </Form>

        </div>}
      </div>
  </div>
}