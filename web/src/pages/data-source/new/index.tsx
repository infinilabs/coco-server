import {
  Button,
  Checkbox,
  Form,
  Input,
  message,
  Modal,
  Switch,
  Spin,
} from 'antd';
import type { FormProps } from 'antd';
import {TypeList, Types} from '@/components/datasource/type';
import {DataSync} from '@/components/datasource/data_sync';
import {createDatasource} from '@/service/api/data-source';
import {getConnector} from '@/service/api/connector';
import GoogleDrive from './google_drive';
import Yuque from './yuque';
import Notion from './notion';
import HugoSite from './hugo_site';
import { useForm } from "antd/es/form/Form";


export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const type = params.get('type')??Types.GoogleDrive;
  const [connector, setConnector] = useState<any>({});
  useEffect(() => {
    getConnector(type).then((res)=>{
      if(res.data?.found === true){
        setConnector(res.data._source || {});
      }
    });
  }, [type]);
  const [createState, setCreateState] = useState({
    isModalOpen: true,
    loading: false,
  });
  const [modelForm] = useForm();
  const onModalOkClick = ()=>{
    modelForm.validateFields().then((values)=>{
      setCreateState((old)=>{
        return {
          ...old,
          loading: true,
        }
      });
      createDatasource({
        name: values.name,
        type: "connector",
        enabled: true,
        connector: {
          id: connector.id,
        }
      }).then(({data})=>{
        setCreateState((old)=>{
          nav(`/data-source/edit/${data._id}`, {});
          return {
            ...old,
            loading: false,
          }
        });
      }).catch(()=>{
        setCreateState((old)=>{
          return {
            ...old,
            loading: false,
          }
        });
      });
    })
  };
  let connectorType = 'Google Drive';
  switch (type) {
    case Types.Yuque:
      connectorType = 'Yuque';
      break;
    case Types.Notion:
      connectorType = 'Notion';
      break;
    case Types.HugoSite:
      connectorType = 'Hugo Site';
      break;
    case Types.GoogleDrive:
      break;
    default:
      return (<Modal title={"连接 " + connector.name}
        open={createState.isModalOpen} 
        onCancel={()=>{nav('/data-source/new-first');}}
        okText={t('common.save')}
        onOk={onModalOkClick} >
        <Spin spinning={createState.loading}>
          <Form form={modelForm} layout="vertical" className="my-2em">
            <Form.Item rules={[{required:true}]} label={<span className="text-gray-500">{t('page.apitoken.columns.name')}</span>} name="name">
              <Input/>
            </Form.Item>
          </Form>
        </Spin>
      </Modal>);
  }

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
      enabled: !!values.enabled,
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
  return <div className="bg-white pt-15px pb-15px min-h-full">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>{t('page.datasource.new.title', {
              connector: connectorType,
            })}</div>
          </div>
        </div>
        {type === Types.GoogleDrive ? <GoogleDrive />:
        <div>
         <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={{connector: {id: type, config: {}}, sync_config: {sync_type: "interval", interval: "60s"}, sync_enabled: true, enabled: true}}
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
            <Form.Item label={t('page.datasource.new.labels.enabled')} name="enabled">
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